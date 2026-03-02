package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
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

type newInputFile struct {
	IsStdin  bool
	Filename string
	Scanner  *bufio.Scanner
}

type readLine struct {
	Filename          string
	Idx               int
	Line              string
	MatchingIPStrings []string
	IsMatch           bool // default is false
}

func newipcmd(w io.Writer, args cliArgStruct) error {

	// null stuff
	log.Debug("starting newipcmd")

	var newInputFiles = []newInputFile{}

	switch len(args.InputFiles) {
	case 0:
		log.Debug("need to read in os.Stdin")
		newInputFiles = append(newInputFiles, newInputFile{IsStdin: true})
	default:
		tmp, err := getFilesFromArgs(args.InputFiles)
		if err != nil {
			log.Fatal("error", err)
		}
		for _, f := range tmp {
			newInputFiles = append(newInputFiles, newInputFile{IsStdin: false, Filename: f})
		}
	}

	// at this point newInputFiles is a list of names or stdin
	// TODO: for LPM, do I want to check LPM across all files together, or in each one?
	//  hrmm.
	for _, f := range newInputFiles {
		matchingLines := getMatchingLines(args, f)
		err := doReports(matchingLines, args, w)
		if err != nil {
			log.Error(err)
		}
	}
	return nil // todo
}

func doReports(matchingLines []*readLine, args cliArgStruct, w io.Writer) error {
	// now do some reporting
	switch {
	case args.Json:
		log.Debug("TODO need to log JSON")
		b, err := json.MarshalIndent(matchingLines, "", "  ")

		if err != nil {
			return err
		}
		fmt.Fprint(w, string(b))
		fmt.Fprint(w, "\n")

	case args.Trie:
		log.Print("TODO need to log trie")
		// create tries and then print them
		//  also need this for LPM I think.
	default:
		log.Debug("need to log text")
		for _, fLine := range matchingLines {
			if fLine.IsMatch {
				fmt.Fprintf(w, "%v:%v:%v\n", fLine.Filename, fLine.Idx, fLine.Line)
			}
		}
	}
	// TODO
	return nil
}

func getMatchingLines(args cliArgStruct, f newInputFile) []*readLine {

	fLines, err := readSingleFile(args, f)
	if err != nil {
		log.Fatal("error opening %v", f)
	}
	log.Debug("Read in %+v from %v", fLines, f.Filename)

	// at this point fLines is []*readLine, for each line in the file I just read

	for _, fLine := range fLines {
		switch {
		case args.Exact:
			//log.Print("need to match exactly")
			//log.Printf("working on line %v", fLine)
			for _, ip := range fLine.MatchingIPStrings {
				if !fLine.IsMatch {
					ipObj := ipaddr.NewIPAddressString(ip).GetAddress()
					//fmt.Printf("comparing %v %v\n", args.Ipaddr, ipObj)
					if ipObj.Equal(args.Ipaddr) {
						fLine.IsMatch = true
					}
				}
			}
		case args.Subnet:
			log.Debug("need to match subnet")
			//log.Printf("working on line %v", fLine)
			for _, ip := range fLine.MatchingIPStrings {
				if !fLine.IsMatch {
					ipObj := ipaddr.NewIPAddressString(ip).GetAddress()
					if args.Ipaddr.Contains(ipObj) {
						fLine.IsMatch = true
					}
				}
			}
		case args.Contains:
			log.Debug("need to match contains")
			//log.Debugf("working on line %v", fLine)
			for _, ip := range fLine.MatchingIPStrings {
				if !fLine.IsMatch {
					ipObj := ipaddr.NewIPAddressString(ip).GetAddress()
					if ipObj.Contains(args.Ipaddr) {
						fLine.IsMatch = true
					}
				}
			}
		case args.Longest:
			log.Fatal("longest match not implemented")
			// TODO
		}
	}

	var matchingLines = []*readLine{}
	for _, fLine := range fLines {
		if fLine.IsMatch {
			matchingLines = append(matchingLines, fLine)
		}
	}

	return matchingLines

}

func readSingleFile(args cliArgStruct, fileName newInputFile) ([]*readLine, error) {

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
		// TODO: only check the regex that
		if args.IsIPv4 {
			rl.MatchingIPStrings = get_ipv4_addresses_from_line(rl.Line, args.IPv4Regex)
		} else {
			rl.MatchingIPStrings = get_ipv6_addresses_from_line(rl.Line, args.IPv4Regex)
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
