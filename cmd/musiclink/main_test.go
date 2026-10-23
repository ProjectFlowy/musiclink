package main

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"
)

func TestReadLinesCanceled(t *testing.T) {
	r, w := io.Pipe()
	defer func() { _ = w.Close() }()
	ctx, cancel := context.WithCancel(t.Context())
	time.AfterFunc(10*time.Millisecond, cancel)
	if _, err := readLines(ctx, r); !errors.Is(err, context.Canceled) {
		t.Fatalf("readLines() error = %v, want context.Canceled", err)
	}
}

func TestParseStdin(t *testing.T) {
	in := strings.NewReader("https://music.yandex.ru/users/music-blog/playlists/2190\n\n  https://www.deezer.com/en/album/983236081  \nhttps://open.spotify.com/show/5CfCWKI5pZ28U0uOzXkDHe\n")
	var out strings.Builder
	c := parseCmd{Output: "pretty", Offline: true, Timeout: time.Second}
	err := c.Run(t.Context(), in, &out)
	if err == nil || err.Error() != "1 of 3 links not parsed" {
		t.Errorf("Run() error = %v", err)
	}
	want := `yandex  playlist  2190  music-blog
deezer  album     983236081
error   musiclink: parse "https://open.spotify.com/show/5CfCWKI5pZ28U0uOzXkDHe": spotify: path "/show/5CfCWKI5pZ28U0uOzXkDHe": unsupported link kind
`
	if out.String() != want {
		t.Errorf("output:\n%s\nwant:\n%s", out.String(), want)
	}
}
