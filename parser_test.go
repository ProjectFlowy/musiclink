package musiclink_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/projectflowy/musiclink"
)

const (
	spotifyTrack = "https://open.spotify.com/track/0RmHZOnMjaCM6mIvLmvxFr"
	shortTrack   = "https://spoti.fi/track"
)

// fakeTransport answers with a redirect to routes[url], or 200 when
// the URL has no route. It never touches the network.
type fakeTransport struct {
	routes map[string]string
	delay  time.Duration
	// block makes requests wait for their context to end.
	block bool

	mu       sync.Mutex
	calls    map[string]int
	bodies   []*body
	agents   []string
	inflight atomic.Int32
	peak     atomic.Int32
	started  chan struct{}
}

func (f *fakeTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	n := f.inflight.Add(1)
	defer f.inflight.Add(-1)
	for {
		peak := f.peak.Load()
		if n <= peak || f.peak.CompareAndSwap(peak, n) {
			break
		}
	}

	url := req.URL.String()
	b := &body{Reader: strings.NewReader("<html>redirecting</html>")}
	f.mu.Lock()
	if f.calls == nil {
		f.calls = map[string]int{}
	}
	f.calls[url]++
	f.bodies = append(f.bodies, b)
	f.agents = append(f.agents, req.Header.Get("User-Agent"))
	f.mu.Unlock()
	if f.started != nil {
		select {
		case f.started <- struct{}{}:
		default:
		}
	}

	if f.block {
		<-req.Context().Done()
		return nil, req.Context().Err()
	}
	if f.delay > 0 {
		time.Sleep(f.delay)
	}

	resp := &http.Response{StatusCode: http.StatusOK, Header: http.Header{}, Body: b, Request: req}
	if loc, ok := f.routes[url]; ok {
		resp.StatusCode = http.StatusTemporaryRedirect
		resp.Header.Set("Location", loc)
	}
	return resp, nil
}

func (f *fakeTransport) callCount(url string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.calls[url]
}

type body struct {
	*strings.Reader
	read, closed atomic.Bool
}

func (b *body) Read(p []byte) (int, error) {
	b.read.Store(true)
	return b.Reader.Read(p)
}

func (b *body) Close() error {
	b.closed.Store(true)
	return nil
}

func newParser(f *fakeTransport, opts ...musiclink.Option) *musiclink.Parser {
	return musiclink.New(append([]musiclink.Option{musiclink.WithHTTPClient(&http.Client{Transport: f})}, opts...)...)
}

