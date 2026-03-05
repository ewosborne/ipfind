package main

import (
	"strings"
	"testing"

	"github.com/seancfoley/ipaddress-go/ipaddr"
)

// helper to simplify creating test arguments
func setupTestArgs(ip string, exact, contains, subnet bool) cliArgStruct {
	args := cliArgStruct{
		Ipstring: ip,
		Exact:    exact,
		Contains: contains,
		Subnet:   subnet,
		Canonize: true, // default behavior
	}
	return argMassage(args)
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
			result, err := processReader(reader, args)

			if err != nil {
				t.Fatalf("processReader failed: %v", err)
			}

			// Check matched line count
			if len(result.conditionMatchedLines) != len(tt.expectedLines) {
				t.Errorf("expected %d matched lines, got %d", len(tt.expectedLines), len(result.conditionMatchedLines))
			}

			// Verify specific line numbers
			for i, lineNum := range tt.expectedLines {
				if i < len(result.conditionMatchedLines) && result.conditionMatchedLines[i].Idx != lineNum {
					t.Errorf("expected match on line %d, got %d", lineNum, result.conditionMatchedLines[i].Idx)
				}
			}

			// Check total IP objects found across all lines
			actualIPCount := 0
			for _, line := range result.conditionMatchedLines {
				actualIPCount += len(line.conditionMatches)
			}
			if actualIPCount != tt.expectedMatchCount {
				t.Errorf("expected %d total IP matches, got %d", tt.expectedMatchCount, actualIPCount)
			}
		})
	}
}

func TestFormatResults(t *testing.T) {
	t.Parallel()
	fp := fileParseResultStruct{
		Filename: "test.txt",
		conditionMatchedLines: []*lineParseResult{
			{
				Idx:  1,
				Line: "1.2.3.4",
				conditionMatches: []*ipaddr.IPAddress{
					ipaddr.NewIPAddressString("1.2.3.4").GetAddress(),
				},
			},
		},
	}

	t.Run("Text Output", func(t *testing.T) {
		t.Parallel()
		var sb strings.Builder
		args := setupTestArgs("1.2.3.4", true, false, false)
		err := formatResults(&sb, fp, args)
		if err != nil {
			t.Fatalf("formatResults failed: %v", err)
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
		err := formatResults(&sb, fp, args)
		if err != nil {
			t.Fatalf("formatResults failed: %v", err)
		}
		if !strings.Contains(sb.String(), `"Line": "1.2.3.4"`) {
			t.Errorf("expected JSON to contain line content, got %q", sb.String())
		}
	})
}
