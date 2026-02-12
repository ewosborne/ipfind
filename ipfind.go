package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"

	"net/netip"

	cli "github.com/urfave/cli"
)

var ipTokenRe = regexp.MustCompile(`\b(?:\d{1,3}\.){3}\d{1,3}(?:/\d{1,2})?\b`)

type queryType int

const (
	qAddress queryType = iota
	qNetwork
)

func parseMaskRange(s string) (int, int, error) {
	if s == "" {
		return -1, -1, nil
	}
	if strings.Contains(s, "-") {
		parts := strings.SplitN(s, "-", 2)
		lo, err := strconv.Atoi(parts[0])
		if err != nil {
			return 0, 0, err
		}
		hi, err := strconv.Atoi(parts[1])
		if err != nil {
			return 0, 0, err
		}
		if lo < 0 || hi < lo || hi > 32 {
			return 0, 0, errors.New("invalid mask range")
		}
		return lo, hi, nil
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return 0, 0, err
	}
	if v < 0 || v > 32 {
		return 0, 0, errors.New("invalid mask range")
	}
	return v, v, nil
}

func findTokens(line string) []string {
	return ipTokenRe.FindAllString(line, -1)
}

func parseQuery(arg string) (queryType, netip.Addr, netip.Prefix, error) {
	if strings.Contains(arg, "/") {
		p, err := netip.ParsePrefix(arg)
		if err != nil {
			return 0, netip.Addr{}, netip.Prefix{}, err
		}
		return qNetwork, netip.Addr{}, p, nil
	}
	a, err := netip.ParseAddr(arg)
	if err != nil {
		return 0, netip.Addr{}, netip.Prefix{}, err
	}
	return qAddress, a, netip.Prefix{}, nil
}

func parseTokenAsPrefix(tok string) (netip.Prefix, bool) {
	if !strings.Contains(tok, "/") {
		return netip.Prefix{}, false
	}
	p, err := netip.ParsePrefix(tok)
	if err != nil {
		return netip.Prefix{}, false
	}
	return p, true
}

func normalizeToken(tok string) (string, bool) {
	if strings.Contains(tok, "/") {
		p, ok := parseTokenAsPrefix(tok)
		if !ok {
			return "", false
		}
		return p.String(), true
	}
	a, err := netip.ParseAddr(tok)
	if err != nil {
		return "", false
	}
	return a.String(), true
}

func addrToUint32(a netip.Addr) uint32 {
	if !a.Is4() {
		return 0
	}
	b := a.As4()
	return uint32(b[0])<<24 | uint32(b[1])<<16 | uint32(b[2])<<8 | uint32(b[3])
}

func prefixRange(p netip.Prefix) (uint32, uint32) {
	base := addrToUint32(p.Addr())
	hostBits := 32 - p.Bits()
	var broadcast uint32
	if hostBits == 0 {
		broadcast = base
	} else {
		broadcast = base | ((uint32(1) << hostBits) - 1)
	}
	return base, broadcast
}

func prefixContainsPrefix(outer, inner netip.Prefix) bool {
	ob, oe := prefixRange(outer)
	ib, ie := prefixRange(inner)
	return ob <= ib && oe >= ie
}

func readAllLines(r io.Reader) ([]string, error) {
	sc := bufio.NewScanner(r)
	var lines []string
	for sc.Scan() {
		lines = append(lines, sc.Text())
	}
	return lines, sc.Err()
}

