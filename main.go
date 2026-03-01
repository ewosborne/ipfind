package main

import (
	"context"
	_ "embed"
	"fmt"
	"log"
	"os"

	"github.com/urfave/cli/v3"
)

//go:embed RootCommandHelpTemplate.txt
var h string

func main() {

	var cliArgs cliArgStruct

	cli.RootCommandHelpTemplate = fmt.Sprintf(`%s
%s
`, cli.RootCommandHelpTemplate, h)

	cli.VersionPrinter = func(cmd *cli.Command) {
		fmt.Printf("version=%s\n", cmd.Root().Version)
	}
	app := &cli.Command{
		Version:                "0.0.2",
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
			&cli.BoolWithInverseFlag{
				Name:        "canonize",
				Usage:       `Canonize input to match mask`,
				Destination: &cliArgs.Canonize,
				Value:       true,
			},
			&cli.BoolWithInverseFlag{
				Name:        "slash",
				Usage:       `Require a subnet mask to recognize a host`,
				Destination: &cliArgs.Slash,
				Value:       true,
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
							Name: "longest",
							Usage: `Find all networks with the longest match 
	 which contains the given network`,
							Aliases:     []string{"l"},
							Destination: &cliArgs.Longest,
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
				}, // Flags:
			},
		}, // MutuallyExclusiveFlags:
		Action: func(ctx context.Context, cmd *cli.Command) error {

			// 1. call argMassage to fix up args
			// 2. call ipcmd(args) to do stuff

			cliArgs := argMassage(cliArgs)
			// fmt.Printf("massaged args:%+v\n", cliArgs)

			// run the command
			return ipcmd(os.Stdout, cliArgs)
		},
	}

	if err := app.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}
