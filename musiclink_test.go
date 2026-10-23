package musiclink_test

import (
	"errors"
	"testing"

	"github.com/projectflowy/musiclink"
)

type parseCase struct {
	url  string
	want musiclink.Link
	err  error
}

// testParse parses each case offline and checks the link or the error.
func testParse(t *testing.T, cases []parseCase) {
	t.Helper()
	p := musiclink.New(musiclink.WithoutNetwork())
	for _, c := range cases {
		t.Run(c.url, func(t *testing.T) {
			got, err := p.Parse(t.Context(), c.url)
			if c.err != nil {
				if !errors.Is(err, c.err) {
					t.Fatalf("Parse() = %+v, %v; want error %v", got, err, c.err)
				}
				return
			}
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}
			if got != c.want {
				t.Errorf("Parse() = %+v, want %+v", got, c.want)
			}
		})
	}
}

func TestParseInvalid(t *testing.T) {
	testParse(t, []parseCase{
		{url: "", err: musiclink.ErrInvalidURL},
		{url: "garbage", err: musiclink.ErrInvalidURL},
		{url: "ftp://music.yandex.ru/album/1", err: musiclink.ErrInvalidURL},
		{url: "music.yandex.ru/album/1", err: musiclink.ErrInvalidURL},
		{url: "https:music.yandex.ru/album/1", err: musiclink.ErrInvalidURL},
		{url: "https:///album/1", err: musiclink.ErrInvalidURL},
		{url: "https://music.yandex.ru:http/album/1", err: musiclink.ErrInvalidURL},
		{url: "https://example.com/album/1", err: musiclink.ErrUnknownHost},
		{url: "https://music.yandex.ru@example.com/album/1", err: musiclink.ErrUnknownHost},
		{url: "https://music.yandex.ru.example.com/album/1", err: musiclink.ErrUnknownHost},
	})
}

func TestPlatformOf(t *testing.T) {
	tests := []struct {
		host string
		want musiclink.Platform
		ok   bool
	}{
		{"music.yandex.ru", musiclink.Yandex, true},
		{"WWW.Music.Yandex.Com:8080", musiclink.Yandex, true},
		{"open.spotify.com", musiclink.Spotify, true},
		{"spoti.fi", musiclink.Spotify, true},
		{"www.deezer.com", musiclink.Deezer, true},
		{"link.deezer.com", musiclink.Deezer, true},
		{"example.com", "", false},
		{"", "", false},
	}
	for _, tt := range tests {
		got, ok := musiclink.PlatformOf(tt.host)
		if got != tt.want || ok != tt.ok {
			t.Errorf("PlatformOf(%q) = %q, %v; want %q, %v", tt.host, got, ok, tt.want, tt.ok)
		}
	}
}

func TestIsShort(t *testing.T) {
	tests := []struct {
		url  string
		want bool
	}{
		{"https://spotify.link/ZbZ3c8F2gzb", true},
		{"https://spoti.fi/3xYzAbC?x=1", true},
		{"https://link.deezer.com/s/34KSFEBUZ020rDp0QZ2Za", true},
		{"https://open.spotify.com/track/0RmHZOnMjaCM6mIvLmvxFr", false},
		{"spotify:track:0RmHZOnMjaCM6mIvLmvxFr", false},
		{"https://example.com/x", false},
		{"garbage", false},
	}
	for _, tt := range tests {
		if got := musiclink.IsShort(tt.url); got != tt.want {
			t.Errorf("IsShort(%q) = %v, want %v", tt.url, got, tt.want)
		}
	}
}

func TestKindString(t *testing.T) {
	for k, want := range map[musiclink.Kind]string{
		musiclink.Track:    "track",
		musiclink.Album:    "album",
		musiclink.Artist:   "artist",
		musiclink.Playlist: "playlist",
		0:                  "Kind(0)",
	} {
		if got := k.String(); got != want {
			t.Errorf("Kind(%d).String() = %q, want %q", k, got, want)
		}
	}
}

func FuzzParse(f *testing.F) {
	for _, s := range []string{
		"https://music.yandex.ru/album/1/track/2",
		"https://music.yandex.ru/users/a/playlists/1",
		"https://open.spotify.com/intl-de/track/0RmHZOnMjaCM6mIvLmvxFr",
		"spotify:user:u:playlist:0QaFCjYt9ww7ZR7ip7Eshq",
		"https://www.deezer.com/en/track/1",
		"https://spoti.fi/x",
		"http://a@b:1/%zz",
	} {
		f.Add(s)
	}
	p := musiclink.New(musiclink.WithoutNetwork())
	f.Fuzz(func(t *testing.T, s string) {
		link, err := p.Parse(t.Context(), s)
		if err == nil && (link.Platform == "" || link.Kind == 0 || link.ID == "") {
			t.Errorf("Parse(%q) = %+v without error", s, link)
		}
	})
}
