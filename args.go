package main

import (
	"log/slog"
	"os"
	"regexp"

	"github.com/charmbracelet/log"
	"github.com/seancfoley/ipaddress-go/ipaddr"
)

type cliArgStruct struct {
	Ipstring                           string
	Exact, Longest, Subnet, Trie       bool
	IsIPv4, IsIPv6, Contains, Canonize bool
	Slash, Json                        bool
	InputFiles                         []string
	Debug                              bool
	Ipaddr                             *ipaddr.IPAddress
	IPv4Regex, IPv6Regex               *regexp.Regexp
}

func argMassage(cliArgs cliArgStruct) cliArgStruct {
	// take in cliArgStruct
	// fix up all the defaults and fiddle with things
	// return it

	// Set up the logger based on the debug flag
	// var logLevel slog.Level
	// //if cmd.Bool("debug") { <--- useless?
	// if cliArgs.Debug {
	// 	logLevel = slog.LevelDebug
	// } else {
	// 	logLevel = slog.LevelInfo
	// }

	// Create a handler with the appropriate level
	// handler := slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
	// 	Level: logLevel,
	// })

	//logger := slog.New(handler)

	handler := log.New(os.Stderr)
	logger := slog.New(handler)

	if cliArgs.Debug {
		log.SetLevel(log.DebugLevel)
		log.Debug(("in debug level"))
	} else {
		log.SetLevel(log.InfoLevel)
	}
	slog.SetDefault(logger)

	// TODO: make sure we have both ip addr and files
	// this is trickier than I thought
	log.Debugf("arglen ipaddr %v flist %v\n", len(cliArgs.Ipstring), len(cliArgs.InputFiles))
	// Longest is default if the others aren't set
	//	cliArgs.Longest = !(cliArgs.Exact || cliArgs.Subnet || cliArgs.Trie || cliArgs.Contains)
	cliArgs.Longest = !(cliArgs.Exact || cliArgs.Subnet || cliArgs.Contains)

	// turn target IP into address object
	cliArgs.Ipaddr = ipaddr.NewIPAddressString(cliArgs.Ipstring).GetAddress()
	if cliArgs.Ipaddr.IsIPv4() {
		cliArgs.IsIPv4 = true
		cliArgs.IsIPv6 = false
	} else if cliArgs.Ipaddr.IsIPv6() {
		cliArgs.IsIPv4 = false
		cliArgs.IsIPv6 = true
	}

	// canonize it unless explicitly disallowed
	// TODO: treat this differently if -e is set? trying it out.
	if cliArgs.Canonize && !cliArgs.Exact { // if Canonize is false, don't convert 1.2.3.4/24 into 1.2.3.0/24
		cliArgs.Ipaddr = cliArgs.Ipaddr.ToPrefixBlock()
	}

	if cliArgs.Slash {
		cliArgs.IPv4Regex = ipv4Regex_withSlash
		cliArgs.IPv6Regex = ipv6Regex_withSlash
	} else {
		cliArgs.IPv4Regex = ipv4Regex_noSlash
		cliArgs.IPv6Regex = ipv6Regex_noSlash
	}

	return cliArgs
}
