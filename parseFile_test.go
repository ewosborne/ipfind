package main

import (
	"bufio"
	"reflect"
	"strings"
	"testing"

	"github.com/seancfoley/ipaddress-go/ipaddr"
)

func TestProcessSingleFile(t *testing.T) {
	tests := []struct {
		name           string
		args           cliArgStruct
		fileContent    string
		wantMatchLines []string
	}{
		{
			name: "Exact match IPv4",
			args: cliArgStruct{
				V4:        true,
				IPv4Regex: ipv4Regex_noSlash,
				Ipaddr:    ipaddr.NewIPAddressString("192.168.1.1").GetAddress(),
				Exact:     true,
			},
			fileContent:    "192.168.1.1\n192.168.1.2\n192.168.1.1/32",
			wantMatchLines: []string{"192.168.1.1", "192.168.1.1/32"},
		},
		{
			name: "Subnet match IPv4",
			args: cliArgStruct{
				V4:        true,
				IPv4Regex: ipv4Regex_noSlash,
				Ipaddr:    ipaddr.NewIPAddressString("192.168.1.0/24").GetAddress(),
				Subnet:    true,
			},
			fileContent:    "192.168.1.1\n10.0.0.1\n192.168.1.0/25",
			wantMatchLines: []string{"192.168.1.1", "192.168.1.0/25"},
		},
		{
			name: "Contains match IPv4",
			args: cliArgStruct{
				V4:        true,
				IPv4Regex: ipv4Regex_noSlash,
				Ipaddr:    ipaddr.NewIPAddressString("192.168.1.1").GetAddress(),
				Contains:  true,
			},
			fileContent:    "192.168.1.0/24\n192.168.0.0/16\n192.168.1.1",
			wantMatchLines: []string{"192.168.1.0/24", "192.168.0.0/16", "192.168.1.1"},
		},
		{
			name: "Longest match IPv4",
			args: cliArgStruct{
				V4:        true,
				IPv4Regex: ipv4Regex_noSlash,
				Ipaddr:    ipaddr.NewIPAddressString("192.168.1.1").GetAddress(),
				Longest:   true,
			},
			fileContent: "192.168.1.0/24\n192.168.0.0/16",
			// In longest match mode, process_single_file currently only populates the trie
			// and returns an empty match list for lines.
			wantMatchLines: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file := inputFile{
				Filename: "test.txt",
				Scanner:  bufio.NewScanner(strings.NewReader(tt.fileContent)),
			}
			got, _, _ := process_single_file(tt.args, file)
			var gotLines []string
			for _, m := range got {
				gotLines = append(gotLines, m.MatchLine)
			}
			if !reflect.DeepEqual(gotLines, tt.wantMatchLines) {
				t.Errorf("process_single_file() %s: got = %v, want %v", tt.name, gotLines, tt.wantMatchLines)
			}
		})
	}
}
