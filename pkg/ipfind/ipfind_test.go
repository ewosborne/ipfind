package ipfind

import (
	"context"
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
				if i < len(matchedLines) && matchedLines[i].Idx != lineNum {
					t.Errorf("expected match on line %d, got %d", lineNum, matchedLines[i].Idx)
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
				Idx:  1,
				Line: "1.2.3.4",
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
		if !strings.Contains(sb.String(), `"Line": "1.2.3.4"`) {
			t.Errorf("expected JSON to contain line content, got %q", sb.String())
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

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			// Create a test file per subtest to avoid cleanup race
			f, _ := os.CreateTemp("", "mode_test_*.txt")
			defer os.Remove(f.Name())
			f.WriteString("1.2.3.4\n5.6.7.8\n")
			f.Close()

			args := Args{
				Ipstring:   "0.0.0.0/0",
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
				if !strings.HasPrefix(output, "[") || !strings.HasSuffix(strings.TrimSpace(output), "]") {
					t.Error("JSON output missing array brackets")
				}
			} else {
				if !strings.Contains(output, "1.2.3.4") || !strings.Contains(output, "5.6.7.8") {
					t.Error("Output missing expected IP matches")
				}
			}
		})
	}
}

func TestGetFileNamesFromArgsSymlinkDir(t *testing.T) {
	t.Parallel()
	tmpDir, err := os.MkdirTemp("", "ipfind_sym_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	testFile := filepath.Join(tmpDir, "test.txt")
	os.WriteFile(testFile, []byte("1.2.3.4\n"), 0644)

	// Create a symlink to the directory
	symlink := filepath.Join(os.TempDir(), fmt.Sprintf("ipfind_symlink_%d", os.Getpid()))
	if err := os.Symlink(tmpDir, symlink); err != nil {
		t.Skipf("skipping symlink test: %v", err)
	}
	defer os.Remove(symlink)

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

func TestRunJSONIndentation(t *testing.T) {
	t.Parallel()

	// Create a temporary file to satisfy Run's file-based logic

	// but we'll use a mocked InputFile for now since Run calls GetInputFilesOrStdin.
	// Actually, easier to just test the logic inside Run directly by passing it a buffer.

	args := setupTestArgs("1.2.3.4", false, false, true) // Subnet mode
	args.Json = true
	// We need to override the InputFiles to skip the directory walking for this test
	// and manually trigger the logic. However, Run is hardcoded to use InputFiles from args.
	// Let's test by creating a small file.

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
    "Idx": 1,
    "Line": "1.2.3.4"
  },
  {
    "Idx": 2,
    "Line": "1.2.3.4"
  }
]
`
	if sb.String() != expected {
		t.Errorf("JSON output indentation mismatch.\nExpected:\n%q\nGot:\n%q", expected, sb.String())
	}
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
		if strings.Contains(output, "10.0.0.0/8") {
			t.Errorf("JSON Output should not contain short prefix /8. Got:\n%s", output)
		}
		// Count occurrences of Indent to ensure correct number of items (should be 2)
		if strings.Count(output, "\"Idx\"") != 2 {
			t.Errorf("Expected 2 JSON objects, got %d. Output:\n%s", strings.Count(output, "\"Idx\""), output)
		}
	})
}
