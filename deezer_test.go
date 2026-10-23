package musiclink_test

import (
	"testing"

	"github.com/projectflowy/musiclink"
)

func TestDeezer(t *testing.T) {
	testParse(t, []parseCase{
		{url: "https://www.deezer.com/en/artist/246791", want: deezer(musiclink.Artist, "246791")},
		{url: "https://www.deezer.com/en/playlist/1996494362", want: deezer(musiclink.Playlist, "1996494362")},
		{url: "https://www.deezer.com/en/album/983236081", want: deezer(musiclink.Album, "983236081")},
		{url: "https://www.deezer.com/en/track/4022917741?host=3873397062&utm_campaign=clipboard-generic&utm_source=user_sharing&utm_content=track-4022917741&deferredFl=1&universal_link=1", want: deezer(musiclink.Track, "4022917741")},
		{url: "https://www.deezer.com/track/4022917741?host=3873397062", want: deezer(musiclink.Track, "4022917741")},
		{url: "https://deezer.com/pt-br/album/1", want: deezer(musiclink.Album, "1")},
		{url: "https://www.deezer.com/en/show/123", err: musiclink.ErrUnsupported},
		{url: "https://www.deezer.com/en/profile/123", err: musiclink.ErrUnsupported},
		{url: "https://www.deezer.com/en/track/abc", err: musiclink.ErrMalformed},
		{url: "https://www.deezer.com/en/", err: musiclink.ErrMalformed},
		{url: "https://link.deezer.com/s/34KSFEBUZ020rDp0QZ2Za", err: musiclink.ErrNetworkDisabled},
	})
}

func deezer(k musiclink.Kind, id string) musiclink.Link {
	return musiclink.Link{Platform: musiclink.Deezer, Kind: k, ID: id}
}
