package musiclink

import "uuid"

// Yandex is Yandex Music.
const Yandex Platform = "yandex"

func init() {
	register(&rules{
		platform: Yandex,
		hosts:    []string{"music.yandex.ru", "music.yandex.com", "music.yandex.by", "music.yandex.kz", "music.yandex.uz"},
		path:     yandexPath,
	})
}

func yandexPath(path string) (Link, error) {
	var buf [maxSegments]string
	segs, ok := split(buf[:0], path, "/")
	if !ok || len(segs) == 0 {
		return Link{}, malformed(Yandex, path)
	}
	switch segs[0] {
	case "artist", "album", "track":
		kind, id := kindNamed(segs[0]), ""
		switch {
		case len(segs) == 2:
			id = segs[1]
		case len(segs) == 4 && kind == Album && segs[2] == "track":
			if !isDigits(segs[1]) {
				return Link{}, badID(Yandex, Album, segs[1])
			}
			kind, id = Track, segs[3]
		default:
			return Link{}, malformed(Yandex, path)
		}
		if !isDigits(id) {
			return Link{}, badID(Yandex, kind, id)
		}
		return Link{Platform: Yandex, Kind: kind, ID: id}, nil
	case "playlists":
		if len(segs) != 2 {
			return Link{}, malformed(Yandex, path)
		}
		if _, err := uuid.Parse(segs[1]); err != nil || len(segs[1]) != 36 {
			return Link{}, badID(Yandex, Playlist, segs[1])
		}
		return Link{Platform: Yandex, Kind: Playlist, ID: segs[1]}, nil
	case "users":
		switch {
		case len(segs) == 4 && segs[2] == "playlists":
			if !isDigits(segs[3]) {
				return Link{}, badID(Yandex, Playlist, segs[3])
			}
			return Link{Platform: Yandex, Kind: Playlist, ID: segs[3], Owner: segs[1]}, nil
		case len(segs) == 1:
			return Link{}, malformed(Yandex, path)
		}
	}
	return Link{}, unsupported(Yandex, path)
}
