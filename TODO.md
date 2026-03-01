TODO handle err and nil, don't be lazy. take AI idiomatic suggestions
TODO tab completion, however that works
TODO turn it into an importable library
TODO readme
TODO goreleaser
TODO with multiple IPs matching one line I see that line multiple times

data/docs.txt:12:it contains other subnets such as 2001:db8::42/120 2001:db8:abcd:efff:1234/80 2001:db8::7/127
data/docs.txt:12:it contains other subnets such as 2001:db8::42/120 2001:db8:abcd:efff:1234/80 2001:db8::7/127

maybe call it 'grip'?  'go recognize IPs'?
or just keep 'ipfind', it's fine.

idea: given a network range and some input, match the network range against the input

ipfind <ip> [input files] {-e|-s|-c|-l|} 
 plus -json, -show-line-numbers, whatever rg has I can steal
 plus ip-specific like canonize, host vs /32
 plus debug and/or verbose.  maybe just different debug levels?

-e / --exact
-s / --subnet: show all subnets in input which contain network range
-c / --contain: show all networks in input which are contained by network range
-l / --longest: like subnet but only show longest match



output options:
json?
show line numbers and file numbers and individual matches
v4, v6, be smart about which one

trie?  how does that figure in here? -c, -s, -l?  



think about parallel vs. serial, speed vs. memory utilization

define behaviors during each case.
what behaviors?
 - canonizing 1.2.3.4/24 to 1.2.3.0/24
 - 1.2.3.4 vs. 1.2.3.4/32

lots of test cases.


can I use iterators without generics?
range over channels? do it without iterators?

make the ip stuff separable so i can rip out that library if i want
so have my own functions that are thin wrappers for his stuff



architecture

main just starts stuff
options are parsed in optparse() or something, not in main. separate file, same package.
 figures out what the input files are 
 massages options
main calls cmd
cmd calls loop
 calls per-input-loop, a loop over each input source
  lineparse runs a loop over each link in each input source
   something grabs multiple matches per line
  lineparse returns line matches to per-file loop
  per-file loop gathers and returns results to cmd.  how?  channels I guess?

cmd accepts output from loops and assembles them.  this is the part I don't understand.
cmd prints them

cmd calls loop(args, file, return-channel)
 loop opens file, processes each line, adds to a per-file return struct
 puts return struct instance on return-channel
cmd reads from channel until done. lets me make it concurrent later.
 just run this as concurrent-safe but with 1 simultaneous worker.

cmd assembles either once it's all done or async, not sure.





// TODO pick between snake and camel case and whatever

figure out when I want to print trie and what flag triggers is.

what about printing trie in both -c and -l cases? and -s?
hrmm.
