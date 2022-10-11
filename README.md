# Tool to extract playlists from iTunes Library XML

An extremely quick and dirty tool to extract the artist, album, and
track names for each playlist in an iTunes library XML. It is
effectively a single use tool so definitely doesn't represent best
practice but it does demonstrate a way of doing custom unmarshalling
of complex XML structures with Go.

## File Structure

The iTunes XML files have a strange structure which is a series of
nested dictionaries formed of
`<key>my-key</key><type>my-value</type>` pairs. The possible
types for the dict values (at least the ones present in the XML dumps
I had access to) are `<string>`, `<integer>`, `<dict>`, and `<array>`
where the array type is an array of dicts. Below is a simplified
version of the file format.

```
<plist>
    <dict>
        <!-- Key value pairs related to global library settings -->
        <!-- XML Dictionary of NumericalTrackID: track dict pairs -->
        <key>Tracks</tracks><dict>
            <key>123</key><dict>
                <key>Name</key><string>never gonna give you up</string>
                <key>Album</key><string>Whenever You Need Somebody</string>
                <key>Artist</key><string>Rick Astley</string>
            </dict>
            <!-- Repeated NumericalTrackID: dict pairs -->
        </dict>
        <!-- Playlists: XML array of playlist dicts -->
        <key>Playlists</key><array>
            <dict>
                <key>Name</key><string>Playlist Name</string>
                <!-- Other playlist metadata that we don't care about -->
                <key>Playlist Items</key><array>
                    <dict>
                        <key>Track ID</key><integer>123</integer>
                    </dict>
                    <dict>
                        <key>Track ID</key><integer>456</integer>
                    </dict>
                </array>
            </dict>
            <!-- Repeated playlist dict elements -->
        </array>
    </dict>
</plist>
```

## Building and running the code

The tool is a Go module and can be built as follows:

```
go build -o "ixpe" github.com/will-dee/itunes-xml-playlist-extract
```

The tool has a help screen which explains the args to run it:

```
./ixpe --help
Usage:
  ixpe [OPTIONS]

Application Options:
  -p, --path=  The path to the iTunes library XML export file
  -o, --out=   The path to the output playlist XML file (default: playlists.txt)
  -d, --debug  Print debug messages
  -f, --format=[csv|table] The output format (default: table)

Help Options:
  -h, --help   Show this help message
```

A placeholder XML library file (`itunes.xml`) is included for the
purposes of testing and playing (without revealing any questionable
music tastes to the world).

```
./ixpe -p ./itunes.xml
cat playlists.txt
+-------------------+-------------+----------------------------+-------------------------+
| Playlist Name     | Artist      | Album                      | Track                   |
+-------------------+-------------+----------------------------+-------------------------+
| My Playlist       | Rick Astley | Whenever You Need Somebody | Never Gonna Give You Up |
| My Playlist       | OK Go       | Oh No                      | Here It Goes Again      |
+-------------------+-------------+----------------------------+-------------------------+
| My Other Playlist | Smash Mouth | Astro Lounge               | All Star                |
| My Other Playlist | Darude      | Before The Storm           | Sandstorm               |
+-------------------+-------------+----------------------------+-------------------------+
```