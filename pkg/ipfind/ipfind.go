package ipfind

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"sync"

	"github.com/charmbracelet/log"
	"github.com/seancfoley/ipaddress-go/ipaddr"
)

var (
	ipv4RegexWithSlash = regexp.MustCompile(`(\d{1,3})\.(\d{1,3})\.(\d{1,3})\.(\d{1,3}(/\d{1,2}))`)
	ipv6RegexWithSlash = regexp.MustCompile(`([:0-9a-fA-F]{2,39}(/[0-9]{1,3}))`)
	ipv4RegexNoSlash   = regexp.MustCompile(`(\d{1,3})\.(\d{1,3})\.(\d{1,3})\.(\d{1,3}(/\d{1,2})?)`)
	ipv6RegexNoSlash   = regexp.MustCompile(`([:0-9a-fA-F]{2,39}(/[0-9]{1,3})?)`)
)

func (l *LineResult) getRegexMatches(ipre *regexp.Regexp, af AddressFamily) []string {
	var ret []string
	switch af {
	case IPv4:
		ret = ipre.FindAllString(l.Line, -1)
	case IPv6:
		for _, m := range ipre.FindAllString(l.Line, -1) {
			if strings.Contains(m, ":") {
				ret = append(ret, m)
			}
		}
	}
	return ret
}

func (fp *FileResult) getLongestSubnetMask() int {
	var ret int
	for _, x := range fp.MatchedLines {
		for _, y := range x.ConditionMatches {
			ret = max(ret, y.GetPrefixLen().Len())
		}
	}
	return ret
}

func (fp *FileResult) getMinimumConditionMatchLines(minimum int) []*LineResult {
	ret := []*LineResult{}
	for _, x := range fp.MatchedLines {
	NextLine:
		for _, y := range x.ConditionMatches {
			if y.GetPrefixLen().Len() >= minimum {
				ret = append(ret, x)
				break NextLine
			}
		}
	}
	return ret
}

func GetInputFilesOrStdin(files []string) ([]InputFile, error) {
	ret := []InputFile{}

	switch len(files) {
	case 0:
		ret = append(ret, InputFile{IsStdin: true})
	default:
		fileNames, err := getFileNamesFromArgs(files)
		if err != nil {
			return nil, fmt.Errorf("walking directories: %w", err)
		}
		for _, f := range fileNames {
			ret = append(ret, InputFile{IsStdin: false, Filename: f})
		}
	}
	return ret, nil
}

func getFileNamesFromArgs(inputFiles []string) ([]string, error) {
	var ret []string
	for _, ifile := range inputFiles {
		info, err := os.Stat(ifile)
		if err != nil {
			return nil, err
		}

		if info.IsDir() {
			// If it's a directory, we walk it.
			// We use filepath.WalkDir, but we need to handle the case where ifile is a symlink to a directory.
			// WalkDir uses Lstat on the root, so it won't recurse into a symlink.
			err := filepath.WalkDir(ifile, func(path string, d fs.DirEntry, err error) error {
				if err != nil {
					return err
				}

				// If this is the root and it's a symlink, WalkDir won't recurse.
				// But we already know it's a directory from os.Stat.
				if path == ifile && d.Type()&fs.ModeSymlink != 0 {
					// We need to manually walk the contents of this symlinked directory.
					items, err := os.ReadDir(ifile)
					if err != nil {
						return err
					}
					for _, item := range items {
						subPath := filepath.Join(ifile, item.Name())
						// Recursively call getFileNamesFromArgs for sub-items
						subFiles, err := getFileNamesFromArgs([]string{subPath})
						if err != nil {
							return err
						}
						ret = append(ret, subFiles...)
					}
					return filepath.SkipDir // We've handled this directory manually
				}

				if !d.IsDir() {
					ret = append(ret, path)
				}
				return nil
			})
			if err != nil {
				return nil, err
			}
		} else {
			ret = append(ret, ifile)
		}
	}
	return ret, nil
}

