# musiclink

[![Go Reference](https://pkg.go.dev/badge/github.com/projectflowy/musiclink.svg)](https://pkg.go.dev/github.com/projectflowy/musiclink)

Parse music streaming links into platform, kind and ID.

```
https://open.spotify.com/track/0RmHZOnMjaCM6mIvLmvxFr?si=698f…  →  spotify track 0RmHZOnMjaCM6mIvLmvxFr
https://music.yandex.ru/album/7653771/track/2219127?utm_…       →  yandex  track 2219127
https://link.deezer.com/s/34KSFEBUZ020rDp0QZ2Za                  →  deezer  track 4022917741
```

- No platform APIs and no metadata: only the link itself.
- Regular links are parsed locally, without allocations.
- The network is used only to expand short links, by following redirects
  without reading response bodies.
- Batch parsing with bounded concurrency and deduplication.

## Install

Library:

```sh
go get github.com/projectflowy/musiclink
```

CLI:

```sh
go install github.com/projectflowy/musiclink/cmd/musiclink@latest
```

Requires Go 1.27.1 or later.

## Usage

```go
p := musiclink.New()

link, err := p.Parse(ctx, "https://open.spotify.com/intl-de/album/17a2qUMfTTjOAUPCw6n8Op?si=x")
switch {
case errors.Is(err, musiclink.ErrUnsupported):
	// a podcast, an episode, a user page…
case err != nil:
	// not a link we understand
default:
	fmt.Println(link.Platform, link.Kind, link.ID) // spotify album 17a2qUMfTTjOAUPCw6n8Op
}

for i, r := range p.ParseBatch(ctx, urls) {
	// results come in the order of urls; each has a Link or an Err
}
```

A `Parser` is safe for concurrent use. Create one and reuse it: its HTTP
client and its limit on network requests are shared by all calls.

`musiclink.PlatformOf(host)` tells which platform a host belongs to, and
`musiclink.IsShort(url)` whether a link needs the network.

### Options

| Option                  | Default                 | Meaning                                                              |
|-------------------------|-------------------------|----------------------------------------------------------------------|
| `WithHTTPClient(c)`     | own client, 10s timeout | Client for short links. It is copied, never modified; the copy does not follow redirects. |
| `WithConcurrency(n)`    | 8                       | Maximum simultaneous network requests per `Parser`, across all calls. |
| `WithMaxRedirects(n)`   | 5                       | Maximum short links in one redirect chain.                           |
| `WithUserAgent(s)`      | `musiclink (+https://github.com/projectflowy/musiclink)` | `User-Agent` of network requests. |
| `WithoutNetwork()`      | network on              | Never use the network; short links fail with `ErrNetworkDisabled`.  |

Timeouts and cancellation come from the `ctx` you pass. There are no retries,
no logging and no cache between calls.

### Errors

Errors are wrapped with context, e.g.
`musiclink: parse "https://music.yandex.ru/album/abc": yandex: album id "abc": malformed link`.
Check them with `errors.Is`:

| Error                 | When                                                                  |
|-----------------------|-----------------------------------------------------------------------|
| `ErrInvalidURL`       | Not an absolute `http`/`https` URL or a known URI.                    |
| `ErrUnknownHost`      | The host is not a known platform, including redirects to other hosts. |
| `ErrUnsupported`      | A known platform, but a kind we do not support: podcasts, episodes, users, videos… |
| `ErrMalformed`        | A known platform, but the path or the ID is wrong.                    |
| `ErrNetworkDisabled`  | A short link with `WithoutNetwork`.                                   |
| `ErrNotRedirect`      | A short link host answered without a redirect.                        |
| `ErrTooManyRedirects` | The chain of short links is longer than `WithMaxRedirects`.           |

## Supported links

Kinds: track, album, artist, playlist. The query and fragment are ignored
(`utm_*`, `si`, `ref_id`…), the host is case-insensitive and may have
`www.` and a port, a trailing `/` is fine.

### Yandex Music

Hosts: `music.yandex.ru`, `.com`, `.by`, `.kz`, `.uz`. IDs are digits.

| Link                                  | Result                                |
|---------------------------------------|---------------------------------------|
| `/artist/<id>`                        | artist                                |
| `/album/<id>`                         | album                                 |
| `/track/<id>`                         | track                                 |
| `/album/<album>/track/<id>`           | track `<id>`                          |
| `/playlists/<uuid>`                   | playlist `<uuid>`                     |
| `/users/<login>/playlists/<id>`       | playlist `<id>`, `Owner` = `<login>`  |

### Spotify

Host: `open.spotify.com`, short links: `spotify.link`, `spoti.fi`.
IDs are 22 base62 characters.

| Link                                              | Result                         |
|---------------------------------------------------|--------------------------------|
| `/track/<id>`, `/album/<id>`, `/artist/<id>`, `/playlist/<id>` | that kind     |
| the same after a locale, `/intl-de/track/<id>`    | that kind                      |
| `/user/<user>/playlist/<id>`                      | playlist `<id>`, no `Owner`: Spotify playlist IDs are global |
| `spotify:track:<id>`, `spotify:user:<user>:playlist:<id>`, … | the same as the paths |

### Deezer

Host: `deezer.com`, short links: `link.deezer.com`. IDs are digits.

| Link                                              | Result         |
|---------------------------------------------------|----------------|
| `/track/<id>`, `/album/<id>`, `/artist/<id>`, `/playlist/<id>` | that kind |
| the same after a locale, `/en/track/<id>`, `/pt-br/album/<id>` | that kind |

## Short links

A short link is fetched with `GET`, without following redirects. The body is
never read: the response is closed right after the headers. A `3xx` with
`Location` is resolved against the request URL and parsed again, and it may
be another short link. Anything else is `ErrNotRedirect`.

How the platforms answer (checked with `curl` in October 2026):

- `spotify.link/<code>` and `spoti.fi/<code>` answer `307` with the
  `open.spotify.com` link, but only to a non-curl `User-Agent`. With curl's
  default agent, or without the header, they answer `200` with an HTML page
  that redirects in JavaScript, which is `ErrNotRedirect`. The default agent
  of this package gets the `307`. An unknown code redirects to
  `https://open.spotify.com/`, which is `ErrMalformed`.
- `link.deezer.com/s/<code>` answers `301` to
  `link.deezer.com/?dest=<url>&…`, which answers `301` to the `deezer.com`
  link: two requests per link. The `User-Agent` does not matter. An unknown
  code redirects to `https://www.deezer.com/deezer-links-404`, which is
  `ErrUnsupported`.

`ParseBatch` parses local links at once in the calling goroutine and expands
short links in a pool of at most `WithConcurrency` goroutines. Equal input
strings are parsed once. When `ctx` is done, links not yet expanded get its
error.

## CLI

```
musiclink parse <url>...     # with no URLs or "-", reads stdin, one link per line
  --output pretty|json       # default pretty
  --offline                  # never use the network
  --concurrency 8
  --max-redirects 5
  --timeout 30s
musiclink version
```

```console
$ musiclink parse https://music.yandex.ru/users/music-blog/playlists/2190 https://link.deezer.com/s/34KSEUilLbcjcyJu2WUbS https://open.spotify.com/show/5CfCWKI5pZ28U0uOzXkDHe
yandex  playlist  2190  music-blog
deezer  playlist  1996494362
error   musiclink: parse "https://open.spotify.com/show/5CfCWKI5pZ28U0uOzXkDHe": spotify: path "/show/5CfCWKI5pZ28U0uOzXkDHe": unsupported link kind
musiclink: error: 1 of 3 links not parsed
```

`--output json` prints an array of `{url, platform, kind, id, owner?}` or
`{url, error}`, in the input order. The exit code is not zero when at least
one link was not parsed.

## Adding a platform

Each platform is one file: `spotify.go`, `yandex.go`, `deezer.go`. It declares
the `Platform` constant and registers its hosts, short link hosts and parse
rules from `init`. A new platform is a new file like these and its
`_test.go` with a table of links.

## License

[MIT](LICENSE)
