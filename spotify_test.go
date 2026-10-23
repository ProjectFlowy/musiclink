package musiclink_test

import (
	"testing"

	"github.com/projectflowy/musiclink"
)

func TestSpotify(t *testing.T) {
	testParse(t, []parseCase{
		{url: "https://open.spotify.com/track/0RmHZOnMjaCM6mIvLmvxFr?si=698f3fa479e84dea", want: spotify(musiclink.Track, "0RmHZOnMjaCM6mIvLmvxFr")},
		{url: "https://open.spotify.com/artist/3jayBSUIK2eqnLCWfJoWwV?si=a917a121e648490a", want: spotify(musiclink.Artist, "3jayBSUIK2eqnLCWfJoWwV")},
		{url: "https://open.spotify.com/track/0RmHZOnMjaCM6mIvLmvxFr?si=92571a9738cc43a6", want: spotify(musiclink.Track, "0RmHZOnMjaCM6mIvLmvxFr")},
		{url: "https://open.spotify.com/album/17a2qUMfTTjOAUPCw6n8Op?si=ZjoYAm10RD-F1-caddeoMg", want: spotify(musiclink.Album, "17a2qUMfTTjOAUPCw6n8Op")},
		{url: "https://open.spotify.com/album/17a2qUMfTTjOAUPCw6n8Op", want: spotify(musiclink.Album, "17a2qUMfTTjOAUPCw6n8Op")},
		{url: "https://open.spotify.com/album/5vAZM4RAN6Y6fKmK828J7d", want: spotify(musiclink.Album, "5vAZM4RAN6Y6fKmK828J7d")},
		{url: "https://open.spotify.com/playlist/0QaFCjYt9ww7ZR7ip7Eshq?si=805c4fbcab2d40c9", want: spotify(musiclink.Playlist, "0QaFCjYt9ww7ZR7ip7Eshq")},
		{url: "https://open.spotify.com/intl-de/track/0RmHZOnMjaCM6mIvLmvxFr", want: spotify(musiclink.Track, "0RmHZOnMjaCM6mIvLmvxFr")},
		{url: "https://open.spotify.com/user/spotify/playlist/0QaFCjYt9ww7ZR7ip7Eshq", want: spotify(musiclink.Playlist, "0QaFCjYt9ww7ZR7ip7Eshq")},
		{url: "spotify:track:0RmHZOnMjaCM6mIvLmvxFr", want: spotify(musiclink.Track, "0RmHZOnMjaCM6mIvLmvxFr")},
		{url: "spotify:artist:3jayBSUIK2eqnLCWfJoWwV", want: spotify(musiclink.Artist, "3jayBSUIK2eqnLCWfJoWwV")},
		{url: "spotify:playlist:0QaFCjYt9ww7ZR7ip7Eshq", want: spotify(musiclink.Playlist, "0QaFCjYt9ww7ZR7ip7Eshq")},
		{url: "spotify:user:spotify:playlist:0QaFCjYt9ww7ZR7ip7Eshq", want: spotify(musiclink.Playlist, "0QaFCjYt9ww7ZR7ip7Eshq")},
		{url: "https://open.spotify.com/show/5CfCWKI5pZ28U0uOzXkDHe", err: musiclink.ErrUnsupported},
		{url: "https://open.spotify.com/episode/5CfCWKI5pZ28U0uOzXkDHe", err: musiclink.ErrUnsupported},
		{url: "https://open.spotify.com/user/spotify", err: musiclink.ErrUnsupported},
		{url: "spotify:show:5CfCWKI5pZ28U0uOzXkDHe", err: musiclink.ErrUnsupported},
		{url: "spotify:user:spotify", err: musiclink.ErrUnsupported},
		{url: "https://open.spotify.com/track/0RmHZOnMjaCM6mIvLmvxF", err: musiclink.ErrMalformed},
		{url: "https://open.spotify.com/track/0RmHZOnMjaCM6mIvLmvxFrr", err: musiclink.ErrMalformed},
		{url: "https://open.spotify.com/track/0RmHZOnMjaCM6mIvLmvx-r", err: musiclink.ErrMalformed},
		{url: "https://open.spotify.com/track", err: musiclink.ErrMalformed},
		{url: "https://open.spotify.com/", err: musiclink.ErrMalformed},
		{url: "https://open.spotify.com/user", err: musiclink.ErrMalformed},
		{url: "spotify:track", err: musiclink.ErrMalformed},
		{url: "https://spoti.fi/3xYzAbC", err: musiclink.ErrNetworkDisabled},
	})
}

func spotify(k musiclink.Kind, id string) musiclink.Link {
	return musiclink.Link{Platform: musiclink.Spotify, Kind: k, ID: id}
}
