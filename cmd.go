package main

/* TODO
figure out what --longest really means
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
	Closer   io.Closer
}

type lineParseResult struct {
	Idx              int
	Line             string
	ipRegexMatches   []string
	conditionMatches []*ipaddr.IPAddress
	isMatch          bool
}

// func get_ip_addresses_from_line(ipre *regexp.Regexp, addressFamily int, line string) []string {
// 	var ret []string
// 	switch addressFamily {
// 	case 4:
// 		ret = ipre.FindAllString(line, -1)
// 	case 6:
// 		for _, m := range ipre.FindAllString(line, -1) {
// 			if strings.Contains(m, ":") {
// 				ret = append(ret, m)
// 			}
// 		}
// 	}
// 	return ret
// }

func (l *lineParseResult) getRegexMatches(ipre *regexp.Regexp, af int) []string {
	var ret []string
	switch af {
	case 4:
		ret = ipre.FindAllString(l.Line, -1)
	case 6:
		for _, m := range ipre.FindAllString(l.Line, -1) {
			if strings.Contains(m, ":") {
				ret = append(ret, m)
			}
		}
	}
	return ret
}

type fileParseResultStruct struct {
	Idx                   int                // line number
	regexMatchedLines     []*lineParseResult // lines that do something
	IPv4Trie              ipaddr.IPv4AddressTrie
	IPv6Trie              ipaddr.IPv6AddressTrie
	conditionMatchedLines []*lineParseResult
	Filename              string
}

func (fp *fileParseResultStruct) getLongestSubnetMask() int {
	var ret int
	for _, x := range fp.conditionMatchedLines {
		for _, y := range x.conditionMatches {
			ret = max(ret, y.GetPrefixLen().Len())
		}
	}
	return ret
}

func (fp *fileParseResultStruct) getMinimumConditionMatchLines(minimum int) []*lineParseResult {
	ret := []*lineParseResult{}
	for _, x := range fp.conditionMatchedLines {
	NextLine:
		for _, y := range x.conditionMatches {
			if y.GetPrefixLen().Len() >= minimum {
				ret = append(ret, x)
				break NextLine
			}
		}
	}
	return ret

}

// TODO I'm pretty sure there are smarter ways to do this
func getInputFilesOrStdin(files []string) ([]inputFile, error) {
	ret := []inputFile{}

	switch len(files) {
	case 0:
		ret = append(ret, inputFile{IsStdin: true})
	default:
		fileNames, err := getFileNamesFromArgs(files)
		if err != nil {
			log.Error("error getting filenames", err)
			return nil, err
		}
		for _, f := range fileNames {
			ret = append(ret, inputFile{IsStdin: false, Filename: f})
		}
	}
	return ret, nil

}

func getScannerFromFile(fileName inputFile) (*bufio.Scanner, io.Closer) {
	if fileName.IsStdin {
		return bufio.NewScanner(os.Stdin), nil
	} else {
		ifh, err := os.Open(fileName.Filename)
		if err != nil {
			log.Fatalf("error opening %v\n", fileName.Filename)
		}
		return bufio.NewScanner(ifh), ifh
	}
}

func ipcmd(w io.Writer, args cliArgStruct) error {

	// null stuff
	log.Debug("starting ipcmd")

	inputFiles, err := getInputFilesOrStdin(args.InputFiles)
	if err != nil {
		log.Error("error getting input filenames", err)
	}

	for _, fileName := range inputFiles {

		// create struct to hold whatever comes out of this file
		fileParseResult := fileParseResultStruct{Filename: fileName.Filename}

		log.Debug("need to load", "file", fileName)
		fileName.Scanner, fileName.Closer = getScannerFromFile(fileName)
		if fileName.Closer != nil { // nil means stdin
			defer fileName.Closer.Close()
		}

		for fileName.Scanner.Scan() {
			fileParseResult.Idx++ // line numbers start at 1
			line := fileName.Scanner.Text()
			if len(line) == 0 {
				continue // skip blank lines
			}

			lineObj := &lineParseResult{Idx: fileParseResult.Idx, Line: line}
			lineObj.ipRegexMatches = lineObj.getRegexMatches(args.IPRegex, args.addressFamily)
			log.Debugf("regex matches are %+v", lineObj.ipRegexMatches)

			// now see if there are condtion matches
			// append each to foundLine.conditionMatches
			// TODO: need to find some way to only print a line once even if it has multiple regex or condition matches
			for _, m := range lineObj.ipRegexMatches {
				log.Debugf("looking at %+v for condition match", m)

				// turn it into an IP address
				ipObject := ipaddr.NewIPAddressString(m).GetAddress()
				log.Debugf("ip address object of regex match:%+v", ipObject)

				// check conditions
				switch {
				case args.Exact:
					log.Debug("comparing %v %v for Exact", ipObject, args.Ipaddr)
					if ipObject.Equal(args.Ipaddr) {
						if !lineObj.isMatch { // only store the line if it's not already a match
							fileParseResult.conditionMatchedLines = append(fileParseResult.conditionMatchedLines, lineObj)
							lineObj.isMatch = true
						}
						lineObj.conditionMatches = append(lineObj.conditionMatches, ipObject)
					} else {
						log.Debug("does not contain")
					}
				case args.Contains:
					log.Debugf("does %v contain %v", ipObject, args.Ipaddr)
					if ipObject.Contains(args.Ipaddr) {
						if !lineObj.isMatch {
							fileParseResult.conditionMatchedLines = append(fileParseResult.conditionMatchedLines, lineObj)
							lineObj.isMatch = true
						}
						lineObj.conditionMatches = append(lineObj.conditionMatches, ipObject)
					} else {
						log.Debug("does not contain")
					}
				case args.Subnet:
					log.Debugf("does %t contain %t", args.Ipaddr, ipObject)
					if args.Ipaddr.Contains(ipObject) {
						if !lineObj.isMatch {
							fileParseResult.conditionMatchedLines = append(fileParseResult.conditionMatchedLines, lineObj)
							lineObj.isMatch = true
						}
						lineObj.conditionMatches = append(lineObj.conditionMatches, ipObject)
					} else {
						log.Debug("does not contain")
					}
				}
			} // for each ipMatch
			fileName.Closer.Close()
		} // for filename.Scanner

		// TODO build longestConditionMatchedLines and maybe feed that to json and text output but not trie.

		switch {
		case args.Json:
			// TODO make this work with text too,
			var lsm int
			if args.Longest {
				lsm = fileParseResult.getLongestSubnetMask()
				log.Debug("lsm is %v", lsm)
			}
			b, err := json.MarshalIndent(fileParseResult.getMinimumConditionMatchLines(lsm), "", "  ")
			if err != nil {
				log.Error(err)
			}
			fmt.Fprintln(w, string(b))
		case args.Trie:
			for _, line := range fileParseResult.conditionMatchedLines {
				switch args.addressFamily {
				case 4:
					for _, pfx := range line.conditionMatches {
						fileParseResult.IPv4Trie.Add(pfx.ToIPv4())
					}
				case 6:
					for _, pfx := range line.conditionMatches {
						fileParseResult.IPv6Trie.Add(pfx.ToIPv6())
					}
				}
			}
			if fileParseResult.IPv4Trie.Size() > 0 {
				fmt.Fprintf(w, "%v\n", fileParseResult.IPv4Trie)
			}
			if fileParseResult.IPv6Trie.Size() > 0 {
				fmt.Fprintf(w, "%v\n", fileParseResult.IPv4Trie)
			}
		default:
			var lsm int
			if args.Longest {
				lsm = fileParseResult.getLongestSubnetMask()
				log.Debug("lsm is %v", lsm)
			}
			log.Debug("printing text")
			for _, line := range fileParseResult.getMinimumConditionMatchLines(lsm) {
				fmt.Fprintf(w, "%v:%v:%v\n", fileParseResult.Filename, line.Idx, line.Line)
			}
		}
	} // for fileName range inputFiles
	return nil
} // func ipcmd

func getFileNamesFromArgs(inputFiles []string) ([]string, error) {
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
