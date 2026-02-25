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
					break NextLine
				}
			case args.Contains:
				if args.Ipaddr.Contains(ip) {
					ret = append(ret, line)
					break NextLine
				}

			case args.Longest:
				fmt.Println("TODO args.Longest")
				break NextLine // need to walk every match, dump in trie.
			}
		}
	}
	return ret
}

// 	// ok now I have matches.
// 	// maybe save everything here?
// 	// save idx, filename, line, match info

// 	// TODO: this isn't valid because I don't want to save fileMatches until I've done the case matching

// 	// do -e, -s, -l, -c stuff.  trie to come later.

// 	for _, line_match := range line_matches {
// 		slog.Debug("comparing", "line", line_match.String(), "args", args.Ipaddr.String())

// 		// so do I just build the trie here?
// 		v4_trie.Add(line_match.ToIPv4())
// 		switch {
// 		case args.Exact:
// 			if args.Ipaddr.Equal(line_match) {
// 				foundMatchingIP = true
// 				matchingIPList = append(matchingIPList, line_match)
// 				slog.Debug("E-MATCH", "EXACT MATCH", line_match.String())
// 			}
// 			// OK do I save the matches somewhere?
// 			// need to save both line and prefix, or maybe just pick between them
// 		case args.Contains:
// 			if args.Ipaddr.Contains(line_match) {
// 				foundMatchingIP = true
// 				matchingIPList = append(matchingIPList, line_match)
// 				slog.Debug("C-MATCH", "ARG", args.Ipaddr.String(), "CONTAINS", line_match.String())
// 			}
// 		case args.Subnet:
// 			if line_match.Contains(args.Ipaddr) {
// 				foundMatchingIP = true
// 				matchingIPList = append(matchingIPList, line_match)
// 				slog.Debug("S-MATCH", "ARG", line_match.String(), "CONTAINS", args.Ipaddr.String())

// 			}
// 		case args.Longest:
// 			// TODO: how do I match the lines with the longest prefix?  gonna need two passes.
// 			//v4_trie.Add(line_match.ToIPv4()) // use v4_trie.LongestPrefixMatch later
// 		}
// 		if foundMatchingIP {
// 			m := dataMatch{Filename: file.Filename, Idx: idx, MatchLine: line, MatchIPs: matchingIPList}
// 			fileMatches = append(fileMatches, m)
// 		}
// 	}
// 	foundMatchingIP = false
// 	matchingIPList = matchingIPList[:0] // clear the list out
// }

// if args.Trie {
// 	fmt.Println(v4_trie.ElementsContaining(args.Ipaddr.ToIPv4()))
// }

// if args.Longest {
// 	fmt.Println(v4_trie.LongestPrefixMatch(args.Ipaddr.ToIPv4()))
// }

// haven't done args.Longest yet
// but print matches

// if len(fileMatches) > 0 {
// 	for _, entry := range fileMatches {
// 		fmt.Printf("file:%v idx:%v line:%v matches:%v\n", entry.Filename, entry.Idx, entry.MatchLine, entry.MatchIPs)
// 	}
// }
