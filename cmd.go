package main

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
)

var ipv4Address = regexp.MustCompile(`(\d{1,3}).(\d{1,3}).(\d{1,3}).(\d{1,3}(/\d{1,2})?)`)

func ipcmd(args cliArgStruct) {

	var a, b, c, d, ipInt int64

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
			fmt.Println("\tidx", idx, "elem", elem)

			// check each found regex against the match criteria somehow and keep or print or skip
			//  could be more than one match on a line and we need to process one of them for exact, one for subnet, and all for longest
			// theory being that if it matches exact or subnet we keep it
			// and if it matches under lpm we need to keep looking and cache this line, not print it

			// first turn the matched ip address to an int32
			//. adjust base for mask else assume /32 I guess.tbd.
			tmp := strings.Split(line, ".")

			a, _ = strconv.ParseInt(tmp[0], 10, 32)
			a = a << 24
			b, _ = strconv.ParseInt(tmp[1], 10, 32)
			b = b << 16
			c, _ = strconv.ParseInt(tmp[2], 10, 32)
			c = c << 8
			d, _ = strconv.ParseInt(tmp[3], 10, 32)

			ipInt = a + b + c + d
			fmt.Println(a, b, c, d, ipInt)

			// TODO: ok to leave this in a 64bit int?  feels weird.

		}

	}

}
