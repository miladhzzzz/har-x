# Capy

This is a Go program that captures DNS queries and responses, TCP/HTTP/TLS handshakes, and HTTP traffic using the gopacket and godet libraries. It takes in a URL and DNS server as input flags, and saves the captured data to text files, a pcap file, and a.har file.

## Installation
To install the program, clone the repository and run the following command:
```shell
go build
```
## Usage
To capture DNS and HTTP traffic, run the program with the following command:
```shell
Copy ./capture -url=<URL> -dns=<DNS server> -output=<output directory>
```
Replace 
<URL>
 with the website you want to capture, 
<DNS server>
 with the DNS server you want to use, and 
<output directory>
 with the directory where you want to save the captured data.

## Output
The program saves the captured data to the following files:

dns.txt
: Contains the DNS query and response.
capture.pcap
: Contains the TCP/HTTP/TLS handshakes and HTTP traffic.
capture.har
: Contains the HTTP traffic in the HAR format.
Dependencies
The program uses the following libraries:

github.com/google/gopacket/pcap
: Provides packet capture functionality.
github.com/miekg/dns
: Provides DNS query and response functionality.
github.com/raff/godet
: Provides Chrome DevTools Protocol functionality.

## License
This program is licensed under the MIT License. See the 

LICENSE
 file for more information.


	"github.com/google/gopacket/pcap"
	"github.com/miekg/dns"