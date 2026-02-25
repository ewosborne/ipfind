package main

import (
	"log/slog"
	"os"

	"github.com/seancfoley/ipaddress-go/ipaddr"
)

type cliArgStruct struct {
	Ipstring                     string
	Exact, Longest, Subnet, Trie bool
	V4, V6, Contains, Canonize   bool
	Slash                        bool
	InputFiles                   []string
	Debug                        bool
	Ipaddr                       *ipaddr.IPAddress
}

func argMassage(cliArgs cliArgStruct) cliArgStruct {
	// take in cliArgStruct
	// fix up all the defaults and fiddle with things
	// return it

	// Set up the logger based on the debug flag
	var logLevel slog.Level
	//if cmd.Bool("debug") { <--- useless?
	if cliArgs.Debug {
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

	// Longest is default if the others aren't set
	cliArgs.Longest = !(cliArgs.Exact || cliArgs.Subnet || cliArgs.Trie || cliArgs.Contains)

	// turn target IP into address object
	cliArgs.Ipaddr = ipaddr.NewIPAddressString(cliArgs.Ipstring).GetAddress()
	if cliArgs.Ipaddr.IsIPv4() {
		cliArgs.V4 = true
		cliArgs.V6 = false
	} else if cliArgs.Ipaddr.IsIPv6() {
		cliArgs.V4 = false
		cliArgs.V6 = true
	}

	// canonize it unless explicitly disallowed
	// TODO: treat this differently if -e is set? trying it out.
	if cliArgs.Canonize && !cliArgs.Exact { // if Canonize is false, don't convert 1.2.3.4/24 into 1.2.3.0/24
		cliArgs.Ipaddr = cliArgs.Ipaddr.ToPrefixBlock()
	}

	return cliArgs
}
