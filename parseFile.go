package main

import (
	"encoding/json"

	"github.com/charmbracelet/log"
	"github.com/seancfoley/ipaddress-go/ipaddr"
)

type dataMatch struct {
	Filename  string
	Idx       int
	MatchLine string
	MatchIPs  []*ipaddr.IPAddress
}

// need this because the debugger can't figure out MatchIPs otherwise
func (dm dataMatch) MarshalJSON() ([]byte, error) {
	var ipStrings []string
	for _, ip := range dm.MatchIPs {
		if ip != nil {
			ipStrings = append(ipStrings, ip.String())
		}
	}
	return json.Marshal(struct {
		Filename  string   `json:"filename"`
		Idx       int      `json:"idx"`
		MatchLine string   `json:"match_line"`
		MatchIPs  []string `json:"match_ips"`
	}{
		Filename:  dm.Filename,
		Idx:       dm.Idx,
		MatchLine: dm.MatchLine,
		MatchIPs:  ipStrings,
	})
}

func process_single_file(args cliArgStruct, file inputFile) ([]dataMatch, ipaddr.IPv4AddressTrie, ipaddr.IPv6AddressTrie) {
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
		idx++                                                                                   // first line is 1, not 0.
		line := scanner.Text()                                                                  // get the line
		dm, ok := scanLine(args, dataMatch{Idx: idx, MatchLine: line, Filename: file.Filename}) // get dataMatch items, one per line
		if !ok {                                                                                // no matches on this line
			continue
		}

		log.Debug("scanned line", "sl", dm)

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
		log.Debug("considering", "line", line)

		//NextLine:
		for _, ip := range line.MatchIPs {
			switch {
			case args.Exact:
				if ip.Equal(args.Ipaddr) {
					ret = append(ret, line)
					if ip.IsIPv4() {
						v4_trie.Add(ip.ToIPv4())
					}

					if ip.IsIPv6() {
						v6_trie.Add(ip.ToIPv6())
					}
					//break NextLine
				}

			case args.Subnet:
				if args.Ipaddr.Contains(ip) {
					ret = append(ret, line)
					if ip.IsIPv4() {
						v4_trie.Add(ip.ToIPv4())
					}

					if ip.IsIPv6() {
						v6_trie.Add(ip.ToIPv6())
					}
					//break NextLine // TODO do I want to break here?  or do the whole line?
				}
			case args.Contains, args.Longest:
				if ip.Contains(args.Ipaddr) {
					ret = append(ret, line)
					if ip.IsIPv4() {
						v4_trie.Add(ip.ToIPv4())
					}

					if ip.IsIPv6() {
						v6_trie.Add(ip.ToIPv6())
					}
					//break NextLine // TODO same question as Subnet.  when do I want to break?
				}

				// TODO: I need to capture maches in ret, not just tries, like everything else.
				// case args.Longest:
				// 	if ip.IsIPv4() {
				// 		v4_trie.Add(ip.ToIPv4())
				// 	}

				// 	if ip.IsIPv6() {
				// 		v6_trie.Add(ip.ToIPv6())
				// 	}
			}
		}
	}

	return ret, v4_trie, v6_trie
}
