package musiclink_test

import (
	"testing"

	"github.com/projectflowy/musiclink"
)

func TestYandex(t *testing.T) {
	testParse(t, []parseCase{
		{url: "https://music.yandex.ru/artist/15691816", want: yandex(musiclink.Artist, "15691816")},
		{url: "https://music.yandex.ru/album/22820453", want: yandex(musiclink.Album, "22820453")},
		{url: "https://music.yandex.ru/album/22820453/track/105406141", want: yandex(musiclink.Track, "105406141")},
		{url: "https://music.yandex.ru/track/105406141", want: yandex(musiclink.Track, "105406141")},
		{url: "https://music.yandex.ru/album/39203154", want: yandex(musiclink.Album, "39203154")},
		{url: "https://music.yandex.ru/album/7653771/track/2219127?utm_medium=copy_link&ref_id=4f797a99-8924-4d0e-95f9-0c8ca5e6295d", want: yandex(musiclink.Track, "2219127")},
		{url: "https://music.yandex.ru/album/7653771?utm_medium=copy_link&ref_id=312253d8-38d0-4636-afb7-254dd0fb3a1b", want: yandex(musiclink.Album, "7653771")},
		{url: "https://music.yandex.ru/artist/3307223?utm_medium=copy_link&ref_id=051833d5-237d-4646-98c7-ff31f593ed43", want: yandex(musiclink.Artist, "3307223")},
		{url: "https://music.yandex.ru/album/7650957/track/53502746?utm_medium=copy_link&ref_id=70d96606-7f39-4069-a68e-1cdb079dcacc", want: yandex(musiclink.Track, "53502746")},
		{url: "https://music.yandex.ru/artist/15691816?utm_source=desktop&utm_medium=copy_link", want: yandex(musiclink.Artist, "15691816")},
		{url: "https://music.yandex.ru/album/27558714/track/115517682?utm_source=desktop&utm_medium=copy_link", want: yandex(musiclink.Track, "115517682")},
		{url: "https://music.yandex.ru/album/22820453?utm_source=desktop&utm_medium=copy_link", want: yandex(musiclink.Album, "22820453")},
		{url: "https://music.yandex.ru/album/27558714/track/117786525?utm_source=desktop&utm_medium=copy_link", want: yandex(musiclink.Track, "117786525")},
		{url: "https://music.yandex.ru/playlists/dccba095-a4b3-bd5e-8e65-b734843f2483", want: yandex(musiclink.Playlist, "dccba095-a4b3-bd5e-8e65-b734843f2483")},
		{url: "https://music.yandex.ru/users/music-blog/playlists/2190", want: musiclink.Link{Platform: musiclink.Yandex, Kind: musiclink.Playlist, ID: "2190", Owner: "music-blog"}},
		{url: "https://music.yandex.com/album/1/", want: yandex(musiclink.Album, "1")},
		{url: "https://MUSIC.Yandex.KZ:443/track/1#top", want: yandex(musiclink.Track, "1")},
		{url: "http://www.music.yandex.by/artist/1", want: yandex(musiclink.Artist, "1")},
		{url: "  https://music.yandex.uz/track/1\n", want: yandex(musiclink.Track, "1")},
		{url: "https://music.yandex.ru/users/a%2Eb/playlists/3", want: musiclink.Link{Platform: musiclink.Yandex, Kind: musiclink.Playlist, ID: "3", Owner: "a.b"}},
		{url: "https://music.yandex.ru/album/abc", err: musiclink.ErrMalformed},
		{url: "https://music.yandex.ru/album/", err: musiclink.ErrMalformed},
		{url: "https://music.yandex.ru/", err: musiclink.ErrMalformed},
		{url: "https://music.yandex.ru/album/1/track/x", err: musiclink.ErrMalformed},
		{url: "https://music.yandex.ru/album/x/track/1", err: musiclink.ErrMalformed},
		{url: "https://music.yandex.ru/album/1/extra", err: musiclink.ErrMalformed},
		{url: "https://music.yandex.ru/users/x/playlists/abc", err: musiclink.ErrMalformed},
		{url: "https://music.yandex.ru/playlists/2190", err: musiclink.ErrMalformed},
		{url: "https://music.yandex.ru/playlists/{dccba095-a4b3-bd5e-8e65-b734843f2483}", err: musiclink.ErrMalformed},
		{url: "https://music.yandex.ru/track/%zz", err: musiclink.ErrMalformed},
		{url: "https://music.yandex.ru/a/b/c/d/e/f/g/h/i", err: musiclink.ErrMalformed},
		{url: "https://music.yandex.ru/users/music-blog", err: musiclink.ErrUnsupported},
		{url: "https://music.yandex.ru/users/music-blog/playlists", err: musiclink.ErrUnsupported},
		{url: "https://music.yandex.ru/podcasts", err: musiclink.ErrUnsupported},
		{url: "https://music.yandex.ru/video/123", err: musiclink.ErrUnsupported},
	})
}

func yandex(k musiclink.Kind, id string) musiclink.Link {
	return musiclink.Link{Platform: musiclink.Yandex, Kind: k, ID: id}
}
