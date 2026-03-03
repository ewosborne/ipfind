package main

/* TODO
figure out what --longest really means
json and trie outputs
etc
*/

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

type foundLineType struct {
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
		TODO: clean all this up. But it feels ok.
		TODO: tests
		TODO: set up writer instead of printf?
	*/

	for _, fileName := range inputFiles {
		var idx int                     // line number
		var foundLines []*foundLineType // lines that do something
		var IPv4Trie = ipaddr.IPv4AddressTrie{}
		var IPv6Trie = ipaddr.IPv6AddressTrie{}
		var matchedLines []*foundLineType

		log.Debug("need to load", "file", fileName)
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

		//var foundLine *foundLineType
		var lineObj *foundLineType
		for fileName.Scanner.Scan() {
			idx++ // line numbers start at 1
			line := fileName.Scanner.Text()
			if len(line) == 0 {
				continue // skip blank lines
			}
			// parse the line into its bits and fill out struct
			ipMatches := get_ip_addresses_from_line(args.IPRegex, args.addressFamily, line)
			log.Debugf("regex matches are %+v", ipMatches)
			lineObj = &foundLineType{Filename: fileName.Filename, Idx: idx, ipRegexMatches: ipMatches, Line: line}
			foundLines = append(foundLines, lineObj)

			// now see if there are condtion matches
			// append each to foundLine.conditionMatches and set foundline.isMatch
			for _, m := range ipMatches {
				log.Debugf("looking at %+v for condition match", m)

				// turn it into an IP address
				ipObject := ipaddr.NewIPAddressString(m).GetAddress()
				log.Debugf("ip address object of regex match:%+v", ipObject)

				switch {
				case args.Exact:
					//log.Printf("comparing %v %v for Exact", ipObject, args.Ipaddr)
					if ipObject.Equal(args.Ipaddr) {
						//log.Printf("found equal match %v:%v", ipObject, lineObj)
						lineObj.isMatch = true
						lineObj.conditionMatches = append(lineObj.conditionMatches, ipObject)
						matchedLines = append(matchedLines, lineObj)

						switch args.addressFamily {
						case 4:
							IPv4Trie.Add(ipObject.ToIPv4())
							//log.Printf("ipv4 trie %v", IPv4Trie)

						case 6:
							IPv6Trie.Add(ipObject.ToIPv6())
							//log.Printf("ipv6 trie %v", IPv6Trie)

						}
					} else {
						log.Debug("does not contain")
					}
				case args.Contains:
					log.Debugf("does %v contain %v", ipObject, args.Ipaddr)
					if ipObject.Contains(args.Ipaddr) {
						log.Debug("FOUND contains match %v", ipObject)
						lineObj.isMatch = true
						lineObj.conditionMatches = append(lineObj.conditionMatches, ipObject)
						matchedLines = append(matchedLines, lineObj)

						switch args.addressFamily {
						case 4:
							IPv4Trie.Add(ipObject.ToIPv4())
						case 6:
							IPv6Trie.Add(ipObject.ToIPv6())
						}
					} else {
						log.Debug("does not contain")
					}
				case args.Subnet:
					log.Debugf("does %t contain %t", args.Ipaddr, ipObject)
					if args.Ipaddr.Contains(ipObject) {
						log.Debugf("FOUND subnet match %v", ipObject)
						lineObj.isMatch = true
						lineObj.conditionMatches = append(lineObj.conditionMatches, ipObject)
						matchedLines = append(matchedLines, lineObj)

						switch args.addressFamily {
						case 4:
							IPv4Trie.Add(ipObject.ToIPv4())
							//log.Printf("ipv4 trie %v", IPv4Trie)

						case 6:
							IPv6Trie.Add(ipObject.ToIPv6())
							//log.Printf("ipv6 trie %v", IPv6Trie)

						}
					} else {
						log.Debug("does not contain")
					}
				}
			} // for each ipMatch
		} // for filename.Scanner

		switch {
		case args.Json:
			b, err := json.MarshalIndent(matchedLines, "", "  ")
			if err != nil {
				log.Error(err)
			}
			fmt.Fprintln(w, string(b))
		case args.Trie:
			log.Info("trie not supported")
			if IPv4Trie.Size() > 0 {
				fmt.Fprintf(w, "%v\n", IPv4Trie)
			}
			if IPv6Trie.Size() > 0 {
				fmt.Fprintf(w, "%v\n", IPv6Trie)

			}
		default:
			//log.Info("printing text")
			for _, line := range matchedLines {
				fmt.Fprintf(w, "%v:%v:%v\n", line.Filename, line.Idx, line.Line)
			}
		}
	} // for fileName range inputFiles
	return nil
} // func ipcmd

func get_ip_addresses_from_line(ipre *regexp.Regexp, addressFamily int, line string) []string {
	var ret []string
	switch addressFamily {
	case 4:
		ret = ipre.FindAllString(line, -1)
	case 6:
		for _, m := range ipre.FindAllString(line, -1) {
			if strings.Contains(m, ":") {
				ret = append(ret, m)
			}
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
