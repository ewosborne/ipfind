package main

import (
	"bufio"
	"encoding/json"
	"fmt"
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

type readLine struct {
	Filename         string
	Idx              int
	Line             string
	IPRegexMatches   []string
	ConditionMatches []string
	IsMatch          bool // default is false
}

type readLineExternal struct {
	Filename string
	Idx      int
	Line     string
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
	// TODO: for LPM, do I want to check LPM across all files together, or in each one?
	//  hrmm.
	// maybe all of it?
	for _, f := range inputFiles {
		matchingLines, lsm := getMatchingLines(args, f)
		err := doReports(matchingLines, args, w, lsm)
		if err != nil {
			log.Error(err)
		}
	}
	return nil // todo
}

func doReports(matchingLines []*readLine, args cliArgStruct, w io.Writer, lsm int) error {
	newML := []*readLineExternal{}
	newInternalML := []*readLine{}

	// have a filter on matchingLines which looks for LSM matches
	// TODO

	if args.Longest {
		for _, tmp := range matchingLines {
			for _, m := range tmp.ConditionMatches {
				if ipaddr.NewIPAddressString(m).GetAddress().GetPrefixLen().Len() == lsm {
					newInternalML = append(newInternalML, tmp)
				}
			}
		}
	}

	if len(newInternalML) > 0 {
		matchingLines = newInternalML
	}

	if !args.Debug {
		for _, tmp := range matchingLines {
			newML = append(newML, &readLineExternal{Idx: tmp.Idx, Filename: tmp.Filename, Line: tmp.Line})
		}
	}

	var b []byte
	var err error
	switch {
	case args.Json:
		if args.Debug {
			b, err = json.MarshalIndent(matchingLines, "", "  ")
			if err != nil {
				return err
			}
			fmt.Fprint(w, string(b))
			fmt.Fprint(w, "\n")
		} else {
			b, err = json.MarshalIndent(newML, "", "  ")
			if err != nil {
				return err
			}
			fmt.Fprint(w, string(b))
			fmt.Fprint(w, "\n")
		}

	case args.Trie:
		//  also need tries for LPM I think.
		IPv4Trie, IPv6Trie := getIPTries(args, matchingLines)
		if IPv4Trie.Size() > 0 {
			fmt.Println(matchingLines[0].Filename)
			fmt.Println(IPv4Trie)
		}
		if IPv6Trie.Size() > 0 {
			fmt.Println(matchingLines[0].Filename)
			fmt.Println(IPv6Trie)
		}
	default:
		log.Debug("need to log text")
		for _, fLine := range matchingLines {
			if fLine.IsMatch {
				fmt.Fprintf(w, "%v:%v:%v\n", fLine.Filename, fLine.Idx, fLine.Line)
			}
		}
	}
	return nil
}

func getIPTries(args cliArgStruct, matchingLines []*readLine) (ipaddr.IPv4AddressTrie, ipaddr.IPv6AddressTrie) {
	IPv4Trie := ipaddr.IPv4AddressTrie{}
	IPv6Trie := ipaddr.IPv6AddressTrie{}

	switch {
	case args.IsIPv4:
		for _, match := range matchingLines {
			for _, line := range match.ConditionMatches {
				IPv4Trie.Add(ipaddr.NewIPAddressString(line).GetAddress().ToIPv4())
			}
		}

	case args.IsIPv6:
		for _, match := range matchingLines {
			for _, line := range match.ConditionMatches {
				IPv6Trie.Add(ipaddr.NewIPAddressString(line).GetAddress().ToIPv6())
			}
		}
	}
	return IPv4Trie, IPv6Trie

}

func getMatchingLines(args cliArgStruct, f inputFile) ([]*readLine, int) {

	fLines, err := readSingleFile(args, f)
	if err != nil {
		log.Fatal("error opening %v", f)
	}
	log.Debug("Read in %+v from %v", fLines, f.Filename)
	var lsm int
	// at this point fLines is []*readLine, for each line in the file I just read

	for _, fLine := range fLines {
		switch {

		case args.Exact:
			//log.Print("need to match exactly")
			//log.Printf("working on line %v", fLine)
			for _, ip := range fLine.IPRegexMatches {
				ipObj := ipaddr.NewIPAddressString(ip).GetAddress()
				//fmt.Printf("comparing %v %v\n", args.Ipaddr, ipObj)
				if ipObj.Equal(args.Ipaddr) {
					fLine.IsMatch = true
					fLine.ConditionMatches = append(fLine.ConditionMatches, ip)
				}
			}
		case args.Subnet:
			log.Debug("need to match subnet")
			//log.Printf("working on line %v", fLine)
			for _, ip := range fLine.IPRegexMatches {
				ipObj := ipaddr.NewIPAddressString(ip).GetAddress()
				// if ipObj.GetPrefixLen().Len() > 21 && ipObj.GetPrefixLen().Len() < 27 {
				// 	continue
				// }
				if args.Ipaddr.Contains(ipObj) {
					fLine.IsMatch = true
					fLine.ConditionMatches = append(fLine.ConditionMatches, ip)
				}
			}
		case args.Contains, args.Longest:
			log.Debug("need to match contains")
			//log.Debugf("working on line %v", fLine)
			for _, ip := range fLine.IPRegexMatches {
				ipObj := ipaddr.NewIPAddressString(ip).GetAddress()
				if ipObj.Contains(args.Ipaddr) {
					fLine.IsMatch = true
					fLine.ConditionMatches = append(fLine.ConditionMatches, ip)
				}
			}

			//	log.Fatal("LSM not supported yet")
			// dump it all in a trie and get
			/*
				target := ipaddr.NewIPAddressString("192.168.1.150").GetAddress().ToIPv4()
				match := trie.LongestPrefixMatch(target)
			*/
		}
	}
	if args.Longest {
		// this works but what I think I really want is to populate matchingLines
		// with things with the matching ip address length.
		Ipv4Trie, Ipv6Trie := getIPTries(args, fLines)

		if args.IsIPv4 && Ipv4Trie.Size() > 0 {
			//fmt.Println("IPv4LSM", Ipv4Trie.LongestPrefixMatch(args.Ipaddr.ToIPv4()).GetPrefixLen().Len())
			lsm = max(lsm, Ipv4Trie.LongestPrefixMatch(args.Ipaddr.ToIPv4()).GetPrefixLen().Len())
		}

		if args.IsIPv6 && Ipv6Trie.Size() > 0 {
			//fmt.Println("IPv6LSM", Ipv6Trie.LongestPrefixMatch(args.Ipaddr.ToIPv6()).GetPrefixLen().Len())
			lsm = max(lsm, Ipv6Trie.LongestPrefixMatch(args.Ipaddr.ToIPv6()).GetPrefixLen().Len())
		}

	}

	var matchingLines = []*readLine{}
	for _, fLine := range fLines {
		if fLine.IsMatch {
			matchingLines = append(matchingLines, fLine)
		}
	}

	return matchingLines, lsm

}

func readSingleFile(args cliArgStruct, fileName inputFile) ([]*readLine, error) {

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

	log.Debug("Need to read in", "file", fileName)

	// now process the file

	var idx = 0
	var linesInFile = []*readLine{}
	for fileName.Scanner.Scan() {
		idx++
		line := fileName.Scanner.Text()
		rl := readLine{Idx: idx, Line: line, Filename: fileName.Filename}

		if len(line) == 0 {
			continue // optimization for blank lines.
		}

		// now find all ipv4 regex matches and ipv6 regex matches from the line
		if args.IsIPv4 {
			rl.IPRegexMatches = get_ipv4_addresses_from_line(rl.Line, args.IPv4Regex)
		} else {
			rl.IPRegexMatches = get_ipv6_addresses_from_line(rl.Line, args.IPv4Regex)
		}

		linesInFile = append(linesInFile, &rl)
	}
	return linesInFile, nil
}

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
