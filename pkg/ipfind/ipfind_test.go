package ipfind

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/seancfoley/ipaddress-go/ipaddr"
)

// helper to simplify creating test arguments
func setupTestArgs(ip string, exact, contains, subnet bool) Args {
	args := Args{
		Ipstring: ip,
		Exact:    exact,
		Contains: contains,
		Subnet:   subnet,
		Canonize: true, // default behavior
	}
	return ArgMassage(args)
}

func TestProcessReader(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name               string
		input              string
		targetIP           string
		mode               string // "exact", "contains", "subnet"
		expectedLines      []int  // line numbers expected to match
		expectedMatchCount int    // total number of IP objects matched
	}{
		// IPv4 POSITIVE
		{"IPv4 Exact Match", "found 1.2.3.4 here", "1.2.3.4", "exact", []int{1}, 1},
		{"IPv4 Subnet Match", "192.168.1.5", "192.168.1.0/24", "subnet", []int{1}, 1},
		{"IPv4 Contains Match", "10.0.0.0/8", "10.0.0.1", "contains", []int{1}, 1},

		// IPv4 NEGATIVE
		{"IPv4 Exact Fail", "found 1.2.3.5 here", "1.2.3.4", "exact", []int{}, 0},
		{"IPv4 Subnet Fail", "192.167.1.5", "192.168.1.0/24", "subnet", []int{}, 0},

		// IPv6 POSITIVE
		{"IPv6 Exact Match", "lookup 2001:db8::1", "2001:db8::1", "exact", []int{1}, 1},
		{"IPv6 Subnet Match", "2001:db8::dead:beef", "2001:db8::/32", "subnet", []int{1}, 1},
		{"IPv6 Contains Match", "2001:db8::/32", "2001:db8::1", "contains", []int{1}, 1},

		// IPv6 NEGATIVE
		{"IPv6 Exact Fail", "2001:db8::2", "2001:db8::1", "exact", []int{}, 0},
		{"IPv6 Subnet Fail", "2002:db8::1", "2001:db8::/32", "subnet", []int{}, 0},

		// EDGE CASES
		{"Empty Lines", "\n\n1.1.1.1\n", "1.1.1.1", "exact", []int{3}, 1},
		{"Multiple IPs on Line", "1.1.1.1 and 2.2.2.2", "1.1.1.1", "exact", []int{1}, 1},
		{"Mixed Garbage", "some text 1.2.3.4 more text", "1.2.3.4", "exact", []int{1}, 1},

		// EDGE CASES (IPv6)
		{"Empty Lines (IPv6)", "\n\n2001:db8::1\n", "2001:db8::1", "exact", []int{3}, 1},
		{"Multiple IPs on Line (IPv6)", "2001:db8::1 and 2001:db8::2", "2001:db8::1", "exact", []int{1}, 1},
		{"Mixed Garbage (IPv6)", "some text 2001:db8::1 more text", "2001:db8::1", "exact", []int{1}, 1},
	}

	for _, tt := range tests {
		tt := tt // capture range variable
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			// Setup args based on mode
			args := setupTestArgs(tt.targetIP, tt.mode == "exact", tt.mode == "contains", tt.mode == "subnet")

			reader := strings.NewReader(tt.input)

			var matchedLines []LineResult
			err := ProcessReader(context.Background(), reader, args, func(lr LineResult) error {
				matchedLines = append(matchedLines, lr)
				return nil
			})

			if err != nil {
				t.Fatalf("ProcessReader failed: %v", err)
			}

			// Check matched line count
			if len(matchedLines) != len(tt.expectedLines) {
				t.Errorf("expected %d matched lines, got %d", len(tt.expectedLines), len(matchedLines))
			}

			// Verify specific line numbers
			for i, lineNum := range tt.expectedLines {
				if i < len(matchedLines) && matchedLines[i].LineNumber != lineNum {
					t.Errorf("expected match on line %d, got %d", lineNum, matchedLines[i].LineNumber)
				}
			}

			// Check total IP objects found across all lines
			actualIPCount := 0
			for _, line := range matchedLines {
				actualIPCount += len(line.ConditionMatches)
			}
			if actualIPCount != tt.expectedMatchCount {
				t.Errorf("expected %d total IP matches, got %d", tt.expectedMatchCount, actualIPCount)
			}
		})
	}
}

