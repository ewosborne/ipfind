package main

import (
	"fmt"
	"log/slog"

	"github.com/seancfoley/ipaddress-go/ipaddr"
)

type dataMatch struct {
	Filename  string
	Idx       int
	MatchLine string
	MatchIPs  []*ipaddr.IPAddress
}

func process_single_file(args cliArgStruct, file inputFile) []dataMatch {
	// process a file
	// return a list of dataMatch objects

	var idx int = 0
	var v4_trie = ipaddr.IPv4AddressTrie{}
	var v6_trie = ipaddr.IPv6AddressTrie{}

	scanner := file.Scanner
	scannedFile := []dataMatch{}
	ret := []dataMatch{}

	// scan each line
	for scanner.Scan() {
		idx++
		line := scanner.Text()                                                                  // get the line
		dm, ok := scanLine(args, dataMatch{Idx: idx, MatchLine: line, Filename: file.Filename}) // get dataMatch items, one per line
		if !ok {                                                                                // no matches on this line
			continue
		}

		// now do stuff with a line we just got
		scannedFile = append(scannedFile, dm)

	}

	// now the file has been read.  we have a dataMatch struct per useful line
	// walk scannedFile and do the Exact/Contains/Subnet/Longest thing
	//  saving the results in a matchlist somewhere I guess.
	//  am I just screening the dataMatch lines again and saving _them_ directly in the matchlist? let's try that.


	/* TODO another way to do this loop is
		walk each line
		for each match on each line
		 add match to trie and add the match object
	
	this would only keep the latest match line by default, need to think through what I want to keep
	*/
	for _, line := range scannedFile {
		slog.Debug("considering", "line", line)

	NextLine:
		for _, ip := range line.MatchIPs {
			switch {
			case args.Exact:
				if ip.Equal(args.Ipaddr) {
					ret = append(ret, line)
					break NextLine
				}

			case args.Subnet:
				if ip.Contains(args.Ipaddr) {
					ret = append(ret, line)
					break NextLine  // TODO do I want to break here?  or do the whole line?
				}
			case args.Contains:
				if args.Ipaddr.Contains(ip) {
					ret = append(ret, line)
					break NextLine // TODO same question as Subnet.  when do I want to break?
				}

			case args.Longest:
				if ip.Contains(args.Ipaddr) {
					if ip.IsIPv4() {
						v4_trie.Add(ip.ToIPv4())
					}

					if ip.IsIPv6() {
						v6_trie.Add(ip.ToIPv6())
					}
					// just dump the whole thing in a trie.  v4 only for now
					// loop every ip address
				}

			}
		}
	}

	// finish up
	if v4_trie.Size() > 0 {
		fmt.Println(file.Filename, "V4 TRIE", v4_trie)
		fmt.Println("LPM", v4_trie.LongestPrefixMatch(args.Ipaddr.ToIPv4()))
		fmt.Printf("\n\n\n")
	}

	if && v6_trie.Size() > 0 {
		fmt.Println(file.Filename, "V6 TRIE", v6_trie)
		fmt.Println("LPM", v6_trie.LongestPrefixMatch(args.Ipaddr.ToIPv6()))
		fmt.Printf("\n\n\n")
	}
	return ret
}
