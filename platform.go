package musiclink

import (
	"fmt"
	"net/url"
	"strings"
)

// rules describe how to parse the links of one platform. Each platform
// lives in its own file and registers its rules from init.
type rules struct {
	platform Platform
	// hosts are the web hosts, without "www.".
	hosts []string
	// shortHosts are the short link hosts, expanded over the network.
	shortHosts []string
	// path parses the path of a link on one of hosts.
	path func(path string) (Link, error)
	// scheme names a URI scheme such as "spotify"; uri parses the part
	// after the colon. Both are optional.
	scheme string
	uri    func(opaque string) (Link, error)
}

// The registry is filled by init functions and only read afterwards.
var (
	domains    = map[string]*rules{}
	shorteners = map[string]Platform{}
	schemes    = map[string]*rules{}
)

func register(r *rules) {
	for _, h := range r.hosts {
		mustAdd(domains, h, r)
	}
	for _, h := range r.shortHosts {
		mustAdd(shorteners, h, r.platform)
	}
	if r.scheme != "" {
		mustAdd(schemes, r.scheme, r)
	}
}

func mustAdd[V any](m map[string]V, k string, v V) {
	if _, ok := m[k]; ok {
		panic("musiclink: " + k + " registered twice")
	}
	m[k] = v
}

// maxSegments bounds the number of path segments; no supported link has more.
const maxSegments = 8

// parseLocal parses s without the network and reports whether it is
// a short link to expand.
func parseLocal(s string) (Link, bool, error) {
	scheme, rest, ok := strings.Cut(s, ":")
	if !ok {
		return Link{}, false, ErrInvalidURL
	}
	scheme = strings.ToLower(scheme)
	if r, ok := schemes[scheme]; ok {
		link, err := r.uri(rest)
		return link, false, err
	}
	if scheme != "https" && scheme != "http" {
		return Link{}, false, ErrInvalidURL
	}
	rest, ok = strings.CutPrefix(rest, "//")
	if !ok {
		return Link{}, false, ErrInvalidURL
	}
	if i := strings.IndexAny(rest, "?#"); i >= 0 {
		rest = rest[:i]
	}
	authority, path := rest, ""
	if i := strings.IndexByte(rest, '/'); i >= 0 {
		authority, path = rest[:i], rest[i:]
	}
	if _, h, ok := strings.CutLast(authority, "@"); ok {
		authority = h
	}
	host, ok := normalizeHost(authority)
	if !ok {
		return Link{}, false, ErrInvalidURL
	}
	if _, ok := shorteners[host]; ok {
		return Link{}, true, nil
	}
	r, ok := domains[host]
	if !ok {
		return Link{}, false, ErrUnknownHost
	}
	link, err := r.path(path)
	return link, false, err
}

// normalizeHost lowercases host and strips the port and "www.".
func normalizeHost(host string) (string, bool) {
	if h, port, ok := strings.CutLast(host, ":"); ok {
		if port != "" && !isDigits(port) {
			return "", false
		}
		host = h
	}
	host = strings.TrimPrefix(strings.ToLower(host), "www.")
	return host, host != ""
}

// split appends the non-empty, unescaped segments of s to dst, usually
// an array on the caller's stack. It reports false when s has too many
// segments or a bad escape.
func split(dst []string, s, sep string) ([]string, bool) {
	for seg := range strings.SplitSeq(s, sep) {
		if seg == "" {
			continue
		}
		if len(dst) == cap(dst) {
			return nil, false
		}
		if strings.Contains(seg, "%") {
			var err error
			if seg, err = url.PathUnescape(seg); err != nil {
				return nil, false
			}
		}
		dst = append(dst, seg)
	}
	return dst, true
}

// kindNamed returns the kind named by a path segment, or zero.
func kindNamed(s string) Kind {
	switch s {
	case "track":
		return Track
	case "album":
		return Album
	case "artist":
		return Artist
	case "playlist":
		return Playlist
	}
	return 0
}

func isDigits(s string) bool {
	if s == "" {
		return false
	}
	for i := range len(s) {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

func isLetters(s string) bool {
	for i := range len(s) {
		if c := s[i] | 0x20; c < 'a' || c > 'z' {
			return false
		}
	}
	return true
}

func isBase62(s string) bool {
	for i := range len(s) {
		c := s[i]
		if (c < '0' || c > '9') && (c < 'a' || c > 'z') && (c < 'A' || c > 'Z') {
			return false
		}
	}
	return true
}

func malformed(p Platform, path string) error {
	return fmt.Errorf("%s: path %q: %w", p, path, ErrMalformed)
}

func badID(p Platform, k Kind, id string) error {
	return fmt.Errorf("%s: %s id %q: %w", p, k, id, ErrMalformed)
}

func unsupported(p Platform, path string) error {
	return fmt.Errorf("%s: path %q: %w", p, path, ErrUnsupported)
}