func TestWriteBufferedResult(t *testing.T) {
	t.Parallel()
	fp := &FileResult{
		InputFile: InputFile{Filename: "test.txt"},
		MatchedLines: []*LineResult{
			{
				LineNumber: 1,
				Line:       "1.2.3.4",
				ConditionMatches: []*ipaddr.IPAddress{
					ipaddr.NewIPAddressString("1.2.3.4").GetAddress(),
				},
			},
		},
	}

	t.Run("Text Output", func(t *testing.T) {
		t.Parallel()
		var sb strings.Builder
		args := setupTestArgs("1.2.3.4", true, false, false)
		firstJson := true
		err := writeBufferedResult(&sb, fp, args, &firstJson)
		if err != nil {
			t.Fatalf("writeBufferedResult failed: %v", err)
		}
		expected := "test.txt:1:1.2.3.4\n"
		if sb.String() != expected {
			t.Errorf("expected %q, got %q", expected, sb.String())
		}
	})

	t.Run("JSON Output", func(t *testing.T) {
		t.Parallel()
		var sb strings.Builder
		args := setupTestArgs("1.2.3.4", true, false, false)
		args.Json = true
		firstJson := true
		err := writeBufferedResult(&sb, fp, args, &firstJson)
		if err != nil {
			t.Fatalf("writeBufferedResult failed: %v", err)
		}
		var lr LineResult
		if err := json.Unmarshal([]byte(sb.String()), &lr); err != nil {
			t.Fatalf("failed to unmarshal JSON: %v. Output: %q", err, sb.String())
		}
		if lr.Line != "1.2.3.4" {
			t.Errorf("expected Line to be 1.2.3.4, got %q", lr.Line)
		}
	})

	fpIPv6 := &FileResult{
		InputFile: InputFile{Filename: "test.txt"},
		MatchedLines: []*LineResult{
			{
				LineNumber: 1,
				Line:       "2001:db8::1",
				ConditionMatches: []*ipaddr.IPAddress{
					ipaddr.NewIPAddressString("2001:db8::1").GetAddress(),
				},
			},
		},
	}

	t.Run("Text Output (IPv6)", func(t *testing.T) {
		t.Parallel()
		var sb strings.Builder
		args := setupTestArgs("2001:db8::1", true, false, false)
		firstJson := true
		err := writeBufferedResult(&sb, fpIPv6, args, &firstJson)
		if err != nil {
			t.Fatalf("writeBufferedResult failed: %v", err)
		}
		expected := "test.txt:1:2001:db8::1\n"
		if sb.String() != expected {
			t.Errorf("expected %q, got %q", expected, sb.String())
		}
	})

	t.Run("JSON Output (IPv6)", func(t *testing.T) {
		t.Parallel()
		var sb strings.Builder
		args := setupTestArgs("2001:db8::1", true, false, false)
		args.Json = true
		firstJson := true
		err := writeBufferedResult(&sb, fpIPv6, args, &firstJson)
		if err != nil {
			t.Fatalf("writeBufferedResult failed: %v", err)
		}
		var lr LineResult
		if err := json.Unmarshal([]byte(sb.String()), &lr); err != nil {
			t.Fatalf("failed to unmarshal JSON: %v. Output: %q", err, sb.String())
		}
		if lr.Line != "2001:db8::1" {
			t.Errorf("expected Line to be 2001:db8::1, got %q", lr.Line)
		}
	})
}

func TestRunModes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		workers int
		sort    bool
		json    bool
	}{
		{"Default Workers (Parallel)", 0, false, false},
		{"Sequential Path", 1, false, false},
		{"Explicit Parallel", 4, false, false},
		{"Sorted Parallel", 2, true, false},
		{"Parallel JSON", 2, false, true},
	}

	ipVariants := []struct {
		suffix   string
		content  string
		ipstring string
		expect1  string
		expect2  string
	}{
		{"IPv4", "1.2.3.4\n5.6.7.8\n", "0.0.0.0/0", "1.2.3.4", "5.6.7.8"},
		{"IPv6", "2001:db8::1\n2001:db8::2\n", "2001:db8::/32", "2001:db8::1", "2001:db8::2"},
	}

	for _, ipv := range ipVariants {
		for _, tt := range tests {
			tt := tt
			ipv := ipv
			t.Run(tt.name+" "+ipv.suffix, func(t *testing.T) {
				t.Parallel()
				// Create a test file per subtest to avoid cleanup race
				f, _ := os.CreateTemp("", "mode_test_*.txt")
				defer os.Remove(f.Name())
				f.WriteString(ipv.content)
				f.Close()

				args := Args{
					Ipstring:   ipv.ipstring,
					Subnet:     true,
					Workers:    tt.workers,
					Sort:       tt.sort,
					Json:       tt.json,
					InputFiles: []string{f.Name()},
				}
				args = ArgMassage(args)

				var sb strings.Builder
				err := Run(context.Background(), &sb, args)
				if err != nil {
					t.Fatalf("Run failed: %v", err)
				}

				output := sb.String()
				if tt.json {
					var results []LineResult
					if err := json.Unmarshal([]byte(output), &results); err != nil {
						t.Fatalf("failed to unmarshal JSON: %v", err)
					}
					found1, found2 := false, false
					for _, r := range results {
						if strings.Contains(r.Line, ipv.expect1) {
							found1 = true
						}
						if strings.Contains(r.Line, ipv.expect2) {
							found2 = true
						}
					}
					if !found1 || !found2 {
						t.Errorf("JSON output missing expected IPs %s or %s", ipv.expect1, ipv.expect2)
					}
				} else {
					if !strings.Contains(output, ipv.expect1) || !strings.Contains(output, ipv.expect2) {
						t.Error("Output missing expected IP matches")
					}
				}
			})
		}
	}
}

