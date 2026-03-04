package main

import (
	"testing"
)

func TestArgMassage(t *testing.T) {
	tests := []struct {
		name     string
		input    cliArgStruct
		wantSub  bool
		wantEx   bool
		wantCont bool
		wantFam  int
	}{
		{
			name: "default to subnet",
			input: cliArgStruct{
				Ipstring: "192.168.1.0/24",
			},
			wantSub: true,
			wantFam: 4,
		},
		{
			name: "explicit exact",
			input: cliArgStruct{
				Ipstring: "10.0.0.1",
				Exact:    true,
			},
			wantEx:  true,
			wantSub: false,
			wantFam: 4,
		},
		{
			name: "explicit contains",
			input: cliArgStruct{
				Ipstring: "2001:db8::/32",
				Contains: true,
			},
			wantCont: true,
			wantSub:  false,
			wantFam:  6,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Note: argMassage calls log.Fatalf on invalid IP, so we only test valid ones here.
			got := argMassage(tt.input)

			if got.Subnet != tt.wantSub {
				t.Errorf("Subnet = %v, want %v", got.Subnet, tt.wantSub)
			}
			if got.Exact != tt.wantEx {
				t.Errorf("Exact = %v, want %v", got.Exact, tt.wantEx)
			}
			if got.Contains != tt.wantCont {
				t.Errorf("Contains = %v, want %v", got.Contains, tt.wantCont)
			}
			if got.addressFamily != tt.wantFam {
				t.Errorf("addressFamily = %v, want %v", got.addressFamily, tt.wantFam)
			}
			if got.Ipaddr == nil {
				t.Fatal("Ipaddr is nil")
			}
		})
	}
}

func TestArgMassageCanonize(t *testing.T) {
	input := cliArgStruct{
		Ipstring: "192.168.1.5/24",
		Canonize: true,
	}
	got := argMassage(input)
	expected := "192.168.1.0/24"
	if got.Ipaddr.String() != expected {
		t.Errorf("Canonize failed: got %v, want %v", got.Ipaddr.String(), expected)
	}

	inputNoCanon := cliArgStruct{
		Ipstring: "192.168.1.5/24",
		Canonize: false,
	}
	gotNoCanon := argMassage(inputNoCanon)
	expectedNoCanon := "192.168.1.5/24"
	if gotNoCanon.Ipaddr.String() != expectedNoCanon {
		t.Errorf("Non-canonize failed: got %v, want %v", gotNoCanon.Ipaddr.String(), expectedNoCanon)
	}
}