func GetReadCloser(fileName InputFile) (io.ReadCloser, error) {
	if fileName.IsStdin {
		return os.Stdin, nil
	}
	ifh, err := os.Open(fileName.Filename)
	if err != nil {
		return nil, fmt.Errorf("opening file %s: %w", fileName.Filename, err)
	}
	info, err := ifh.Stat()
	if err != nil {
		ifh.Close()
		return nil, fmt.Errorf("stat file %s: %w", fileName.Filename, err)
	}
	if info.IsDir() {
		ifh.Close()
		return nil, fmt.Errorf("%s is a directory", fileName.Filename)
	}
	return ifh, nil
}

func MatchesCondition(ipObject, targetIP *ipaddr.IPAddress, args Args) bool {
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

// ProcessReader now uses a callback to stream matches back to the caller immediately.
func ProcessReader(ctx context.Context, r io.Reader, args Args, onMatch func(LineResult) error) error {
	idx := 0
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		// Respect context cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		idx++
		line := scanner.Text()
		if len(line) == 0 {
			continue
		}

		lineObj := LineResult{LineNumber: idx, Line: line}
		lineObj.IPRegexMatches = lineObj.getRegexMatches(args.IPRegex, args.AddressFamily)

		matched := false
		for _, m := range lineObj.IPRegexMatches {
			ipObject := ipaddr.NewIPAddressString(m).GetAddress()
			if ipObject == nil {
				continue
			}

			if MatchesCondition(ipObject, args.Ipaddr, args) {
				matched = true
				lineObj.ConditionMatches = append(lineObj.ConditionMatches, ipObject)
			}
		}

		if matched {
			if err := onMatch(lineObj); err != nil {
				return err
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("reading input: %w", err)
	}
	return nil
}

// Run is the main entry point for the library logic.
func Run(ctx context.Context, w io.Writer, args Args) error {
	log.Debug("starting ipcmd", "workers", args.Workers, "sort", args.Sort)

	inputFiles, err := GetInputFilesOrStdin(args.InputFiles)
	if err != nil {
		return fmt.Errorf("getting input: %w", err)
	}

	// Case 1: Exactly 1 worker, no sorting - exact original behavior (streaming)
	if args.Workers == 1 && !args.Sort {
		return runSequential(ctx, w, inputFiles, args)
	}

	// Case 2: Parallel or sorting required - buffer each file's result
	return runParallel(ctx, w, inputFiles, args)
}

func runSequential(ctx context.Context, w io.Writer, inputFiles []InputFile, args Args) error {
	firstJson := true
	if args.Json {
		fmt.Fprintln(w, "[")
	}

	for _, fileName := range inputFiles {
		rc, err := GetReadCloser(fileName)
		if err != nil {
			log.Error("skipping file", "file", fileName.Filename, "error", err)
			continue
		}

		fr := NewFileResult(fileName)
		onMatch := func(lr LineResult) error {
			if args.Longest || args.Trie {
				fr.MatchedLines = append(fr.MatchedLines, &lr)
				return nil
			}

			if args.Json {
				if !firstJson {
					fmt.Fprintln(w, ",")
				}
				firstJson = false
				b, err := json.MarshalIndent(lr, "  ", "  ")
				if err != nil {
					return err
				}
				fmt.Fprintf(w, "  %s", string(b))
				return nil
			}

			fmt.Fprintf(w, "%v:%v:%v\n", fileName.Filename, lr.LineNumber, lr.Line)
			return nil
		}

		err = ProcessReader(ctx, rc, args, onMatch)
		if !fileName.IsStdin {
			rc.Close()
		}
		if err != nil {
			log.Error("error processing file", "file", fileName.Filename, "error", err)
			continue
		}

		if args.Longest || args.Trie {
			if err := writeBufferedResult(w, fr, args, &firstJson); err != nil {
				return err
			}
		}
	}

	if args.Json {
		fmt.Fprintln(w, "\n]")
	}
	return nil
}

func runParallel(ctx context.Context, w io.Writer, inputFiles []InputFile, args Args) error {
	numWorkers := args.Workers
	if numWorkers <= 0 {
		numWorkers = runtime.NumCPU() * 2
	}

	results := make(chan *FileResult, len(inputFiles))
	jobs := make(chan InputFile, len(inputFiles))

	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for fileName := range jobs {
				fr := NewFileResult(fileName)
				rc, err := GetReadCloser(fileName)
				if err != nil {
					log.Error("skipping file", "file", fileName.Filename, "error", err)
					results <- fr
					continue
				}

				onMatch := func(lr LineResult) error {
					fr.MatchedLines = append(fr.MatchedLines, &lr)
					return nil
				}

				err = ProcessReader(ctx, rc, args, onMatch)
				if !fileName.IsStdin {
					rc.Close()
				}
				if err != nil {
					log.Error("error processing file", "file", fileName.Filename, "error", err)
				}
				results <- fr
			}
		}()
	}

	for _, f := range inputFiles {
		jobs <- f
	}
	close(jobs)

	// Wait for workers in a separate goroutine
	go func() {
		wg.Wait()
		close(results)
	}()

	firstJson := true
	if args.Json {
		fmt.Fprintln(w, "[")
	}

	if args.Sort {
		var allResults []*FileResult
		for fr := range results {
			allResults = append(allResults, fr)
		}
		sort.Slice(allResults, func(i, j int) bool {
			return allResults[i].Filename < allResults[j].Filename
		})
		for _, fr := range allResults {
			if err := writeBufferedResult(w, fr, args, &firstJson); err != nil {
				return err
			}
		}
	} else {
		// Stream file-by-file as they complete
		for fr := range results {
			if err := writeBufferedResult(w, fr, args, &firstJson); err != nil {
				return err
			}
		}
	}

	if args.Json {
		fmt.Fprintln(w, "\n]")
	}

	return nil
}

