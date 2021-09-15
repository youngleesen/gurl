// Gurl is a Go implemented CLI cURL-like tool for humans
// gurl [flags] [METHOD] URL [ITEM [ITEM]]
package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
)

const (
	version              = "0.1.0"
	printReqHeader uint8 = 1 << (iota - 1)
	printReqBody
	printRespHeader
	printRespBody
)

var (
	ver, form, pretty, raw, download, insecureSSL bool
	auth, proxy, printV, body                     string
	printOption                                   uint8
	benchN, benchC                                int

	isjson           = flag.Bool("json", true, "Send the data as a JSON object")
	method           = flag.String("method", "GET", "HTTP method")
	URL              = flagEnv("url", "", "HTTP request URL")
	jsonmap          = map[string]interface{}{}
	contentJsonRegex = regexp.MustCompile(`application/(.*)json`)
)

func flagEnv(name, value, usage string) *string {
	if value == "" {
		value = os.Getenv("GURL_" + strings.ToUpper(name))
	}
	return flag.String(name, value, usage)
}

func flagEnvVar(p *string, name, value, usage string) {
	if value == "" {
		value = os.Getenv("GURL_" + strings.ToUpper(name))
	}
	flag.StringVar(p, name, value, usage)
}

func init() {
	flag.BoolVar(&ver, "v", false, "Print Version Number")
	flag.BoolVar(&raw, "raw", false, "Print JSON Raw Format")
	flag.StringVar(&printV, "print", "A", "Print request and response")
	flag.BoolVar(&form, "f", false, "Submitting as a form")
	flag.BoolVar(&download, "d", false, "Download the url content as file")
	flag.BoolVar(&insecureSSL, "i", false, "Allow connections to SSL sites without certs")

	flagEnvVar(&auth, "auth", "", "HTTP authentication username:password, USER[:PASS]")
	flagEnvVar(&proxy, "proxy", "", "Proxy host and port, PROXY_URL")
	flag.IntVar(&benchN, "b.n", 0, "Number of bench requests to run")
	flag.IntVar(&benchC, "b.c", 100, "Number of bench requests to run concurrently.")
	flag.StringVar(&body, "body", "", "Raw data send as body")
}

