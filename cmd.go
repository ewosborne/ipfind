package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"regexp"

	"github.com/seancfoley/ipaddress-go/ipaddr"
)

// TODO handle err and nil, don't be lazy
// TODO logging and debugs
// TODO test cases
// TODO print entire routing table https://seancfoley.github.io/IPAddress/ipaddress.html#address-tries
// TODO dump everything into a trie and use it?
// TODO get rid of this enormously bloated ipaddress library? or make better use of it.
// TODO tab completion, however that works

type afArgsStruct struct {
	targetAF, targetAFBits int
	ipRE                   *regexp.Regexp
}

var (
	ipv4Regex = regexp.MustCompile(`(\d{1,3}).(\d{1,3}).(\d{1,3}).(\d{1,3}(/\d{1,2})?[^.])`)
	ipv6Regex = regexp.MustCompile(`([:0-9a-fA-F]{2,39}(/[0-9]{1,3})?)`)
	afArgs    afArgsStruct
)

func get_input(args cliArgStruct) *bufio.Scanner {
	if len(args.inputFile) > 0 {
		file, _ := os.Open(args.inputFile)
		return bufio.NewScanner(file)
	} else {
		return bufio.NewScanner(os.Stdin)
	}
}

func has_matching_subnet(matches []string, target *ipaddr.IPAddress, regex *regexp.Regexp) bool {
	//fmt.Printf("%v matches", len(matches))
	for _, match := range matches {
		matchIP := ipaddr.NewIPAddressString(match).GetAddress()
		//fmt.Printf("\n comparing %T %v %T %v\n", target, target, matchIP, matchIP)
		if matchIP.Equal(target) {
			//fmt.Println(" MATCH!")
			return true
		}
	}
	//fmt.Printf("FALSE\n!")
	return false
}

func has_containing_subnet(matches []string, target *ipaddr.IPAddress, regex *regexp.Regexp) bool {
	// returns true if any match in matches is a subnet which contains target
	for _, match := range matches {
		matchIP := ipaddr.NewIPAddressString(match).GetAddress()
		//fmt.Printf("\n comparing %T %v %T %v\n", target, target, matchIP, matchIP)
		if matchIP.Contains(target) {
			//fmt.Println(" MATCH!")
			return true
		}
	}
	//fmt.Printf("FALSE\n!")
	return false
}

func get_longest_line_subnet(matches []string, targetIPAddr *ipaddr.IPAddress) (int, string, error) {
	var longest_line_subnet int
	longest_line_subnet = -1
	var longest_line_subnet_network string
	// walks matches, looks for and returns masklen of longest line

	// NOTE WELL: match is a regex match and doesn't necessarily contain the ip address in question!
	for _, match := range matches {
		// turn match to address
		// get its masklen
		// update longest_line_subnet
		tmp := ipaddr.NewIPAddressString(match).GetAddress()
		if !tmp.Contains(targetIPAddr) {
			continue
		}
		mlen := ipaddr.NewIPAddressString(match).GetAddress().GetPrefixLen().Len()
		// mlen comes out to 0 if it's a host address, I don't like that

		if mlen == 0 {
			mlen = afArgs.targetAFBits
		}

		//fmt.Printf("longest check: %v\n", match)
		if mlen > longest_line_subnet {
			longest_line_subnet = mlen
			longest_line_subnet_network = match
		}
	}

	if longest_line_subnet >= 0 {
		//fmt.Println("returning", longest_line_subnet)
		return longest_line_subnet, longest_line_subnet_network, nil // should only get called if it'll match TODO better error handling
	} else {
		return 0, "", errors.New("couldn't find any matches in this line")
	}
}

func ipcmd(args cliArgStruct) {

	// longestCache := make(map[int][]string)

	//fmt.Printf("args in ipcmd:%+v\n", args)

	switch ipv6Regex.MatchString(args.ipaddr) {
	case true:
		afArgs = afArgsStruct{
			targetAFBits: 128,
			targetAF:     6,
			ipRE:         ipv6Regex,
		}
	case false:
		afArgs = afArgsStruct{
			targetAFBits: 32,
			targetAF:     4,
			ipRE:         ipv4Regex,
		}
	}

	targetIPAddr := ipaddr.NewIPAddressString(args.ipaddr).GetAddress()

	scanner := get_input(args)

	// NOTES

	// args.longest has to make two passes over input or do something bespoke and clever
	if args.longest {
		//fmt.Println("entering args.longest special path")

		var outer_longest int
		longest_cache := make(map[int][]string)

		/* basically this is a two-pass version of args.subnet
		first do the args.subnet thing and walk all input lines
		ditch the ones with no match
		store the entire line of the others in a map, k=masklen, v=[]lines
		*/

		for scanner.Scan() {
			line := scanner.Text()
			matches := afArgs.ipRE.FindAllString(line, -1)
			if matches == nil {
				continue
			}

			// find longest subnet in all matches
			longest_line_subnet, longest_line_string, _ := get_longest_line_subnet(matches, targetIPAddr) // TODO handle error

			// hack
			if args.networkOnly {
				longest_cache[longest_line_subnet] = append(longest_cache[longest_line_subnet], longest_line_string)
			} else if !args.networkOnly {
				longest_cache[longest_line_subnet] = append(longest_cache[longest_line_subnet], line)
			}
			outer_longest = max(outer_longest, longest_line_subnet)
		}

		//fmt.Printf("longest match seen in entire input is %v\n", outer_longest)
		//fmt.Printf("longest match seen is %v\n", outer_longest)
		for _, tmp := range longest_cache[outer_longest] {
			// TODO this is where I handle -n maybe?
			fmt.Printf(" %v\n", tmp)
		}
	} else {

		// the other ones just make one pass over the whole thing
		for scanner.Scan() {
			line := scanner.Text()

			// find all IP addresses in the line
			matches := afArgs.ipRE.FindAllString(line, -1)

			// if there are no ip addresses in the line, done
			if matches == nil {
				continue
			}

			// then there are four conditions I handle here

			switch {
			case args.exact:
				if has_matching_subnet(matches, targetIPAddr, afArgs.ipRE) {
					switch {
					case args.networkOnly:
						// I guess?  TODO
						fmt.Printf("%v\n", matches[0])
					case !args.networkOnly:
						fmt.Printf("%v\n", line)
					}
				}

			case args.subnet:
				if has_containing_subnet(matches, targetIPAddr, afArgs.ipRE) {
					switch {
					case args.networkOnly:
						// I guess?  TODO
						fmt.Printf("%v\n", matches[0])
					case !args.networkOnly:
						fmt.Printf("%v\n", line)
					}
				}
			}
		}
	}
}
