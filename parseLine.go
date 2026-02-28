package main

import (
	"regexp"
	"slices"
	"strings"

	"github.com/charmbracelet/log"
	"github.com/seancfoley/ipaddress-go/ipaddr"
)

// this file has routes which parse an individual line and return stuff

// first two regexes require /mask, last two it's optional
var (
	ipv4Regex_withSlash = regexp.MustCompile(`(\d{1,3}).(\d{1,3}).(\d{1,3}).(\d{1,3}(/\d{1,2}))`)
	ipv6Regex_withSlash = regexp.MustCompile(`([:0-9a-fA-F]{2,39}(/[0-9]{1,3}))`)
	ipv4Regex_noSlash   = regexp.MustCompile(`(\d{1,3}).(\d{1,3}).(\d{1,3}).(\d{1,3}(/\d{1,2})?)`)
	ipv6Regex_noSlash   = regexp.MustCompile(`([:0-9a-fA-F]{2,39}(/[0-9]{1,3})?)`)
)

func scanLine(args cliArgStruct, ret dataMatch) (dataMatch, bool) {
	var v4_line_matches = []*ipaddr.IPAddress{}
	var v6_line_matches = []*ipaddr.IPAddress{}

	if args.V4 {
		v4_line_matches = get_ipv4_addresses_from_line(ret.MatchLine, args.IPv4Regex)
	}

	if args.V6 {
		v6_line_matches = get_ipv6_addresses_from_line(ret.MatchLine, args.IPv6Regex)
	}

	// note well that this is _regex matches_, not _criteria matches_.
	line_matches := slices.Concat(v4_line_matches, v6_line_matches)
	if len(line_matches) == 0 {
		return ret, false
	} else {

		// TODO wtf why isn't this set?
		ret.MatchIPs = line_matches
		//ret.MatchIPs = slices.Clone(line_matches)
		//copy(ret.MatchIPs, line_matches)
	}

	return ret, true
}

func get_ip_addresses_from_line(ipre *regexp.Regexp, line string) []*ipaddr.IPAddress {
	ret := []*ipaddr.IPAddress{}
	ipStrings := ipre.FindAllString(line, -1)
	if ipStrings == nil { // no matches
		return nil
	}
	log.Debug("FindAllString", "v4", ipStrings)

	for _, ipString := range ipStrings {
		log.Debug("before", "addrString", ipString)
		converted := ipaddr.NewIPAddressString(ipString).GetAddress()
		log.Debug("after", "converted", converted.String())
		if converted != nil { // no successful conversions, matches must have been bogus
			ret = append(ret, converted)
		}

	}

	return ret
}

// TODO can I do this part in parallel?  ipv6 in particular is expensive.
// syncMap maybe? ick.
//
//	channels?
func get_ipv4_addresses_from_line(line string, ipv4Regex *regexp.Regexp) []*ipaddr.IPAddress {
	return get_ip_addresses_from_line(ipv4Regex, line)
}

func get_ipv6_addresses_from_line(line string, ipv6Regex *regexp.Regexp) []*ipaddr.IPAddress {

	// hack because the regex is getting messy but this seems ok.
	ret := []*ipaddr.IPAddress{}
	for _, m := range get_ip_addresses_from_line(ipv6Regex, line) {
		if strings.Contains(m.String(), ":") {
			ret = append(ret, m)
		}
	}
	return ret

}
