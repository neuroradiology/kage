// Package cli wires kage's command surface: the cobra tree, the global flags,
// and the fang-rendered help and errors. The actual work lives in the clone,
// browser, sanitize, asset, and urlx packages; this layer only parses flags and
// prints progress.
package cli

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"

	"github.com/charmbracelet/fang"
	"github.com/spf13/cobra"

	"github.com/tamnd/kage/pack"
	"github.com/tamnd/kage/zim"
)

// Execute builds the root command and runs it through fang. main passes the
// signal-aware context so Ctrl-C cancels the in-flight clone and flushes resume
// state. It returns the process exit code.
func Execute(ctx context.Context) int {
	// A kage binary with a ZIM appended runs as an offline viewer for that site,
	// ignoring its arguments. A normal build has no trailer and falls through.
	if ra, size, ok := pack.Embedded(); ok {
		return runEmbeddedViewer(ctx, ra, size)
	}

	root := newRoot()
	opts := []fang.Option{
		fang.WithVersion(Version),
	}
	if err := fang.Execute(ctx, root, opts...); err != nil {
		return 1
	}
	return 0
}

// newRoot assembles the command tree.
func newRoot() *cobra.Command {
	root := &cobra.Command{
		Use:   "kage",
		Short: "Clone any website for offline viewing, with the JavaScript stripped out",
		Long: "kage (影, \"shadow\") renders each page in headless Chrome, snapshots the\n" +
			"final DOM, removes every script and event handler, and localises the CSS,\n" +
			"images, and fonts so the saved copy looks like the live site but runs no\n" +
			"code. The result is a plain folder you can open straight from disk.",
		Version:       fmt.Sprintf("%s (commit %s, built %s)", Version, Commit, Date),
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.AddCommand(newCloneCmd())
	root.AddCommand(newServeCmd())
	root.AddCommand(newPackCmd())
	root.AddCommand(newOpenCmd())
	return root
}

// runEmbeddedViewer serves the ZIM appended to this executable on an ephemeral
// local port and opens the browser. It runs until the context is cancelled
// (Ctrl-C) and ignores all command-line arguments: a packed binary is the site,
// not the kage CLI.
func runEmbeddedViewer(ctx context.Context, ra io.ReaderAt, size int64) int {
	r, err := zim.NewReader(ra, size)
	if err != nil {
		fmt.Fprintln(os.Stderr, "kage: corrupt embedded archive:", err)
		return 1
	}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		fmt.Fprintln(os.Stderr, "kage: cannot start viewer:", err)
		return 1
	}
	url := "http://" + ln.Addr().String()
	fmt.Fprintln(os.Stderr, "serving offline site at "+url+"  (Ctrl-C to stop)")
	_ = pack.OpenInBrowser(url)

	srv := &http.Server{Handler: pack.Handler(r)}
	go func() {
		<-ctx.Done()
		_ = srv.Close()
	}()
	if err := srv.Serve(ln); err != nil && err != http.ErrServerClosed {
		fmt.Fprintln(os.Stderr, "kage:", err)
		return 1
	}
	return 0
}
