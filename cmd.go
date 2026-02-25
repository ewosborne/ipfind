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