func TestParseShort(t *testing.T) {
	tests := []struct {
		name   string
		url    string
		routes map[string]string
		want   musiclink.Link
	}{
		{
			name:   "redirect",
			url:    shortTrack,
			routes: map[string]string{shortTrack: spotifyTrack + "?si=1"},
			want:   spotify(musiclink.Track, "0RmHZOnMjaCM6mIvLmvxFr"),
		},
		{
			name: "chain",
			url:  "https://spotify.link/a",
			routes: map[string]string{
				"https://spotify.link/a": "https://spoti.fi/b",
				"https://spoti.fi/b":     spotifyTrack,
			},
			want: spotify(musiclink.Track, "0RmHZOnMjaCM6mIvLmvxFr"),
		},
		{
			name:   "relative location",
			url:    "https://spoti.fi/a",
			routes: map[string]string{"https://spoti.fi/a": "/b", "https://spoti.fi/b": spotifyTrack},
			want:   spotify(musiclink.Track, "0RmHZOnMjaCM6mIvLmvxFr"),
		},
		{
			name:   "spotify uri",
			url:    "https://spoti.fi/a",
			routes: map[string]string{"https://spoti.fi/a": "spotify:album:17a2qUMfTTjOAUPCw6n8Op"},
			want:   spotify(musiclink.Album, "17a2qUMfTTjOAUPCw6n8Op"),
		},
		{
			name: "deezer",
			url:  "https://link.deezer.com/s/34KSFEBUZ020rDp0QZ2Za",
			routes: map[string]string{
				"https://link.deezer.com/s/34KSFEBUZ020rDp0QZ2Za":                                                     "https://link.deezer.com/?dest=https%3A%2F%2Fwww.deezer.com%2Ftrack%2F4022917741%3Fhost%3D3873397062",
				"https://link.deezer.com/?dest=https%3A%2F%2Fwww.deezer.com%2Ftrack%2F4022917741%3Fhost%3D3873397062": "https://www.deezer.com/track/4022917741?host=3873397062",
			},
			want: deezer(musiclink.Track, "4022917741"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &fakeTransport{routes: tt.routes}
			got, err := newParser(f).Parse(t.Context(), tt.url)
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}
			if got != tt.want {
				t.Errorf("Parse() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestParseShortErrors(t *testing.T) {
	tests := []struct {
		name   string
		routes map[string]string
		want   error
	}{
		{"not a redirect", nil, musiclink.ErrNotRedirect},
		{"loop", map[string]string{shortTrack: "https://spoti.fi/b", "https://spoti.fi/b": shortTrack}, musiclink.ErrTooManyRedirects},
		{"unknown host", map[string]string{shortTrack: "https://example.com/track/1"}, musiclink.ErrUnknownHost},
		{"unsupported", map[string]string{shortTrack: "https://open.spotify.com/show/5CfCWKI5pZ28U0uOzXkDHe"}, musiclink.ErrUnsupported},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &fakeTransport{routes: tt.routes}
			_, err := newParser(f).Parse(t.Context(), shortTrack)
			if !errors.Is(err, tt.want) {
				t.Fatalf("Parse() error = %v, want %v", err, tt.want)
			}
		})
	}
}

func TestParseRedirectLimit(t *testing.T) {
	f := &fakeTransport{routes: map[string]string{shortTrack: "https://spoti.fi/b", "https://spoti.fi/b": shortTrack}}
	_, err := newParser(f, musiclink.WithMaxRedirects(3)).Parse(t.Context(), shortTrack)
	if !errors.Is(err, musiclink.ErrTooManyRedirects) {
		t.Fatalf("Parse() error = %v, want ErrTooManyRedirects", err)
	}
	if got := f.callCount(shortTrack) + f.callCount("https://spoti.fi/b"); got != 3 {
		t.Errorf("requests = %d, want 3", got)
	}
}

func TestParseBodyNotRead(t *testing.T) {
	f := &fakeTransport{routes: map[string]string{shortTrack: spotifyTrack}}
	if _, err := newParser(f).Parse(t.Context(), shortTrack); err != nil {
		t.Fatal(err)
	}
	if _, err := newParser(&fakeTransport{}).Parse(t.Context(), shortTrack); !errors.Is(err, musiclink.ErrNotRedirect) {
		t.Fatal(err)
	}
	for _, b := range f.bodies {
		if b.read.Load() || !b.closed.Load() {
			t.Errorf("body read = %v, closed = %v; want unread and closed", b.read.Load(), b.closed.Load())
		}
	}
}

func TestParseUserAgent(t *testing.T) {
	f := &fakeTransport{routes: map[string]string{shortTrack: spotifyTrack}}
	if _, err := newParser(f, musiclink.WithUserAgent("test/1")).Parse(t.Context(), shortTrack); err != nil {
		t.Fatal(err)
	}
	if f.agents[0] != "test/1" {
		t.Errorf("User-Agent = %q, want test/1", f.agents[0])
	}
}

func TestWithHTTPClientNotModified(t *testing.T) {
	c := &http.Client{Transport: &fakeTransport{routes: map[string]string{shortTrack: spotifyTrack}}}
	if _, err := musiclink.New(musiclink.WithHTTPClient(c)).Parse(t.Context(), shortTrack); err != nil {
		t.Fatal(err)
	}
	if c.CheckRedirect != nil {
		t.Error("client CheckRedirect was set")
	}
}

func TestWithoutNetwork(t *testing.T) {
	f := &fakeTransport{routes: map[string]string{shortTrack: spotifyTrack}}
	p := newParser(f, musiclink.WithoutNetwork())
	if _, err := p.Parse(t.Context(), shortTrack); !errors.Is(err, musiclink.ErrNetworkDisabled) {
		t.Errorf("Parse() error = %v, want ErrNetworkDisabled", err)
	}
	res := p.ParseBatch(t.Context(), []string{shortTrack})
	if !errors.Is(res[0].Err, musiclink.ErrNetworkDisabled) {
		t.Errorf("ParseBatch() error = %v, want ErrNetworkDisabled", res[0].Err)
	}
	if n := f.callCount(shortTrack); n != 0 {
		t.Errorf("requests = %d, want 0", n)
	}
}

func TestParseConcurrencyShared(t *testing.T) {
	routes := map[string]string{}
	for i := range 20 {
		routes[fmt.Sprintf("https://spoti.fi/%d", i)] = spotifyTrack
	}
	f := &fakeTransport{routes: routes, delay: 5 * time.Millisecond}
	p := newParser(f, musiclink.WithConcurrency(2))
	var wg sync.WaitGroup
	for u := range routes {
		wg.Go(func() {
			if _, err := p.Parse(t.Context(), u); err != nil {
				t.Error(err)
			}
		})
	}
	wg.Wait()
	if peak := f.peak.Load(); peak > 2 {
		t.Errorf("peak requests = %d, want <= 2", peak)
	}
}

func TestParseBatch(t *testing.T) {
	f := &fakeTransport{routes: map[string]string{
		shortTrack:           spotifyTrack,
		"https://spoti.fi/x": "https://example.com/",
	}}
	urls := []string{
		"https://music.yandex.ru/album/1",
		shortTrack,
		"garbage",
		"https://spoti.fi/x",
		"https://open.spotify.com/show/5CfCWKI5pZ28U0uOzXkDHe",
		shortTrack,
		"https://music.yandex.ru/album/1",
		shortTrack,
	}
	got := newParser(f).ParseBatch(t.Context(), urls)
	want := []struct {
		link musiclink.Link
		err  error
	}{
		{link: yandex(musiclink.Album, "1")},
		{link: spotify(musiclink.Track, "0RmHZOnMjaCM6mIvLmvxFr")},
		{err: musiclink.ErrInvalidURL},
		{err: musiclink.ErrUnknownHost},
		{err: musiclink.ErrUnsupported},
		{link: spotify(musiclink.Track, "0RmHZOnMjaCM6mIvLmvxFr")},
		{link: yandex(musiclink.Album, "1")},
		{link: spotify(musiclink.Track, "0RmHZOnMjaCM6mIvLmvxFr")},
	}
	if len(got) != len(want) {
		t.Fatalf("len = %d, want %d", len(got), len(want))
	}
	for i, w := range want {
		if got[i].Link != w.link || !errors.Is(got[i].Err, w.err) || (w.err == nil) != (got[i].Err == nil) {
			t.Errorf("[%d] %s = %+v, %v; want %+v, %v", i, urls[i], got[i].Link, got[i].Err, w.link, w.err)
		}
	}
	if n := f.callCount(shortTrack); n != 1 {
		t.Errorf("requests to %s = %d, want 1", shortTrack, n)
	}
}

func TestParseBatchConcurrency(t *testing.T) {
	routes := map[string]string{}
	var urls []string
	for i := range 30 {
		u := fmt.Sprintf("https://spoti.fi/%d", i)
		routes[u] = spotifyTrack
		urls = append(urls, u)
	}
	f := &fakeTransport{routes: routes, delay: 5 * time.Millisecond}
	res := newParser(f, musiclink.WithConcurrency(3)).ParseBatch(t.Context(), urls)
	for i, r := range res {
		if r.Err != nil {
			t.Errorf("[%d] error = %v", i, r.Err)
		}
	}
	if peak := f.peak.Load(); peak > 3 {
		t.Errorf("peak requests = %d, want <= 3", peak)
	}
}

func TestParseBatchCancel(t *testing.T) {
	var urls []string
	for i := range 10 {
		urls = append(urls, fmt.Sprintf("https://spoti.fi/%d", i))
	}
	urls = append(urls, "https://music.yandex.ru/track/1")
	f := &fakeTransport{block: true, started: make(chan struct{}, 1)}
	ctx, cancel := context.WithCancel(t.Context())
	go func() {
		<-f.started
		cancel()
	}()
	res := newParser(f, musiclink.WithConcurrency(2)).ParseBatch(ctx, urls)
	for i, r := range res[:10] {
		if !errors.Is(r.Err, context.Canceled) {
			t.Errorf("[%d] error = %v, want context.Canceled", i, r.Err)
		}
	}
	if r := res[10]; r.Err != nil || r.Link != yandex(musiclink.Track, "1") {
		t.Errorf("local link = %+v, %v", r.Link, r.Err)
	}
}

func TestParseCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	_, err := newParser(&fakeTransport{block: true}).Parse(ctx, shortTrack)
	if !errors.Is(err, context.Canceled) {
		t.Errorf("Parse() error = %v, want context.Canceled", err)
	}
}
