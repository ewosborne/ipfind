package main_test

import (
	"os"
	"os/exec"
	"path/filepath"
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

	// Exact: should include the plain token and the line with the plain token
	outExact := runCmd(t, bin, "-e", "1.2.3.4", samplePath)
	if !strings.Contains(outExact, "plain ip 1.2.3.4 appears here") {
		t.Fatalf("exact on sample.txt missing expected plain line; got:\n%s", outExact)
	}
	if !strings.Contains(outExact, "weird: 1.2.3.4/31 and 1.2.3.4") {
		t.Fatalf("exact on sample.txt missing weird line; got:\n%s", outExact)
	}

	// Subnet: should include known CIDR-containing lines that contain 1.2.3.4
	outSubnet := runCmd(t, bin, "-s", "1.2.3.4", samplePath)
	expectSubnetLines := []string{
		"cidr2: 1.0.0.0/8",
		"cidr3: 1.2.3.0/24",
		"cidr4: 1.2.3.0/24, 1.2.3.128/25, 1.0.0.0/8",
		"small net: 1.2.3.4/32",
		"overlap: 1.2.3.0/25",
		"weird: 1.2.3.4/31 and 1.2.3.4",
	}
	for _, ex := range expectSubnetLines {
		if !strings.Contains(outSubnet, ex) {
			t.Fatalf("subnet on sample.txt missing expected line %q; got:\n%s", ex, outSubnet)
		}
	}

	// Longest: should return the /32 line only (most-specific)
	outLongest := runCmd(t, bin, "-l", "1.2.3.4", samplePath)
	if !strings.Contains(outLongest, "small net: 1.2.3.4/32") {
		t.Fatalf("longest on sample.txt missing /32 line; got:\n%s", outLongest)
	}
	if strings.Contains(outLongest, "cidr2: 1.0.0.0/8") {
		t.Fatalf("longest should not include less specific CIDRs like /8; got:\n%s", outLongest)
	}

	// Mask-range: ensure returned lines include at least one prefix between 20-28
	outMask := runCmd(t, bin, "-s", "--mask-range", "20-28", "1.2.3.4", samplePath)
	if !strings.Contains(outMask, "cidr3: 1.2.3.0/24") && !strings.Contains(outMask, "cidr4:") {
		t.Fatalf("mask-range on sample.txt did not return expected /24 lines; got:\n%s", outMask)
	}
}
