package musiclink

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const defaultUserAgent = "musiclink (+https://github.com/projectflowy/musiclink)"

// Parser parses links. It is safe for concurrent use.
type Parser struct {
	client       *http.Client
	sem          chan struct{}
	maxRedirects int
	userAgent    string
	offline      bool
}

type options struct {
	client       *http.Client
	concurrency  int
	maxRedirects int
	userAgent    string
	offline      bool
}

// Option configures a Parser.
type Option func(*options)

// WithHTTPClient sets the client used to expand short links. The client is
// copied and never modified; the copy does not follow redirects.
// The default client has a 10 second timeout.
func WithHTTPClient(c *http.Client) Option {
	return func(o *options) {
		if c != nil {
			o.client = c
		}
	}
}

// WithConcurrency limits the number of simultaneous network requests made
// by the Parser across all calls. The default is 8; n < 1 keeps it.
func WithConcurrency(n int) Option {
	return func(o *options) {
		if n > 0 {
			o.concurrency = n
		}
	}
}

// WithMaxRedirects limits how many short links a chain may go through.
// The default is 5; n < 1 keeps it.
func WithMaxRedirects(n int) Option {
	return func(o *options) {
		if n > 0 {
			o.maxRedirects = n
		}
	}
}

// WithUserAgent sets the User-Agent header for network requests.
func WithUserAgent(ua string) Option {
	return func(o *options) {
		o.userAgent = ua
	}
}

// WithoutNetwork disables the network. Short links then fail with
// ErrNetworkDisabled.
func WithoutNetwork() Option {
	return func(o *options) {
		o.offline = true
	}
}

// New returns a Parser configured by opts.
func New(opts ...Option) *Parser {
	o := options{
		client:       &http.Client{Timeout: 10 * time.Second},
		concurrency:  8,
		maxRedirects: 5,
		userAgent:    defaultUserAgent,
	}
	for _, opt := range opts {
		opt(&o)
	}
	client := *o.client
	client.CheckRedirect = func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	}
	return &Parser{
		client:       &client,
		sem:          make(chan struct{}, o.concurrency),
		maxRedirects: o.maxRedirects,
		userAgent:    o.userAgent,
		offline:      o.offline,
	}
}

// Parse parses rawURL: an absolute http or https URL or a Spotify URI.
// The query and fragment are ignored. Short links are expanded over
// the network, bounded by ctx.
func (p *Parser) Parse(ctx context.Context, rawURL string) (Link, error) {
	link, short, err := p.parseLocal(rawURL)
	if err == nil && short {
		link, err = p.expand(ctx, strings.TrimSpace(rawURL))
	}
	if err != nil {
		return Link{}, fmt.Errorf("musiclink: parse %q: %w", rawURL, err)
	}
	return link, nil
}

// ParseBatch parses rawURLs and returns a result for each, in the same order.
// Local links are parsed right away, short links are expanded concurrently.
// Equal inputs are parsed once. When ctx is done, links not yet expanded
// get its error.
func (p *Parser) ParseBatch(ctx context.Context, rawURLs []string) []Result {
	results := make([]Result, len(rawURLs))
	first := make(map[string]int, len(rawURLs))
	var shorts []int
	for i, raw := range rawURLs {
		if _, ok := first[raw]; ok {
			continue
		}
		first[raw] = i
		link, short, err := p.parseLocal(raw)
		if err == nil && short {
			shorts = append(shorts, i)
			continue
		}
		results[i] = result(raw, link, err)
	}

	var next atomic.Int64
	var wg sync.WaitGroup
	for range min(cap(p.sem), len(shorts)) {
		wg.Go(func() {
			for {
				n := int(next.Add(1)) - 1
				if n >= len(shorts) {
					return
				}
				i := shorts[n]
				raw := rawURLs[i]
				if err := ctx.Err(); err != nil {
					results[i] = result(raw, Link{}, err)
					continue
				}
				link, err := p.expand(ctx, strings.TrimSpace(raw))
				results[i] = result(raw, link, err)
			}
		})
	}
	wg.Wait()

	for i, raw := range rawURLs {
		if j := first[raw]; j != i {
			results[i] = results[j]
		}
	}
	return results
}

func result(raw string, link Link, err error) Result {
	if err != nil {
		return Result{Err: fmt.Errorf("musiclink: parse %q: %w", raw, err)}
	}
	return Result{Link: link}
}

// parseLocal parses rawURL without the network and reports whether it is
// a short link to expand.
func (p *Parser) parseLocal(rawURL string) (Link, bool, error) {
	link, short, err := parseLocal(strings.TrimSpace(rawURL))
	if err == nil && short && p.offline {
		return Link{}, false, ErrNetworkDisabled
	}
	return link, short, err
}

// expand follows a chain of short links until it reaches a regular link.
func (p *Parser) expand(ctx context.Context, short string) (Link, error) {
	for range p.maxRedirects {
		loc, err := p.follow(ctx, short)
		if err != nil {
			return Link{}, err
		}
		link, isShort, err := parseLocal(loc)
		if err != nil {
			return Link{}, fmt.Errorf("redirect to %q: %w", loc, err)
		}
		if !isShort {
			return link, nil
		}
		short = loc
	}
	return Link{}, fmt.Errorf("%d short links in a row: %w", p.maxRedirects, ErrTooManyRedirects)
}

// follow requests a short link and returns its redirect target.
// The response body is never read.
func (p *Parser) follow(ctx context.Context, short string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, short, nil)
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrInvalidURL, err)
	}
	if p.userAgent != "" {
		req.Header.Set("User-Agent", p.userAgent)
	}

	select {
	case p.sem <- struct{}{}:
	case <-ctx.Done():
		return "", ctx.Err()
	}
	resp, err := p.client.Do(req)
	<-p.sem
	if err != nil {
		return "", err
	}
	_ = resp.Body.Close()

	loc, err := resp.Location()
	if resp.StatusCode < 300 || resp.StatusCode > 399 || err != nil {
		return "", fmt.Errorf("%s responded with status %d: %w", short, resp.StatusCode, ErrNotRedirect)
	}
	return loc.String(), nil
}
