package main

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"strings"

	flags "github.com/jessevdk/go-flags"
)

var Args struct {
	Path    string `short:"p" long:"path" description:"The path to the iTunes library XML export file" required:"yep"`
	OutPath string `short:"o" long:"out" description:"The path to the output playlist XML file" default:"playlists.txt"`
	Debug   bool   `short:"d" long:"debug" description:"Print debug messages"`
	Format  string `short:"f" long:"format" description:"The output format" choice:"csv" choice:"table" default:"table"`
}

func init() {
	if _, err := flags.Parse(&Args); err != nil {
		os.Exit(1)
	}
}

type Dict struct {
	XMLName xml.Name `xml:"dict"`
	KVs     map[string]interface{}
}

type Array struct {
	XMLName xml.Name `xml:"array"`
	Dicts   []Dict   `xml:"dict"`
}

func (di *Dict) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	kvs := make(map[string]interface{})
	// Loop through all the tokens in this element until we find a closing element
	// that matches our start element
	var key string
	for {
		t, err := d.Token()
		if err != nil {
			return err
		}
		switch ty := t.(type) {
		case xml.EndElement:
			if ty.Name.Local == start.Name.Local {
				// We're done
				di.KVs = kvs
				return nil
			}
		case xml.StartElement:
			if ty.Name.Local == "key" {
				// We're parsing a key
				var k string
				if err := d.DecodeElement(&k, &ty); err != nil {
					return err
				}
				key = k
			}
			if ty.Name.Local == "integer" {
				// We're parsing an integer value
				var v int
				if err := d.DecodeElement(&v, &ty); err != nil {
					return err
				}
				kvs[key] = v
			}
			if ty.Name.Local == "string" {
				// We're parsing an integer value
				var v string
				if err := d.DecodeElement(&v, &ty); err != nil {
					return err
				}
				kvs[key] = v
			}
			if ty.Name.Local == "dict" {
				// We're parsing a nested dict value
				var v Dict
				if err := d.DecodeElement(&v, &ty); err != nil {
					return err
				}
				kvs[key] = v
			}
			if ty.Name.Local == "array" {
				var v Array
				if err := d.DecodeElement(&v, &ty); err != nil {
					return err
				}
				kvs[key] = v
			}
		}
	}
}

type ITunesLib struct {
	XMLName xml.Name `xml:"plist"`
	D       Dict     `xml:"dict"`
}

type Track struct {
	Artist string
	Album  string
	Name   string
}

type Playlist struct {
	Name   string
	Tracks []Track
}

type Playlists []Playlist

// WriteCSV writes the set of playlists to the given writer in CSV format.
// It writes a header row and fields: playlist name, artist, album, and track.
// An error is returned if any issues are encountered during this process.
func (ps Playlists) WriteCSV(w io.Writer) error {
	// Write header row
	if _, err := w.Write([]byte("Playlist Name, Artist, Album, Track\n")); err != nil {
		return err
	}
	// Write playlist data
	for _, p := range ps {
		for _, t := range p.Tracks {
			if _, err := w.Write([]byte(fmt.Sprintf("%s,%s,%s,%s\n", p.Name, t.Artist, t.Album, t.Name))); err != nil {
				return err
			}
		}
	}
	return nil
}

