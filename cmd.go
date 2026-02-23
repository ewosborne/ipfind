package main

/*
	TODO
	clean up output
	handle line vs network print match
*/

import (
	"bufio"
	"fmt"
	"log/slog"
	"os"
	"regexp"
	"slices"
	"strings"

	"github.com/seancfoley/ipaddress-go/ipaddr"
)

var (
	ipv4Regex = regexp.MustCompile(`(\d{1,3}).(\d{1,3}).(\d{1,3}).(\d{1,3}(/\d{1,2})?)`)
	ipv6Regex = regexp.MustCompile(`([:0-9a-fA-F]{2,39}(/[0-9]{1,3})?)`)
)

type foundmatch struct {
	Idx  int
	Addr *ipaddr.IPAddress
	Line string
}

type templateData struct {
	PrintIdx bool
	Items    []foundmatch
}

func (f foundmatch) String() string {
	return fmt.Sprintf("fm idx: %v  addr:%v  line(%v)", f.Idx, f.Addr, f.Line)
}

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

func get_ipv6_addresses_from_line(line string) []*ipaddr.IPAddress {

	// hack because the regex is getting messy but this seems ok.
	ret := []*ipaddr.IPAddress{}
	for _, m := range get_ip_addresses_from_line(ipv6Regex, line) {
		if strings.Contains(m.String(), ":") {
			ret = append(ret, m)
		}
	}
	return ret

}

func getHostbits(match *ipaddr.IPAddress) int {
	plen := match.GetPrefixLen().Len() // grr if there's no explicit /mask it's 0 not 32 or 128.  wtf.
	if plen == 0 {
		plen = match.GetBitCount()
	}
	return plen
}

func ipcmd(args cliArgStruct) error {
	slog.Debug("ipcmd", "args", args)

	var matchlist []foundmatch

	// ok now do things

	var idx int = 0
	// TODO DualIPv4v6AssociativeTries but maybe it sucks?
	v4_trie := ipaddr.NewIPv4AddressAssociativeTrie()
	v6_trie := ipaddr.NewIPv6AddressAssociativeTrie()

	var longest_subnet_seen int
	longest_subnets := make(map[int][]foundmatch)
	scanner := get_input_scanner(args)

	for scanner.Scan() {
		// get line from scanner and start counting at 1
		idx++

		line := scanner.Text()

		slog.Debug("scanned:", "idx", idx, "line", line)

		//fmt.Printf("\nline %v\n", line)
		v4_matches := get_ipv4_addresses_from_line(line)
		v6_matches := get_ipv6_addresses_from_line(line)

		slog.Debug("placeholder", "len", len(v4_matches))
		//fmt.Printf("v4 matches%v\n", v4_matches)

		matches := slices.Concat(v4_matches, v6_matches)
		if len(matches) == 0 {
			continue
		}

		/*
			OK now I have v4 matches
			do the -e, -s, -l, -t stuff
		*/

		for _, match := range matches {
			slog.Debug("comparing", "a", args.Ipaddr.String(), "b", match.String())

			// just slop it all into a trie
			// .Put() adds the foundmatch struct along with the prefix.
			//  not sure which I want yet.
			//trie.Put(match.ToIPv4(), foundmatch{idx: idx, addr: match, line: line})

			switch {
			case len(args.Ipstring) == 0: // no target IP address, just populate trie
				if match.IsIPv4() {
					v4_trie.Add(match.ToIPv4())
				} else if match.IsIPv6() {
					v6_trie.Add(match.ToIPv6())
				}
				continue // stop looking

			case args.Exact:
				if args.Ipaddr.Equal(match) {
					slog.Debug("FOUND MATCH", "match", match.String(), "ipaddr", args.Ipaddr.String(), "idx", idx, "line", line)
					matchlist = append(matchlist, foundmatch{Idx: idx, Addr: match, Line: line})
				}

			case args.Subnet:
				if match.Contains(args.Ipaddr) {
					slog.Debug("CONTAINS",
						"match", match.String(),
						"args", args.Ipaddr.String(),
						"idx", idx, "line", line)
					matchlist = append(matchlist, foundmatch{Idx: idx, Addr: match, Line: line})

				}
			case args.Longest:
				// just like Subnet but I need to track the longest match
				if match.Contains(args.Ipaddr) {
					plen := getHostbits(match)
					longest_subnet_seen = max(plen, longest_subnet_seen)
					matchlist = longest_subnets[longest_subnet_seen]

					longest_subnets[plen] = append(longest_subnets[plen], foundmatch{Idx: idx, Addr: match, Line: line}) // TODO
				}
			}
		} // for _, m := range matches
	}

	// now finish it off
	if args.Trie {

		// populate tries
		for _, m := range matchlist {
			if m.Addr.IsIPv4() {
				//v4_trie.Add(m.addr.ToIPv4())
				v4_trie.Put(m.Addr.ToIPv4(), m)

			} else if m.Addr.IsIPv6() {
				//v6_trie.Add(m.addr.ToIPv6())
				v6_trie.Put(m.Addr.ToIPv6(), m)
				//trie.Put(match.ToIPv4(), foundmatch{idx: idx, addr: match, line: line})
			}
		}

		// print tries
		if v4_trie.Size() > 0 {
			fmt.Println(v4_trie)
		}
		if v6_trie.Size() > 0 {
			fmt.Println(v6_trie)
		}

	} else { // not trie, just print matchlist
		for _, m := range matchlist {
			fmt.Printf("%v %v (%v)\n", m.Idx, m.Addr, m.Line)
		}
	}
	return nil
}
