package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/seancfoley/ipaddress-go/ipaddr"
)

func TestDisplayOutput(t *testing.T) {
	tests := []struct {
		name         string
		args         cliArgStruct
		matchedLines []dataMatch
		ipv4Trie     ipaddr.IPv4AddressTrie
		ipv6Trie     ipaddr.IPv6AddressTrie
		wantOutput   string
	}{
		{
			name: "Simple text output",
			args: cliArgStruct{},
			matchedLines: []dataMatch{
				{Filename: "test.txt", Idx: 1, MatchLine: "1.1.1.1 is here"},
			},
			wantOutput: "test.txt:1:1.1.1.1 is here\n",
		},
		{
			name: "Simple IPv6 text output",
			args: cliArgStruct{},
			matchedLines: []dataMatch{
				{Filename: "test6.txt", Idx: 10, MatchLine: "2001:db8::1 is here"},
			},
			wantOutput: "test6.txt:10:2001:db8::1 is here\n",
		},
		{
			name: "JSON output",
			args: cliArgStruct{Json: true},
			matchedLines: []dataMatch{
				{
					Filename: "test.txt", Idx: 1, MatchLine: "1.1.1.1 is here",
					MatchIPs: []*ipaddr.IPAddress{ipaddr.NewIPAddressString("1.1.1.1").GetAddress()},
				},
			},
			wantOutput: "[\n  {\n    \"filename\": \"test.txt\",\n    \"idx\": 1,\n    \"match_line\": \"1.1.1.1 is here\",\n    \"match_ips\": [\n      \"1.1.1.1\"\n    ]\n  }\n]",
		},
		{
			name: "JSON IPv6 output",
			args: cliArgStruct{Json: true},
			matchedLines: []dataMatch{
				{
					Filename: "test6.txt", Idx: 10, MatchLine: "2001:db8::1 is here",
					MatchIPs: []*ipaddr.IPAddress{ipaddr.NewIPAddressString("2001:db8::1").GetAddress()},
				},
			},
			wantOutput: "[\n  {\n    \"filename\": \"test6.txt\",\n    \"idx\": 10,\n    \"match_line\": \"2001:db8::1 is here\",\n    \"match_ips\": [\n      \"2001:db8::1\"\n    ]\n  }\n]",
		},
		{
			name: "Longest match output",
			args: cliArgStruct{
				Longest: true,
				Ipaddr:  ipaddr.NewIPAddressString("1.1.1.1").GetAddress(),
			},
			ipv4Trie: func() ipaddr.IPv4AddressTrie {
				t := ipaddr.IPv4AddressTrie{}
				t.Add(ipaddr.NewIPAddressString("1.1.1.0/24").GetAddress().ToIPv4())
				return t
			}(),
			wantOutput: "IPv4 LPM 1.1.1.0/24\n",
		},
		{
			name: "IPv6 LPM output",
			args: cliArgStruct{
				Longest: true,
				Ipaddr:  ipaddr.NewIPAddressString("2001:db8::1").GetAddress(),
			},
			ipv6Trie: func() ipaddr.IPv6AddressTrie {
				t := ipaddr.IPv6AddressTrie{}
				t.Add(ipaddr.NewIPAddressString("2001:db8::/32").GetAddress().ToIPv6())
				return t
			}(),
			wantOutput: "IPv6 LPM 2001:db8::/32\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var w bytes.Buffer
			displayOutput(&w, tt.args, tt.matchedLines, tt.ipv4Trie, tt.ipv6Trie)
			got := w.String()
			// JSON marshaling might have trailing newline or not depending on implementation
			if tt.args.Json {
				if strings.TrimSpace(got) != strings.TrimSpace(tt.wantOutput) {
					t.Errorf("displayOutput() got = %v, want %v", got, tt.wantOutput)
				}
			} else {
				if got != tt.wantOutput {
					t.Errorf("displayOutput() got = %v, want %v", got, tt.wantOutput)
				}
			}
		})
	}
}
