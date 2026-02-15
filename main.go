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
}

func main() {

	var cliArgs cliArgStruct

	cli.VersionPrinter = func(cmd *cli.Command) {
		fmt.Printf("version=%s\n", cmd.Root().Version)
	}
	app := &cli.Command{

		Version: "0.0.1",
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
				},
			},
		},

		Name:  "ipfind",
		Usage: "find this ip",
		Action: func(ctx context.Context, cmd *cli.Command) error {

			//fmt.Println("hello fom action L:", ipaddr, cmd.Bool("longest"))
			//fmt.Println("you want me to find IP", cliArgs.ipaddr)

			// set longest if neither of the other are set
			cliArgs.longest = !(cliArgs.exact || cliArgs.subnet)

			//fmt.Printf("exact: %v longest: %v, subnet: %v.\n	", cliArgs.exact, cliArgs.longest, cliArgs.subnet)
			ipcmd(cliArgs)
			return nil
		},
	}

	if err := app.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}
