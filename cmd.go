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

	var idx int = 0

	scanner := get_input_scanner(args)

	for scanner.Scan() {
		// get line from scanner and start counting at 1
		idx++

		line := scanner.Text()
		slog.Debug("scanned:", "idx", idx, "line", line)

		//fmt.Printf("\nline %v\n", line)
		v4_matches := get_ipv4_addresses_from_line(line)
		slog.Debug("placeholder", "len", len(v4_matches))
		//fmt.Printf("v4 matches%v\n", v4_matches)

		/* TODO v6 regexp is all messed up and turns 4.218.236.160/30 into v6 matches [0.0.0.218 0.0.0.236 0.0.0.160/30]
		v6_matches := get_ipv6_addresses_from_line(line)
		fmt.Printf("v6 matches %v\n\n", v6_matches)
		*/

		/* TODO: maybe combine v4_matches and v6_matches? */
		// matches := append(v4_matches, v6_matches...)
		matches := v4_matches

		/*
			OK now I have v4 matches
			do the -e, -s, -l, -t stuff
		*/

		for _, match := range matches {
			slog.Debug("comparing", "a", args.Ipaddr.String(), "b", match.String())

			switch {
			case args.Exact:
				if args.Ipaddr.Equal(match) {
					slog.Debug("FOUND MATCH", match.String(), args.Ipaddr.String())
					//fmt.Println("FOUND MATCH", match.String(), args.Ipaddr.String(), idx, line)

					// TODO now what?
				}

			case args.Subnet:
				if match.Contains(args.Ipaddr) {
					//slog.Debug("CONTAINS", match.String(), args.Ipaddr.String(), idx, line)
					fmt.Println("CONTAINS", match.String(), args.Ipaddr.String(), idx, line)

					// TODO now what?

				}
				//fmt.Println("TODO subnet match")

			case args.Longest:
				fmt.Println("TODO longest match")

			case args.Trie:
				fmt.Println("TODO trie")

			}
		}
	}

	return nil
}
