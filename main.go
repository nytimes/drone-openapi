package main

import (
	"fmt"
	"os"

	_ "github.com/joho/godotenv/autoload"
	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"
)

var version string // build number set at compile-time

const (
	specsFlag    = "specs"
	teamFlag     = "team"
	uploaderFlag = "uploader-url"
	keyFlag      = "key"
	tokenFlag    = "token"
)

func main() {
	app := cli.NewApp()
	app.Name = "openapi spec uploader plugin"
	app.Usage = "openapi spec uploader plugin"
	app.Action = run
	app.Version = version
	app.Flags = []cli.Flag{
		&cli.StringSliceFlag{
			Name:    specsFlag,
			Usage:   "spec files to upload (can be specified multiple times)",
			EnvVars: []string{"PLUGIN_SPECS"},
		},
		&cli.StringFlag{
			Name:    teamFlag,
			Usage:   "the team the specs belong to",
			EnvVars: []string{"PLUGIN_TEAM"},
		},
		&cli.StringFlag{
			Name:    uploaderFlag,
			Usage:   "the url of the uploader",
			EnvVars: []string{"PLUGIN_UPLOADER_URL"},
		},
		&cli.StringFlag{
			Name:    keyFlag,
			Usage:   "api key to upload the spec",
			EnvVars: []string{"PLUGIN_KEY"},
		},
		&cli.StringFlag{
			Name:    tokenFlag,
			Usage:   "google credential to upload the spec",
			EnvVars: []string{"PLUGIN_TOKEN", "GOOGLE_CREDENTIALS"},
		},
		// &cli.StringFlag{
		// 	Name:   "workspace",
		// 	Usage:  "the workspace",
		// 	EnvVars: "PLUGIN_PATH,DRONE_WORKSPACE",
		// },
	}

	if err := app.Run(os.Args); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func run(c *cli.Context) error {
	plugin := Plugin{
		Config: Config{
			// plugin-specific parameters
			Specs:    c.StringSlice(specsFlag),
			Team:     c.String(teamFlag),
			Uploader: c.String(uploaderFlag),
			Key:      c.String(keyFlag),
			Token:    c.String(tokenFlag),
		},
	}

	// return plugin.Exec()

	if err := plugin.Validate(); err != nil {
		return errors.Wrap(err, "validation failed")
	}

	if err := plugin.Exec(); err != nil {
		return errors.Wrap(err, "exec failed")
	}

	return nil
}
