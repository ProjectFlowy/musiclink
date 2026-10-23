package musiclink

import "strings"

// Spotify is Spotify.
const Spotify Platform = "spotify"

func init() {
	register(&rules{
		platform:   Spotify,
		hosts:      []string{"open.spotify.com"},
		shortHosts: []string{"spotify.link", "spoti.fi"},
		path:       spotifyPath,
		scheme:     "spotify",
		uri:        spotifyURI,
	})
}

func spotifyPath(path string) (Link, error) {
	var buf [maxSegments]string
	segs, ok := split(buf[:0], path, "/")
	if !ok {
		return Link{}, malformed(Spotify, path)
	}
	if len(segs) > 0 && strings.HasPrefix(segs[0], "intl-") {
		segs = segs[1:]
	}
	return spotifySegments(path, segs)
}

// spotifyURI parses the part of a Spotify URI after "spotify:".
func spotifyURI(s string) (Link, error) {
	var buf [maxSegments]string
	segs, ok := split(buf[:0], s, ":")
	if !ok {
		return Link{}, malformed(Spotify, s)
	}
	return spotifySegments(s, segs)
}

// spotifySegments parses Spotify path or URI segments.
func spotifySegments(path string, segs []string) (Link, error) {
	if len(segs) >= 2 && segs[0] == "user" {
		if len(segs) != 4 || segs[2] != "playlist" {
			return Link{}, unsupported(Spotify, path)
		}
		// Spotify playlist IDs are global, so the owner is not needed.
		segs = segs[2:]
	}
	if len(segs) == 0 || segs[0] == "user" {
		return Link{}, malformed(Spotify, path)
	}
	kind := kindNamed(segs[0])
	if kind == 0 {
		return Link{}, unsupported(Spotify, path)
	}
	if len(segs) != 2 {
		return Link{}, malformed(Spotify, path)
	}
	if len(segs[1]) != 22 || !isBase62(segs[1]) {
		return Link{}, badID(Spotify, kind, segs[1])
	}
	return Link{Platform: Spotify, Kind: kind, ID: segs[1]}, nil
}
