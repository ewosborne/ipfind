package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"

	"github.com/seancfoley/ipaddress-go/ipaddr"
	"github.com/urfave/cli/v3"
)

type cliArgStruct struct {
	Ipstring                             string
	Exact, Longest, Subnet, Trie, V4, V6 bool
	InputFile                            string
	Debug                                bool
	Ipaddr                               *ipaddr.IPAddress
}

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
				}, // Flags:
			},
		}, // MutuallyExclusiveFlags:
		Action: func(ctx context.Context, cmd *cli.Command) error {
			// Set up the logger based on the debug flag
			var logLevel slog.Level
			if cmd.Bool("debug") {
				logLevel = slog.LevelDebug
			} else {
				logLevel = slog.LevelInfo
			}

			// Create a handler with the appropriate level
			handler := slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
				Level: logLevel,
			})
			logger := slog.New(handler)
			slog.SetDefault(logger)

			// Longest is default if the other two aren't set
			cliArgs.Longest = !(cliArgs.Exact || cliArgs.Subnet)

			// turn target IP into address object
			cliArgs.Ipaddr = ipaddr.NewIPAddressString(cliArgs.Ipstring).GetAddress()

			// run the command
			return ipcmd(cliArgs)
		},
	}

	if err := app.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}
