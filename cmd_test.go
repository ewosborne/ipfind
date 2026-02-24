package main

import (
	"bufio"
	"bytes"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/seancfoley/ipaddress-go/ipaddr"
)

func mustParseIP(s string) *ipaddr.IPAddress {
	a := ipaddr.NewIPAddressString(s).GetAddress()
	if a == nil {
		panic("invalid IP: " + s)
	}
	return a
}

func ipAddrsToStrings(addrs []*ipaddr.IPAddress) []string {
	out := make([]string, len(addrs))
	for i, a := range addrs {
		out[i] = a.String()
	}
	return out
}

func TestGetIPv4AddressesFromLine(t *testing.T) {
	type ipExtractCase struct {
		name    string
		line    string
		wantLen int
		wantIPs []string
	}

	cases := []ipExtractCase{
		{name: "empty", line: "", wantLen: 0},
		{name: "no IPs", line: "hello world", wantLen: 0},
		{name: "single host", line: "plain ip 1.2.3.4 appears", wantLen: 1, wantIPs: []string{"1.2.3.4"}},
		{name: "single CIDR", line: "cidr: 10.0.0.0/8", wantLen: 1, wantIPs: []string{"10.0.0.0/8"}},
		{name: "multiple IPs", line: "a 1.2.3.4 b 5.6.7.8 c", wantLen: 2, wantIPs: []string{"1.2.3.4", "5.6.7.8"}},
		{name: "IPv6 in line", line: "addr ::1 here", wantLen: 0},
		{name: "invalid octets", line: "999.999.999.999", wantLen: 0},
		{name: "mixed valid invalid", line: "1.2.3.4 and 256.1.1.1", wantLen: 1, wantIPs: []string{"1.2.3.4"}},
		{name: "sample line", line: "1.0.0.0/8 classA", wantLen: 1, wantIPs: []string{"1.0.0.0/8"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := get_ipv4_addresses_from_line(tc.line)
			if len(got) != tc.wantLen {
				t.Errorf("get_ipv4_addresses_from_line(%q) len = %d, want %d", tc.line, len(got), tc.wantLen)
			}
			if tc.wantIPs != nil && !slices.Equal(ipAddrsToStrings(got), tc.wantIPs) {
				t.Errorf("get_ipv4_addresses_from_line(%q) = %v, want %v", tc.line, ipAddrsToStrings(got), tc.wantIPs)
			}
		})
	}
}

func TestGetIPv6AddressesFromLine(t *testing.T) {
	type ipExtractCase struct {
		name    string
		line    string
		wantLen int
		wantIPs []string
	}

	cases := []ipExtractCase{
		{name: "empty", line: "", wantLen: 0},
		{name: "no IPs", line: "hello world", wantLen: 0},
		{name: "loopback", line: "addr ::1 here", wantLen: 1, wantIPs: []string{"::1"}},
		{name: "IPv6 CIDR", line: "net 2001:db8::/32", wantLen: 1, wantIPs: []string{"2001:db8::/32"}},
		{name: "link local", line: "fe80::1", wantLen: 1, wantIPs: []string{"fe80::1"}},
		{name: "IPv4 in line", line: "1.2.3.4 only", wantLen: 0},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := get_ipv6_addresses_from_line(tc.line)
			if len(got) != tc.wantLen {
				t.Errorf("get_ipv6_addresses_from_line(%q) len = %d, want %d", tc.line, len(got), tc.wantLen)
			}
			if tc.wantIPs != nil && !slices.Equal(ipAddrsToStrings(got), tc.wantIPs) {
				t.Errorf("get_ipv6_addresses_from_line(%q) = %v, want %v", tc.line, ipAddrsToStrings(got), tc.wantIPs)
			}
		})
	}
}

func TestGetFilesFromArgs(t *testing.T) {
	tmp := t.TempDir()

	f1 := filepath.Join(tmp, "a.txt")
	if err := os.WriteFile(f1, []byte("a"), 0644); err != nil {
		t.Fatal(err)
	}
	f2 := filepath.Join(tmp, "b.txt")
	if err := os.WriteFile(f2, []byte("b"), 0644); err != nil {
		t.Fatal(err)
	}

	subdir := filepath.Join(tmp, "sub")
	if err := os.Mkdir(subdir, 0755); err != nil {
		t.Fatal(err)
	}
	f3 := filepath.Join(subdir, "c.txt")
	if err := os.WriteFile(f3, []byte("c"), 0644); err != nil {
		t.Fatal(err)
	}

	type filesCase struct {
		name      string
		input     []string
		wantFiles []string
		wantErr   bool
	}

	cases := []filesCase{
		{
			name:      "single file",
			input:     []string{f1},
			wantFiles: []string{f1},
		},
		{
			name:      "directory",
			input:     []string{tmp},
			wantFiles: []string{f1, f2, f3},
		},
		{
			name:      "file and dir",
			input:     []string{f1, subdir},
			wantFiles: []string{f1, f3},
		},
		{
			name:    "nonexistent",
			input:   []string{filepath.Join(tmp, "nonexistent")},
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := getFilesFromArgs(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Errorf("getFilesFromArgs(%v) want error, got nil", tc.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("getFilesFromArgs(%v) err = %v", tc.input, err)
			}
			slices.Sort(got)
			slices.Sort(tc.wantFiles)
			if !slices.Equal(got, tc.wantFiles) {
				t.Errorf("getFilesFromArgs(%v) = %v, want %v", tc.input, got, tc.wantFiles)
			}
		})
	}
}

