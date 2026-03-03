package main

import (
	"bufio"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/charmbracelet/log"
	"github.com/seancfoley/ipaddress-go/ipaddr"
)

var (
	ipv4Regex_withSlash = regexp.MustCompile(`(\d{1,3})\.(\d{1,3})\.(\d{1,3})\.(\d{1,3}(/\d{1,2}))`)
	ipv6Regex_withSlash = regexp.MustCompile(`([:0-9a-fA-F]{2,39}(/[0-9]{1,3}))`)
	ipv4Regex_noSlash   = regexp.MustCompile(`(\d{1,3})\.(\d{1,3})\.(\d{1,3})\.(\d{1,3}(/\d{1,2})?)`)
	ipv6Regex_noSlash   = regexp.MustCompile(`([:0-9a-fA-F]{2,39}(/[0-9]{1,3})?)`)
)

type inputFile struct {
	IsStdin  bool
	Filename string
	Scanner  *bufio.Scanner
}

type foundLine struct {
	Filename         string
	Idx              int
	Line             string
	ipRegexMatches   []string
	conditionMatches []*ipaddr.IPAddress
	isMatch          bool // default is false
}

func ipcmd(w io.Writer, args cliArgStruct) error {

	// null stuff
	log.Debug("starting ipcmd")

	var inputFiles = []inputFile{}

	switch len(args.InputFiles) {
	case 0:
		log.Debug("need to read in os.Stdin")
		inputFiles = append(inputFiles, inputFile{IsStdin: true})
	default:
		tmp, err := getFilesFromArgs(args.InputFiles)
		if err != nil {
			log.Fatal("error", err)
		}
		for _, f := range tmp {
			inputFiles = append(inputFiles, inputFile{IsStdin: false, Filename: f})
		}
	}

	// at this point inputFiles is a list of names or stdin

	/*
		open and process each file. leave room for goroutines.
	*/

	for _, fileName := range inputFiles {
		var idx int                 // line number
		var foundLines []*foundLine // lines that do something
		var IPv4Trie = ipaddr.IPv4AddressTrie{}
		var IPv6Trie = ipaddr.IPv6AddressTrie{}

		log.Info("need to load", "file", fileName)
		if fileName.IsStdin {
			fileName.Scanner = bufio.NewScanner(os.Stdin)
		} else {
			ifh, err := os.Open(fileName.Filename)
			if err != nil {
				log.Fatalf("error opening %v\n", fileName.Filename)
			}
			defer ifh.Close()

			fileName.Scanner = bufio.NewScanner(ifh)
		}

		for fileName.Scanner.Scan() {
			idx++ // line numbers start at 1
			line := fileName.Scanner.Text()
			if len(line) == 0 {
				continue // skip blank lines
			}
			// parse the line into its bits and fill out struct
			ipMatches := get_ip_addresses_from_line(args.IPRegex, line)
			log.Debugf("regex matches are %+v", ipMatches)
			foundLine := &foundLine{Idx: idx, ipRegexMatches: ipMatches, Line: line}
			//log.Printf("found ip regex match %+v", foundLine)
			foundLines = append(foundLines, foundLine)

			// now see if there are condtion matches
			// append each to foundLine.conditionMatches and set foundline.isMatch
			for _, m := range ipMatches {
				log.Printf("looking at %+v for condition match", m)

				// turn it into an IP address
				ipObject := ipaddr.NewIPAddressString(m).GetAddress()
				log.Printf("ip address object of regex match:%+v", ipObject)

				// do I populate the trie here just in case I need it?
				// sure
				switch args.addressFamily {
				case 4:
					IPv4Trie.Add(ipObject.ToIPv4())
					//log.Printf("ipv4 trie %v", IPv4Trie)

				case 6:
					IPv6Trie.Add(ipObject.ToIPv6())
					//log.Printf("ipv6 trie %v", IPv6Trie)

				}

				switch {
				case args.Exact:
					log.Printf("comparing %v %v for Exact", ipObject, args.Ipaddr)
					if ipObject.Equal(args.Ipaddr) {
						log.Printf("found equal match %v", ipObject)
						foundLine.conditionMatches = append(foundLine.conditionMatches, ipObject)
					}
				case args.Contains:
					log.Printf("comparing %v %v for Contains", ipObject, args.Ipaddr)
					if ipObject.Contains(args.Ipaddr) {
						log.Printf("found contains match %v", ipObject)
						foundLine.conditionMatches = append(foundLine.conditionMatches, ipObject)
					}
				case args.Subnet:
					log.Printf("comparing %v %v for Subnet", ipObject, args.Ipaddr)
					if args.Ipaddr.Contains(ipObject) {
						log.Printf("found subnet match %v", ipObject)
						foundLine.conditionMatches = append(foundLine.conditionMatches, ipObject)
					}
				}
			}

		} // for filename.Scanner
		log.Print("need to show matched line here but scope thing")
	} // for fileName range inputFiles
	return nil
} // func ipcmd

