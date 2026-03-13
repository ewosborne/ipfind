package main

import (
	"context"
	_ "embed"
	"fmt"
	"os"

	"github.com/charmbracelet/log"
	"github.com/ewosborne/ipfind/pkg/ipfind"
	"github.com/urfave/cli/v3"
)

//go:embed RootCommandHelpTemplate.txt
var h string

func main() {

	var cliArgs ipfind.Args

	cli.RootCommandHelpTemplate = fmt.Sprintf(`%s
%s
`, cli.RootCommandHelpTemplate, h)

	cli.VersionPrinter = func(cmd *cli.Command) {
		fmt.Printf("version=%s\n", cmd.Root().Version)
	}
	app := &cli.Command{
		Version:                "0.1.0",
		UseShortOptionHandling: true,
		EnableShellCompletion:  true,
		Name:                   "ipfind",
		Usage: `Search for networks matching, containing, or contained 
  in a specified IP address.`,

		Arguments: []cli.Argument{
			&cli.StringArg{
				Name:        "ip",
				Destination: &cliArgs.Ipstring,
			},
			&cli.StringArgs{
				Name:        "file",
				Min:         0,
				Max:         -1,
				Destination: &cliArgs.InputFiles,
			},
		},
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:        "debug",
				Aliases:     []string{"d"},
				Usage:       "Debug output",
				Destination: &cliArgs.Debug,
			},
			&cli.BoolFlag{
				Name: "longest",
				Usage: `Find all networks with the longest match 
	 which contains the given network`,
				Aliases:     []string{"l"},
				Destination: &cliArgs.Longest,
			},
			&cli.BoolWithInverseFlag{
				Name:        "canonize",
				Usage:       `Canonize input to match mask`,
				Destination: &cliArgs.Canonize,
				Value:       true,
			},
			&cli.IntFlag{
				Name:        "workers",
				Aliases:     []string{"w"},
				Usage:       "Number of parallel workers (0 = 2 per CPU)",
				Destination: &cliArgs.Workers,
				Value:       0,
			},
			&cli.BoolFlag{
				Name:        "sort",
				Usage:       "Sort output by filename",
				Destination: &cliArgs.Sort,
			},
			&cli.StringFlag{
				Name:        "pretext",
				Usage:       "Text to look for before match",
				Destination: &cliArgs.Pretext,
			},
		},
		MutuallyExclusiveFlags: []cli.MutuallyExclusiveFlags{
			{
				Flags: [][]cli.Flag{
					{
						&cli.BoolFlag{
							Name:        "json",
							Aliases:     []string{"j"},
							Usage:       "JSON output",
							Destination: &cliArgs.Json,
						},
					},
					{
						&cli.BoolFlag{
							Name:        "trie",
							Aliases:     []string{"t"},
							Usage:       "Trie",
							Destination: &cliArgs.Trie,
						},
					},
				},
			},
			{
				Flags: [][]cli.Flag{
					{
						&cli.BoolFlag{
							Name:        "exact",
							Usage:       "Find exact network & subnet matches",
							Aliases:     []string{"e"},
							Destination: &cliArgs.Exact,
						},
					},
					{
						&cli.BoolFlag{
							Name:        "subnets-of",
							Usage:       "Find all networks which are subnets of the specified network",
							Aliases:     []string{"s"},
							Destination: &cliArgs.Subnet,
						},
					},
					{
						&cli.BoolFlag{
							Name:        "contains",
							Usage:       "Find all networks which contain the specified network",
							Aliases:     []string{"c"},
							Destination: &cliArgs.Contains,
						},
					},
				},
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if cliArgs.Debug {
				log.SetLevel(log.DebugLevel)
			} else {
				log.SetLevel(log.InfoLevel)
			}

			cliArgs = ipfind.ArgMassage(cliArgs)
			return ipfind.Run(ctx, os.Stdout, cliArgs)
		},
	}

	if err := app.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}
