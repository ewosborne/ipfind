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
		wantTrieV4Size int
		wantTrieV6Size int
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
			wantTrieV4Size: 1, // Only one unique address 192.168.1.1
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
			wantTrieV4Size: 2,
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
			wantTrieV4Size: 3,
		},
		{
			name: "Longest match IPv4",
			args: cliArgStruct{
				V4:        true,
				IPv4Regex: ipv4Regex_noSlash,
				Ipaddr:    ipaddr.NewIPAddressString("192.168.1.1").GetAddress(),
				Longest:   true,
			},
			fileContent:    "192.168.1.0/24\n192.168.0.0/16",
			wantMatchLines: nil,
			wantTrieV4Size: 2,
		},
		{
			name: "Multiple Exact match IPv4 on one line",
			args: cliArgStruct{
				V4:        true,
				IPv4Regex: ipv4Regex_noSlash,
				Ipaddr:    ipaddr.NewIPAddressString("192.168.1.1").GetAddress(),
				Exact:     true,
			},
			fileContent:    "192.168.1.1, 192.168.1.1, 192.168.1.2",
			wantMatchLines: []string{"192.168.1.1, 192.168.1.1, 192.168.1.2", "192.168.1.1, 192.168.1.1, 192.168.1.2"},
			wantTrieV4Size: 1,
		},
		{
			name: "Multiple Subnet match IPv4 on one line",
			args: cliArgStruct{
				V4:        true,
				IPv4Regex: ipv4Regex_noSlash,
				Ipaddr:    ipaddr.NewIPAddressString("192.168.1.0/24").GetAddress(),
				Subnet:    true,
			},
			fileContent:    "192.168.1.1, 192.168.1.2, 10.0.0.1",
			wantMatchLines: []string{"192.168.1.1, 192.168.1.2, 10.0.0.1", "192.168.1.1, 192.168.1.2, 10.0.0.1"},
			wantTrieV4Size: 2,
		},
		{
			name: "Multiple Contains match IPv4 on one line",
			args: cliArgStruct{
				V4:        true,
				IPv4Regex: ipv4Regex_noSlash,
				Ipaddr:    ipaddr.NewIPAddressString("192.168.1.1").GetAddress(),
				Contains:  true,
			},
			fileContent:    "192.168.1.0/24, 192.168.1.0/25, 10.0.0.0/8",
			wantMatchLines: []string{"192.168.1.0/24, 192.168.1.0/25, 10.0.0.0/8", "192.168.1.0/24, 192.168.1.0/25, 10.0.0.0/8"},
			wantTrieV4Size: 2,
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
			wantTrieV6Size: 1,
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
			wantTrieV6Size: 2,
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
			wantTrieV6Size: 3,
		},
		{
			name: "Multiple Exact match IPv6 on one line",
			args: cliArgStruct{
				V6:        true,
				IPv6Regex: ipv6Regex_noSlash,
				Ipaddr:    ipaddr.NewIPAddressString("2001:db8::1").GetAddress(),
				Exact:     true,
			},
			fileContent:    "2001:db8::1, 2001:db8::1, 2001:db8::2",
			wantMatchLines: []string{"2001:db8::1, 2001:db8::1, 2001:db8::2", "2001:db8::1, 2001:db8::1, 2001:db8::2"},
			wantTrieV6Size: 1,
		},
		{
			name: "Multiple Subnet match IPv6 on one line",
			args: cliArgStruct{
				V6:        true,
				IPv6Regex: ipv6Regex_noSlash,
				Ipaddr:    ipaddr.NewIPAddressString("2001:db8::/32").GetAddress(),
				Subnet:    true,
			},
			fileContent:    "2001:db8::1, 2001:db8::2, fe80::1",
			wantMatchLines: []string{"2001:db8::1, 2001:db8::2, fe80::1", "2001:db8::1, 2001:db8::2, fe80::1"},
			wantTrieV6Size: 2,
		},
		{
			name: "Multiple Contains match IPv6 on one line",
			args: cliArgStruct{
				V6:        true,
				IPv6Regex: ipv6Regex_noSlash,
				Ipaddr:    ipaddr.NewIPAddressString("2001:db8::1").GetAddress(),
				Contains:  true,
			},
			fileContent:    "2001:db8::/32, 2001:db8::/48, 2001::/16",
			wantMatchLines: []string{"2001:db8::/32, 2001:db8::/48, 2001::/16", "2001:db8::/32, 2001:db8::/48, 2001::/16", "2001:db8::/32, 2001:db8::/48, 2001::/16"},
			wantTrieV6Size: 3,
		},
		{
			name: "Longest match IPv4 multiple on line",
			args: cliArgStruct{
				V4:        true,
				IPv4Regex: ipv4Regex_noSlash,
				Ipaddr:    ipaddr.NewIPAddressString("192.168.1.1").GetAddress(),
				Longest:   true,
			},
			fileContent:    "192.168.1.0/24, 192.168.0.0/16",
			wantMatchLines: nil,
			wantTrieV4Size: 2,
		},
		{
			name: "Longest match IPv6 multiple on line",
			args: cliArgStruct{
				V6:        true,
				IPv6Regex: ipv6Regex_noSlash,
				Ipaddr:    ipaddr.NewIPAddressString("2001:db8::1").GetAddress(),
				Longest:   true,
			},
			fileContent:    "2001:db8::/32, 2001::/16",
			wantMatchLines: nil,
			wantTrieV6Size: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			file := inputFile{
				Filename: "test.txt",
				Scanner:  bufio.NewScanner(strings.NewReader(tt.fileContent)),
			}
			got, v4t, v6t := process_single_file(tt.args, file)
			var gotLines []string
			for _, m := range got {
				gotLines = append(gotLines, m.MatchLine)
			}
			if !reflect.DeepEqual(gotLines, tt.wantMatchLines) {
				t.Errorf("process_single_file() %s: gotLines = %v, want %v", tt.name, gotLines, tt.wantMatchLines)
			}
			if v4t.Size() != tt.wantTrieV4Size {
				t.Errorf("process_single_file() %s: v4 trie size = %v, want %v", tt.name, v4t.Size(), tt.wantTrieV4Size)
			}
			if v6t.Size() != tt.wantTrieV6Size {
				t.Errorf("process_single_file() %s: v6 trie size = %v, want %v", tt.name, v6t.Size(), tt.wantTrieV6Size)
			}
		})
	}
}
