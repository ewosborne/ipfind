package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/urfave/cli/v3"
)

func main() {

	var cliArgs cliArgStruct

	cli.VersionPrinter = func(cmd *cli.Command) {
		fmt.Printf("version=%s\n", cmd.Root().Version)
	}
	app := &cli.Command{
		Version:                "0.0.2",
		UseShortOptionHandling: true,
		EnableShellCompletion:  true,
		Name:                   "ipfind",
		Usage:                  "find this ip",

		Arguments: []cli.Argument{
			&cli.StringArg{
				Name:        "ip",
				Destination: &cliArgs.Ipstring,
			},
			&cli.StringArgs{
				Name: "file",
				//Usage:       "input file",
				//Aliases:     []string{"f"},
				Min:         0,
				Max:         -1,
				Destination: &cliArgs.InputFiles,
			},
		},
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:        "debug",
				Aliases:     []string{"d"},
				Usage:       "print debug output",
				Destination: &cliArgs.Debug,
			},
			&cli.BoolFlag{
				Name:        "trie",
				Usage:       "print trie",
				Aliases:     []string{"t"},
				Destination: &cliArgs.Trie,
			},
			&cli.BoolWithInverseFlag{
				Name:        "canonize",
				Usage:       "do not canonize to logical mastk",
				Destination: &cliArgs.Canonize,
				Value:       true,
			},
		},
		MutuallyExclusiveFlags: []cli.MutuallyExclusiveFlags{
			// {
			// 	Flags: [][]cli.Flag{
			// 		{
			// 			&cli.BoolFlag{
			// 				Name:        "v4",
			// 				Usage:       "force ipv4",
			// 				Destination: &cliArgs.V4,
			// 			},
			// 		},
			// 		{
			// 			&cli.BoolFlag{
			// 				Name:        "v6",
			// 				Usage:       "force ipv6",
			// 				Destination: &cliArgs.V6,
			// 			},
			// 		},
			// 	},
			// },
			{
				Flags: [][]cli.Flag{
					{
						&cli.BoolFlag{
							Name:        "exact",
							Usage:       "exact match",
							Aliases:     []string{"e"},
							Destination: &cliArgs.Exact,
						},
					},
					{
						&cli.BoolFlag{
							Name:        "longest",
							Usage:       "longest match",
							Aliases:     []string{"l"},
							Destination: &cliArgs.Longest,
						},
					},
					{
						&cli.BoolFlag{
							Name:        "subnet",
							Usage:       "subnet match",
							Aliases:     []string{"s"},
							Destination: &cliArgs.Subnet,
						},
					},
					{
						&cli.BoolFlag{
							Name:        "contains",
							Usage:       "find all networks and hosts contained in the specified subnet",
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
			return ipcmd(cliArgs)
		},
	}

	if err := app.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}