func TestGetFileNamesFromArgsSymlinkDir(t *testing.T) {
	t.Parallel()

	runSymlinkTest := func(t *testing.T, content string) {
		t.Helper()
		tmpDir, err := os.MkdirTemp("", "ipfind_sym_test_*")
		if err != nil {
			t.Fatalf("failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		testFile := filepath.Join(tmpDir, "test.txt")
		os.WriteFile(testFile, []byte(content), 0644)

		// Create a symlink to the directory (in unique dir to avoid parallel test conflicts)
		symlinkDir, _ := os.MkdirTemp("", "ipfind_symlink_*")
		defer os.RemoveAll(symlinkDir)
		symlink := filepath.Join(symlinkDir, "link")
		if err := os.Symlink(tmpDir, symlink); err != nil {
			t.Skipf("skipping symlink test: %v", err)
		}

		files, err := getFileNamesFromArgs([]string{symlink})
		if err != nil {
			t.Fatalf("getFileNamesFromArgs failed: %v", err)
		}

		found := false
		for _, f := range files {
			// It should find the file inside the symlinked directory
			if strings.HasSuffix(f, "test.txt") {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("failed to find file inside symlinked directory. Found files: %v", files)
		}
	}

	t.Run("IPv4", func(t *testing.T) {
		t.Parallel()
		runSymlinkTest(t, "1.2.3.4\n")
	})

	t.Run("IPv6", func(t *testing.T) {
		t.Parallel()
		runSymlinkTest(t, "2001:db8::1\n")
	})
}

func TestRunJSONIndentation(t *testing.T) {
	t.Parallel()

	t.Run("IPv4", func(t *testing.T) {
		t.Parallel()
		args := setupTestArgs("1.2.3.4", false, false, true) // Subnet mode
		args.Json = true

		tmpFile, err := os.CreateTemp("", "ipfind_test_*.txt")
		if err != nil {
			t.Fatalf("failed to create temp file: %v", err)
		}
		defer os.Remove(tmpFile.Name())

		if _, err := tmpFile.WriteString("1.2.3.4\n1.2.3.4\n"); err != nil {
			t.Fatalf("failed to write to temp file: %v", err)
		}
		tmpFile.Close()

		args.InputFiles = []string{tmpFile.Name()}

		var sb strings.Builder
		err = Run(context.Background(), &sb, args)
		if err != nil {
			t.Fatalf("Run failed: %v", err)
		}

		expected := `[
  {
    "LineNumber": 1,
    "Line": "1.2.3.4"
  },
  {
    "LineNumber": 2,
    "Line": "1.2.3.4"
  }
]
`
		if sb.String() != expected {
			t.Errorf("JSON output indentation mismatch.\nExpected:\n%q\nGot:\n%q", expected, sb.String())
		}
	})

	t.Run("IPv6", func(t *testing.T) {
		t.Parallel()
		args := setupTestArgs("2001:db8::/32", false, false, true) // Subnet mode
		args.Json = true

		tmpFile, err := os.CreateTemp("", "ipfind_test_*.txt")
		if err != nil {
			t.Fatalf("failed to create temp file: %v", err)
		}
		defer os.Remove(tmpFile.Name())

		if _, err := tmpFile.WriteString("2001:db8::1\n2001:db8::1\n"); err != nil {
			t.Fatalf("failed to write to temp file: %v", err)
		}
		tmpFile.Close()

		args.InputFiles = []string{tmpFile.Name()}

		var sb strings.Builder
		err = Run(context.Background(), &sb, args)
		if err != nil {
			t.Fatalf("Run failed: %v", err)
		}

		expected := `[
  {
    "LineNumber": 1,
    "Line": "2001:db8::1"
  },
  {
    "LineNumber": 2,
    "Line": "2001:db8::1"
  }
]
`
		if sb.String() != expected {
			t.Errorf("JSON output indentation mismatch.\nExpected:\n%q\nGot:\n%q", expected, sb.String())
		}
	})
}

func TestLongestFlag(t *testing.T) {
	t.Parallel()

	tmpFile, err := os.CreateTemp("", "ipfind_longest_*.txt")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	// Write a mix of short and long prefixes
	// 10.0.0.0/8 should be ignored when /24 is present
	content := "10.0.0.0/8\n10.1.1.0/24\n10.2.2.0/24\n"
	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatalf("failed to write to temp file: %v", err)
	}
	tmpFile.Close()

	t.Run("Text Output Longest", func(t *testing.T) {
		args := setupTestArgs("10.0.0.0/8", false, false, true) // Subnet mode
		args.Longest = true
		args.InputFiles = []string{tmpFile.Name()}

		var sb strings.Builder
		err = Run(context.Background(), &sb, args)
		if err != nil {
			t.Fatalf("Run failed: %v", err)
		}

		output := sb.String()
		if strings.Contains(output, "10.0.0.0/8") {
			t.Errorf("Output should not contain short prefix /8 when /24 exists. Got:\n%s", output)
		}
		if !strings.Contains(output, "10.1.1.0/24") || !strings.Contains(output, "10.2.2.0/24") {
			t.Errorf("Output should contain both longest matches. Got:\n%s", output)
		}
	})

	t.Run("JSON Output Longest", func(t *testing.T) {
		args := setupTestArgs("10.0.0.0/8", false, false, true)
		args.Longest = true
		args.Json = true
		args.InputFiles = []string{tmpFile.Name()}

		var sb strings.Builder
		err = Run(context.Background(), &sb, args)
		if err != nil {
			t.Fatalf("Run failed: %v", err)
		}

		output := sb.String()
		var results []LineResult
		if err := json.Unmarshal([]byte(output), &results); err != nil {
			t.Fatalf("failed to unmarshal JSON: %v. Output:\n%s", err, output)
		}
		if len(results) != 2 {
			t.Errorf("Expected 2 JSON objects, got %d", len(results))
		}
		for _, r := range results {
			if strings.Contains(r.Line, "10.0.0.0/8") {
				t.Error("JSON Output should not contain short prefix /8")
			}
		}
	})

	// IPv6 variants
	tmpFileIPv6, err := os.CreateTemp("", "ipfind_longest_ipv6_*.txt")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFileIPv6.Name())

	contentIPv6 := "2001:db8::/32\n2001:db8:1::/48\n2001:db8:2::/48\n"
	if _, err := tmpFileIPv6.WriteString(contentIPv6); err != nil {
		t.Fatalf("failed to write to temp file: %v", err)
	}
	tmpFileIPv6.Close()

	t.Run("Text Output Longest (IPv6)", func(t *testing.T) {
		args := setupTestArgs("2001:db8::/32", false, false, true) // Subnet mode
		args.Longest = true
		args.InputFiles = []string{tmpFileIPv6.Name()}

		var sb strings.Builder
		err = Run(context.Background(), &sb, args)
		if err != nil {
			t.Fatalf("Run failed: %v", err)
		}

		output := sb.String()
		if strings.Contains(output, "2001:db8::/32") {
			t.Errorf("Output should not contain short prefix /32 when /48 exists. Got:\n%s", output)
		}
		if !strings.Contains(output, "2001:db8:1::/48") || !strings.Contains(output, "2001:db8:2::/48") {
			t.Errorf("Output should contain both longest matches. Got:\n%s", output)
		}
	})

	t.Run("JSON Output Longest (IPv6)", func(t *testing.T) {
		args := setupTestArgs("2001:db8::/32", false, false, true)
		args.Longest = true
		args.Json = true
		args.InputFiles = []string{tmpFileIPv6.Name()}

		var sb strings.Builder
		err = Run(context.Background(), &sb, args)
		if err != nil {
			t.Fatalf("Run failed: %v", err)
		}

		output := sb.String()
		var results []LineResult
		if err := json.Unmarshal([]byte(output), &results); err != nil {
			t.Fatalf("failed to unmarshal JSON: %v. Output:\n%s", err, output)
		}
		if len(results) != 2 {
			t.Errorf("Expected 2 JSON objects, got %d", len(results))
		}
		for _, r := range results {
			if strings.Contains(r.Line, "2001:db8::/32") {
				t.Error("JSON Output should not contain short prefix /32")
			}
		}
	})
}

// --- Fuzz targets ---

func FuzzProcessReader(f *testing.F) {
	f.Add("1.2.3.4", "0.0.0.0/0", false, false, true, false)
	f.Add("2001:db8::1", "::/0", false, false, true, false)
	f.Add("log entry 192.168.1.1 from user", "192.168.1.0/24", false, false, true, false)
	f.Fuzz(func(t *testing.T, input string, targetIP string, exact, contains, subnet, slash bool) {
		args := Args{
			Ipstring: targetIP,
			Exact:    exact,
			Contains: contains,
			Subnet:   subnet,
			Slash:    slash,
			Canonize: true,
		}
		args = ArgMassage(args)
		if args.Ipaddr == nil {
			t.Skip("invalid target IP in fuzz input")
		}

		var count int
		err := ProcessReader(context.Background(), strings.NewReader(input), args,
			func(lr LineResult) error {
				count++
				return nil
			})
		if err != nil {
			// ProcessReader should only return error if scanner fails (not possible with strings.Reader)
			// or if onMatch returns error (which we don't).
			t.Fatalf("ProcessReader failed: %v", err)
		}
	})
}

func FuzzArgMassage(f *testing.F) {
	f.Add("1.2.3.4", false, false, true, false, true)
	f.Add("2001:db8::/32", false, true, false, true, false)
	f.Fuzz(func(t *testing.T, ipstring string, exact, contains, subnet, slash, canonize bool) {
		args := ArgMassage(Args{
			Ipstring: ipstring,
			Exact:    exact,
			Contains: contains,
			Subnet:   subnet,
			Slash:    slash,
			Canonize: canonize,
		})
		if args.Ipaddr != nil {
			_ = args.Ipaddr.String()
		}
	})
}

func FuzzMatchesCondition(f *testing.F) {
	f.Add("1.2.3.4", "1.2.3.0/24", false, false, true)
	f.Add("2001:db8::1", "2001:db8::/32", false, false, true)
	f.Fuzz(func(t *testing.T, ipStr, targetStr string, exact, contains, subnet bool) {
		ipObj := ipaddr.NewIPAddressString(ipStr).GetAddress()
		targetObj := ipaddr.NewIPAddressString(targetStr).GetAddress()
		if ipObj == nil || targetObj == nil {
			t.Skip()
		}
		args := Args{Exact: exact, Contains: contains, Subnet: subnet}
		_ = MatchesCondition(ipObj, targetObj, args)
	})
}

func FuzzGetRegexMatches(f *testing.F) {
	f.Add("log entry 192.168.1.1 from user", true)
	f.Add("::1", false)
	f.Fuzz(func(t *testing.T, line string, isIPv4 bool) {
		lr := LineResult{Line: line}
		af := IPv6
		re := ipv6RegexNoSlash
		if isIPv4 {
			af = IPv4
			re = ipv4RegexNoSlash
		}
		_ = lr.getRegexMatches(re, af)
	})
}

// --- Trie mode ---

func TestTrieMode(t *testing.T) {
	t.Parallel()

	t.Run("IPv4", func(t *testing.T) {
		t.Parallel()
		tmpFile, err := os.CreateTemp("", "ipfind_trie_*.txt")
		if err != nil {
			t.Fatalf("failed to create temp file: %v", err)
		}
		defer os.Remove(tmpFile.Name())
		tmpFile.WriteString("10.0.0.0/24\n10.0.1.0/24\n")
		tmpFile.Close()

		args := setupTestArgs("10.0.0.0/8", false, false, true)
		args.Trie = true
		args.InputFiles = []string{tmpFile.Name()}

		var sb strings.Builder
		err = Run(context.Background(), &sb, args)
		if err != nil {
			t.Fatalf("Run failed: %v", err)
		}
		output := sb.String()
		if output == "" {
			t.Error("Trie output should not be empty")
		}
		if !strings.Contains(output, "10.0.0.0") || !strings.Contains(output, "10.0.1.0") {
			t.Errorf("Trie output should contain matched prefixes. Got:\n%s", output)
		}
	})

	t.Run("IPv6", func(t *testing.T) {
		t.Parallel()
		tmpFile, err := os.CreateTemp("", "ipfind_trie_ipv6_*.txt")
		if err != nil {
			t.Fatalf("failed to create temp file: %v", err)
		}
		defer os.Remove(tmpFile.Name())
		tmpFile.WriteString("2001:db8::/32\n2001:db8:1::/48\n")
		tmpFile.Close()

		args := setupTestArgs("2001:db8::/16", false, false, true)
		args.Trie = true
		args.InputFiles = []string{tmpFile.Name()}

		var sb strings.Builder
		err = Run(context.Background(), &sb, args)
		if err != nil {
			t.Fatalf("Run failed: %v", err)
		}
		output := sb.String()
		if output == "" {
			t.Error("Trie output should not be empty")
		}
		if !strings.Contains(output, "2001:db8") {
			t.Errorf("Trie output should contain matched prefixes. Got:\n%s", output)
		}
	})
}

// --- Slash flag ---

func TestSlashFlag(t *testing.T) {
	t.Parallel()

	tmpFile, err := os.CreateTemp("", "ipfind_slash_*.txt")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.WriteString("10.0.0.1\n10.0.0.0/8\n192.168.1.1\n")
	tmpFile.Close()

	args := Args{
		Ipstring:   "10.0.0.0/8",
		Subnet:     true,
		Slash:      true,
		Canonize:   true,
		InputFiles: []string{tmpFile.Name()},
	}
	args = ArgMassage(args)

	var sb strings.Builder
	err = Run(context.Background(), &sb, args)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	output := sb.String()
	// With Slash=true, only CIDR notation matches; 10.0.0.1 and 192.168.1.1 should NOT match
	if strings.Contains(output, "10.0.0.1") || strings.Contains(output, "192.168.1.1") {
		t.Errorf("Slash mode should not match plain IPs. Got:\n%s", output)
	}
	if !strings.Contains(output, "10.0.0.0/8") {
		t.Errorf("Slash mode should match CIDR. Got:\n%s", output)
	}
}

// --- Canonize=false ---

func TestCanonizeDisabled(t *testing.T) {
	t.Parallel()

	args := setupTestArgs("192.168.1.0/24", false, false, true)
	args.Canonize = false
	args = ArgMassage(args)

	// With canonize disabled, target stays as 192.168.1.0/24 (prefix block)
	// 192.168.1.5 should still match in subnet mode
	input := "192.168.1.5"
	var matched bool
	err := ProcessReader(context.Background(), strings.NewReader(input), args,
		func(lr LineResult) error {
			matched = true
			return nil
		})
	if err != nil {
		t.Fatalf("ProcessReader failed: %v", err)
	}
	if !matched {
		t.Error("Expected match for 192.168.1.5 in subnet 192.168.1.0/24")
	}
}

// --- Invalid IP ---

func TestInvalidIP(t *testing.T) {
	t.Parallel()

	args := ArgMassage(Args{Ipstring: "not-an-ip"})
	if args.Ipaddr != nil {
		t.Error("Invalid IP should result in nil Ipaddr")
	}

	args = ArgMassage(Args{Ipstring: ""})
	// Empty string may or may not produce nil depending on library; ensure no panic

	// ProcessReader with nil Ipaddr should not panic
	args = ArgMassage(Args{Ipstring: "invalid"})
	var count int
	err := ProcessReader(context.Background(), strings.NewReader("1.2.3.4"), args,
		func(lr LineResult) error {
			count++
			return nil
		})
	if err != nil {
		t.Fatalf("ProcessReader should not fail: %v", err)
	}
	if count > 0 {
		t.Error("With nil Ipaddr, no matches should occur")
	}
}

// --- Context cancellation ---

func TestProcessReaderContextCancellation(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	args := setupTestArgs("0.0.0.0/0", false, false, true)
	err := ProcessReader(ctx, strings.NewReader("1.2.3.4\n2.3.4.5\n"), args,
		func(lr LineResult) error { return nil })
	if err != context.Canceled {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}

func TestProcessReaderOnMatchError(t *testing.T) {
	t.Parallel()

	args := setupTestArgs("0.0.0.0/0", false, false, true)
	errSentinel := fmt.Errorf("onMatch error")
	callCount := 0
	err := ProcessReader(context.Background(), strings.NewReader("1.2.3.4\n2.3.4.5\n"), args,
		func(lr LineResult) error {
			callCount++
			return errSentinel
		})
	if err != errSentinel {
		t.Errorf("expected onMatch error to propagate, got %v", err)
	}
	if callCount != 1 {
		t.Errorf("expected onMatch to be called once before error, got %d", callCount)
	}
}

// --- GetInputFilesOrStdin ---

func TestGetInputFilesOrStdin(t *testing.T) {
	t.Parallel()

	t.Run("Empty returns stdin", func(t *testing.T) {
		t.Parallel()
		files, err := GetInputFilesOrStdin([]string{})
		if err != nil {
			t.Fatalf("GetInputFilesOrStdin failed: %v", err)
		}
		if len(files) != 1 || !files[0].IsStdin {
			t.Errorf("expected stdin InputFile, got %v", files)
		}
	})

	t.Run("With files", func(t *testing.T) {
		t.Parallel()
		tmpFile, err := os.CreateTemp("", "ipfind_gifs_*.txt")
		if err != nil {
			t.Fatalf("failed to create temp file: %v", err)
		}
		tmpFile.Close()
		defer os.Remove(tmpFile.Name())

		files, err := GetInputFilesOrStdin([]string{tmpFile.Name()})
		if err != nil {
			t.Fatalf("GetInputFilesOrStdin failed: %v", err)
		}
		if len(files) != 1 || files[0].IsStdin {
			t.Errorf("expected single file InputFile, got %v", files)
		}
		if files[0].Filename != tmpFile.Name() {
			t.Errorf("expected filename %q, got %q", tmpFile.Name(), files[0].Filename)
		}
	})

	t.Run("Nonexistent path returns error", func(t *testing.T) {
		t.Parallel()
		_, err := GetInputFilesOrStdin([]string{"/nonexistent/path/that/does/not/exist"})
		if err == nil {
			t.Error("expected error for nonexistent path")
		}
	})
}

// --- getFileNamesFromArgs ---

func TestGetFileNamesFromArgs(t *testing.T) {
	t.Parallel()

	t.Run("Nonexistent path returns error", func(t *testing.T) {
		t.Parallel()
		_, err := getFileNamesFromArgs([]string{"/nonexistent/path/that/does/not/exist"})
		if err == nil {
			t.Error("expected error for nonexistent path")
		}
	})

	t.Run("Plain directory walks files", func(t *testing.T) {
		t.Parallel()
		tmpDir, err := os.MkdirTemp("", "ipfind_dir_*")
		if err != nil {
			t.Fatalf("failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		f1 := filepath.Join(tmpDir, "a.txt")
		f2 := filepath.Join(tmpDir, "b.txt")
		os.WriteFile(f1, []byte("1.2.3.4"), 0644)
		os.WriteFile(f2, []byte("5.6.7.8"), 0644)

		files, err := getFileNamesFromArgs([]string{tmpDir})
		if err != nil {
			t.Fatalf("getFileNamesFromArgs failed: %v", err)
		}
		if len(files) != 2 {
			t.Errorf("expected 2 files, got %d: %v", len(files), files)
		}
	})

	t.Run("Nested directory walks recursively", func(t *testing.T) {
		t.Parallel()
		tmpDir, err := os.MkdirTemp("", "ipfind_nested_*")
		if err != nil {
			t.Fatalf("failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		subDir := filepath.Join(tmpDir, "sub")
		os.Mkdir(subDir, 0755)
		os.WriteFile(filepath.Join(tmpDir, "root.txt"), []byte("1.2.3.4"), 0644)
		os.WriteFile(filepath.Join(subDir, "nested.txt"), []byte("5.6.7.8"), 0644)

		files, err := getFileNamesFromArgs([]string{tmpDir})
		if err != nil {
			t.Fatalf("getFileNamesFromArgs failed: %v", err)
		}
		if len(files) != 2 {
			t.Errorf("expected 2 files, got %d: %v", len(files), files)
		}
	})
}

// --- GetReadCloser errors ---

func TestGetReadCloserErrors(t *testing.T) {
	t.Parallel()

	_, err := GetReadCloser(InputFile{Filename: "/nonexistent/path/that/does/not/exist"})
	if err == nil {
		t.Error("expected error for nonexistent file")
	}

	tmpDir, err := os.MkdirTemp("", "ipfind_read_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)
	_, err = GetReadCloser(InputFile{Filename: tmpDir})
	if err == nil {
		t.Error("expected error when opening directory")
	}
}

// --- Run with stdin ---

func TestRunWithStdin(t *testing.T) {
	// Do not run in parallel - we replace os.Stdin
	oldStdin := os.Stdin
	defer func() { os.Stdin = oldStdin }()

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("Pipe failed: %v", err)
	}
	os.Stdin = r

	w.WriteString("1.2.3.4\n5.6.7.8\n")
	w.Close()

	args := setupTestArgs("0.0.0.0/0", false, false, true)
	args.InputFiles = []string{}

	var sb strings.Builder
	err = Run(context.Background(), &sb, args)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	output := sb.String()
	if !strings.Contains(output, "1.2.3.4") || !strings.Contains(output, "5.6.7.8") {
		t.Errorf("expected stdin content in output, got %q", output)
	}
}

// --- Run skips failed files ---

func TestRunSkipsFailedFiles(t *testing.T) {
	t.Parallel()

	tmpDir, err := os.MkdirTemp("", "ipfind_skip_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	validFile := filepath.Join(tmpDir, "valid.txt")
	os.WriteFile(validFile, []byte("1.2.3.4\n"), 0644)

	brokenSymlink := filepath.Join(tmpDir, "broken")
	if err := os.Symlink("/nonexistent/target", brokenSymlink); err != nil {
		t.Skipf("skipping: cannot create symlink: %v", err)
	}

	args := setupTestArgs("0.0.0.0/0", false, false, true)
	args.InputFiles = []string{tmpDir}

	var sb strings.Builder
	err = Run(context.Background(), &sb, args)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if !strings.Contains(sb.String(), "1.2.3.4") {
		t.Error("Run should process valid file after skipping broken symlink")
	}
}

// --- Empty / no-match files ---

func TestEmptyAndNoMatchFiles(t *testing.T) {
	t.Parallel()

	tmpFile, err := os.CreateTemp("", "ipfind_empty_*.txt")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	args := setupTestArgs("1.2.3.4", true, false, false)
	args.InputFiles = []string{tmpFile.Name()}

	var sb strings.Builder
	err = Run(context.Background(), &sb, args)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if sb.Len() != 0 {
		t.Errorf("expected empty output for empty file, got %q", sb.String())
	}

	// No-match file
	tmpFile2, _ := os.CreateTemp("", "ipfind_nomatch_*.txt")
	tmpFile2.WriteString("5.6.7.8\n")
	tmpFile2.Close()
	defer os.Remove(tmpFile2.Name())

	args.InputFiles = []string{tmpFile2.Name()}
	sb.Reset()
	err = Run(context.Background(), &sb, args)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if sb.Len() != 0 {
		t.Errorf("expected empty output for no-match file, got %q", sb.String())
	}
}

// --- Mixed IPv4/IPv6 ---

func TestMixedAddressFamily(t *testing.T) {
	t.Parallel()

	input := "1.2.3.4\n2001:db8::1\n"
	args := setupTestArgs("1.2.3.4", true, false, false)

	var matched []string
	err := ProcessReader(context.Background(), strings.NewReader(input), args,
		func(lr LineResult) error {
			matched = append(matched, lr.Line)
			return nil
		})
	if err != nil {
		t.Fatalf("ProcessReader failed: %v", err)
	}
	if len(matched) != 1 || matched[0] != "1.2.3.4" {
		t.Errorf("IPv4 target should only match IPv4 line, got %v", matched)
	}

	args = setupTestArgs("2001:db8::1", true, false, false)
	matched = nil
	err = ProcessReader(context.Background(), strings.NewReader(input), args,
		func(lr LineResult) error {
			matched = append(matched, lr.Line)
			return nil
		})
	if err != nil {
		t.Fatalf("ProcessReader failed: %v", err)
	}
	if len(matched) != 1 || matched[0] != "2001:db8::1" {
		t.Errorf("IPv6 target should only match IPv6 line, got %v", matched)
	}
}

// --- MatchesCondition ---

func TestMatchesCondition(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		ipStr     string
		targetStr string
		exact     bool
		contains  bool
		subnet    bool
		want      bool
	}{
		{"exact match", "1.2.3.4", "1.2.3.4", true, false, false, true},
		{"exact no match", "1.2.3.5", "1.2.3.4", true, false, false, false},
		{"contains match", "10.0.0.0/8", "10.0.0.1", false, true, false, true},
		{"contains no match", "10.0.0.0/24", "10.0.1.1", false, true, false, false},
		{"subnet match", "192.168.1.5", "192.168.1.0/24", false, false, true, true},
		{"subnet no match", "192.167.1.5", "192.168.1.0/24", false, false, true, false},
		{"IPv6 exact", "2001:db8::1", "2001:db8::1", true, false, false, true},
		{"IPv6 subnet", "2001:db8::dead", "2001:db8::/32", false, false, true, true},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ipObj := ipaddr.NewIPAddressString(tt.ipStr).GetAddress()
			targetObj := ipaddr.NewIPAddressString(tt.targetStr).GetAddress()
			if ipObj == nil || targetObj == nil {
				t.Skip("invalid IP in test case")
			}
			args := Args{Exact: tt.exact, Contains: tt.contains, Subnet: tt.subnet}
			got := MatchesCondition(ipObj, targetObj, args)
			if got != tt.want {
				t.Errorf("MatchesCondition(%q, %q) = %v, want %v", tt.ipStr, tt.targetStr, got, tt.want)
			}
		})
	}

	t.Run("default case", func(t *testing.T) {
		t.Parallel()
		ipObj := ipaddr.NewIPAddressString("1.2.3.4").GetAddress()
		targetObj := ipaddr.NewIPAddressString("1.2.3.4").GetAddress()
		args := Args{} // No mode flags set
		got := MatchesCondition(ipObj, targetObj, args)
		if got != false {
			t.Error("MatchesCondition with no flags should return false")
		}
	})
}

// --- getLongestSubnetMask, getMinimumConditionMatchLines ---

func TestGetLongestSubnetMask(t *testing.T) {
	t.Parallel()

	fp := &FileResult{
		MatchedLines: []*LineResult{
			{ConditionMatches: []*ipaddr.IPAddress{
				ipaddr.NewIPAddressString("10.0.0.0/8").GetAddress(),
				ipaddr.NewIPAddressString("10.1.0.0/16").GetAddress(),
			}},
		},
	}
	got := fp.getLongestSubnetMask()
	if got != 16 {
		t.Errorf("getLongestSubnetMask = %d, want 16", got)
	}
}

func TestGetMinimumConditionMatchLines(t *testing.T) {
	t.Parallel()

	fp := &FileResult{
		MatchedLines: []*LineResult{
			{ConditionMatches: []*ipaddr.IPAddress{
				ipaddr.NewIPAddressString("10.0.0.0/8").GetAddress(),
			}},
			{ConditionMatches: []*ipaddr.IPAddress{
				ipaddr.NewIPAddressString("10.1.0.0/24").GetAddress(),
			}},
		},
	}
	lines := fp.getMinimumConditionMatchLines(16)
	if len(lines) != 1 {
		t.Errorf("getMinimumConditionMatchLines(16) = %d lines, want 1", len(lines))
	}
	if len(lines) > 0 && len(lines[0].ConditionMatches) > 0 {
		pl := lines[0].ConditionMatches[0].GetPrefixLen().Len()
		if pl != 24 {
			t.Errorf("expected /24 line, got prefix len %d", pl)
		}
	}
}

// --- Regex edge cases ---

func TestRegexEdgeCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		input  string
		target string
		mode   string
		want   bool
	}{
		{"IPv6 compressed", "::1", "::1", "exact", true},
		{"IPv6 expanded", "0:0:0:0:0:0:0:1", "::1", "exact", true},
		{"IPv4 standard", "10.10.10.10", "10.10.10.10", "exact", true},
		{"malformed IPv4 high octet", "1.2.3.256", "1.2.3.4", "exact", false},
		{"malformed IPv4 short", "1.2.3", "1.2.3.4", "exact", false},
		{"IPv6 standard", "2001:db8::1", "2001:db8::1", "exact", true},
		{"URL with IP", "http://192.168.1.1/path", "192.168.1.1", "exact", true},
		{"log format", "[2024-01-01] 10.0.0.1 connected", "10.0.0.1", "exact", true},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			args := setupTestArgs(tt.target, tt.mode == "exact", tt.mode == "contains", tt.mode == "subnet")
			var matched bool
			err := ProcessReader(context.Background(), strings.NewReader(tt.input), args,
				func(lr LineResult) error {
					matched = true
					return nil
				})
			if err != nil {
				t.Fatalf("ProcessReader failed: %v", err)
			}
			if matched != tt.want {
				t.Errorf("input %q target %q: got match=%v, want %v", tt.input, tt.target, matched, tt.want)
			}
		})
	}
}

