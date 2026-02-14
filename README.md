NAME:
   ipfind - search lines for IPv4 addresses and CIDRs

USAGE:
    [global options] command [command options] [arguments...]

COMMANDS:
   help, h  Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --exact, -e          print lines with exact token matches to query
   --subnet, -s         print lines with CIDR blocks that contain the query
   --longest-match, -l  print lines containing the most-specific CIDR(s) that match the query
   --mask-range value   mask range MIN-MAX to filter candidate CIDRs
   --version, -v        print version and exit
   --help, -h           show help
