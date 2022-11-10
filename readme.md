You start each peer by typing `go run main.go xxxx`, with xxxx being a port listed in the port addresses text file. When a peer is open you can either type `enter` or `exit` in the command line. Enter tries to enter the critical section, while exit exits the critical section.

The file "port-addresses.txt" has 3 different ports listed (5000, 5001 and 5002) and you therefore have to run three different peers with these 3 ports.
