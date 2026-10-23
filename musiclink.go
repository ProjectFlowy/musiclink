// Package musiclink parses links to music streaming platforms into
// a platform, an entity kind and an ID.
//
// Parsing is local and does not talk to platform APIs. The network is used
// only to expand known short links by following their redirects.
package musiclink

import (
	"errors"
	"strconv"
	"strings"
)

// Platform identifies a music streaming platform.
type Platform string

// Kind is the kind of entity a link points to.
type Kind uint8

const (
	Track Kind = iota + 1
	Album
	Artist
	Playlist
)

func (k Kind) String() string {
	switch k {
	case Track:
		return "track"
	case Album:
		return "album"
	case Artist:
		return "artist"
	case Playlist:
		return "playlist"
	}
	return "Kind(" + strconv.Itoa(int(k)) + ")"
}

// Link is a parsed link.
type Link struct {
	Platform Platform
	Kind     Kind
	ID       string
	// Owner is the playlist owner when the platform addresses playlists
	// by owner and ID. It is empty otherwise.
	Owner string
}

// Result is the outcome of parsing one link in a batch.
type Result struct {
	Link Link
	Err  error
}

// Errors returned by the parser are wrapped; use errors.Is to check them.
var (
	ErrInvalidURL       = errors.New("invalid URL")
	ErrUnknownHost      = errors.New("unknown host")
	ErrUnsupported      = errors.New("unsupported link kind")
	ErrMalformed        = errors.New("malformed link")
	ErrNetworkDisabled  = errors.New("network disabled")
	ErrNotRedirect      = errors.New("not a redirect")
	ErrTooManyRedirects = errors.New("too many redirects")
)

// PlatformOf reports the platform a host belongs to, including short link
// hosts. The host may carry a port and a "www." prefix, in any case.
func PlatformOf(host string) (Platform, bool) {
	host, ok := normalizeHost(host)
	if !ok {
		return "", false
	}
	if r, ok := domains[host]; ok {
		return r.platform, true
	}
	p, ok := shorteners[host]
	return p, ok
}

// IsShort reports whether rawURL is a link on a known short link host.
func IsShort(rawURL string) bool {
	_, short, err := parseLocal(strings.TrimSpace(rawURL))
	return err == nil && short
}