func TestRunSequentialModes(t *testing.T) {
	t.Parallel()

	t.Run("JSON output multi-line", func(t *testing.T) {
		tmpFile, _ := os.CreateTemp("", "seq_json_*.txt")
		defer os.Remove(tmpFile.Name())
		tmpFile.WriteString("1.2.3.4\n1.2.3.5\n")
		tmpFile.Close()

		args := setupTestArgs("1.2.3.0/24", false, false, true)
		args.Json = true
		args.Workers = 1
		args.InputFiles = []string{tmpFile.Name()}

		var sb strings.Builder
		err := Run(context.Background(), &sb, args)
		if err != nil {
			t.Fatalf("Run failed: %v", err)
		}
		output := sb.String()
		var results []LineResult
		if err := json.Unmarshal([]byte(output), &results); err != nil {
			t.Fatalf("failed to unmarshal JSON: %v. Output:\n%s", err, output)
		}
		if len(results) != 2 {
			t.Errorf("Expected 2 JSON objects, got %d", len(results))
		}
		if results[0].Line != "1.2.3.4" || results[1].Line != "1.2.3.5" {
			t.Errorf("JSON output missing expected matches. Got:\n%v", results)
		}
	})

	t.Run("Skips invalid files", func(t *testing.T) {
		tmpDir, _ := os.MkdirTemp("", "seq_skip_*.txt")
		defer os.RemoveAll(tmpDir)

		args := setupTestArgs("1.2.3.4", true, false, false)
		args.Workers = 1
		// Passing a directory as a file argument.
		// getFileNamesFromArgs will return the directory path if it's explicitly passed (not walked).
		// Wait, if I pass it explicitly, os.Stat(ifile) is IsDir, and it walks it.
		// If the dir is empty, ret will be empty.

		// Let's use a file we can't open (like a directory) by passing it as a single file.
		// Actually, GetInputFilesOrStdin calls getFileNamesFromArgs.

		validFile, _ := os.CreateTemp("", "valid_*.txt")
		validFile.WriteString("1.2.3.4\n")
		validFile.Close()
		defer os.Remove(validFile.Name())

		// We'll use the existing TestRunSkipsFailedFiles logic but force sequential
		args.InputFiles = []string{validFile.Name()}
		// To trigger the error branch in runSequential, we need GetReadCloser to fail.
		// We can achieve this by passing a directory path directly in InputFiles list
		// but since Run calls GetInputFilesOrStdin, it might be tricky.

		// Let's just call runSequential directly if we want to test its internals.
		inputFiles := []InputFile{
			{Filename: "/nonexistent/file"}, // This will fail GetReadCloser
			{Filename: validFile.Name()},    // This will succeed
		}

		var sb strings.Builder
		err := runSequential(context.Background(), &sb, inputFiles, args)
		if err != nil {
			t.Fatalf("runSequential should not fail when skipping files: %v", err)
		}
		if !strings.Contains(sb.String(), "1.2.3.4") {
			t.Error("runSequential failed to process valid file after error")
		}
	})

	t.Run("Trie mode sequential", func(t *testing.T) {
		tmpFile, _ := os.CreateTemp("", "seq_trie_*.txt")
		defer os.Remove(tmpFile.Name())
		tmpFile.WriteString("10.0.0.1\n")
		tmpFile.Close()

		args := setupTestArgs("10.0.0.0/8", false, false, true)
		args.Trie = true
		args.Workers = 1
		args.InputFiles = []string{tmpFile.Name()}

		var sb strings.Builder
		err := Run(context.Background(), &sb, args)
		if err != nil {
			t.Fatalf("Run failed: %v", err)
		}
		if !strings.Contains(sb.String(), "10.0.0.1") {
			t.Errorf("Trie sequential output missing match: %s", sb.String())
		}
	})
}

