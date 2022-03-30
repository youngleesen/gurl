package main

import (
	"compress/gzip"
	"fmt"
	"github.com/bingoohuang/gg/pkg/iox"
	"github.com/bingoohuang/goup/shapeio"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

func downloadFile(req *Request, res *http.Response, filename string) {
	fd, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE, 0o666)
	if err != nil {
		log.Fatalf("create download file %q failed: %v", filename, err)
	}

	if !isWindows() {
		fmt.Println(Color(res.Proto, Magenta), Color(res.Status, Green))
		for k, val := range res.Header {
			fmt.Println(Color(k, Gray), ":", Color(strings.Join(val, " "), Cyan))
		}
	} else {
		fmt.Println(res.Proto, res.Status)
		for k, val := range res.Header {
			fmt.Println(k, ":", strings.Join(val, " "))
		}
	}
	fmt.Println("")
	var total int64
	if contentLength := res.Header.Get("Content-Length"); contentLength != "" {
		total, _ = strconv.ParseInt(contentLength, 10, 64)
	}
	fmt.Printf("Downloading to %q\n", filename)
	pb := NewProgressBar(total).Start()
	mw := io.MultiWriter(fd, pb)

	br := res.Body
	if limitRate > 0 {
		br = shapeio.NewReader(br, shapeio.WithRateLimit(float64(limitRate)))
	}
	if res.Header.Get("Content-Encoding") == "gzip" {
		reader, err := gzip.NewReader(br)
		if err != nil {
			log.Fatalf("create download file %q failed: %v", filename, err)
		}
		br = reader
	}

	// disable timeout for downloading.
	// A zero value for t means I/O operations will not time out.
	if err := req.ConnInfo.Conn.SetDeadline(time.Time{}); err != nil {
		log.Printf("failed to set deadline: %v", err)
	}
	if _, err := io.Copy(mw, br); err != nil {
		log.Fatal("Can't Write the body into file", err)
	}
	pb.Finish()
	iox.Close(fd, br)
	fmt.Println()
}
