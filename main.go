package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/urfave/cli/v3"
)

func main() {
	var addr string

	cli.VersionPrinter = func(cmd *cli.Command) {
		fmt.Printf("version=%s\n", cmd.Root().Version)
	}
	app := &cli.Command{

		Version: "0.0.1",
		MutuallyExclusiveFlags: []cli.MutuallyExclusiveFlags{
			{
				Required: true,
				Flags: [][]cli.Flag{
					{
						&cli.StringFlag{
							Name:    "exact",
							Usage:   "exact match",
							Aliases: []string{"e"},
						},
					},
					{
						&cli.StringFlag{
							Name:    "longest",
							Usage:   "longest match",
							Aliases: []string{"l"},
						},
					},
					{
						&cli.StringFlag{
							Name:    "subnet",
							Usage:   "subnet match",
							Aliases: []string{"s"},
						},
					},
				},
			},
		},

		Name:  "ipfind",
		Usage: "find this ip",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if cmd.NArg() > 0 {
				fmt.Println("args", cmd.Args(), cmd.NArg())
				fmt.Println("first", cmd.Args().Get(0))
			}
			fmt.Println("hello from action", cmd.String("addr"), addr)
			ip()
			return nil
		},
	}

	if err := app.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}
