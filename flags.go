package main

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/bingoohuang/gg/pkg/fla9"
	"github.com/bingoohuang/gg/pkg/man"
	"go.uber.org/atomic"
)

var (
	disableKeepAlive, ver, form, pretty, enableTlcp bool
	ugly, raw, freeInnerJSON, gzipOn, countingItems bool
	auth, proxy, printV, body, think, method, dns   string
	caFile, tlcpCerts                               string
	uploadFiles, urls                               []string
	printOption                                     uint16
	benchN, benchC, confirmNum                      int
	currentN                                        atomic.Int64
	timeout                                         time.Duration
	limitRate                                       = NewRateLimitFlag()
	download                                        = &fla9.StringBool{}

	jsonmap = map[string]interface{}{}
)

func init() {
	flagEnv(&urls, "url,u", "", "", "URL")
	fla9.StringVar(&method, "method,m", "GET", "")
	fla9.StringVar(&tlcpCerts, "tlcp-certs", "", "format: sign.cert.pem,sign.key.pem,enc.cert.pem,enc.key.pem")

	fla9.BoolVar(&disableKeepAlive, "k", false, "")
	fla9.BoolVar(&enableTlcp, "tlcp", false, "")
	fla9.BoolVar(&ver, "version,v", false, "")
	fla9.BoolVar(&raw, "raw,r", false, "")
	fla9.BoolVar(&ugly, "ugly", false, "")
	fla9.BoolVar(&countingItems, "count", false, "")
	fla9.StringVar(&printV, "print,p", "A", "")
	fla9.StringVar(&caFile, "ca", "", "")
	fla9.BoolVar(&form, "f", false, "")
	fla9.BoolVar(&gzipOn, "gzip", false, "")
	fla9.Var(download, "d", "")
	fla9.DurationVar(&timeout, "t", time.Minute, "")
	fla9.StringsVar(&uploadFiles, "F", nil, "")
	fla9.Var(limitRate, "L", "")
	fla9.StringVar(&think, "think", "0", "")

	flagEnvVar(&auth, "auth", "", "", `AUTH`)
	flagEnvVar(&proxy, "proxy,P", "", "", `PROXY`)
	fla9.IntVar(&benchN, "n", 1, "")
	fla9.IntVar(&confirmNum, "confirm", 0, "")
	fla9.IntVar(&benchC, "c", 1, "")
	fla9.StringVar(&body, "body,b", "", "")
	fla9.StringVar(&dns, "dns", "", "")
}

const (
	printReqHeader uint16 = 1 << iota
	printReqURL
	printReqBody
	printRespOption
	printRespHeader
	printRespCode
	printRespBody
	printReqSession
	printVerbose
	printHTTPTrace
	printDebug
	quietFileUploadDownloadProgressing
	freeInnerJSONTag
)

func parsePrintOption(s string) {
	AdjustPrintOption(&s, 'A', printReqHeader|printReqBody|printRespHeader|printRespBody|printReqSession|printVerbose|printHTTPTrace)
	AdjustPrintOption(&s, 'a', printReqHeader|printReqBody|printRespHeader|printRespBody|printReqSession|printVerbose|printHTTPTrace)
	AdjustPrintOption(&s, 'H', printReqHeader)
	AdjustPrintOption(&s, 'B', printReqBody)
	AdjustPrintOption(&s, 'o', printRespOption)
	AdjustPrintOption(&s, 'h', printRespHeader)
	AdjustPrintOption(&s, 'b', printRespBody)
	AdjustPrintOption(&s, 's', printReqSession)
	AdjustPrintOption(&s, 'v', printVerbose)
	AdjustPrintOption(&s, 't', printHTTPTrace)
	AdjustPrintOption(&s, 'c', printRespCode)
	AdjustPrintOption(&s, 'u', printReqURL)
	AdjustPrintOption(&s, 'q', quietFileUploadDownloadProgressing)
	AdjustPrintOption(&s, 'f', freeInnerJSONTag)
	AdjustPrintOption(&s, 'd', printDebug)

	if s != "" {
		log.Fatalf("unknown print option: %s", s)
	}
}

func AdjustPrintOption(s *string, r rune, flags uint16) {
	if strings.ContainsRune(*s, r) {
		printOption |= flags
		*s = strings.ReplaceAll(*s, string(r), "")
	}
}

func HasPrintOption(flags uint16) bool {
	return printOption&flags == flags
}

