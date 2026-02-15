package main

import (
	"bufio"
	"fmt"
	"os"
	"regexp"

	"github.com/seancfoley/ipaddress-go/ipaddr"
)

// TODO ipv6
// TODO print entire routing table https://seancfoley.github.io/IPAddress/ipaddress.html#address-tries
var ipv4AddressRE = regexp.MustCompile(`(\d{1,3}).(\d{1,3}).(\d{1,3}).(\d{1,3}(/\d{1,2})?)`)

func ipcmd(args cliArgStruct) {

	fmt.Printf("args in ipcmd:%+v\n", args)

	findIPv4Addr := ipaddr.NewIPAddressString(args.ipaddr)
	fmt.Printf("looking for %v\n", findIPv4Addr)

	var scanner *bufio.Scanner
	var longest_mask_seen int
	longest_matches := make(map[int][]string)
	if len(args.inputFile) > 0 {
		file, _ := os.Open(args.inputFile)
		scanner = bufio.NewScanner(file)
	} else {
		scanner = bufio.NewScanner(os.Stdin)
	}
	for scanner.Scan() {
		line := scanner.Text()
		fmt.Printf("\nline is #%s#\n", line)
		for _, elem := range ipv4AddressRE.FindAllString(line, -1) {

			// turn regex match into ip object
			found := ipaddr.NewIPAddressString(elem).GetAddress()

			if args.exact {
				gpl := found.GetPrefixLen()
				plen := gpl.Len()
				if gpl == nil || plen == 32 {
					fmt.Printf("EXACT MATCH %v\n", found)
				}
			} else if args.subnet {
				if found.Contains(findIPv4Addr.GetAddress()) {
					fmt.Println("SUBNET CONTAINS", found, findIPv4Addr)
				}
			} else if args.longest {
				fmt.Println("TODO: longest")

				// anything which matches args.subnet is a candidate for longest
				if found.Contains(findIPv4Addr.GetAddress()) {
					m := found.GetPrefixLen().Len()
					longest_mask_seen = max(longest_mask_seen, m)
					if longest_mask_seen == m {
						gpl := found.GetPrefixLen()
						plen := gpl.Len()
						if gpl == nil {
							plen = 32
						}
						fmt.Println("LONGEST CANDIDATE", found, plen, findIPv4Addr)

						// TODO: do I want elem or line here? maybe make this a flag?
						// TODO: if elem then maybe check for unique?
						//longest_matches[plen] = append(longest_matches[plen], elem)
						longest_matches[plen] = append(longest_matches[plen], line)
					}
				}
			}

		}
	}
	if args.longest {
		fmt.Printf("match map %v", longest_matches)
		fmt.Printf("best lines %v", longest_matches[longest_mask_seen])
	}
}