func writeBufferedResult(w io.Writer, fr *FileResult, args Args, firstJson *bool) error {
	var lsm int
	if args.Longest {
		lsm = fr.getLongestSubnetMask()
	}

	if args.Trie {
		for _, line := range fr.MatchedLines {
			switch args.AddressFamily {
			case IPv4:
				for _, pfx := range line.ConditionMatches {
					fr.IPv4Trie.Add(pfx.ToIPv4())
				}
			case IPv6:
				for _, pfx := range line.ConditionMatches {
					fr.IPv6Trie.Add(pfx.ToIPv6())
				}
			}
		}
		if fr.IPv4Trie.Size() > 0 {
			fmt.Fprintf(w, "%v\n", fr.IPv4Trie)
		}
		if fr.IPv6Trie.Size() > 0 {
			fmt.Fprintf(w, "%v\n", fr.IPv6Trie)
		}
		return nil
	}

	// Normal text or JSON
	for _, line := range fr.getMinimumConditionMatchLines(lsm) {
		if args.Json {
			if !*firstJson {
				fmt.Fprintln(w, ",")
			}
			*firstJson = false
			b, err := json.MarshalIndent(line, "  ", "  ")
			if err != nil {
				return err
			}
			fmt.Fprintf(w, "  %s", string(b))
		} else {
			fmt.Fprintf(w, "%v:%v:%v\n", fr.Filename, line.LineNumber, line.Line)
		}
	}
	return nil
}

func ArgMassage(cliArgs Args) Args {
	// Subnet is default if the others aren't set
	cliArgs.Subnet = !(cliArgs.Exact || cliArgs.Contains)

	// turn target IP into address object
	cliArgs.Ipaddr = ipaddr.NewIPAddressString(cliArgs.Ipstring).GetAddress()
	if cliArgs.Ipaddr == nil {
		log.Error("invalid IP", "ip", cliArgs.Ipstring)
	}

	// canonize it unless explicitly disallowed
	if cliArgs.Canonize && cliArgs.Ipaddr != nil {
		cliArgs.Ipaddr = cliArgs.Ipaddr.ToPrefixBlock()
	}

	if cliArgs.Slash {
		cliArgs.IPv4Regex = ipv4RegexWithSlash
		cliArgs.IPv6Regex = ipv6RegexWithSlash
	} else {
		cliArgs.IPv4Regex = ipv4RegexNoSlash
		cliArgs.IPv6Regex = ipv6RegexNoSlash
	}

	if cliArgs.Ipaddr != nil {
		if cliArgs.Ipaddr.IsIPv4() {
			cliArgs.AddressFamily = IPv4
			cliArgs.IPRegex = cliArgs.IPv4Regex
		} else if cliArgs.Ipaddr.IsIPv6() {
			cliArgs.AddressFamily = IPv6
			cliArgs.IPRegex = cliArgs.IPv6Regex
		}
	}

	return cliArgs
}
