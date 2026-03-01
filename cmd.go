package main

/* this is the main command loop.  it figures out what files need to be parsed and calls
   parseFile for each one.  concurrency-ready but 1 worker to start.
*/
import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"

	"github.com/charmbracelet/log"
	"github.com/seancfoley/ipaddress-go/ipaddr"
)

type inputFile struct {
	Filename string
	IsStdin  bool
	Scanner  *bufio.Scanner
}

func displayOutput(w io.Writer, args cliArgStruct, matchedLines []dataMatch, ipv4Trie ipaddr.IPv4AddressTrie, ipv6Trie ipaddr.IPv6AddressTrie) {

	// three output formats: text, json, trie

	if args.Json {
		b, err := json.MarshalIndent(matchedLines, "", "  ")
		if err != nil {
			log.Error(err)
		}
		fmt.Fprint(w, string(b))
	} else if args.Trie {
		if ipv4Trie.Size() > 0 {
			fmt.Fprintln(w, ipv4Trie)
		}

		if ipv6Trie.Size() > 0 {
			fmt.Fprintln(w, ipv6Trie)
		}
	} else { // default mode is per line
		// if args.Longest {
		// 	if ipv4Trie.Size() > 0 {
		// 		fmt.Fprintln(w, "IPv4 LPM", ipv4Trie.LongestPrefixMatch(args.Ipaddr.ToIPv4()))
		// 	}
		// 	if ipv6Trie.Size() > 0 {
		// 		fmt.Fprintln(w, "IPv6 LPM", ipv6Trie.LongestPrefixMatch(args.Ipaddr.ToIPv6()))
		// 	}
		// }
		for _, m := range matchedLines {
			log.Debugf("%v:%v:%v:%v\n", m.Filename, m.Idx, m.MatchLine, m.MatchIPs)
			fmt.Fprintf(w, "%v:%v:%v\n", m.Filename, m.Idx, m.MatchLine)
		}
	}
}

func ipcmd(w io.Writer, args cliArgStruct) error {

	iFiles, err := get_inputFiles(args)
	if err != nil {
		return fmt.Errorf("failed to get input files: %w", err)
	}

	// walk stuff.  this needs a rewrite with channels and a worker pool
	for _, i := range iFiles {
		//fmt.Printf("need to process file %v\n", i.Filename)
		// launch a goroutine per file maybe?  for now just do it in order

		if i.IsStdin {
			i.Scanner = bufio.NewScanner(os.Stdin)
		} else {
			ifh, err := os.Open(i.Filename)
			if err != nil {
				panic("lazy exit here")
			}
			defer ifh.Close()
			i.Scanner = bufio.NewScanner(ifh)

		}
		matchedLines, ipv4Trie, ipv6Trie := process_single_file(args, i)
		displayOutput(w, args, matchedLines, ipv4Trie, ipv6Trie)

	}

	return nil
} // func ipcmd

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

func get_inputFiles(args cliArgStruct) ([]inputFile, error) {
	// returns a list of input files

	var iFiles = []inputFile{}

	// build list of files or stdin
	if len(args.InputFiles) == 0 {
		log.Debug("reading from stdin")
		iFiles = append(iFiles, inputFile{IsStdin: true})
	} else {
		log.Debug("ifiles are", "ifiles", args.InputFiles)
		// InputFiles is a slice. each element in the slice is either a file or a directory name.
		files, err := getFilesFromArgs(args.InputFiles)
		if err != nil {
			return nil, err
		}

		slices.Sort(files)

		log.Debug("files to walk are", "file", files)
		for _, file := range files {
			iFiles = append(iFiles, inputFile{IsStdin: false, Filename: file})

		}
	}

	return iFiles, nil
}