func TestRunParallelErrorHandling(t *testing.T) {
	t.Parallel()

	validFile, _ := os.CreateTemp("", "parallel_valid_*.txt")
	validFile.WriteString("1.2.3.4\n")
	validFile.Close()
	defer os.Remove(validFile.Name())

	args := setupTestArgs("1.2.3.4", true, false, false)
	args.Workers = 2

	inputFiles := []InputFile{
		{Filename: "/nonexistent/file/parallel"}, // Fail GetReadCloser
		{Filename: validFile.Name()},             // Success
	}

	var sb strings.Builder
	err := runParallel(context.Background(), &sb, inputFiles, args)
	if err != nil {
		t.Fatalf("runParallel should not fail when skipping files: %v", err)
	}
	if !strings.Contains(sb.String(), "1.2.3.4") {
		t.Error("runParallel failed to process valid file after error")
	}
}

func TestProcessReaderScannerError(t *testing.T) {
	t.Parallel()

	// A reader that returns an error to trigger scanner.Err()
	// Large line to trigger bufio.Scanner error
	largeLine := strings.Repeat("a", 128*1024)
	r := strings.NewReader(largeLine)

	args := setupTestArgs("1.2.3.4", true, false, false)
	err := ProcessReader(context.Background(), r, args, func(lr LineResult) error { return nil })
	if err == nil || !strings.Contains(err.Error(), "token too long") {
		t.Errorf("expected token too long error, got %v", err)
	}
}

