package musiclink_test

import (
	"fmt"
	"net/http"
	"net/url"
	"testing"

	"github.com/projectflowy/musiclink"
)

var benchLinks = []struct{ name, url string }{
	{"yandex/track", "https://music.yandex.ru/album/7653771/track/2219127?utm_medium=copy_link&ref_id=4f797a99-8924-4d0e-95f9-0c8ca5e6295d"},
	{"yandex/playlist-uuid", "https://music.yandex.ru/playlists/dccba095-a4b3-bd5e-8e65-b734843f2483"},
	{"yandex/user-playlist", "https://music.yandex.ru/users/music-blog/playlists/2190"},
	{"spotify/track", "https://open.spotify.com/track/0RmHZOnMjaCM6mIvLmvxFr?si=698f3fa479e84dea"},
	{"spotify/intl", "https://open.spotify.com/intl-de/album/17a2qUMfTTjOAUPCw6n8Op"},
	{"spotify/uri", "spotify:user:spotify:playlist:0QaFCjYt9ww7ZR7ip7Eshq"},
	{"deezer/track", "https://www.deezer.com/en/track/4022917741?host=3873397062&utm_campaign=clipboard-generic&utm_source=user_sharing"},
	{"upper-case-host", "https://MUSIC.YANDEX.RU/album/22820453"},
	{"err/unknown-host", "https://example.com/track/1"},
	{"err/malformed", "https://music.yandex.ru/album/abc"},
	{"err/unsupported", "https://open.spotify.com/show/5CfCWKI5pZ28U0uOzXkDHe"},
}

func BenchmarkParse(b *testing.B) {
	p := musiclink.New()
	for _, l := range benchLinks {
		b.Run(l.name, func(b *testing.B) {
			ctx := b.Context()
			b.ReportAllocs()
			for b.Loop() {
				_, _ = p.Parse(ctx, l.url)
			}
		})
	}
}

// BenchmarkURLParse is a baseline: net/url.Parse alone, before any rules.
func BenchmarkURLParse(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		_, _ = url.Parse(benchLinks[0].url)
	}
}

func BenchmarkParseBatch(b *testing.B) {
	for _, n := range []int{100, 10000} {
		urls := make([]string, n)
		for i := range urls {
			urls[i] = fmt.Sprintf("https://music.yandex.ru/album/%d/track/%d", i, i)
		}
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			p := musiclink.New()
			ctx := b.Context()
			b.ReportAllocs()
			for b.Loop() {
				p.ParseBatch(ctx, urls)
			}
			b.ReportMetric(float64(b.Elapsed().Nanoseconds())/float64(b.N*n), "ns/link")
		})
	}
}

// BenchmarkParseShort expands a short link through an instant in-memory
// transport: the cost of the HTTP client machinery without the network.
func BenchmarkParseShort(b *testing.B) {
	f := &fakeTransport{routes: map[string]string{shortTrack: spotifyTrack}}
	p := musiclink.New(musiclink.WithHTTPClient(&http.Client{Transport: f}))
	ctx := b.Context()
	b.ReportAllocs()
	for b.Loop() {
		if _, err := p.Parse(ctx, shortTrack); err != nil {
			b.Fatal(err)
		}
	}
}
