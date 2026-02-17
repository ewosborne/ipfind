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
	ipv4Regex = regexp.MustCompile(`(\d{1,3}).(\d{1,3}).(\d{1,3}).(\d{1,3}(/\d{1,2})?)`)
	ipv6Regex = regexp.MustCompile(`([:0-9a-fA-F]{2,39}(/[0-9]{1,3})?)`)
	afArgs    afArgsStruct
)

func get_input_scanner(args cliArgStruct) *bufio.Scanner {
	if len(args.inputFile) > 0 {
		file, _ := os.Open(args.inputFile)
		return bufio.NewScanner(file)
	} else {
		return bufio.NewScanner(os.Stdin)
	}
}

// returns true if there's a subnet anywhere in matches which is equal to the target
func has_matching_subnet(matches []string, target *ipaddr.IPAddress, regex *regexp.Regexp) bool {
	for _, match := range matches {
		matchIP := ipaddr.NewIPAddressString(match).GetAddress()
		if matchIP.Equal(target) {
			return true
		}
	}
	return false
}

// returns true if any match in matches is a subnet which contains target
func has_containing_subnet(matches []string, target *ipaddr.IPAddress, regex *regexp.Regexp) bool {
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

// returns both the longest matching masklen in a set of matches and the thing it matched against
// I forget why I have that second one but I think it's to do with printing the line vs. just the network
func get_longest_line_subnet(matches []string, targetIPAddr *ipaddr.IPAddress) (int, string, error) {
	var longest_line_subnet int
	longest_line_subnet = -1
	var longest_line_subnet_network string
	// walks matches, looks for and returns masklen of longest line

	fmt.Printf("%v matches: %v\n", len(matches), matches)

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

	longestCache := make(map[int][]string)

	fmt.Printf("args in ipcmd:%+v\n", args)

	targetIPAddr := ipaddr.NewIPAddressString(args.ipaddr).GetHostAddress()

	if targetIPAddr.IsIPv4() {
		afArgs = afArgsStruct{
			targetAFBits: ipaddr.IPv4BitCount,
			targetAF:     int(ipaddr.IPv4),
			ipRE:         ipv4Regex,
		}
	} else if targetIPAddr.IsIPv6() {
		afArgs = afArgsStruct{
			targetAFBits: ipaddr.IPv6BitCount,
			targetAF:     int(ipaddr.IPv6),
			ipRE:         ipv6Regex,
		}
	}

	scanner := get_input_scanner(args)

	// NOTES

	// read in a line
	// parse the line to []matches
	// if we're looking for exact, that's easy.  just walk matches, check for Equal().  print network or line.

	for scanner.Scan() {
		line := scanner.Text()
		//fmt.Printf("SCANNED |%v|\n", line)
		ipaddrs := afArgs.ipRE.FindAllString(line, -1)
		if ipaddrs == nil { // this line has no ip addresses at all
			continue
		}

		switch {
		case args.exact:
			// check each ipaddr to see if it's an exact match for what we're looking for
			for _, ip := range ipaddrs {
				// change ip to object
				ipobj := ipaddr.NewIPAddressString(ip).GetHostAddress()
				if ipobj.Equal(targetIPAddr) {
					switch args.networkOnly {
					case true:
						fmt.Printf("%v\n", targetIPAddr)
					case false:
						fmt.Printf("%v\n", line)
					}
					break // go scan next line
				} // ipob.Equal()
			} // for range ipaddrs
		case args.subnet, args.longest:
			/* I want to do this extensibly.  the only difference between args.subnet and args.longest is that longest has a post-process.
			if args.subnet - just like args.exact except it's Contains() instead of Equal()
			if args.longest - just like args.subnet except we save the item in a map instead of printing it.
			*/
			for _, ip := range ipaddrs {
				// change ip to object
				ipobj := ipaddr.NewIPAddressString(ip).GetAddress()
				//fmt.Printf("IPOBJ |%v|\n", ipobj)
				//if ipobj.Contains(targetIPAddr) || ipobj.Equal(targetIPAddr) {
				if ipobj.Contains(targetIPAddr) {
					switch {
					case args.subnet:
						switch args.networkOnly {
						case true:
							fmt.Printf("%v\n", ipobj)
						case false:
							fmt.Printf("%v\n", line)
						}
					case args.longest:
						// get ip mask
						var maskLength int
						a := ipobj.GetPrefixLen()
						if a == nil {
							// raw host
							maskLength = afArgs.targetAFBits // no mask is nil not 32 or 128.  no idea why.
						} else {
							maskLength = a.Len()
						}

						//fmt.Printf("for %v len is %v\n", ipobj, maskLength)

						switch args.networkOnly {
						case true:
							//fmt.Printf("%v\n", targetIPAddr)
							longestCache[maskLength] = append(longestCache[maskLength], ipobj.String())
						case false:
							//fmt.Printf("%v\n", line)
							longestCache[maskLength] = append(longestCache[maskLength], line)

						}
					} // outer switch
				}
			} // for range ipaddrs
		} // case args.subnet, args.longest
	} // for scanner.Scan()

	if args.longest {
		// find longest mask seen
		var longestMask int
		for key := range longestCache {
			longestMask = max(longestMask, key)
		}

		//fmt.Printf("longest seen |%v|\n", longestMask)
		for _, item := range longestCache[longestMask] {
			fmt.Printf("\t%v\n", item)
		}

	}
} // ipcmd
// TODO here is the special processing for args.longest

// if we're looking for subenet match, almost as easy. just walk matches, check for Contains().  print network or line.
// in both cases it's OK to stop after the first match

/*
 longest is special.

 make one pass over the whole input as if it was subnet match, collecting masklen and (network or line)
 dump it all in a map of k=masklen, v=list of returned items
 then report on that map by finding the longest key and printing each line
 so it's sort of a special case of subnet matching
*/

// // args.longest has to make two passes over input or do something bespoke and clever
// if args.longest {
// 	//fmt.Println("entering args.longest special path")

// 	var outer_longest int
// 	longest_cache := make(map[int][]string)

// 	/* basically this is a two-pass version of args.subnet
// 	first do the args.subnet thing and walk all input lines
// 	ditch the ones with no match
// 	store the entire line of the others in a map, k=masklen, v=[]lines
// 	*/

// 	for scanner.Scan() {
// 		line := scanner.Text()
// 		matches := afArgs.ipRE.FindAllString(line, -1)
// 		if matches == nil {
// 			continue
// 		}

// 		// find longest subnet in all matches
// 		longest_line_subnet, longest_line_string, _ := get_longest_line_subnet(matches, targetIPAddr) // TODO handle error

// 		// hack
// 		if args.networkOnly {
// 			longest_cache[longest_line_subnet] = append(longest_cache[longest_line_subnet], longest_line_string)
// 		} else if !args.networkOnly {
// 			longest_cache[longest_line_subnet] = append(longest_cache[longest_line_subnet], line)
// 		}
// 		outer_longest = max(outer_longest, longest_line_subnet)
// 	}

// 	//fmt.Printf("longest match seen in entire input is %v\n", outer_longest)
// 	//fmt.Printf("longest match seen is %v\n", outer_longest)
// 	for _, tmp := range longest_cache[outer_longest] {
// 		// TODO this is where I handle -n maybe?
// 		fmt.Printf(" %v\n", tmp)
// 	}
// } else {

// 	// the other ones just make one pass over the whole thing
// 	for scanner.Scan() {
// 		line := scanner.Text()

// 		// find all IP addresses in the line
// 		matches := afArgs.ipRE.FindAllString(line, -1)

// 		// if there are no ip addresses in the line, done
// 		if matches == nil {
// 			continue
// 		}

// 		// then there are four conditions I handle here

// 		switch {
// 		case args.exact:
// 			if has_matching_subnet(matches, targetIPAddr, afArgs.ipRE) {
// 				switch args.networkOnly {
// 				case true:
// 					// print the first instance of an exact match in the line
// 					fmt.Printf("%v\n", matches[0])
// 				case false:
// 					// print the whole line
// 					fmt.Printf("%v\n", line)
// 				}
// 			}

// 		case args.subnet:
// 			if has_containing_subnet(matches, targetIPAddr, afArgs.ipRE) {
// 				switch args.networkOnly {
// 				case true:
// 					// print the first instance of an exact match in the line
// 					fmt.Printf("%v\n", matches[0])
// 				case false:
// 					// print the whole line
// 					fmt.Printf("%v\n", line)
// 				}
// 			}
// 		}
// 	}
// }
