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

func getReadCloser(fileName inputFile) (io.ReadCloser, error) {
	if fileName.IsStdin {
		return os.Stdin, nil
	} else {
		ifh, err := os.Open(fileName.Filename)
		if err != nil {
			return nil, err
		}
		return ifh, nil
	}
}

func matchesCondition(ipObject, targetIP *ipaddr.IPAddress, args cliArgStruct) bool {
	switch {
	case args.Exact:
		return ipObject.Equal(targetIP)
	case args.Contains:
		return ipObject.Contains(targetIP)
	case args.Subnet:
		return targetIP.Contains(ipObject)
	default:
		return false
	}
}

func processReader(r io.Reader, args cliArgStruct) (fileParseResultStruct, error) {
	fileParseResult := fileParseResultStruct{}
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		fileParseResult.Idx++ // line numbers start at 1
		line := scanner.Text()
		if len(line) == 0 {
			continue // skip blank lines
		}

		lineObj := &lineParseResult{Idx: fileParseResult.Idx, Line: line}
		lineObj.ipRegexMatches = lineObj.getRegexMatches(args.IPRegex, args.addressFamily)

		for _, m := range lineObj.ipRegexMatches {
			ipObject := ipaddr.NewIPAddressString(m).GetAddress()
			if ipObject == nil {
				continue
			}

			if matchesCondition(ipObject, args.Ipaddr, args) {
				if !lineObj.isMatch {
					fileParseResult.conditionMatchedLines = append(fileParseResult.conditionMatchedLines, lineObj)
					lineObj.isMatch = true
				}
				lineObj.conditionMatches = append(lineObj.conditionMatches, ipObject)
			}
		}
	}
	return fileParseResult, scanner.Err()
}

func formatResults(w io.Writer, fp fileParseResultStruct, args cliArgStruct) error {
	var lsm int
	if args.Longest {
		lsm = fp.getLongestSubnetMask()
	}

	switch {
	case args.Json:
		b, err := json.MarshalIndent(fp.getMinimumConditionMatchLines(lsm), "", "  ")
		if err != nil {
			return err
		}
		fmt.Fprintln(w, string(b))
	case args.Trie:
		for _, line := range fp.conditionMatchedLines {
			switch args.addressFamily {
			case 4:
				for _, pfx := range line.conditionMatches {
					fp.IPv4Trie.Add(pfx.ToIPv4())
				}
			case 6:
				for _, pfx := range line.conditionMatches {
					fp.IPv6Trie.Add(pfx.ToIPv6())
				}
			}
		}
		if fp.IPv4Trie.Size() > 0 {
			fmt.Fprintf(w, "%v\n", fp.IPv4Trie)
		}
		if fp.IPv6Trie.Size() > 0 {
			fmt.Fprintf(w, "%v\n", fp.IPv6Trie)
		}
	default:
		for _, line := range fp.getMinimumConditionMatchLines(lsm) {
			fmt.Fprintf(w, "%v:%v:%v\n", fp.Filename, line.Idx, line.Line)
		}
	}
	return nil
}

func ipcmd(w io.Writer, args cliArgStruct) error {
	log.Debug("starting ipcmd")

	inputFiles, err := getInputFilesOrStdin(args.InputFiles)
	if err != nil {
		return err
	}

	for _, fileName := range inputFiles {
		rc, err := getReadCloser(fileName)
		if err != nil {
			log.Errorf("error opening %v: %v", fileName.Filename, err)
			continue
		}

		fileParseResult, err := processReader(rc, args)
		if !fileName.IsStdin {
			rc.Close()
		}
		if err != nil {
			log.Errorf("error processing %v: %v", fileName.Filename, err)
			continue
		}

		fileParseResult.Filename = fileName.Filename
		if err := formatResults(w, fileParseResult, args); err != nil {
			log.Errorf("error formatting results for %v: %v", fileName.Filename, err)
		}
	}
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
