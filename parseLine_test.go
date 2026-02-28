package main

import (
	"reflect"
	"regexp"
	"testing"
)

func TestScanLine(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		args    cliArgStruct
		line    string
		wantOK  bool
		wantIPs []string
	}{
		{
			name: "IPv4 with slash",
			args: cliArgStruct{
				V4:        true,
				IPv4Regex: ipv4Regex_withSlash,
			},
			line:    "Found 192.168.1.0/24 here",
			wantOK:  true,
			wantIPs: []string{"192.168.1.0/24"},
		},
		{
			name: "IPv4 no slash, regex with slash",
			args: cliArgStruct{
				V4:        true,
				IPv4Regex: ipv4Regex_withSlash,
			},
			line:    "Found 192.168.1.1 here",
			wantOK:  false,
			wantIPs: nil,
		},
		{
			name: "IPv4 no slash, regex no slash",
			args: cliArgStruct{
				V4:        true,
				IPv4Regex: ipv4Regex_noSlash,
			},
			line:    "Found 192.168.1.1 here",
			wantOK:  true,
			wantIPs: []string{"192.168.1.1"},
		},
		{
			name: "IPv6 with slash",
			args: cliArgStruct{
				V6:        true,
				IPv6Regex: ipv6Regex_withSlash,
			},
			line:    "Found 2001:db8::/32 here",
			wantOK:  true,
			wantIPs: []string{"2001:db8::/32"},
		},
		{
			name: "IPv6 no slash, regex no slash",
			args: cliArgStruct{
				V6:        true,
				IPv6Regex: ipv6Regex_noSlash,
			},
			line:    "Found 2001:db8::1 here",
			wantOK:  true,
			wantIPs: []string{"2001:db8::1"},
		},
		{
			name: "Both IPv4 and IPv6",
			args: cliArgStruct{
				V4:        true,
				V6:        true,
				IPv4Regex: ipv4Regex_noSlash,
				IPv6Regex: ipv6Regex_noSlash,
			},
			line:    "1.2.3.4 and 2001:db8::",
			wantOK:  true,
			wantIPs: []string{"1.2.3.4", "2001:db8::"},
		},
		{
			name: "No matches",
			args: cliArgStruct{
				V4:        true,
				V6:        true,
				IPv4Regex: ipv4Regex_noSlash,
				IPv6Regex: ipv6Regex_noSlash,
			},
			line:    "No IPs here",
			wantOK:  false,
			wantIPs: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dm := dataMatch{MatchLine: tt.line}
			got, ok := scanLine(tt.args, dm)
			if ok != tt.wantOK {
				t.Errorf("scanLine() ok = %v, wantOK %v", ok, tt.wantOK)
			}
			if ok {
				var gotIPs []string
				for _, ip := range got.MatchIPs {
					gotIPs = append(gotIPs, ip.String())
				}
				if !reflect.DeepEqual(gotIPs, tt.wantIPs) {
					t.Errorf("scanLine() gotIPs = %v, want %v", gotIPs, tt.wantIPs)
				}
			}
		})
	}
}

func TestGetIpAddressesFromLine(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		re      *regexp.Regexp
		line    string
		wantIPs []string
	}{
		{
			name:    "Multiple IPv4",
			re:      ipv4Regex_noSlash,
			line:    "1.1.1.1, 2.2.2.2/24",
			wantIPs: []string{"1.1.1.1", "2.2.2.2/24"},
		},
		{
			name:    "Multiple IPv6",
			re:      ipv6Regex_noSlash,
			line:    "2001:db8::1, 2001:db8::2/64",
			wantIPs: []string{"2001:db8::1", "2001:db8::2/64"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := get_ip_addresses_from_line(tt.re, tt.line)
			var gotIPs []string
			for _, ip := range got {
				gotIPs = append(gotIPs, ip.String())
			}
			if !reflect.DeepEqual(gotIPs, tt.wantIPs) {
				t.Errorf("get_ip_addresses_from_line() = %v, want %v", gotIPs, tt.wantIPs)
			}
		})
	}
}

func TestIPv4RegexBug(t *testing.T) {
	t.Parallel()
	line := "111G333G444G999"
	if ipv4Regex_noSlash.MatchString(line) {
		t.Errorf("ipv4Regex_noSlash matched invalid IP %s", line)
	}

	lineWithSlash := "111G333G444G999/24"
	if ipv4Regex_withSlash.MatchString(lineWithSlash) {
		t.Errorf("ipv4Regex_withSlash matched invalid IP %s", lineWithSlash)
	}
}
