package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// helper: run ipcmd with provided file content and return output lines (trimmed)
func runWithContent(t *testing.T, args cliArgStruct, content string) []string {
	t.Helper()
	dir := t.TempDir()
	fpath := filepath.Join(dir, "input.txt")
	if err := os.WriteFile(fpath, []byte(content), 0o644); err != nil {
		t.Fatalf("write temp file: %v", err)
	}
	args.inputFile = fpath

	// capture stdout
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stdout = w

	// run
	ipcmd(args)

	// restore and collect
	_ = w.Close()
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	os.Stdout = old

	// split and normalize
	out := strings.Split(buf.String(), "\n")
	var lines []string
	for _, l := range out {
		l = strings.TrimSpace(strings.TrimPrefix(l, "\t"))
		if l == "" {
			continue
		}
		lines = append(lines, l)
	}
	return lines
}

// assertContainsAll checks that every string in wants is present in at least one element of got.
func assertContainsAll(t *testing.T, got []string, wants []string, caseName string) {
	t.Helper()
	for _, need := range wants {
		found := false
		for _, v := range got {
			if strings.Contains(v, need) {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("%s: expected output to contain %q but it did not. Got output lines:\n%v", caseName, need, got)
		}
	}
}

func TestIPCmd_ExactCases(t *testing.T) {
	cases := []struct {
		name         string
		args         cliArgStruct
		content      string
		wantContains []string
		wantCount    int
		wantAllEqual string
	}{
		{name: "ipv4 exact", args: cliArgStruct{ipaddr: "192.168.1.5", exact: true}, content: "line1 192.168.1.5 some\nother 192.168.1.6\nhost 192.168.1.5/32 extra\n", wantContains: []string{"192.168.1.5"}, wantCount: -1},
		{name: "ipv4 exact network-only", args: cliArgStruct{ipaddr: "10.0.0.1", exact: true, networkOnly: true}, content: "a 10.0.0.1\nb 10.0.0.1 extra\n", wantCount: 2, wantAllEqual: "10.0.0.1"},
		{name: "ipv6 exact", args: cliArgStruct{ipaddr: "2001:db8::1", exact: true}, content: "foo 2001:db8::1\nbar 2001:db8::2\n", wantContains: []string{"2001:db8::1"}, wantCount: -1},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := runWithContent(t, c.args, c.content)
			if c.wantCount >= 0 {
				if len(got) != c.wantCount {
					t.Fatalf("%s: expected count %d got %d; output=%#v", c.name, c.wantCount, len(got), got)
				}
			} else if len(got) == 0 {
				t.Fatalf("%s: expected at least one result, got none", c.name)
			}
			if c.wantAllEqual != "" {
				for _, v := range got {
					if v != c.wantAllEqual {
						t.Fatalf("%s: expected all outputs equal %q but found %q; output=%#v", c.name, c.wantAllEqual, v, got)
					}
				}
			}
			assertContainsAll(t, got, c.wantContains, c.name)
		})
	}
}

func TestIPCmd_SubnetAndLongest(t *testing.T) {
	cases := []struct {
		name         string
		args         cliArgStruct
		content      string
		wantContains []string
		wantCount    int
	}{
		{name: "subnet network-only ipv4", args: cliArgStruct{ipaddr: "192.168.1.5", subnet: true, networkOnly: true}, content: "one 192.168.0.0/16\ntwo 10.0.0.0/8\nthree 192.168.1.0/24\n", wantContains: []string{"192.168.0.0/16", "192.168.1.0/24"}, wantCount: 2},
		{name: "longest network-only ipv4", args: cliArgStruct{ipaddr: "192.168.1.5", longest: true, networkOnly: true}, content: "one 192.168.0.0/16\ntwo 10.0.0.0/8\nthree 192.168.1.0/24\n", wantContains: []string{"192.168.1.0/24"}, wantCount: 1},
		{name: "subnet ipv6", args: cliArgStruct{ipaddr: "2001:db8::5", subnet: true, networkOnly: true}, content: "a 2001:db8::/32\nb 2001:db8:1::/48\n", wantContains: []string{"2001:db8::/32"}, wantCount: 1},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := runWithContent(t, c.args, c.content)
			if c.wantCount >= 0 && len(got) != c.wantCount {
				t.Fatalf("%s: expected count %d got %d; output=%#v", c.name, c.wantCount, len(got), got)
			}
			assertContainsAll(t, got, c.wantContains, c.name)
		})
	}
}

func TestIPCmd_InvalidAndMalformed(t *testing.T) {
	cases := []struct {
		name         string
		args         cliArgStruct
		content      string
		wantContains []string
		wantCount    int
	}{
		{name: "invalid ip strings", args: cliArgStruct{ipaddr: "1.2.3.4", exact: true}, content: "bad 999.999.999.999 notanip ::gggg\nvalid 1.2.3.4\n", wantContains: []string{"valid 1.2.3.4"}, wantCount: 1},
		{name: "malformed cidr ignored", args: cliArgStruct{ipaddr: "192.168.1.5", subnet: true}, content: "badcidr 192.168.1.0/33\nok 192.168.1.0/24\n", wantContains: []string{"192.168.1.0/24"}, wantCount: 1},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := runWithContent(t, c.args, c.content)
			if c.wantCount >= 0 && len(got) != c.wantCount {
				t.Fatalf("%s: expected count %d got %d; output=%#v", c.name, c.wantCount, len(got), got)
			}
			assertContainsAll(t, got, c.wantContains, c.name)
		})
	}
}

func TestIPCmd_EdgeCases(t *testing.T) {
	cases := []struct {
		name         string
		args         cliArgStruct
		content      string
		wantContains []string
		wantCount    int
	}{
		{name: "multiple ips in line", args: cliArgStruct{ipaddr: "192.168.1.5", exact: true}, content: "mix 10.0.0.1 192.168.1.5 tail\n", wantContains: []string{"192.168.1.5"}, wantCount: 1},
		{name: "punctuation around ip", args: cliArgStruct{ipaddr: "192.168.1.5", exact: true}, content: "addr:(192.168.1.5), other\n", wantContains: []string{"192.168.1.5"}, wantCount: 1},
		{name: "longest prefers host over network", args: cliArgStruct{ipaddr: "192.168.1.5", longest: true, networkOnly: true}, content: "net 192.168.1.0/24\nhost 192.168.1.5\n", wantContains: []string{"192.168.1.5"}, wantCount: 1},
		{name: "ipv6 case insensitivity", args: cliArgStruct{ipaddr: "2001:DB8::1", exact: true}, content: "line 2001:db8::1\n", wantContains: []string{"2001:db8::1"}, wantCount: 1},
		{name: "ip with /32 matches exact", args: cliArgStruct{ipaddr: "192.168.1.5", exact: true}, content: "entry 192.168.1.5/32\n", wantContains: []string{"192.168.1.5"}, wantCount: 1},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := runWithContent(t, c.args, c.content)
			if c.wantCount >= 0 && len(got) != c.wantCount {
				t.Fatalf("%s: expected count %d got %d; output=%#v", c.name, c.wantCount, len(got), got)
			}
			assertContainsAll(t, got, c.wantContains, c.name)
		})
	}
}
