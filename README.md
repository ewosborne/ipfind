ipfind (Go) - find IPv4 addresses and CIDRs in text

Build/run:

```
go run ipfind.go [flags] <query> [file]
```

Examples:

- Exact token match (token equals query after normalization):

```
go run ipfind.go -e 1.2.3.4 sample.txt
go run ipfind.go -e 1.2.3.0/24 sample.txt
```

- Subnet match (print lines containing CIDRs that contain the query):

```
go run ipfind.go -s 1.2.3.4 sample.txt
go run ipfind.go -s 1.2.3.0/28 sample.txt
```

- Longest-match (print lines containing the most-specific CIDR(s) that match):

```
go run ipfind.go -l 1.2.3.4 sample.txt
```

- Mask range filter (only consider CIDRs with prefixlen in range):

```
go run ipfind.go -s --mask-range 20-28 1.2.3.4 sample.txt
```

If `file` is omitted or `-` is used, the tool reads from stdin.

Find IP addresses in a file

