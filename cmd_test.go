package main

import (
	"fmt"
	"strings"
	"testing"

	"github.com/seancfoley/ipaddress-go/ipaddr"
)

func TestGetIPv4AddressesFromLine_TableDriven(t *testing.T) {
	tests := []struct {
		name         string
		line         string
		wantNil      bool
		wantCount    int
		wantPrefixes []int // expected prefix lengths to be present (order not required)
	}{
		{name: "no match", line: "no ip here", wantNil: true},
		{name: "invalid ip", line: "999.999.999.999", wantNil: true},
		{name: "plain and prefix", line: "look 8.8.8.8 and 192.168.1.0/24", wantCount: 2, wantPrefixes: []int{0, 24}},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got := get_ipv4_addresses_from_line(tc.line)
			if tc.wantNil {
				if got != nil {
					t.Fatalf("expected nil, got %v", got)
				}
				return
			}
			if got == nil {
				t.Fatalf("expected %d matches, got nil", tc.wantCount)
			}
			if len(got) != tc.wantCount {
				t.Fatalf("expected %d matches, got %d (%v)", tc.wantCount, len(got), got)
			}

			// collect prefixes found
			found := make(map[int]bool)
			for _, a := range got {
				if a == nil {
					t.Fatalf("nil address in results")
				}
				if !a.IsIPv4() {
					t.Fatalf("expected IPv4, got %v", a)
				}
				found[a.GetPrefixLen().Len()] = true
			}
			for _, p := range tc.wantPrefixes {
				if !found[p] {
					t.Fatalf("expected prefix %d to be present, prefixes found: %v", p, found)
				}
			}

			// sanity-check getHostbits behaviour for plain vs pref address when relevant
			if tc.name == "plain and prefix" {
				var plainAddr, prefAddr *ipaddr.IPAddress
				for _, a := range got {
					if a.GetPrefixLen().Len() == 24 {
						prefAddr = a
					} else {
						plainAddr = a
					}
				}
				if getHostbits(prefAddr) != 24 {
					t.Fatalf("expected hostbits 24 for prefAddr, got %d", getHostbits(prefAddr))
				}
				if getHostbits(plainAddr) != 32 {
					t.Fatalf("expected hostbits 32 for plainAddr, got %d", getHostbits(plainAddr))
				}
			}
		})
	}
}

func TestFoundmatchString_TableDriven(t *testing.T) {
	tests := []struct {
		name string
		idx  int
		addr string
		line string
	}{
		{name: "simple", idx: 7, addr: "10.0.0.1/8", line: "hello"},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			a := ipaddr.NewIPAddressString(tc.addr).GetAddress()
			fm := foundmatch{Idx: tc.idx, Addr: a, Line: tc.line}
			s := fm.String()
			if !strings.Contains(s, fmt.Sprintf("idx: %d", tc.idx)) || !strings.Contains(s, "line("+tc.line+")") {
				t.Fatalf("unexpected foundmatch string: %s", s)
			}
		})
	}
}
