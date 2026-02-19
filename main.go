package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/urfave/cli/v3"
)

type cliArgStruct struct {
	ipaddr                 string
	exact, longest, subnet bool
	inputFile              string
	networkOnly            bool
	debug                  bool
}

func main() {

	var cliArgs cliArgStruct

	cli.VersionPrinter = func(cmd *cli.Command) {
		fmt.Printf("version=%s\n", cmd.Root().Version)
	}
	app := &cli.Command{
		Version:                "0.0.1",
		UseShortOptionHandling: true,
		EnableShellCompletion:  true,

		Arguments: []cli.Argument{
			&cli.StringArg{
				Name:        "ip",
				Destination: &cliArgs.ipaddr,
			},
			&cli.StringArg{
				Name:        "filename",
				Destination: &cliArgs.inputFile,
			},
		},
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:        "debug",
				Aliases:     []string{"d"},
				Usage:       "print debug output",
				Destination: &cliArgs.debug,
			},
			&cli.BoolFlag{
				Name:        "network-only",
				Aliases:     []string{"n"},
				Value:       false,
				Usage:       "show only matched networks, not the entire line",
				Destination: &cliArgs.networkOnly,
			},
		},
		MutuallyExclusiveFlags: []cli.MutuallyExclusiveFlags{
			{
				Flags: [][]cli.Flag{
					{
						&cli.BoolFlag{
							Name:        "exact",
							Usage:       "exact match",
							Aliases:     []string{"e"},
							Destination: &cliArgs.exact,
						},
					},
					{
						&cli.BoolFlag{
							Name:        "longest",
							Usage:       "longest match",
							Aliases:     []string{"l"},
							Destination: &cliArgs.longest,
						},
					},
					{
						&cli.BoolFlag{
							Name:        "subnet",
							Usage:       "subnet match",
							Aliases:     []string{"s"},
							Destination: &cliArgs.subnet,
						},
					},
				}, // Flags:
			},
		}, // MutuallyExclusiveFlags:

		Name:  "ipfind",
		Usage: "find this ip",
		Action: func(ctx context.Context, cmd *cli.Command) error {

			// set longest if neither of the other are set
			cliArgs.longest = !(cliArgs.exact || cliArgs.subnet)

			ipcmd(cliArgs)
			return nil
		},
	}

	if err := app.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}
