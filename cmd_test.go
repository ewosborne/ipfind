package main

import (
	"bytes"
	"os"
	"regexp"
	"strings"
	"testing"
)

func TestGetIPAddressesFromLine(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		family   int
		regex    *regexp.Regexp
		expected []string
	}{
		{
			name:     "IPv4 simple",
			line:     "Found 192.168.1.1 and 10.0.0.1/24",
			family:   4,
			regex:    ipv4Regex_noSlash,
			expected: []string{"192.168.1.1", "10.0.0.1/24"},
		},
		{
			name:     "IPv4 with slash only",
			line:     "Found 192.168.1.1 and 10.0.0.0/24",
			family:   4,
			regex:    ipv4Regex_withSlash,
			expected: []string{"10.0.0.0/24"},
		},
		{
			name:     "IPv6 simple",
			line:     "Check 2001:db8::1 or 2001:db8:abcd::/64",
			family:   6,
			regex:    ipv6Regex_noSlash,
			expected: []string{"2001:db8::1", "2001:db8:abcd::/64"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := get_ip_addresses_from_line(tt.regex, tt.family, tt.line)
			if len(got) != len(tt.expected) {
				t.Fatalf("got %v, want %v", got, tt.expected)
			}
			for i := range got {
				if got[i] != tt.expected[i] {
					t.Errorf("at index %d: got %s, want %s", i, got[i], tt.expected[i])
				}
			}
		})
	}
}

func TestIPCmd(t *testing.T) {
	// Setup a temporary file
	content := `
192.168.1.1
192.168.2.0/24
10.0.0.5
2001:db8::1
2001:db8:ffff::/48
`
	tmpFile, err := os.CreateTemp("", "ipfind_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatal(err)
	}
	tmpFile.Close()

	tests := []struct {
		name       string
		target     string
		mode       string // "exact", "subnet", "contains"
		format     string // "text", "json", "trie"
		wantInSub  string
		wantNotSub string
	}{
		{
			name:      "IPv4 Subnet Match",
			target:    "192.168.0.0/16",
			mode:      "subnet",
			format:    "text",
			wantInSub: "192.168.1.1",
		},
		{
			name:      "IPv4 Exact Match",
			target:    "192.168.2.0/24",
			mode:      "exact",
			format:    "text",
			wantInSub: "192.168.2.0/24",
		},
		{
			name:       "IPv4 Exact Negative",
			target:     "192.168.2.1",
			mode:       "exact",
			format:     "text",
			wantNotSub: "192.168.2.0/24",
		},
		{
			name:      "IPv4 Contains Match",
			target:    "192.168.2.15",
			mode:      "contains",
			format:    "text",
			wantInSub: "192.168.2.0/24",
		},
		{
			name:      "IPv6 Subnet Match",
			target:    "2001:db8::/32",
			mode:      "subnet",
			format:    "text",
			wantInSub: "2001:db8::1",
		},
		{
			name:      "JSON Output",
			target:    "10.0.0.0/8",
			mode:      "subnet",
			format:    "json",
			wantInSub: `"Line": "10.0.0.5"`,
		},
		{
			name:      "Trie Output IPv4",
			target:    "192.168.0.0/16",
			mode:      "subnet",
			format:    "trie",
			wantInSub: "192.168.1.1",
		},
		{
			name:      "Trie Output IPv6",
			target:    "2001:db8::/32",
			mode:      "subnet",
			format:    "trie",
			wantInSub: "2001:db8::1",
		},
		{
			name:       "Negative Subnet Match",
			target:     "10.0.0.0/24",
			mode:       "subnet",
			format:     "text",
			wantNotSub: "192.168.1.1",
		},
		{
			name:       "Negative Contains Match",
			target:     "192.168.1.1",
			mode:       "contains",
			format:     "text",
			wantNotSub: "192.168.2.0/24",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := cliArgStruct{
				Ipstring:   tt.target,
				InputFiles: []string{tmpFile.Name()},
			}
			switch tt.mode {
			case "exact":
				args.Exact = true
			case "contains":
				args.Contains = true
			case "subnet":
				args.Subnet = true
			}
			switch tt.format {
			case "json":
				args.Json = true
			case "trie":
				args.Trie = true
			}

			// Use argMassage to initialize internal fields (regex, IP object, etc.)
			args = argMassage(args)

			var buf bytes.Buffer
			err := ipcmd(&buf, args)
			if err != nil {
				t.Errorf("ipcmd failed: %v", err)
			}

			out := buf.String()
			if tt.wantInSub != "" && !strings.Contains(out, tt.wantInSub) {
				t.Errorf("expected output to contain %q, but got:\n%s", tt.wantInSub, out)
			}
			if tt.wantNotSub != "" && strings.Contains(out, tt.wantNotSub) {
				t.Errorf("expected output NOT to contain %q, but got:\n%s", tt.wantNotSub, out)
			}
		})
	}
}