// func doReports(matchingLines []*foundLine, args cliArgStruct, w io.Writer, lsm int) error {
// 	newML := []*readLineExternal{}
// 	newInternalML := []*foundLine{}

// 	// have a filter on matchingLines which looks for LSM matches
// 	// TODO

// 	if args.Longest {
// 		for _, tmp := range matchingLines {
// 			for _, m := range tmp.ConditionMatches {
// 				if ipaddr.NewIPAddressString(m).GetAddress().GetPrefixLen().Len() == lsm {
// 					newInternalML = append(newInternalML, tmp)
// 				}
// 			}
// 		}
// 	}

// 	if len(newInternalML) > 0 {
// 		matchingLines = newInternalML
// 	}

// 	if !args.Debug {
// 		for _, tmp := range matchingLines {
// 			newML = append(newML, &readLineExternal{Idx: tmp.Idx, Filename: tmp.Filename, Line: tmp.Line})
// 		}
// 	}

// 	var b []byte
// 	var err error
// 	switch {
// 	case args.Json:
// 		if args.Debug {
// 			b, err = json.MarshalIndent(matchingLines, "", "  ")
// 			if err != nil {
// 				return err
// 			}
// 			fmt.Fprint(w, string(b))
// 			fmt.Fprint(w, "\n")
// 		} else {
// 			b, err = json.MarshalIndent(newML, "", "  ")
// 			if err != nil {
// 				return err
// 			}
// 			fmt.Fprint(w, string(b))
// 			fmt.Fprint(w, "\n")
// 		}

// 	case args.Trie:
// 		//  also need tries for LPM I think.
// 		IPv4Trie, IPv6Trie := getIPTries(args, matchingLines)
// 		if IPv4Trie.Size() > 0 {
// 			fmt.Println(matchingLines[0].Filename)
// 			fmt.Println(IPv4Trie)
// 		}
// 		if IPv6Trie.Size() > 0 {
// 			fmt.Println(matchingLines[0].Filename)
// 			fmt.Println(IPv6Trie)
// 		}
// 	default:
// 		log.Debug("need to log text")
// 		for _, fLine := range matchingLines {
// 			if fLine.IsMatch {
// 				fmt.Fprintf(w, "%v:%v:%v\n", fLine.Filename, fLine.Idx, fLine.Line)
// 			}
// 		}
// 	}
// 	return nil
// }

// func getIPTries(args cliArgStruct, matchingLines []*foundLine) (ipaddr.IPv4AddressTrie, ipaddr.IPv6AddressTrie) {
// 	IPv4Trie := ipaddr.IPv4AddressTrie{}
// 	IPv6Trie := ipaddr.IPv6AddressTrie{}

// 	switch args.addressFamily {
// 	case 4:
// 		for _, match := range matchingLines {
// 			for _, line := range match.conditionMatches {
// 				IPv4Trie.Add(ipaddr.NewIPAddressString(line).GetAddress().ToIPv4())
// 			}
// 		}

// 	case 6:
// 		for _, match := range matchingLines {
// 			for _, line := range match.conditionMatches {
// 				IPv6Trie.Add(ipaddr.NewIPAddressString(line).GetAddress().ToIPv6())
// 			}
// 		}
// 	}
// 	return IPv4Trie, IPv6Trie

// }

// func getMatchingLines(args cliArgStruct, f inputFile) ([]*foundLine, int) {

// 	fLines, err := readSingleFile(args, f)
// 	if err != nil {
// 		log.Fatal("error opening %v", f)
// 	}
// 	log.Debug("Read in %+v from %v", fLines, f.Filename)
// 	var lsm int
// 	// at this point fLines is []*readLine, for each line in the file I just read

// 	for _, fLine := range fLines {
// 		switch {

// 		case args.Exact:
// 			//log.Print("need to match exactly")
// 			//log.Printf("working on line %v", fLine)
// 			for _, ip := range fLine.ipRegexMatches {
// 				ipObj := ipaddr.NewIPAddressString(ip).GetAddress()
// 				//fmt.Printf("comparing %v %v\n", args.Ipaddr, ipObj)
// 				if ipObj.Equal(args.Ipaddr) {
// 					fLine.isMatch = true
// 					fLine.conditionMatches = append(fLine.conditionMatches, ip)
// 				}
// 			}
// 		case args.Subnet:
// 			log.Debug("need to match subnet")
// 			//log.Printf("working on line %v", fLine)
// 			for _, ip := range fLine.ipRegexMatches {
// 				ipObj := ipaddr.NewIPAddressString(ip).GetAddress()
// 				// if ipObj.GetPrefixLen().Len() > 21 && ipObj.GetPrefixLen().Len() < 27 {
// 				// 	continue
// 				// }
// 				if args.Ipaddr.Contains(ipObj) {
// 					fLine.isMatch = true
// 					fLine.conditionMatches = append(fLine.conditionMatches, ip)
// 				}
// 			}
// 		case args.Contains:
// 			log.Debug("need to match contains")
// 			//log.Debugf("working on line %v", fLine)
// 			for _, ip := range fLine.ipRegexMatches {
// 				ipObj := ipaddr.NewIPAddressString(ip).GetAddress()
// 				if ipObj.Contains(args.Ipaddr) {
// 					fLine.isMatch = true
// 					fLine.conditionMatches = append(fLine.conditionMatches, ip)
// 				}
// 			}

