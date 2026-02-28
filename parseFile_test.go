package main

import (
	"bufio"
	"reflect"
	"strings"
	"testing"

	"github.com/seancfoley/ipaddress-go/ipaddr"
)

func TestProcessSingleFile(t *testing.T) {
	t.Parallel()
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
		{
			name: "Exact match IPv6",
			args: cliArgStruct{
				V6:        true,
				IPv6Regex: ipv6Regex_noSlash,
				Ipaddr:    ipaddr.NewIPAddressString("2001:db8::1").GetAddress(),
				Exact:     true,
			},
			fileContent:    "2001:db8::1\n2001:db8::2\n2001:db8::1/128",
			wantMatchLines: []string{"2001:db8::1", "2001:db8::1/128"},
		},
		{
			name: "Subnet match IPv6",
			args: cliArgStruct{
				V6:        true,
				IPv6Regex: ipv6Regex_noSlash,
				Ipaddr:    ipaddr.NewIPAddressString("2001:db8::/32").GetAddress(),
				Subnet:    true,
			},
			fileContent:    "2001:db8::1\nfe80::1\n2001:db8:1234::/48",
			wantMatchLines: []string{"2001:db8::1", "2001:db8:1234::/48"},
		},
		{
			name: "Contains match IPv6",
			args: cliArgStruct{
				V6:        true,
				IPv6Regex: ipv6Regex_noSlash,
				Ipaddr:    ipaddr.NewIPAddressString("2001:db8::1").GetAddress(),
				Contains:  true,
			},
			fileContent:    "2001:db8::/32\n2001::/16\n2001:db8::1",
			wantMatchLines: []string{"2001:db8::/32", "2001::/16", "2001:db8::1"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
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
