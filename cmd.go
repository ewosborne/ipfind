package main

import (
	"bufio"
	"fmt"
	"os"
	"regexp"

	"github.com/seancfoley/ipaddress-go/ipaddr"
)

// TODO ipv6 - or is this done?  test it, at least.
// TODO handle err and nil, don't be lazy
// TODO logging
// TODO print entire routing table https://seancfoley.github.io/IPAddress/ipaddress.html#address-tries
var (
	ipv4AddressRE = regexp.MustCompile(`(\d{1,3}).(\d{1,3}).(\d{1,3}).(\d{1,3}(/\d{1,2})?)`)
	ipv6Regex     = regexp.MustCompile(`([0-9a-fA-F]{0,4}:){2,7}[0-9a-fA-F]{0,4}(/\d{1,3})?`)
	targetAFBits  = 32
	ipRE          = ipv4AddressRE
)

func get_input(args cliArgStruct) *bufio.Scanner {
	if len(args.inputFile) > 0 {
		file, _ := os.Open(args.inputFile)
		return bufio.NewScanner(file)
	} else {
		return bufio.NewScanner(os.Stdin)
	}
}

func ipcmd(args cliArgStruct) {

	fmt.Printf("args in ipcmd:%+v\n", args)

	if ipv6Regex.Match([]byte(args.ipaddr)) {
		targetAFBits = 128
		ipRE = ipv6Regex
	}

	targetIPAddr := ipaddr.NewIPAddressString(args.ipaddr)
	fmt.Printf("looking for %v\n", targetIPAddr)

	var scanner *bufio.Scanner
	var longest_mask_seen int

	longest_matches := make(map[int][]string)
	scanner = get_input(args)
	for scanner.Scan() {
		line := scanner.Text()
		fmt.Printf("\nline is #%s#\n", line)
		for _, elem := range ipRE.FindAllString(line, -1) {

			// turn regex match into ip object
			found := ipaddr.NewIPAddressString(elem).GetAddress()

			if args.exact {
				gpl := found.GetPrefixLen()
				plen := gpl.Len()
				if gpl == nil || plen == targetAFBits {
					fmt.Printf("EXACT MATCH %v\n", found)
				}
			} else if args.subnet {
				if found.Contains(targetIPAddr.GetAddress()) {
					fmt.Println("SUBNET CONTAINS", found, targetIPAddr)
				}
			} else if args.longest {
				// anything which matches args.subnet is a candidate for longest
				if found.Contains(targetIPAddr.GetAddress()) {
					m := found.GetPrefixLen().Len()
					longest_mask_seen = max(longest_mask_seen, m)
					if longest_mask_seen == m {
						gpl := found.GetPrefixLen()
						plen := gpl.Len()
						if gpl == nil {
							plen = targetAFBits
						}
						fmt.Println("LONGEST CANDIDATE", found, plen, targetIPAddr)

						// TODO: do I want elem or line here? maybe make this a flag?
						// TODO: if elem then maybe check for unique?
						// TODO: for now I want line, make --network-only a flag.
						//longest_matches[plen] = append(longest_matches[plen], elem)
						// fmt.Println("network only? args:", args.networkOnly)

						longest_matches[plen] = append(longest_matches[plen], line)

					}
				}
			}

			// if foundMatch {
			// 	longest_matches[plen] = append(longest_matches[plen], line)
			// }
		}
	}
	if args.longest {
		fmt.Printf("match map %v\n", longest_matches)
		fmt.Printf("best lines %v\n", longest_matches[longest_mask_seen])
	}
}
