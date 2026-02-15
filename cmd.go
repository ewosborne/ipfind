package main

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
)

var ipv4Address = regexp.MustCompile(`(\d{1,3}).(\d{1,3}).(\d{1,3}).(\d{1,3}(/\d{1,2})?)`)

func ipcmd(args cliArgStruct) {
	fmt.Printf("args in ipcmd:%+v\n", args)
	fmt.Println("you want me to find", args.ipaddr)

	/*
		read from file or stdin (so far just stdin)
		if it's exact or subnet then match and print line by line
		if it's longest then line by line and store any matching lines but don't print
		 then do a second pass to find the longest prefi
	*/
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := scanner.Text()
		fmt.Printf("line is #%s#\n", line)
		for idx, elem := range ipv4Address.FindAllString(line, -1) {
			fmt.Println("idx", idx, "elem", elem, "line", line)
		}

	}

}