func parsePrintOption(s string) {
	if strings.ContainsRune(s, 'A') {
		printOption = printReqHeader | printReqBody | printRespHeader | printRespBody
		return
	}

	if strings.ContainsRune(s, 'H') {
		printOption |= printReqHeader
	}
	if strings.ContainsRune(s, 'B') {
		printOption |= printReqBody
	}
	if strings.ContainsRune(s, 'h') {
		printOption |= printRespHeader
	}
	if strings.ContainsRune(s, 'b') {
		printOption |= printRespBody
	}
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile | log.Lmicroseconds)
	flag.Usage = usage

	var flagArgs []string
	var nonFlagArgs []string
	for i, arg := range os.Args[1:] {
		if strings.HasPrefix(arg, "-") {
			flagArgs = os.Args[i+1:]
			break
		} else {
			nonFlagArgs = append(nonFlagArgs, arg)
		}
	}

	flag.CommandLine.Parse(flagArgs)
	pretty = !raw

	nonFlagArgs = append(nonFlagArgs, flag.Args()...)
	if len(nonFlagArgs) > 0 {
		nonFlagArgs = filter(nonFlagArgs)
	}
	if ver {
		fmt.Println("Version:", version)
		os.Exit(2)
	}
	parsePrintOption(printV)
	if printOption&printReqBody != printReqBody {
		defaultSetting.DumpBody = false
	}
	var stdin []byte
	if runtime.GOOS != "windows" {
		fi, err := os.Stdin.Stat()
		if err != nil {
			panic(err)
		}
		if fi.Size() != 0 {
			stdin, err = ioutil.ReadAll(os.Stdin)
			if err != nil {
				log.Fatal("Read from Stdin", err)
			}
		}
	}

	if *URL == "" {
		usage()
	}
	if strings.HasPrefix(*URL, ":") {
		urlb := []byte(*URL)
		if *URL == ":" {
			*URL = "http://localhost/"
		} else if len(*URL) > 1 && urlb[1] != '/' {
			*URL = "http://localhost" + *URL
		} else {
			*URL = "http://localhost" + string(urlb[1:])
		}
	}
	if !strings.HasPrefix(*URL, "http://") && !strings.HasPrefix(*URL, "https://") {
		*URL = "http://" + *URL
	}
	u, err := url.Parse(*URL)
	if err != nil {
		log.Fatal(err)
	}
	if auth != "" {
		if userpass := strings.Split(auth, ":"); len(userpass) == 2 {
			u.User = url.UserPassword(userpass[0], userpass[1])
		} else {
			u.User = url.User(auth)
		}
	}
	*URL = u.String()
	req := getHTTP(*method, *URL, nonFlagArgs)
	if u.User != nil {
		password, _ := u.User.Password()
		req.SetBasicAuth(u.User.Username(), password)
	}
	// Insecure SSL Support
	if insecureSSL {
		req.SetTLSClientConfig(&tls.Config{InsecureSkipVerify: true})
	}
	// Proxy Support
	if proxy != "" {
		purl, err := url.Parse(proxy)
		if err != nil {
			log.Fatal("Proxy Url parse err", err)
		}
		req.SetProxy(http.ProxyURL(purl))
	} else {
		eurl, err := http.ProxyFromEnvironment(req.Req)
		if err != nil {
			log.Fatal("Environment Proxy Url parse err", err)
		}
		req.SetProxy(http.ProxyURL(eurl))
	}
	if body != "" {
		req.Body(body)
	}
	if len(stdin) > 0 {
		var j interface{}
		d := json.NewDecoder(bytes.NewReader(stdin))
		d.UseNumber()
		err = d.Decode(&j)
		if err != nil {
			req.Body(stdin)
		} else {
			req.JsonBody(j)
		}
	}

	// AB bench
	if benchN > 0 {
		req.Debug(false)
		RunBench(req)
		return
	}
	res, err := req.Response()
	if err != nil {
		log.Fatalln("can't get the url", err)
	}

	filename := ""
	if disposition := res.Header.Get("Content-Disposition"); disposition != "" {
		if _, params, _ := mime.ParseMediaType(disposition); params != nil {
			filename = params["filename"]
		}
	}
	if download || filename != "" {
		downloadFile(u, res, filename)
		return
	}

	if runtime.GOOS != "windows" {
		fi, err := os.Stdout.Stat()
		if err != nil {
			panic(err)
		}
		if fi.Mode()&os.ModeDevice == os.ModeDevice {
			var dumpHeader, dumpBody []byte
			dump := req.DumpRequest()
			dps := strings.Split(string(dump), "\n")
			for i, line := range dps {
				if len(strings.Trim(line, "\r\n ")) == 0 {
					dumpHeader = []byte(strings.Join(dps[:i], "\n"))
					dumpBody = []byte(strings.Join(dps[i:], "\n"))
					break
				}
			}
			if printOption&printReqHeader == printReqHeader {
				fmt.Println(ColorfulRequest(string(dumpHeader)))
				fmt.Println("")
			}
			if printOption&printReqBody == printReqBody {
				if string(dumpBody) != "\r\n" {
					fmt.Println(string(dumpBody))
					fmt.Println("")
				}
			}
			if printOption&printRespHeader == printRespHeader {
				fmt.Println(Color(res.Proto, Magenta), Color(res.Status, Green))
				for k, v := range res.Header {
					fmt.Printf("%s: %s\n", Color(k, Gray), Color(strings.Join(v, " "), Cyan))
				}
				fmt.Println("")
			}
			if printOption&printRespBody == printRespBody {
				body, isJSON := formatResponseBody(res, req, pretty)
				fmt.Println(ColorfulResponse(body, isJSON))
			}
		} else {
			body, _ := formatResponseBody(res, req, pretty)
			_, err = os.Stdout.WriteString(body)
			if err != nil {
				log.Fatal(err)
			}
		}
	} else {
		var dumpHeader, dumpBody []byte
		dump := req.DumpRequest()
		dps := strings.Split(string(dump), "\n")
		for i, line := range dps {
			if len(strings.Trim(line, "\r\n ")) == 0 {
				dumpHeader = []byte(strings.Join(dps[:i], "\n"))
				dumpBody = []byte(strings.Join(dps[i:], "\n"))
				break
			}
		}
		if printOption&printReqHeader == printReqHeader {
			fmt.Println(string(dumpHeader))
			fmt.Println("")
		}
		if printOption&printReqBody == printReqBody {
			fmt.Println(string(dumpBody))
			fmt.Println("")
		}
		if printOption&printRespHeader == printRespHeader {
			fmt.Println(res.Proto, res.Status)
			for k, v := range res.Header {
				fmt.Println(k, ":", strings.Join(v, " "))
			}
			fmt.Println("")
		}
		if printOption&printRespBody == printRespBody {
			body, _ := formatResponseBody(res, req, pretty)
			fmt.Println(body)
		}
	}
}

func downloadFile(u *url.URL, res *http.Response, filename string) {
	if filename == "" {
		_, filename = filepath.Split(u.Path)
	}
	fd, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		log.Fatal("can't create file", err)
	}
	if runtime.GOOS != "windows" {
		fmt.Println(Color(res.Proto, Magenta), Color(res.Status, Green))
		for k, v := range res.Header {
			fmt.Println(Color(k, Gray), ":", Color(strings.Join(v, " "), Cyan))
		}
	} else {
		fmt.Println(res.Proto, res.Status)
		for k, v := range res.Header {
			fmt.Println(k, ":", strings.Join(v, " "))
		}
	}
	fmt.Println("")
	var total int64
	if contentLength := res.Header.Get("Content-Length"); contentLength != "" {
		total, _ = strconv.ParseInt(contentLength, 10, 64)
	}
	fmt.Printf("Downloading to \"%s\"\n", filename)
	pb := NewProgressBar(total)
	pb.Start()
	mw := io.MultiWriter(fd, pb)
	if _, err := io.Copy(mw, res.Body); err != nil {
		log.Fatal("Can't Write the body into file", err)
	}
	pb.Finish()
	fd.Close()
	res.Body.Close()
}