func TestGetFileNamesFromArgsError(t *testing.T) {
	t.Parallel()

	// Test non-existent path
	_, err := getFileNamesFromArgs([]string{"/nonexistent/path"})
	if err == nil {
		t.Error("expected error for non-existent path")
	}
}

func TestRunModeSelection(t *testing.T) {
	t.Parallel()

	// Case 1: Workers == 1, Sort == false -> Sequential
	// Case 2: Workers != 1 or Sort == true -> Parallel

	tmpFile, _ := os.CreateTemp("", "mode_sel_*.txt")
	defer os.Remove(tmpFile.Name())
	tmpFile.WriteString("1.2.3.4\n")
	tmpFile.Close()

	t.Run("Force Parallel via Sort", func(t *testing.T) {
		args := setupTestArgs("1.2.3.4", true, false, false)
		args.Workers = 1
		args.Sort = true
		args.InputFiles = []string{tmpFile.Name()}

		var sb strings.Builder
		err := Run(context.Background(), &sb, args)
		if err != nil {
			t.Fatalf("Run failed: %v", err)
		}
		if !strings.Contains(sb.String(), "1.2.3.4") {
			t.Error("Parallel (sorted) output missing match")
		}
	})

	t.Run("Force Parallel via Workers > 1", func(t *testing.T) {
		args := setupTestArgs("1.2.3.4", true, false, false)
		args.Workers = 2
		args.InputFiles = []string{tmpFile.Name()}

		var sb strings.Builder
		err := Run(context.Background(), &sb, args)
		if err != nil {
			t.Fatalf("Run failed: %v", err)
		}
		if !strings.Contains(sb.String(), "1.2.3.4") {
			t.Error("Parallel output missing match")
		}
	})
}
