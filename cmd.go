package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"regexp"

	"github.com/seancfoley/ipaddress-go/ipaddr"
)

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

func ipcmd(args cliArgStruct) {

	longestCache := make(map[int][]string)
	var outputData []string

	if args.debug {
		log.Printf("args in ipcmd:%+v\n", args)
	}

	targetIPAddr := ipaddr.NewIPAddressString(args.ipaddr).GetHostAddress()

	switch {
	case targetIPAddr.IsIPv4():
		afArgs = afArgsStruct{
			targetAFBits: ipaddr.IPv4BitCount,
			targetAF:     int(ipaddr.IPv4),
			ipRE:         ipv4Regex,
		}
	case targetIPAddr.IsIPv6():
		afArgs = afArgsStruct{
			targetAFBits: ipaddr.IPv6BitCount,
			targetAF:     int(ipaddr.IPv6),
			ipRE:         ipv6Regex,
		}
	default:
		log.Fatalf("couldn't figure out address family for %v\n", args.ipaddr)
	}

	scanner := get_input_scanner(args)

	// NOTES

	// read in a line
	// parse the line to []matches
	// if we're looking for exact, that's easy.  just walk matches, check for Equal().  print network or line.

	for scanner.Scan() {
		line := scanner.Text()
		if args.debug {
			log.Printf("SCANNED |%v|\n", line)
		}
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
						//fmt.Printf("%v\n", targetIPAddr)
						outputData = append(outputData, targetIPAddr.String())
					case false:
						//fmt.Printf("%v\n", line)
						outputData = append(outputData, line)

					}
					break // go scan next line
				} // ipob.Equal()
			} // for range ipaddrs
		case args.subnet, args.longest:
			/*
				the only difference between args.subnet and args.longest is that longest has a post-process.
				if args.subnet - just like args.exact except it's Contains() instead of Equal()
				if args.longest - just like args.subnet except we save the item in a map instead of printing it.
			*/
			for _, ip := range ipaddrs {
				// change ip to object
				ipobj := ipaddr.NewIPAddressString(ip).GetAddress()
				if args.debug {
					fmt.Printf("IPOBJ |%v|\n", ipobj)
				}
				//if ipobj.Contains(targetIPAddr) || ipobj.Equal(targetIPAddr) {
				if ipobj.Contains(targetIPAddr) {
					switch {
					case args.subnet:
						switch args.networkOnly {
						case true:
							//fmt.Printf("%v\n", ipobj)
							outputData = append(outputData, ipobj.String())

						case false:
							//fmt.Printf("%v\n", line)
							outputData = append(outputData, line)

						}
					case args.longest:
						// get ip mask
						var maskLength int
						a := ipobj.GetPrefixLen()
						if a == nil {
							// raw host gets nil prefix len rather than 32 or 128.  no idea why.
							maskLength = afArgs.targetAFBits
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

		if args.debug {
			fmt.Printf("longest seen |%v|\n", longestMask)
		}
		outputData = longestCache[longestMask]
	}

	for _, item := range outputData {
		fmt.Printf("\t%v\n", item)
	}

} // ipcmd
