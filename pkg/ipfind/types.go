package ipfind

import (
	"regexp"

	"github.com/seancfoley/ipaddress-go/ipaddr"
)

type AddressFamily int

const (
	IPv4 AddressFamily = 4
	IPv6 AddressFamily = 6
)

type InputFile struct {
	IsStdin  bool
	Filename string
}

type LineResult struct {
	Idx              int
	Line             string
	IPRegexMatches   []string            `json:"-"` // Don't include in JSON output
	ConditionMatches []*ipaddr.IPAddress `json:"-"` // Don't include in JSON output
	IsMatch          bool                `json:"-"` // Don't include in JSON output
}

type FileResult struct {
	InputFile    // Struct embedding
	Idx          int
	IPv4Trie     ipaddr.IPv4AddressTrie
	IPv6Trie     ipaddr.IPv6AddressTrie
	MatchedLines []*LineResult
}

type Args struct {
	Ipstring                      string
	Exact, Longest, Subnet, Trie  bool
	Contains, Canonize            bool
	AddressFamily                 AddressFamily
	Slash, Json                   bool
	InputFiles                    []string
	Debug                         bool
	Ipaddr                        *ipaddr.IPAddress
	IPv4Regex, IPv6Regex, IPRegex *regexp.Regexp
	Workers                       int
	Sort                          bool
	Pretext                       string
}

func NewFileResult(f InputFile) *FileResult {
	return &FileResult{
		InputFile: f,
		IPv4Trie:  ipaddr.IPv4AddressTrie{},
		IPv6Trie:  ipaddr.IPv6AddressTrie{},
	}
}
