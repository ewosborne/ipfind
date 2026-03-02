package main

import (
	"io/fs"
	"path/filepath"
)

/* this is the main command loop.  it figures out what files need to be parsed and calls
   parseFile for each one.  concurrency-ready but 1 worker to start.
*/

// type inputFile struct {
// 	Filename string
// 	IsStdin  bool
// 	Scanner  *bufio.Scanner
// }

// func displayOutput(w io.Writer, args cliArgStruct, matchedLines []dataMatch, ipv4Trie ipaddr.IPv4AddressTrie, ipv6Trie ipaddr.IPv6AddressTrie) {

// 	// three output formats: text, json, trie

// 	if args.Json {
// 		b, err := json.MarshalIndent(matchedLines, "", "  ")
// 		if err != nil {
// 			log.Error(err)
// 		}
// 		fmt.Fprint(w, string(b))
// 	} else if args.Trie {
// 		if ipv4Trie.Size() > 0 {
// 			fmt.Fprintln(w, ipv4Trie)
// 		}

// 		if ipv6Trie.Size() > 0 {
// 			fmt.Fprintln(w, ipv6Trie)
// 		}
// 	} else { // default mode is per line
// 		/* OK
// 		if it's Longest I need to find the longest masklen and print only that
// 		by setting machedLines to whatever's in that trie.
// 		otherwise print everything else.
// 		*/

// 		if args.Longest {
// 			tmp := []dataMatch{}
// 			// find longest mask in the trie
// 			var longest int
// 			if ipv4Trie.Size() > 0 {
// 				iter := ipv4Trie.BlockSizeNodeIterator(true)
// 				//fmt.Printf("%v\n", ipv4Trie)
// 				for iter.HasNext() {
// 					node := iter.Next()
// 					addr := node.GetKey()
// 					if addr.IsPrefixed() {
// 						longest = max(longest, addr.GetNetworkPrefixLen().Len())

// 					}
// 				}
// 			} else {
// 				iter := ipv6Trie.BlockSizeNodeIterator(true)
// 				//fmt.Printf("%v\n", ipv4Trie)
// 				for iter.HasNext() {
// 					node := iter.Next()
// 					addr := node.GetKey()
// 					if addr.IsPrefixed() {
// 						longest = max(longest, addr.GetNetworkPrefixLen().Len())

// 					}
// 				}
// 			}
// 			log.Debugf("longest is /%v\n", longest)
// 			// TODO print only matches with that mask
// 			for _, m := range matchedLines {
// 				for _, ip := range m.MatchIPs {
// 					if ip.GetNetworkPrefixLen().Len() == longest {
// 						//log.Debugf("%v:%v:%v:%v\n", m.Filename, m.Idx, m.MatchLine, m.MatchIPs)
// 						//fmt.Fprintf(w, "%v:%v:%v\n", m.Filename, m.Idx, m.MatchLine)
// 						tmp = append(tmp, m)
// 					}
// 				}
// 			}
// 			matchedLines = tmp
// 		}
// 		for _, m := range matchedLines {
// 			log.Debugf("%v:%v:%v:%v\n", m.Filename, m.Idx, m.MatchLine, m.MatchIPs)
// 			fmt.Fprintf(w, "%v:%v:%v\n", m.Filename, m.Idx, m.MatchLine)
// 		}
// 	}
// }

// func ipcmd(w io.Writer, args cliArgStruct) error {

// 	iFiles, err := get_inputFiles(args)
// 	if err != nil {
// 		return fmt.Errorf("failed to get input files: %w", err)
// 	}

// 	// walk stuff.  this needs a rewrite with channels and a worker pool
// 	for _, i := range iFiles {
// 		//fmt.Printf("need to process file %v\n", i.Filename)
// 		// launch a goroutine per file maybe?  for now just do it in order

// 		if i.IsStdin {
// 			i.Scanner = bufio.NewScanner(os.Stdin)
// 		} else {
// 			ifh, err := os.Open(i.Filename)
// 			if err != nil {
// 				log.Errorf("could not open file %v\n", i.Filename)
// 			}
// 			defer ifh.Close()
// 			i.Scanner = bufio.NewScanner(ifh)

// 		}
// 	}

// 	return nil
// } // func ipcmd

// v   keep this function
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

// func get_inputFiles(args cliArgStruct) ([]inputFile, error) {
// 	// returns a list of input files

// 	var iFiles = []inputFile{}

// 	// build list of files or stdin
// 	if len(args.InputFiles) == 0 {
// 		log.Debug("reading from stdin")
// 		iFiles = append(iFiles, inputFile{IsStdin: true})
// 	} else {
// 		log.Debug("ifiles are", "ifiles", args.InputFiles)
// 		// InputFiles is a slice. each element in the slice is either a file or a directory name.
// 		files, err := getFilesFromArgs(args.InputFiles)
// 		if err != nil {
// 			return nil, err
// 		}

// 		slices.Sort(files)

// 		log.Debug("files to walk are", "file", files)
// 		for _, file := range files {
// 			iFiles = append(iFiles, inputFile{IsStdin: false, Filename: file})

// 		}
// 	}

// 	return iFiles, nil
// }
