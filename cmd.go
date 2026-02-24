package main

/*
	TODO
	clean up output
	handle line vs network print match

	'contains' flag: pass 4.0.0.0/8 as an arg with -c/--contains, show me all subnets which this contains.


	rewrite file handling in goroutines, or at least prep for that
	   one goroutine per file (or if stdin, that's just one file)
	   roll it all up into a per-file map at the top level

	think about how this interacts with trie support
*/

import (
	"bufio"
	"fmt"
	"io/fs"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/seancfoley/ipaddress-go/ipaddr"
)

var (
	ipv4Regex = regexp.MustCompile(`(\d{1,3}).(\d{1,3}).(\d{1,3}).(\d{1,3}(/\d{1,2})?)`)
	ipv6Regex = regexp.MustCompile(`([:0-9a-fA-F]{2,39}(/[0-9]{1,3})?)`)
)

func getFilesFromArgs(inputFiles []string) []string {
	var ret []string
	for _, ifile := range inputFiles {
		err := filepath.WalkDir(ifile, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				fmt.Println("error", err)
				os.Exit(1)
			}

			if !d.IsDir() {
				ret = append(ret, path)
			}
			return nil
		})
		if err != nil {
			log.Fatal("error", err)
		}
	}
	return ret

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

type inputFile struct {
	Filename string
	IsStdin  bool
	Scanner  *bufio.Scanner
}

type dataMatch struct {
	Filename  string
	Idx       int
	MatchLine string
	MatchIPs  []*ipaddr.IPAddress
}

func process_single_file(args cliArgStruct, file inputFile) {
	// process a file
	// return something?

	var idx int = 0
	var longest_subnet_seen = 0
	var fileMatches = []dataMatch{}
	var v4_line_matches = []*ipaddr.IPAddress{}
	var v6_line_matches = []*ipaddr.IPAddress{}

	scanner := file.Scanner

	for scanner.Scan() {

		var foundMatchingIP bool = false
		var matchingIPList = []*ipaddr.IPAddress{}
		idx++ // start counting at 1
		line := scanner.Text()
		slog.Debug("processed", "idx", idx, "line", line)

		if args.V4 {
			v4_line_matches = get_ipv4_addresses_from_line(line)
		} else if args.V6 {
			v6_line_matches = get_ipv6_addresses_from_line(line)
		}

		// note well that this is _regex matches_, not _criteria matches_.
		line_matches := slices.Concat(v4_line_matches, v6_line_matches)
		if len(line_matches) == 0 {
			continue // no matches on this line
		}

		// ok now I have matches.
		// maybe save everything here?
		// save idx, filename, line, match info

		// TODO: this isn't valid because I don't want to save fileMatches until I've done the case matching

		// do -e, -s, -l, -c stuff.  trie to come later.

		for _, line_match := range line_matches {
			slog.Debug("comparing", "line", line_match.String())
			switch {
			case args.Exact:
				if args.Ipaddr.Equal(line_match) {
					foundMatchingIP = true
					matchingIPList = append(matchingIPList, line_match)
					//fmt.Println("EXACT MATCH", line_match)
				}
				// OK do I save the matches somewhere?
				// need to save both line and prefix, or maybe just pick between them
			case args.Contains:
				if args.Ipaddr.Contains(line_match) {
					foundMatchingIP = true
					matchingIPList = append(matchingIPList, line_match)

					//fmt.Println("ARG", args.Ipaddr, "CONTAINS", line_match)
				}
			case args.Subnet:
				if line_match.Contains(args.Ipaddr) {
					foundMatchingIP = true
					matchingIPList = append(matchingIPList, line_match)

					//fmt.Println(line_match, "CONTAINS ARG", args.Ipaddr)

				}
			case args.Longest:
				//fmt.Println("TODO longest")
				if line_match.Contains(args.Ipaddr) {
					plen := getHostbits(line_match)
					longest_subnet_seen = max(longest_subnet_seen, plen)
					if plen == longest_subnet_seen {
						fmt.Println(" LONGEST SO FAR", longest_subnet_seen, line_match)
					}
				}
				// just run args.Subnet logic and remember the longest match seen
				//  need to handle multiple matchesS
			}
			if foundMatchingIP {
				m := dataMatch{Filename: file.Filename, Idx: idx, MatchLine: line, MatchIPs: matchingIPList}
				fileMatches = append(fileMatches, m)
			}
		}
		foundMatchingIP = false
		matchingIPList = matchingIPList[:0] // clear the list out
	}

	// haven't done args.Longest yet
	// but print matches
	if len(fileMatches) > 0 {
		for _, entry := range fileMatches {
			fmt.Printf("file:%v idx:%v line:%v matches:%v\n", entry.Filename, entry.Idx, entry.MatchLine, entry.MatchIPs)
		}
	}
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

	var iFiles = []inputFile{}

	// build list of files or stdin
	// TODO: preserve filename here for later reporting.
	if len(args.InputFiles) == 0 {
		slog.Debug("reading from stdin")
		iFiles = append(iFiles, inputFile{Filename: "os.Stdin", IsStdin: true, Scanner: bufio.NewScanner(os.Stdin)})
	} else {
		slog.Debug("ifiles are", "ifiles", args.InputFiles)
		// InputFiles is a slice. each element in the slice is either a file or a directory name.
		files := getFilesFromArgs(args.InputFiles)

		slices.Sort(files)

		slog.Debug("files to walk are", "file", files)
		for _, file := range files {
			tmp, _ := os.Open(file)
			iFiles = append(iFiles, inputFile{IsStdin: false, Filename: file, Scanner: bufio.NewScanner(tmp)})

		}
	}

	// walk stuff
	for _, i := range iFiles {
		fmt.Printf("need to process file %v\n", i.Filename)
		// launch a goroutine per file maybe?  for now just do it in order
		process_single_file(args, i)
	}

	return nil
} // func ipcmd