func main() {
	app := &cli.App{
		Name:  "ipfind",
		Usage: "search lines for IPv4 addresses and CIDRs",
		Flags: []cli.Flag{
			cli.BoolFlag{Name: "exact, e", Usage: "print lines with exact token matches to query"},
			cli.BoolFlag{Name: "subnet, s", Usage: "print lines with CIDR blocks that contain the query"},
			cli.BoolFlag{Name: "longest-match, l", Usage: "print lines containing the most-specific CIDR(s) that match the query"},
			cli.StringFlag{Name: "mask-range", Usage: "mask range MIN-MAX to filter candidate CIDRs"},
			cli.BoolFlag{Name: "version, v", Usage: "print version and exit"},
		},
		Action: func(c *cli.Context) error {
			// print version and exit if requested
			if c.Bool("version") {
				fmt.Println("0.1.0")
				return nil
			}
			exact := c.Bool("exact")
			subnet := c.Bool("subnet")
			longest := c.Bool("longest-match")
			maskRange := c.String("mask-range")

			// allow flags to appear after positional args by scanning c.Args()
			for i := 0; i < c.NArg(); i++ {
				a := c.Args().Get(i)
				switch a {
				case "-e", "--exact":
					exact = true
				case "-s", "--subnet":
					subnet = true
				case "-l", "--longest-match":
					longest = true
				case "--mask-range":
					if i+1 < c.NArg() {
						maskRange = c.Args().Get(i + 1)
						i++
					}
				default:
					if strings.HasPrefix(a, "--mask-range=") {
						maskRange = strings.SplitN(a, "=", 2)[1]
					}
				}
			}

			modeCount := 0
			if exact {
				modeCount++
			}
			if subnet {
				modeCount++
			}
			if longest {
				modeCount++
			}
			// If no mode flag provided, default to exact
			if modeCount == 0 {
				exact = true
				modeCount = 1
			}
			if modeCount > 1 {
				return cli.NewExitError("flags -e/--exact, -s/--subnet, and -l/--longest-match are mutually exclusive", 2)
			}

			if c.NArg() < 1 {
				return cli.NewExitError("query (IP or IP/mask) is required", 2)
			}
			queryArg := c.Args().Get(0)
			if strings.HasPrefix(queryArg, "-") {
				return cli.NewExitError("mode flags must be followed by the query IP; use: ipfind -s <IP> [file] [--mask-range <MIN-MAX>]", 2)
			}
			fileArg := ""
			if c.NArg() >= 2 {
				fileArg = c.Args().Get(1)
				if strings.HasPrefix(fileArg, "-") {
					fileArg = ""
				}
			}

			qtype, qaddr, qpref, err := parseQuery(queryArg)
			if err != nil {
				return cli.NewExitError(fmt.Sprintf("invalid query: %v", err), 2)
			}

			loMask, hiMask, err := parseMaskRange(maskRange)
			if err != nil {
				return cli.NewExitError(fmt.Sprintf("invalid mask-range: %v", err), 2)
			}

			var fh *os.File
			if fileArg == "" || fileArg == "-" {
				fh = os.Stdin
			} else {
				fh, err = os.Open(fileArg)
				if err != nil {
					return cli.NewExitError(fmt.Sprintf("error opening file: %v", err), 2)
				}
				defer fh.Close()
			}

			lines, err := readAllLines(fh)
			if err != nil {
				return cli.NewExitError(fmt.Sprintf("read error: %v", err), 1)
			}

			var outLines []string

			if exact {
				var qnorm string
				if qtype == qAddress {
					qnorm = qaddr.String()
				} else {
					qnorm = qpref.String()
				}
				for _, ln := range lines {
					toks := findTokens(ln)
					matched := false
					for _, t := range toks {
						n, ok := normalizeToken(t)
						if !ok {
							continue
						}
						if n == qnorm {
							matched = true
							break
						}
					}
					if matched {
						outLines = append(outLines, ln)
					}
				}
			} else if subnet {
				for _, ln := range lines {
					toks := findTokens(ln)
					matched := false
					for _, t := range toks {
						p, ok := parseTokenAsPrefix(t)
						if !ok {
							continue
						}
						if loMask >= 0 {
							if p.Bits() < loMask || p.Bits() > hiMask {
								continue
							}
						}
						if qtype == qAddress {
							if p.Contains(qaddr) {
								matched = true
								break
							}
						} else {
							if prefixContainsPrefix(p, qpref) {
								matched = true
								break
							}
						}
					}
					if matched {
						outLines = append(outLines, ln)
					}
				}
			} else if longest {
				type cand struct {
					bits int
					idx  int
					line string
				}
				var cands []cand
				for i, ln := range lines {
					toks := findTokens(ln)
					for _, t := range toks {
						p, ok := parseTokenAsPrefix(t)
						if !ok {
							continue
						}
						if loMask >= 0 {
							if p.Bits() < loMask || p.Bits() > hiMask {
								continue
							}
						}
						okContains := false
						if qtype == qAddress {
							if p.Contains(qaddr) {
								okContains = true
							}
						} else {
							if prefixContainsPrefix(p, qpref) {
								okContains = true
							}
						}
						if okContains {
							cands = append(cands, cand{p.Bits(), i, ln})
						}
					}
				}
				if len(cands) > 0 {
					maxBits := 0
					for _, c := range cands {
						if c.bits > maxBits {
							maxBits = c.bits
						}
					}
					seen := map[string]bool{}
					for _, c := range cands {
						if c.bits == maxBits && !seen[c.line] {
							outLines = append(outLines, c.line)
							seen[c.line] = true
						}
					}
				}
			}

			seen := map[string]bool{}
			for _, l := range outLines {
				if !seen[l] {
					fmt.Println(l)
					seen[l] = true
				}
			}

			return nil
		},
	}

	if err := app.Run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
