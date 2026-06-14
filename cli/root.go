// Package cli wires kage's command surface: the cobra tree, the global flags,
// and the fang-rendered help and errors. The actual work lives in the clone,
// browser, sanitize, asset, and urlx packages; this layer only parses flags and
// prints progress.
package cli

import (
	"context"
	"fmt"

	"github.com/charmbracelet/fang"
	"github.com/spf13/cobra"
)

// Execute builds the root command and runs it through fang. main passes the
// signal-aware context so Ctrl-C cancels the in-flight clone and flushes resume
// state. It returns the process exit code.
func Execute(ctx context.Context) int {
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
	return root
}
