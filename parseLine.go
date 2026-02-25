package main

import (
	"log/slog"
	"regexp"
	"slices"
	"strings"

	"github.com/seancfoley/ipaddress-go/ipaddr"
)

// this file has routes which parse an individual line and return stuff

var (
	ipv4Regex       = regexp.MustCompile(`(\d{1,3}).(\d{1,3}).(\d{1,3}).(\d{1,3}(/\d{1,2})?)`)
	ipv6Regex       = regexp.MustCompile(`([:0-9a-fA-F]{2,39}(/[0-9]{1,3})?)`)
	v4_line_matches = []*ipaddr.IPAddress{}
	v6_line_matches = []*ipaddr.IPAddress{}
)

func scanLine(args cliArgStruct, ret dataMatch) (dataMatch, bool) {

	if args.V4 {
		v4_line_matches = get_ipv4_addresses_from_line(ret.MatchLine)
	}

	if args.V6 {
		v6_line_matches = get_ipv6_addresses_from_line(ret.MatchLine)
	}

	// 	// note well that this is _regex matches_, not _criteria matches_.
	line_matches := slices.Concat(v4_line_matches, v6_line_matches)
	if len(line_matches) == 0 {
		return ret, false
	} else {
		ret.MatchIPs = line_matches
	}

	return ret, true
}

func get_ip_addresses_from_line(ipre *regexp.Regexp, line string) []*ipaddr.IPAddress {
	ret := []*ipaddr.IPAddress{}
	ipStrings := ipre.FindAllString(line, -1)
	if ipStrings == nil { // no matches
		return nil
	}
	slog.Debug("FindAllString", "v4", ipStrings)

	for _, ipString := range ipStrings {
		slog.Debug("before", "addrString", ipString)
		converted := ipaddr.NewIPAddressString(ipString).GetAddress()
		slog.Debug("after", "converted", converted.String())
		if converted != nil { // no successful conversions, matches must have been bogus
			ret = append(ret, converted)
		}

	}

	return ret
}

func get_ipv4_addresses_from_line(line string) []*ipaddr.IPAddress {
	return get_ip_addresses_from_line(ipv4Regex, line)
}

func get_ipv6_addresses_from_line(line string) []*ipaddr.IPAddress {

	// hack because the regex is getting messy but this seems ok.
	ret := []*ipaddr.IPAddress{}
	for _, m := range get_ip_addresses_from_line(ipv6Regex, line) {
		if strings.Contains(m.String(), ":") {
			ret = append(ret, m)
		}
	}
	return ret

}