// 			//	log.Fatal("LSM not supported yet")
// 			// dump it all in a trie and get
// 			/*
// 				target := ipaddr.NewIPAddressString("192.168.1.150").GetAddress().ToIPv4()
// 				match := trie.LongestPrefixMatch(target)
// 			*/
// 		}
// 	}
// 	if args.Longest {
// 		// this works but what I think I really want is to populate matchingLines
// 		// with things with the matching ip address length.
// 		Ipv4Trie, Ipv6Trie := getIPTries(args, fLines)

// 		if args.IsIPv4 && Ipv4Trie.Size() > 0 {
// 			//fmt.Println("IPv4LSM", Ipv4Trie.LongestPrefixMatch(args.Ipaddr.ToIPv4()).GetPrefixLen().Len())
// 			lsm = max(lsm, Ipv4Trie.LongestPrefixMatch(args.Ipaddr.ToIPv4()).GetPrefixLen().Len())
// 		}

// 		if args.IsIPv6 && Ipv6Trie.Size() > 0 {
// 			//fmt.Println("IPv6LSM", Ipv6Trie.LongestPrefixMatch(args.Ipaddr.ToIPv6()).GetPrefixLen().Len())
// 			lsm = max(lsm, Ipv6Trie.LongestPrefixMatch(args.Ipaddr.ToIPv6()).GetPrefixLen().Len())
// 		}

// 	}

// 	var matchingLines = []*foundLine{}
// 	for _, fLine := range fLines {
// 		if fLine.isMatch {
// 			matchingLines = append(matchingLines, fLine)
// 		}
// 	}

// 	return matchingLines, lsm

// }

// func readSingleFile(args cliArgStruct, fileName inputFile) ([]*foundLine, error) {

// 	if fileName.IsStdin {
// 		fileName.Scanner = bufio.NewScanner(os.Stdin)
// 	} else {
// 		ifh, err := os.Open(fileName.Filename)
// 		if err != nil {
// 			log.Fatalf("error opening %v\n", fileName.Filename)
// 		}
// 		defer ifh.Close()

// 		fileName.Scanner = bufio.NewScanner(ifh)
// 	}

// 	log.Debug("Need to read in", "file", fileName)

// 	// now process the file

// 	var idx = 0
// 	var linesInFile = []*foundLine{}
// 	for fileName.Scanner.Scan() {
// 		idx++
// 		line := fileName.Scanner.Text()
// 		rl := foundLine{Idx: idx, Line: line, Filename: fileName.Filename}

// 		if len(line) == 0 {
// 			continue // optimization for blank lines.
// 		}

// 		// now find all ipv4 regex matches and ipv6 regex matches from the line
// 		if args.IsIPv4 {
// 			rl.ipRegexMatches = get_ipv4_addresses_from_line(rl.Line, args.IPv4Regex)
// 		} else {
// 			rl.ipRegexMatches = get_ipv6_addresses_from_line(rl.Line, args.IPv4Regex)
// 		}

// 		linesInFile = append(linesInFile, &rl)
// 	}
// 	return linesInFile, nil
// }

func get_ip_addresses_from_line(ipre *regexp.Regexp, line string) []string {
	return ipre.FindAllString(line, -1)
}

func get_ipv4_addresses_from_line(line string, ipv4Regex *regexp.Regexp) []string {
	return get_ip_addresses_from_line(ipv4Regex, line)
}

func get_ipv6_addresses_from_line(line string, ipv6Regex *regexp.Regexp) []string {

	// hack because the regex is getting messy but this seems ok.
	ret := []string{}
	for _, m := range get_ip_addresses_from_line(ipv6Regex, line) {
		if strings.Contains(m, ":") {
			ret = append(ret, m)
		}
	}
	return ret

}

func getFilesFromArgs(inputFiles []string) ([]string, error) {
	var ret []string
	for _, ifile := range inputFiles {
		err := filepath.WalkDir(ifile, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}

			if !d.IsDir() {
				ret = append(ret, path)
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	return ret, nil
}
