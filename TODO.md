TODO handle err and nil, don't be lazy
TODO test cases
TODO more work on trie branch
TODO tab completion, however that works


ipf <optional ip> -f/--file <ifile> 

-t/--trie: given an IP, show containg trie. given no ip, generate trie for every IP found in the file.
    ^^ q: if there's no IP, do I do v4 and v6?
-e/--exact: given an IP, show all exact matches
-s/--subnet: given an IP, show all containing subnets
-l/--longest: given an IP, show all LPM


t/e/s/l are mutually exclusive to each other.
t assumes s because that's the only way that makes sense?
 or is there a use for t and l?

-d/--debug: dump debug statements with log.Printf, etc.
-v/--verbose: not sure what this does yet.
-4, -6: force ipv4, ipv6 AFs. useful for --trie I guess.



output:
show matching line: --show line
show just matching network: --show network
show line number: --show number 
some sort of json dump: --json?  not sure what this would look like.


basic flow:

determine input stream 
read from it
find all lines with IPs
find all matching IPs
continue doing stuff