// WriteTable writes out the playlists data as a human-readable table.
// The column widths are set to match the widest entry and the columns are
// padded for readability. An error is returned in the event of any processing
// issues.
func (ps Playlists) WriteTable(w io.Writer) error {
	// Loop through playlists once to work out how wide each field needs to be
	// Set baseline widths based on the desired column headers.
	plNameWidth := 13 // 'Playlist Name'
	artistWidth := 6  // 'Artist'
	albumWidth := 5   // 'Album'
	trackWidth := 5   // 'Track'
	for _, p := range ps {
		if len(p.Name) > plNameWidth {
			plNameWidth = len(p.Name)
		}
		for _, t := range p.Tracks {
			if len(t.Artist) > artistWidth {
				artistWidth = len(t.Artist)
			}
			if len(t.Album) > albumWidth {
				albumWidth = len(t.Album)
			}
			if len(t.Name) > trackWidth {
				trackWidth = len(t.Name)
			}
		}
	}
	// Pad the calculated widths by 2 so that the table fields have a space at either end.
	plNameWidth += 2
	artistWidth += 2
	albumWidth += 2
	trackWidth += 2
	// Actually write the table
	buf := bytes.NewBuffer(nil)
	// Write the header row
	colWidths := []int{plNameWidth, artistWidth, albumWidth, trackWidth}
	writeDividerRow := func() {
		for _, cw := range colWidths {
			buf.WriteString("+")
			buf.WriteString(strings.Repeat("-", cw))
		}
		buf.WriteString("+\n")
	}
	writeDividerRow()
	colHeaders := []string{"Playlist Name", "Artist", "Album", "Track"}
	for i := 0; i < 4; i++ {
		buf.WriteString("|")
		n, _ := buf.WriteString(fmt.Sprintf(" %s ", colHeaders[i]))
		// Right-pad with spaces
		if n < colWidths[i] {
			buf.WriteString(strings.Repeat(" ", colWidths[i]-n))
		}
	}
	buf.WriteString("|\n")
	writeDividerRow()
	if _, err := w.Write(buf.Bytes()); err != nil {
		return err
	}
	buf.Reset()
	// Write Platlist data
	for _, p := range ps {
		for _, t := range p.Tracks {
			colItems := []string{p.Name, t.Artist, t.Album, t.Name}
			for i := 0; i < 4; i++ {
				buf.WriteString("|")
				n, _ := buf.WriteString(fmt.Sprintf(" %s ", colItems[i]))
				// Right-pad with spaces
				if n < colWidths[i] {
					buf.WriteString(strings.Repeat(" ", colWidths[i]-n))
				}
			}
			buf.WriteString("|\n")
		}
		// Finish section
		writeDividerRow()
		if _, err := w.Write(buf.Bytes()); err != nil {
			return err
		}
		buf.Reset()
	}

	return nil
}

func StringOrDefault(val interface{}, alt string) string {
	s, ok := val.(string)
	if !ok {
		return alt
	}
	return s
}

func PrintMsg(msg string) {
	if Args.Debug {
		fmt.Println(msg)
	}
}

func main() {
	itunesBytes, err := ioutil.ReadFile(Args.Path)
	if err != nil {
		log.Fatalf("Failed to load iTunes library file: '%s'\n", err.Error())
	}

	var i ITunesLib
	if err := xml.Unmarshal(itunesBytes, &i); err != nil {
		log.Fatalf("Failed to parse iTunes library file: %s", err.Error())
	}

	// Extract the tracks as a helpful object
	rawTracks := i.D.KVs["Tracks"].(Dict)
	tracks := make(map[string]Track)
	for trackID, trackDict := range rawTracks.KVs {
		var t Track
		td := trackDict.(Dict)
		t.Artist = StringOrDefault(td.KVs["Artist"], "Unknown Artist")
		t.Album = StringOrDefault(td.KVs["Album"], "Unknown Album")
		t.Name = StringOrDefault(td.KVs["Name"], "Unknown Name")
		tracks[trackID] = t
	}
	PrintMsg(fmt.Sprintf("Library contains %d tracks", len(tracks)))

	rawPlaylists := i.D.KVs["Playlists"].(Array)
	PrintMsg(fmt.Sprintf("Library contains %d playlists", len(rawPlaylists.Dicts)))

	// Convert the playlists into something useful
	// Start at index 5 to lose the enormous default 'Library',
	// 'Downloaded', 'Music', 'Podcasts' and 'Albums' playlists.
	var playlists Playlists
	for _, d := range rawPlaylists.Dicts[5:] {
		var p Playlist
		p.Name = StringOrDefault(d.KVs["Name"], "Unknown Playlist")
		pTracks, ok := d.KVs["Playlist Items"].(Array)
		if !ok {
			PrintMsg(fmt.Sprintf("Error: Playlist %s has no tracks", p.Name))
			continue
		}
		for _, t := range pTracks.Dicts {
			trackID := t.KVs["Track ID"].(int)
			tk := tracks[strconv.Itoa(trackID)]
			p.Tracks = append(p.Tracks, tk)
		}
		playlists = append(playlists, p)
	}

	PrintMsg(fmt.Sprintf("Parsed %d playlists successfully", len(playlists)))

	// Output the playlists helpfully
	f, _ := os.Create(Args.OutPath)
	defer f.Close()
	if Args.Format == "csv" {
		if err := playlists.WriteCSV(f); err != nil {
			log.Fatalf("Failed to write playlist csv to file %s: %s", Args.OutPath, err.Error())
		}
	} else if Args.Format == "table" {
		if err := playlists.WriteTable(f); err != nil {
			log.Fatalf("Failed to write playlist table to file %s: %s", Args.OutPath, err.Error())
		}
	}
	PrintMsg(fmt.Sprintf("Successfully wrote playlists to %s", Args.OutPath))
}
