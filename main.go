package main

import (
	"os"

	"github.com/rancher/wrangler/v2/pkg/signals"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"

	"github.com/oneblock-ai/steve/v2/pkg/debug"
	stevecli "github.com/oneblock-ai/steve/v2/pkg/server/cli"
	"github.com/oneblock-ai/steve/v2/pkg/version"
)

var (
	config      stevecli.Config
	debugconfig debug.Config
)

func main() {
	app := cli.NewApp()
	app.Name = "steve"
	app.Version = version.FriendlyVersion()
	app.Usage = ""
	app.Flags = append(
		stevecli.Flags(&config),
		debug.Flags(&debugconfig)...)
	app.Action = run

	if err := app.Run(os.Args); err != nil {
		logrus.Fatal(err)
	}
}

func run(_ *cli.Context) error {
	ctx := signals.SetupSignalContext()
	debugconfig.MustSetupDebug()
	s, err := config.ToServer(ctx)
	if err != nil {
		return err
	}
	return s.ListenAndServe(ctx, config.HTTPSListenPort, config.HTTPListenPort, nil)
}
