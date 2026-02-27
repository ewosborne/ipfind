package main

/* this is the main command loop.  it figures out what files need to be parsed and calls
   parseFile for each one.  concurrency-ready but 1 worker to start.
*/
import (
	"bufio"
	"encoding/json"
	"fmt"
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

func displayOutput(args cliArgStruct, matchedLines []dataMatch, ipv4Trie ipaddr.IPv4AddressTrie, ipv6Trie ipaddr.IPv6AddressTrie) {

	// need a redo.  three output formats: text, json, trie

	if args.Json {
		b, err := json.MarshalIndent(matchedLines, "", "  ")
		if err != nil {
			log.Error(err)
		}
		fmt.Print(string(b))
	} else if args.Trie {
		if args.V4 {
			if args.Longest {
				fmt.Println("IPv4 LPM", ipv4Trie.LongestPrefixMatch(args.Ipaddr.ToIPv4()))
			}
			if args.Trie && ipv4Trie.Size() > 0 {
				fmt.Println(ipv4Trie)
			}
		}
		if args.V6 {
			fmt.Println("IPv6 LPM", ipv6Trie.LongestPrefixMatch(args.Ipaddr.ToIPv6()))
		}
		if args.Trie && ipv6Trie.Size() > 0 {
			fmt.Println(ipv6Trie)
		}
	} else {
		for _, m := range matchedLines {
			log.Debugf("%v:%v:%v:%v\n", m.Filename, m.Idx, m.MatchLine, m.MatchIPs)
			fmt.Printf("%v:%v:%v\n", m.Filename, m.Idx, m.MatchLine)
		}
	}
}

func ipcmd(args cliArgStruct) error {

	iFiles, err := get_inputFiles(args)
	if err != nil {
		panic("need to handle this error but lazy")
	}

	// walk stuff.  this needs a rewrite with channels and a worker pool.
	for _, i := range iFiles {
		//fmt.Printf("need to process file %v\n", i.Filename)
		// launch a goroutine per file maybe?  for now just do it in order
		matchedLines, ipv4Trie, ipv6Trie := process_single_file(args, i)
		displayOutput(args, matchedLines, ipv4Trie, ipv6Trie)

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
		iFiles = append(iFiles, inputFile{Filename: "os.Stdin", IsStdin: true, Scanner: bufio.NewScanner(os.Stdin)})
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
			tmp, _ := os.Open(file)
			iFiles = append(iFiles, inputFile{IsStdin: false, Filename: file, Scanner: bufio.NewScanner(tmp)})

		}
	}

	return iFiles, nil
}
