package musiclink

// Deezer is Deezer.
const Deezer Platform = "deezer"

func init() {
	register(&rules{
		platform:   Deezer,
		hosts:      []string{"deezer.com"},
		shortHosts: []string{"link.deezer.com"},
		path:       deezerPath,
	})
}

func deezerPath(path string) (Link, error) {
	var buf [maxSegments]string
	segs, ok := split(buf[:0], path, "/")
	if !ok {
		return Link{}, malformed(Deezer, path)
	}
	if len(segs) > 0 && kindNamed(segs[0]) == 0 && isLocale(segs[0]) {
		segs = segs[1:]
	}
	if len(segs) == 0 {
		return Link{}, malformed(Deezer, path)
	}
	kind := kindNamed(segs[0])
	if kind == 0 {
		return Link{}, unsupported(Deezer, path)
	}
	if len(segs) != 2 {
		return Link{}, malformed(Deezer, path)
	}
	if !isDigits(segs[1]) {
		return Link{}, badID(Deezer, kind, segs[1])
	}
	return Link{Platform: Deezer, Kind: kind, ID: segs[1]}, nil
}

// isLocale reports whether s looks like "en" or "pt-br".
func isLocale(s string) bool {
	if len(s) == 5 && s[2] == '-' {
		return isLetters(s[:2]) && isLetters(s[3:])
	}
	return len(s) == 2 && isLetters(s)
}
