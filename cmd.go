package main

/* this is the main command loop.  it figures out what files need to be parsed and calls
   parseFile for each one.  concurrency-ready but 1 worker to start.
*/
import (
	"bufio"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"slices"

	"github.com/charmbracelet/log"
)

type inputFile struct {
	Filename string
	IsStdin  bool
	Scanner  *bufio.Scanner
}

func ipcmd(args cliArgStruct) error {

	handler := log.New(os.Stderr)
	logger := slog.New(handler)
	logger.Error("meow?")

	slog.Debug("ipcmd", "args", args)
	log.Print("ugly log here")

	iFiles, err := get_inputFiles(args)
	if err != nil {
		panic("need to handle this error but lazy")
	}

	// walk stuff.  this needs a rewrite with channels and a worker pool.
	for _, i := range iFiles {
		//fmt.Printf("need to process file %v\n", i.Filename)
		// launch a goroutine per file maybe?  for now just do it in order
		matchedLines := process_single_file(args, i)
		for _, m := range matchedLines {
			fmt.Printf("match:%+v\n", m) // this is where any fancy output goes I think
		}
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
	// TODO: preserve filename here for later reporting.
	if len(args.InputFiles) == 0 {
		slog.Debug("reading from stdin")
		iFiles = append(iFiles, inputFile{Filename: "os.Stdin", IsStdin: true, Scanner: bufio.NewScanner(os.Stdin)})
	} else {
		slog.Debug("ifiles are", "ifiles", args.InputFiles)
		// InputFiles is a slice. each element in the slice is either a file or a directory name.
		files, err := getFilesFromArgs(args.InputFiles)
		if err != nil {
			return nil, err
		}

		slices.Sort(files)

		slog.Debug("files to walk are", "file", files)
		for _, file := range files {
			tmp, _ := os.Open(file)
			iFiles = append(iFiles, inputFile{IsStdin: false, Filename: file, Scanner: bufio.NewScanner(tmp)})

		}
	}

	return iFiles, nil
}

/*
	TODO
	clean up output
	handle line vs network print match?

	rewrite file handling in goroutines, or at least prep for that
	   one goroutine per file (or if stdin, that's just one file)
	   roll it all up into a per-file map at the top level

	think about how this interacts with trie support.  -ct, -st, -lt, -et vs. just -t.

	learn how to use the debugger

	get some solid tests cases in place

	better logging?  charmbracelet log? less structured?

	I think ipaddr has ip ranges too, use those?  make this more of a general-purpose grep thing?

	<search network - host, network, range, etc.>
	  contained-in
	    longest match vs. subnets
      contains
	  exact

things to think through:
output format (trie, prefix, line)
performance


*/
