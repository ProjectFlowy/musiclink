// Command musiclink parses music streaming links.
package main

import (
	"bufio"
	"cmp"
	"context"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"runtime/debug"
	"strings"
	"syscall"
	"time"

	"github.com/alecthomas/kong"

	"github.com/projectflowy/musiclink"
)

// version is set at build time with -ldflags "-X main.version=...".
var version string

type cli struct {
	Parse   parseCmd   `cmd:"" help:"Parse links into platform, kind and ID."`
	Version versionCmd `cmd:"" help:"Print the version."`
}

type parseCmd struct {
	URLs []string `arg:"" name:"url" help:"Links to parse, separated by spaces. A single \"-\" reads them from stdin, one per line."`

	Output       string        `enum:"pretty,json" default:"pretty" help:"Output format: pretty or json."`
	Offline      bool          `help:"Do not use the network; short links fail."`
	Concurrency  int           `default:"8" help:"Maximum simultaneous network requests."`
	MaxRedirects int           `default:"5" help:"Maximum short links in a redirect chain."`
	Timeout      time.Duration `default:"30s" help:"Give up on links not parsed in this time."`
}

func (c parseCmd) Run(ctx context.Context, in io.Reader, out io.Writer) error {
	if c.Timeout <= 0 {
		return errors.New("timeout must be positive")
	}
	urls := c.URLs
	if len(urls) == 1 && urls[0] == "-" {
		if isTerminal(in) {
			_, _ = fmt.Fprintln(os.Stderr, "Reading links from stdin, one per line. End with Ctrl-D.")
		}
		var err error
		if urls, err = readLines(ctx, in); err != nil {
			return err
		}
	}

	opts := []musiclink.Option{musiclink.WithConcurrency(c.Concurrency), musiclink.WithMaxRedirects(c.MaxRedirects)}
	if c.Offline {
		opts = append(opts, musiclink.WithoutNetwork())
	}
	ctx, cancel := context.WithTimeout(ctx, c.Timeout)
	defer cancel()
	results := musiclink.New(opts...).ParseBatch(ctx, urls)

	if c.Output == "json" {
		err := printJSON(out, urls, results)
		return cmp.Or(err, failed(results))
	}
	printPretty(out, results)
	return failed(results)
}

// readLines reads non-empty lines until EOF or until ctx is done.
func readLines(ctx context.Context, r io.Reader) ([]string, error) {
	type result struct {
		lines []string
		err   error
	}
	done := make(chan result, 1)
	go func() {
		var res result
		s := bufio.NewScanner(r)
		for s.Scan() {
			if line := strings.TrimSpace(s.Text()); line != "" {
				res.lines = append(res.lines, line)
			}
		}
		res.err = s.Err()
		done <- res
	}()
	select {
	case res := <-done:
		if res.err != nil {
			return nil, fmt.Errorf("read stdin: %w", res.err)
		}
		return res.lines, nil
	case <-ctx.Done():
		// The reader stays blocked; the process is about to exit anyway.
		return nil, fmt.Errorf("read stdin: %w", context.Cause(ctx))
	}
}

func isTerminal(r io.Reader) bool {
	f, ok := r.(*os.File)
	if !ok {
		return false
	}
	st, err := f.Stat()
	return err == nil && st.Mode()&os.ModeCharDevice != 0
}

func failed(results []musiclink.Result) error {
	n := 0
	for _, r := range results {
		if r.Err != nil {
			n++
		}
	}
	if n > 0 {
		return fmt.Errorf("%d of %d links not parsed", n, len(results))
	}
	return nil
}

type resultView struct {
	URL      string `json:"url"`
	Platform string `json:"platform,omitempty"`
	Kind     string `json:"kind,omitempty"`
	ID       string `json:"id,omitempty"`
	Owner    string `json:"owner,omitempty"`
	Error    string `json:"error,omitempty"`
}

func printJSON(out io.Writer, urls []string, results []musiclink.Result) error {
	views := make([]resultView, len(results))
	for i, r := range results {
		views[i] = resultView{URL: urls[i]}
		if r.Err != nil {
			views[i].Error = r.Err.Error()
			continue
		}
		views[i].Platform = string(r.Link.Platform)
		views[i].Kind = r.Link.Kind.String()
		views[i].ID = r.Link.ID
		views[i].Owner = r.Link.Owner
	}
	if err := json.MarshalWrite(out, views, json.Deterministic(true), jsontext.WithIndent("  ")); err != nil {
		return err
	}
	_, err := io.WriteString(out, "\n")
	return err
}

// printPretty prints a row per result. Columns are aligned across all rows;
// the last cell of a row is not padded and does not widen its column.
func printPretty(out io.Writer, results []musiclink.Result) {
	rows := make([][]string, len(results))
	for i, r := range results {
		if r.Err != nil {
			rows[i] = []string{"error", r.Err.Error()}
			continue
		}
		rows[i] = []string{string(r.Link.Platform), r.Link.Kind.String(), r.Link.ID}
		if r.Link.Owner != "" {
			rows[i] = append(rows[i], r.Link.Owner)
		}
	}
	var widths []int
	for _, row := range rows {
		for i, cell := range row[:len(row)-1] {
			if i == len(widths) {
				widths = append(widths, 0)
			}
			widths[i] = max(widths[i], len(cell))
		}
	}
	for _, row := range rows {
		for i, cell := range row[:len(row)-1] {
			_, _ = fmt.Fprintf(out, "%-*s  ", widths[i], cell)
		}
		_, _ = fmt.Fprintln(out, row[len(row)-1])
	}
}

type versionCmd struct{}

func (versionCmd) Run(out io.Writer) error {
	v := version
	if info, ok := debug.ReadBuildInfo(); ok && v == "" {
		v = info.Main.Version
	}
	_, err := fmt.Fprintln(out, "musiclink", cmp.Or(v, "(devel)"))
	return err
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	// After the first signal, a second one kills the process right away.
	context.AfterFunc(ctx, stop)

	var c cli
	k := kong.Parse(&c,
		kong.Name("musiclink"),
		kong.Description("Parse music streaming links into platform, kind and ID."),
		kong.UsageOnError(),
		kong.BindTo(ctx, (*context.Context)(nil)),
		kong.BindTo(os.Stdin, (*io.Reader)(nil)),
		kong.BindTo(os.Stdout, (*io.Writer)(nil)),
	)
	k.FatalIfErrorf(k.Run())
}
