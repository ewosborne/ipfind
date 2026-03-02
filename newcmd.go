package main

import (
	"bufio"
	"fmt"
	"io"
	"os"

	"github.com/charmbracelet/log"
	"github.com/seancfoley/ipaddress-go/ipaddr"
)

type newInputFile struct {
	IsStdin  bool
	Filename string
	Scanner  *bufio.Scanner
}

type readLine struct {
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
		for _, f := range args.InputFiles {
			log.Debug("need to read in", "filename", f)
			newInputFiles = append(newInputFiles, newInputFile{IsStdin: false, Filename: f})
		}
	}

	// at this point newInputFiles is a list of names or stdin
	for _, f := range newInputFiles {

		// at this point the line has been read in and has string matches, idx, line
		// so we're done reading in the file
		fLines, err := readSingleFile(args, f)
		if err != nil {
			log.Fatal("error opening %v", f)
		}
		log.Debug("Read in %+v from %v", fLines, f.Filename)

		// at this point fLines contains []readLine, for each line in the file I just read
		//fmt.Printf("ARGS are %+v %+v\n", args, args.Ipaddr)

		for _, fLine := range fLines {
			switch {
			case args.Exact:
				//log.Print("need to match exactly")
				//log.Printf("working on line %v", fLine)
				for _, ip := range fLine.MatchingIPStrings {
					ipObj := ipaddr.NewIPAddressString(ip).GetAddress()
					//fmt.Printf("comparing %v %v\n", args.Ipaddr, ipObj)
					if ipObj.Equal(args.Ipaddr) {
						//fmt.Println("MATCH!")
						fLine.IsMatch = true // TODO I think I need to pass a pointer here, not sure I understand why
						// TODO break out of the line early, we found a match
					}
				}
			case args.Subnet:
				log.Print("need to match subnet")
				log.Printf("working on line %v", fLine)
				for _, ip := range fLine.MatchingIPStrings {
					ipObj := ipaddr.NewIPAddressString(ip).GetAddress()
					if args.Ipaddr.Contains(ipObj) {
						fLine.IsMatch = true
						// TODO break out of the line early, we found a match
					}
				}
			case args.Contains:
				log.Print("need to match contains")
				log.Printf("working on line %v", fLine)
				for _, ip := range fLine.MatchingIPStrings {
					ipObj := ipaddr.NewIPAddressString(ip).GetAddress()
					if ipObj.Contains(args.Ipaddr) {
						fLine.IsMatch = true
						// TODO break out of the line early, we found a match
					}
				}
			case args.Longest:
				log.Print("need to match longest")
				// TODO
			}
		}

		// now do some reporting
		switch {
		case args.Json:
			log.Print("TODO need to log JSON")
		case args.Trie:
			log.Print("TODO need to log trie")
		default:
			log.Print("need to log text")
			for _, fLine := range fLines {
				if fLine.IsMatch {
					fmt.Printf("%v:%v\n", fLine.Idx, fLine.Line)
				}
			}
		}

		// TODO
	}

	return nil // todo
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
		rl := readLine{Idx: idx, Line: line}

		if len(line) == 0 {
			continue // optimization for blank lines.
		}

		// now find all ipv4 regex matches and ipv6 regex matches from the line
		// TODO: only check the regex that
		if args.IsIPv4 {
			rl.MatchingIPStrings = args.IPv4Regex.FindAllString(line, -1)
		} else {
			rl.MatchingIPStrings = args.IPv6Regex.FindAllString(line, -1)
		}

		linesInFile = append(linesInFile, &rl)
	}
	return linesInFile, nil
}