func TestProcessSingleFile(t *testing.T) {
	type processCase struct {
		name        string
		input       string
		targetIP    string
		mode        string // "exact", "contains", "subnet", "longest"
		wantContain string
		wantLines   int
	}

	cases := []processCase{
		{
			name:        "exact match",
			input:       "1.2.3.4/32\n",
			targetIP:    "1.2.3.4",
			mode:        "exact",
			wantContain: "1.2.3.4/32",
			wantLines:   1,
		},
		{
			name:        "contains match",
			input:       "10.1.2.3\n",
			targetIP:    "10.0.0.0/8",
			mode:        "contains",
			wantContain: "10.1.2.3",
			wantLines:   1,
		},
		{
			name:        "subnet match",
			input:       "1.2.3.0/24\n",
			targetIP:    "1.2.3.4",
			mode:        "subnet",
			wantContain: "1.2.3.0/24",
			wantLines:   1,
		},
		{
			name:        "no match",
			input:       "10.0.0.0/8\n",
			targetIP:    "1.2.3.4",
			mode:        "contains",
			wantContain: "",
			wantLines:   0,
		},
		{
			name:        "longest prefix",
			input:       "1.0.0.0/8\n1.2.3.0/24\n",
			targetIP:    "1.2.3.4",
			mode:        "longest",
			wantContain: "1.2.3.0/24",
			wantLines:   1,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			args := cliArgStruct{
				Ipaddr: mustParseIP(tc.targetIP),
				V4:     true,
				V6:     false,
			}
			switch tc.mode {
			case "exact":
				args.Exact = true
			case "contains":
				args.Contains = true
			case "subnet":
				args.Subnet = true
			case "longest":
				args.Longest = true
			default:
				t.Fatalf("unknown mode %q", tc.mode)
			}

			file := inputFile{
				Filename: "test.txt",
				IsStdin:  false,
				Scanner:  bufio.NewScanner(strings.NewReader(tc.input)),
			}

			old := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w
			defer func() {
				os.Stdout = old
				w.Close()
			}()

			process_single_file(args, file)
			w.Close()

			var buf bytes.Buffer
			buf.ReadFrom(r)
			got := buf.String()

			if tc.wantContain != "" && !strings.Contains(got, tc.wantContain) {
				t.Errorf("output %q does not contain %q", got, tc.wantContain)
			}
			if tc.wantLines > 0 {
				lines := strings.Split(strings.TrimSpace(got), "\n")
				if len(lines) < tc.wantLines {
					t.Errorf("output has %d lines, want at least %d: %q", len(lines), tc.wantLines, got)
				}
			}
			if tc.wantLines == 0 && tc.wantContain == "" && got != "" {
				t.Errorf("expected no output, got %q", got)
			}
		})
	}
}

func TestIpcmd(t *testing.T) {
	tmp := t.TempDir()
	f := filepath.Join(tmp, "input.txt")
	content := "1.2.3.0/24\n10.0.0.0/8\n"
	if err := os.WriteFile(f, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	type ipcmdCase struct {
		name        string
		args        cliArgStruct
		wantContain string
		wantErr     bool
	}

	cases := []ipcmdCase{
		{
			name: "file input exact match",
			args: cliArgStruct{
				Ipaddr:     mustParseIP("1.2.3.4"),
				V4:         true,
				V6:         false,
				Exact:      false,
				Subnet:     true,
				InputFiles: []string{f},
			},
			wantContain: "1.2.3.0/24",
		},
		{
			name: "file input contains match",
			args: cliArgStruct{
				Ipaddr:     mustParseIP("10.0.0.0/8"),
				V4:         true,
				V6:         false,
				Contains:   true,
				InputFiles: []string{f},
			},
			wantContain: "10.0.0.0/8",
		},
		{
			name: "nonexistent file",
			args: cliArgStruct{
				Ipaddr:     mustParseIP("1.2.3.4"),
				V4:         true,
				V6:         false,
				Subnet:     true,
				InputFiles: []string{filepath.Join(tmp, "nonexistent")},
			},
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			old := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w
			defer func() {
				os.Stdout = old
				w.Close()
			}()

			err := ipcmd(tc.args)
			w.Close()

			var buf bytes.Buffer
			buf.ReadFrom(r)
			got := buf.String()

			if tc.wantErr {
				if err == nil {
					t.Errorf("ipcmd want error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("ipcmd err = %v", err)
			}
			if tc.wantContain != "" && !strings.Contains(got, tc.wantContain) {
				t.Errorf("output %q does not contain %q", got, tc.wantContain)
			}
		})
	}
}