const help = `gurl is a Go implemented cURL-like cli tool for humans.
Usage:
	gurl [flags] [METHOD] URL [URL] [ITEM [ITEM]]
flags:
  -u                HTTP request URL
  -method -m        HTTP method
  -k                Disable keepalive
  -version -v       Print Version Number
  -raw -r           Print JSON Raw format other than pretty
  -ugly             Print JSON In Ugly compact Format
  -C                Print items counting in colored output
  -ca               Ca root certificate file to verify TLS
  -f                Submitting the data as a form
  -gzip             Gzip request body or not
  -d                Download the url content as file, yes/n
  -t                Timeout for read and write, default 1m
  -F filename       Upload a file, e.g. gurl :2110 -F 1.png -F 2.png
  -L limit          Limit rate /s, like 10K, append :req/:rsp to specific the limit direction
  -think            Think time, like 5s, 100ms, 100ms-5s, 100-200ms and etc.
  -auth=USER[:PASS] HTTP authentication username:password, USER[:PASS]
  -proxy=PROXY_URL  Proxy host and port, PROXY_URL
  -n=1 -c=1         Number of requests and concurrency to run
  -confirm=0        Should confirm after number of requests 
  -body,b           Send RAW data as body, or @filename to load body from the file's content
  -print -p         String specifying what the output should contain, default will print all information
                       H: request headers  B: request body,  u: request URL
                       h: response headers  b: response body, c: status code
                       s: http conn session v: Verbose t: HTTP trace
                       q: keep quiet for file uploading/downloading progress
                       f: expand inner JSON string as JSON object
  -dns              Specified custom DNS resolver address, format: [DNS_SERVER]:[PORT]
  -version,v        Show Version Number
  -tlcp             使用传输层密码协议(TLCP)，TLCP协议遵循《GB/T 38636-2020 信息安全技术 传输层密码协议》。
METHOD:
  gurl defaults to either GET (if there is no request data) or POST (with request data).
URL:
  The only one needed to perform a request is a URL. The default scheme is http://,
  which can be omitted from the argument; example.org works just fine.
ITEM:
  Can be any of: Query      : key=value  Header: key:value       Post data: key=value
                 Force query: key==value key==@/path/file
                 JSON data  : key:=value Upload: key@/path/file
                 File content as body: @/path/file
Example:
  gurl beego.me
  gurl :8080
Envs:
  1. URL:         URL
  2. PROXY:       Proxy host and port， like: http://proxy.cn, https://user:pass@proxy.cn
  3. AUTH:        HTTP authentication username:password, USER[:PASS]
  4. TLS_VERIFY:  Enable client verifies the server's certificate chain and host name.
more help information please refer to https://github.com/bingoohuang/gurl
`

func usage() {
	fmt.Print(help)
	os.Exit(2)
}

type RateLimitDirection int

const (
	RateLimitBoth RateLimitDirection = iota
	RateLimitRequest
	RateLimitResponse
)

func NewRateLimitFlag() *RateLimitFlag {
	return &RateLimitFlag{}
}

type RateLimitFlag struct {
	Val       *uint64
	Direction RateLimitDirection
}

func (i *RateLimitFlag) Enabled() bool { return i.Val != nil && *i.Val > 0 }

func (i *RateLimitFlag) String() string {
	if !i.Enabled() {
		return "0"
	}

	s := man.Bytes(*i.Val)
	switch i.Direction {
	case RateLimitRequest:
		return s + ":req"
	case RateLimitResponse:
		return s + ":rsp"
	}

	return s
}

func (i *RateLimitFlag) Set(value string) (err error) {
	dirPos := strings.IndexByte(value, ':')
	i.Direction = RateLimitBoth
	if dirPos > 0 {
		switch dir := value[dirPos+1:]; strings.ToLower(dir) {
		case "req":
			i.Direction = RateLimitRequest
		case "rsp":
			i.Direction = RateLimitResponse
		default:
			log.Fatalf("unknown rate limit %s", value)
		}
		value = value[:dirPos]
	}

	val, err := man.ParseBytes(value)
	if err != nil {
		return err
	}

	i.Val = &val
	return nil
}

func (i *RateLimitFlag) IsForReq() bool {
	return i.Enabled() && (i.Direction == RateLimitRequest || i.Direction == RateLimitBoth)
}

func (i *RateLimitFlag) IsForRsp() bool {
	return i.Enabled() && (i.Direction == RateLimitResponse || i.Direction == RateLimitBoth)
}

func (i *RateLimitFlag) Float64() float64 { return float64(*i.Val) }
