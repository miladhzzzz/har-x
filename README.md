# HAR-X
### HAR eXploration and Rapid Intelligence

This is a Python script that parses and analyzes HAR (HTTP Archive) files. HAR files usually contain  ~30K lines of code and we parse that and create analyses visualization, json summary, a complete list of all requests in csv sub ~5s !!


## What is Har?

**"har files are pcaps of the http realm!"**

HAR (HTTP Archive) is a JSON-formatted file that captures the network traffic of a web page. It contains information about each request made by the page, including the URL, method, response status, response time, and other details. 

## How is har files created?

When you type in a URL into your browser theres 3 stages:
1. your computer tries to resolve the hostname you provided meaning it will ask a DNS server for IP addresses.(DNS)
2. a tcp connection will be established to the IP address we got from DNS , handshakes , tls , http are negotiated. (TCP)
3. your browser makes a HTTP get request to the url using TCP connection we established earlier.(HTTP)

Finally you see your web page! yay!

#### Browser
so when we are dealing with web applications if we want to know whats exactly going on in all the layers mentioned above we need to query dns servers, use wireshark to capture packets and use Developer Tools in our browser to capture everything and then analyze it!


#### Capy Cli Tool!
well! **Capy** will do all that for you its a Go program that does the above stages and saves the files in your output directory.
so you can install capy with these methods:

#### install
install capy using our install script:
```shell
curl -Lo https://github.com/miladhzzzz/har-x/hack/install.sh | su - sh
```
#### Usage 
```shell
./capy -url=<URL> -dns=<DNS server> -output=<output directory>
```

#### build from scratch!
Or you can build capy just make sure you have Go installed:
```shell
git clone https://github.com/miladhzzzz/har-x
cd har-x/capy
go mod download
cd cmd && go build -o ../bin/capy
sudo cp ../../bin/capy /usr/local/bin
```
**theres also a build.sh file in root dir of /har-x that you can use to automate your build's**


## HAR analyzing with Python

To install the required libraries, run the following command:

```shell
pip install -r requirements.txt
```

## Usage

To run the script, open a terminal and navigate to the directory containing the script. Then, run the following command:

```shell
python har.analyze.py -f example.har -o output.csv --plot
```

This will parse the HAR file located at `example.har`, create a csv document containing details about all the requests ,
summarize the data in JSON format and output to terminal, and generate analyses graphs.

You also can save the json output with following command:

```shell
python har.analyze.py -f example.har -o output.csv --plot >> summary.json
```

This will do all the above plus saving the summary as a json file in the working directory.

## Command-Line Arguments

The script accepts the following command-line arguments:

- `-f` or `--file`: The path to the HAR file to be parsed.
- `-s` or `--status`: The status code range to include (e.g. 200-299).
- `-c` or `--content`: The content type to include (e.g. text/html).
- `-g` or `--group`: The column to group by (e.g. url).
- `-o` or `--output`: The output file path if not present will use STDOUT as output.
- `--summarize`: Summarizes the HAR file and returns json.
- `--plot`: Generates analyses graph.

## Output

The script outputs the following:

- You can find examples of outputs for google.com and perplexity.ai in /output directory.

- A table of the filtered data either in terminal output or a csv file.
![](output/google.csv)

- A Multipane graph analyzing different aspects and visualizing them.
  ![](output/google.png)

- A summary of the data in JSON format.
```shell
{
    "total_requests": 42,
    "failed_request": 12,
    "average_time": 265.2077380945452,
    "fastest_request": 4.899999999906868,
    "slowest_request": 1201.3089999941421,
    "status_counts": {
        "200": 30,
        "204": 12
    },
    "content_types": {
        "text/html": 16,
        "text/javascript": 13,
        "image/png": 3,
        "text/plain": 3,
        "application/json": 2,
        "font/woff2": 1,
        "image/x-icon": 1,
        "image/webp": 1,
        "text/css": 1,
        "image/jpeg": 1
    }
}
```

## Why This Tool is Useful

The HAR Parser tool is useful for web developers, cybersecurity analysts, and web problem analysts who want to analyze the network traffic of a web page and identify performance issues. It can help developers identify slow-loading resources, identify resource usage patterns, and compare the performance of different pages or versions of a website. Additionally, the tool can be used to analyze web services like Netflix and provide valuable insights into their system architecture and functionality. This can help identify potential performance issues or bottlenecks and areas for optimization. Overall, the HAR Parser tool is a valuable tool for anyone involved in web development, performance testing, or system analysis.

## License

This script is licensed under the MIT License. See the `LICENSE` file for more information.