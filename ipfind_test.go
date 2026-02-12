package main_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func buildBinary(t *testing.T, dir string) string {
	t.Helper()
	bin := filepath.Join(dir, "ipfind")
	cmd := exec.Command("go", "build", "-o", bin, ".")
	cmd.Env = append(os.Environ())
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("build failed: %v\n%s", err, string(out))
	}
	return bin
}

func runCmd(t *testing.T, bin string, args ...string) string {
	t.Helper()
	cmd := exec.Command(bin, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		// include output for debugging
		t.Fatalf("command failed: %v\n%s", err, string(out))
	}
	return string(out)
}

func TestExactSubnetLongest(t *testing.T) {
	td := t.TempDir()
	bin := buildBinary(t, td)

	// create sample file
	sample := `plain ip 1.2.3.4 appears here
cidrs: 1.2.3.0/24, 1.2.3.128/25, 1.0.0.0/8
small net: 1.2.3.4/32
overlap: 1.2.3.0/25
weird: 1.2.3.4/31 and 1.2.3.4
`
	samplePath := filepath.Join(td, "sample.txt")
	if err := os.WriteFile(samplePath, []byte(sample), 0644); err != nil {
		t.Fatal(err)
	}

	// Exact: should include lines with literal 1.2.3.4 token
	out := runCmd(t, bin, "-e", "1.2.3.4", samplePath)
	if !strings.Contains(out, "plain ip 1.2.3.4 appears here") {
		t.Fatalf("exact missing expected line; got:\n%s", out)
	}
	if !strings.Contains(out, "weird: 1.2.3.4/31 and 1.2.3.4") {
		t.Fatalf("exact missing weird line; got:\n%s", out)
	}

	// Subnet: should include lines that have CIDRs containing the IP
	out = runCmd(t, bin, "-s", "1.2.3.4", samplePath)
	if !strings.Contains(out, "cidrs: 1.2.3.0/24") {
		t.Fatalf("subnet missing expected cidr; got:\n%s", out)
	}
	if !strings.Contains(out, "small net: 1.2.3.4/32") {
		t.Fatalf("subnet missing /32 line; got:\n%s", out)
	}

	// Longest-match: should only return the line with the most specific prefix (/32)
	out = runCmd(t, bin, "-l", "1.2.3.4", samplePath)
	if !strings.Contains(out, "small net: 1.2.3.4/32") {
		t.Fatalf("longest missing /32; got:\n%s", out)
	}

	// Mask range: restrict to prefixes between 20 and 28 (so /24 matches, /32 excluded)
	out = runCmd(t, bin, "-s", "--mask-range", "20-28", "1.2.3.4", samplePath)
	if !strings.Contains(out, "cidrs: 1.2.3.0/24") {
		t.Fatalf("mask-range missing expected /24; got:\n%s", out)
	}
	if strings.Contains(out, "small net: 1.2.3.4/32") {
		t.Fatalf("mask-range should not include /32; got:\n%s", out)
	}
}

func TestSampleTxtIntegration(t *testing.T) {
	// use repository sample.txt if present
	samplePath := "sample.txt"
	if _, err := os.Stat(samplePath); err != nil {
		if os.IsNotExist(err) {
			t.Skip("sample.txt not present in repo; skipping")
		}
		t.Fatalf("stat sample.txt: %v", err)
	}

	td := t.TempDir()
	bin := buildBinary(t, td)

	// Exact: expect at least one occurrence of '1.2.3.4'
	outExact := runCmd(t, bin, "-e", "1.2.3.4", samplePath)
	if !strings.Contains(outExact, "1.2.3.4") {
		t.Fatalf("exact on sample.txt did not contain 1.2.3.4; got:\n%s", outExact)
	}

	// Subnet: expect at least one CIDR (a '/' in the output)
	outSubnet := runCmd(t, bin, "-s", "1.2.3.4", samplePath)
	if !strings.Contains(outSubnet, "/") {
		t.Fatalf("subnet on sample.txt returned no CIDR-like output; got:\n%s", outSubnet)
	}

	// Longest: output lines should be subset of subnet output
	outLongest := runCmd(t, bin, "-l", "1.2.3.4", samplePath)
	if strings.TrimSpace(outLongest) == "" {
		t.Fatalf("longest returned no lines; got empty output")
	}
	subnetLines := map[string]bool{}
	for _, ln := range strings.Split(strings.TrimRight(outSubnet, "\n"), "\n") {
		subnetLines[strings.TrimSpace(ln)] = true
	}
	for _, ln := range strings.Split(strings.TrimRight(outLongest, "\n"), "\n") {
		if ln = strings.TrimSpace(ln); ln == "" {
			continue
		}
		if !subnetLines[ln] {
			t.Fatalf("longest output line not present in subnet output: %q", ln)
		}
	}

	// Mask-range: ensure returned CIDR prefixes fall within range 20-28
	outMask := runCmd(t, bin, "-s", "--mask-range", "20-28", "1.2.3.4", samplePath)
	for _, ln := range strings.Split(strings.TrimRight(outMask, "\n"), "\n") {
		// find tokens with /
		toks := strings.Fields(ln)
		found := false
		for _, tkn := range toks {
			if !strings.Contains(tkn, "/") {
				continue
			}
			parts := strings.SplitN(tkn, "/", 2)
			if len(parts) != 2 {
				continue
			}
			pref := parts[1]
			if pref == "" {
				continue
			}
			// extract leading digits to handle trailing punctuation
			numstr := ""
			for _, ch := range pref {
				if ch >= '0' && ch <= '9' {
					numstr += string(ch)
				} else {
					break
				}
			}
			if numstr == "" {
				continue
			}
			n, err := strconv.Atoi(numstr)
			if err != nil {
				continue
			}
			if n >= 20 && n <= 28 {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("mask-range returned a line with no CIDR in 20-28: %s", ln)
		}
	}
}
