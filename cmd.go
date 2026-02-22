package main

import (
	"bufio"
	"fmt"
	"log/slog"
	"os"
	"regexp"

	"github.com/seancfoley/ipaddress-go/ipaddr"
)

var (
	ipv4Regex = regexp.MustCompile(`(\d{1,3}).(\d{1,3}).(\d{1,3}).(\d{1,3}(/\d{1,2})?)`)
	ipv6Regex = regexp.MustCompile(`([:0-9a-fA-F]{2,39}(/[0-9]{1,3})?)`)
	//afArgs    afArgsStruct
)

func get_input_scanner(args cliArgStruct) *bufio.Scanner {
	if len(args.InputFile) > 0 {
		file, _ := os.Open(args.InputFile)
		return bufio.NewScanner(file)
	} else {
		return bufio.NewScanner(os.Stdin)
	}
}

func get_ip_addresses_from_line(ipre *regexp.Regexp, line string) []*ipaddr.IPAddress {
	ret := []*ipaddr.IPAddress{}
	ipStrings := ipre.FindAllString(line, -1)
	if ipStrings == nil { // no matches
		return nil
	}
	slog.Debug("FindAllString", "v4", ipStrings)

	for _, ipString := range ipStrings {
		slog.Debug("before", "addrString", ipString)
		converted := ipaddr.NewIPAddressString(ipString).GetAddress()
		slog.Debug("after", "converted", converted.String())
		if converted != nil { // no successful conversions, matches must have been bogus
			ret = append(ret, converted)
		}

	}

	if len(ret) == 0 { // not sure if this ever gets triggered
		return nil
	} else {
		return ret
	}
}

func get_ipv4_addresses_from_line(line string) []*ipaddr.IPAddress {
	return get_ip_addresses_from_line(ipv4Regex, line)
}

/* TODO regex is messy, see below */
// func get_ipv6_addresses_from_line(line string) []*ipaddr.IPAddress {
// 	return get_ip_addresses_from_line(ipv6Regex, line)
// }

func ipcmd(args cliArgStruct) error {
	slog.Debug("ipcmd", "args", args)

	// ok now do things

	scanner := get_input_scanner(args)

	for scanner.Scan() {
		// get line from scanner

		line := scanner.Text()
		slog.Debug("scanned:", "line", line)

		fmt.Printf("\nline %v\n", line)
		v4_matches := get_ipv4_addresses_from_line(line)
		fmt.Printf("v4 matches%v\n", v4_matches)

		/* TODO v6 regexp is all messed up and turns 4.218.236.160/30 into v6 matches [0.0.0.218 0.0.0.236 0.0.0.160/30]
		v6_matches := get_ipv6_addresses_from_line(line)
		fmt.Printf("v6 matches %v\n\n", v6_matches)
		*/

		/*
			OK now I have v4 matches
			do the -e, -s, -l, -t stuff
		*/
	}

	return nil
}
