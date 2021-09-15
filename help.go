package main

import (
	"fmt"
	"os"
)

const help = `gurl is a Go implemented CLI cURL-like tool for humans.
Usage:
	gurl [flags] [METHOD] URL [ITEM [ITEM]]
flags:
  -auth=USER[:PASS] Pass a username:password pair as the argument
  -b.n=0 -b.c=100   Number of requests and concurrency to run
  -body=""          Send RAW data as body
  -f                Submitting the data as a form
  -j                Send the data in a JSON object as application/json
  -raw              Print JSON Raw format other than pretty
  -i                Allow connections to SSL sites without certs
  -proxy=PROXY_URL  Proxy with host and port
  -print=A          String specifying what the output should contain, default will print all information
                       H: request headers  B: request body  h: response headers  b: response body
  -v                Show Version Number
METHOD:
  gurl defaults to either GET (if there is no request data) or POST (with request data).
URL:
  The only one needed to perform a request is a URL. The default scheme is http://,
  which can be omitted from the argument; example.org works just fine.
ITEM:
  Can be any of: Query      : key=value  Header: key:value       Post data: key=value 
                 Force query: key==value
                 JSON data  : key:=value Upload: key@/path/file
Example:
  gurl beego.me
more help information please refer to https://github.com/bingoohuang/gurl
`

func usage() {
	fmt.Print(help)
	os.Exit(2)
}