package main

/*
	TODO
	handle -t with no specified ip address
	clean up output
	handle line vs network print match
	ipv6 support, need better regexp or a different approach

*/

import (
	"bufio"
	"fmt"
	"log/slog"
	"os"
	"regexp"
	"strings"

	"github.com/seancfoley/ipaddress-go/ipaddr"
)

var (
	ipv4Regex = regexp.MustCompile(`(\d{1,3}).(\d{1,3}).(\d{1,3}).(\d{1,3}(/\d{1,2})?)`)
	//ipv6Regex = regexp.MustCompile(`[^0-9a-fA-F]*([:0-9a-fA-F]{2,39}(/[0-9]{1,3})?)[^0-9a-fA-F]*`)
	ipv6Regex = regexp.MustCompile(`([:0-9a-fA-F]{2,39}(/[0-9]{1,3})?)`)

	//afArgs    afArgsStruct
)

type foundmatch struct {
	idx  int
	addr *ipaddr.IPAddress
	line string
}

func (f foundmatch) String() string {
	return fmt.Sprintf("fm idx: %v  addr:%v  line(%v)", f.idx, f.addr, f.line)
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
		if match.IsIPv4() {
			plen = 32
		} else if match.IsIPv6() {
			plen = 128
		}
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

		// matches := append(v4_matches, v6_matches...)
		// matches := slices.Concat(v4_matches, v6_matches)
		matches := append(v4_matches, v6_matches...)
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

			// if I have no target IP just dump it all into a trie and continue
			if len(args.Ipstring) == 0 {
				// 	trie.Add(match.ToIPv4())
				if match.IsIPv4() {
					v4_trie.Add(match.ToIPv4()) // TODO handle ipv6
				} else if match.IsIPv6() {
					v6_trie.Add(match.ToIPv6())
				}
				continue
			}

			switch {
			case len(args.Ipstring) == 0: // no target IP address, just populate trie
				// trie.Add(match.ToIPv4()) // TODO: need to handle ipv6 here
				if match.IsIPv4() {
					v4_trie.Add(match.ToIPv4()) // TODO handle ipv6
				} else if match.IsIPv6() {
					v6_trie.Add(match.ToIPv6())
				}
				continue // stop looking

			case args.Exact:
				if args.Ipaddr.Equal(match) {
					//fmt.Println("FOUND MATCH", match.String(), args.Ipaddr.String(), idx, line)
					// TODO now what? need a consistent output format.
					matchlist = append(matchlist, foundmatch{idx: idx, addr: match, line: line})
				}

			case args.Subnet:
				if match.Contains(args.Ipaddr) {
					slog.Debug("CONTAINS", "match", match.String(), "args", args.Ipaddr.String(), "idx", idx, "line", line)
					// TODO now what?
					//  * add to some list of matches?  track both line and address?  TBD.
					matchlist = append(matchlist, foundmatch{idx: idx, addr: match, line: line})

				}
			case args.Longest:
				// just like Subnet but I need to track the longest match
				if match.Contains(args.Ipaddr) {
					plen := getHostbits(match)
					longest_subnet_seen = max(plen, longest_subnet_seen)

					//fmt.Println("LM plen", match, plen)
					longest_subnets[plen] = append(longest_subnets[plen], foundmatch{idx: idx, addr: match, line: line}) // TODO
				}

			case args.Trie:
				fmt.Println("TODO trie")

			}
		}
	}

	// deal with args.Longest second pass here
	if args.Longest {
		matchlist = longest_subnets[longest_subnet_seen]
	}
	//fmt.Println("MATCHLIST")

	for _, m := range matchlist {
		// maybe make the trie here?
		if args.Trie {
			if m.addr.IsIPv4() {
				v4_trie.Add(m.addr.ToIPv4()) // TODO handle ipv6
			} else if m.addr.IsIPv6() {
				v6_trie.Add(m.addr.ToIPv6())
			}
		} else {
			fmt.Println(m)
		}
	}
	if args.Trie {
		if v4_trie.Size() > 0 {
			fmt.Println(v4_trie)
		}
		if v6_trie.Size() > 0 {

			fmt.Println(v6_trie)
		}
	}
	return nil
}
