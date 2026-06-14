// Command kage clones a website into a self-contained offline folder: it renders
// every page in headless Chrome, strips all JavaScript, and localises the CSS,
// images, and fonts so the saved copy looks like the live site but runs no code.
package main

import (
	"context"
	"os"
	"os/signal"

	"github.com/tamnd/kage/cli"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	os.Exit(cli.Execute(ctx))
}
