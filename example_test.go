package musiclink_test

import (
	"context"
	"errors"
	"fmt"

	"github.com/projectflowy/musiclink"
)

func Example() {
	p := musiclink.New()
	link, err := p.Parse(context.Background(), "https://music.yandex.ru/album/7653771/track/2219127?utm_medium=copy_link")
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(link.Platform, link.Kind, link.ID)
	// Output: yandex track 2219127
}

func ExampleParser_ParseBatch() {
	p := musiclink.New(musiclink.WithoutNetwork())
	results := p.ParseBatch(context.Background(), []string{
		"https://open.spotify.com/intl-de/album/17a2qUMfTTjOAUPCw6n8Op?si=x",
		"https://music.yandex.ru/users/music-blog/playlists/2190",
		"https://open.spotify.com/show/5CfCWKI5pZ28U0uOzXkDHe",
	})
	for _, r := range results {
		if r.Err != nil {
			fmt.Println("error:", errors.Is(r.Err, musiclink.ErrUnsupported))
			continue
		}
		fmt.Printf("%s %s %s owner=%q\n", r.Link.Platform, r.Link.Kind, r.Link.ID, r.Link.Owner)
	}
	// Output:
	// spotify album 17a2qUMfTTjOAUPCw6n8Op owner=""
	// yandex playlist 2190 owner="music-blog"
	// error: true
}

func ExampleIsShort() {
	fmt.Println(musiclink.IsShort("https://spoti.fi/3xYzAbC"))
	fmt.Println(musiclink.IsShort("https://open.spotify.com/track/0RmHZOnMjaCM6mIvLmvxFr"))
	// Output:
	// true
	// false
}

func ExamplePlatformOf() {
	p, ok := musiclink.PlatformOf("www.deezer.com")
	fmt.Println(p, ok)
	// Output: deezer true
}
