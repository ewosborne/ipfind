package main

import (
	"testing"

	"github.com/seancfoley/ipaddress-go/ipaddr"
)

func TestArgMassage(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		in   cliArgStruct
		want cliArgStruct
	}{
		{
			name: "Default to Longest",
			in: cliArgStruct{
				Ipstring: "1.1.1.1",
				Slash:    true,
			},
			want: cliArgStruct{
				Ipstring:  "1.1.1.1",
				Longest:   true,
				V4:        true,
				V6:        false,
				Canonize:  false, // This might be true because Destination: &cliArgs.Canonize has Value: true in main.go
				Slash:     true,
				IPv4Regex: ipv4Regex_withSlash,
				IPv6Regex: ipv6Regex_withSlash,
				Ipaddr:    ipaddr.NewIPAddressString("1.1.1.1").GetAddress(),
			},
		},
		{
			name: "Exact match with Canonize set to false",
			in: cliArgStruct{
				Ipstring: "1.1.1.1/24",
				Exact:    true,
				Canonize: false,
			},
			want: cliArgStruct{
				Ipstring: "1.1.1.1/24",
				Exact:    true,
				Longest:  false,
				V4:       true,
				V6:       false,
				Canonize: false,
				Ipaddr:   ipaddr.NewIPAddressString("1.1.1.1/24").GetAddress(),
			},
		},
		{
			name: "IPv6 Default to Longest",
			in: cliArgStruct{
				Ipstring: "2001:db8::1",
				Slash:    true,
			},
			want: cliArgStruct{
				Ipstring:  "2001:db8::1",
				Longest:   true,
				V4:        false,
				V6:        true,
				Slash:     true,
				IPv4Regex: ipv4Regex_withSlash,
				IPv6Regex: ipv6Regex_withSlash,
				Ipaddr:    ipaddr.NewIPAddressString("2001:db8::1").GetAddress(),
			},
		},
		{
			name: "IPv6 Exact match",
			in: cliArgStruct{
				Ipstring: "2001:db8::/32",
				Exact:    true,
			},
			want: cliArgStruct{
				Ipstring: "2001:db8::/32",
				Exact:    true,
				Longest:  false,
				V4:       false,
				V6:       true,
				Ipaddr:   ipaddr.NewIPAddressString("2001:db8::/32").GetAddress(),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := argMassage(tt.in)

			// Compare relevant fields as comparing regexes and complex objects might be tricky
			if got.Ipstring != tt.want.Ipstring {
				t.Errorf("argMassage().Ipstring = %v, want %v", got.Ipstring, tt.want.Ipstring)
			}
			if got.Longest != tt.want.Longest {
				t.Errorf("argMassage().Longest = %v, want %v", got.Longest, tt.want.Longest)
			}
			if got.Exact != tt.want.Exact {
				t.Errorf("argMassage().Exact = %v, want %v", got.Exact, tt.want.Exact)
			}
			if got.V4 != tt.want.V4 {
				t.Errorf("argMassage().V4 = %v, want %v", got.V4, tt.want.V4)
			}
			if got.V6 != tt.want.V6 {
				t.Errorf("argMassage().V6 = %v, want %v", got.V6, tt.want.V6)
			}
			if got.Ipaddr != nil && tt.want.Ipaddr != nil {
				if !got.Ipaddr.Equal(tt.want.Ipaddr) {
					t.Errorf("argMassage().Ipaddr = %v, want %v", got.Ipaddr, tt.want.Ipaddr)
				}
			}
		})
	}
}
