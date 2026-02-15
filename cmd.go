package main

import (
	"bufio"
	"fmt"
	"os"
	"regexp"

	"github.com/seancfoley/ipaddress-go/ipaddr"
)

// TODO: handle mask.
var ipv4Address = regexp.MustCompile(`(\d{1,3}).(\d{1,3}).(\d{1,3}).(\d{1,3}(/\d{1,2})?)`)

//var ipv4Address = regexp.MustCompile(`(\d{1,3}).(\d{1,3}).(\d{1,3}).(\d{1,3})`)

func ipcmd(args cliArgStruct) {

	fmt.Printf("args in ipcmd:%+v\n", args)

	findIPv4Addr := ipaddr.NewIPAddressString(args.ipaddr)

	var scanner *bufio.Scanner
	if len(args.inputFile) > 0 {
		file, _ := os.Open(args.inputFile)
		scanner = bufio.NewScanner(file)
	} else {
		scanner = bufio.NewScanner(os.Stdin)
	}
	for scanner.Scan() {
		line := scanner.Text()
		fmt.Printf("line is #%s#\n", line)
		for idx, elem := range ipv4Address.FindAllString(line, -1) {
			//fmt.Println("\tidx", idx, "elem", elem)
			ipv4AddrStr := ipaddr.NewIPAddressString(elem)
			fmt.Printf("\t%v: found %v\n", idx, ipv4AddrStr)

			if ipv4AddrStr.GetAddress().Contains(findIPv4Addr.GetAddress()) {
				fmt.Printf("\t\t%v CONTAINS	%v\n", ipv4AddrStr, findIPv4Addr)
			}
		}
	}
}
