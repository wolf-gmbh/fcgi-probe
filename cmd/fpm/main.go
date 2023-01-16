package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"

	"github.com/spf13/pflag"
	"github.com/yookoala/gofast"
)

var (
	docroot   = "/var/www/html/public"
	index     = "index.php"
	method    = http.MethodGet
	body      = ""
	statusMin = 200
	statusMax = 299
	silent    = false
	headers   = []string{}
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "\nERROR: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	pflag.CommandLine.SortFlags = false
	pflag.StringVar(&docroot, "document-root", docroot, "the fpm document root")
	pflag.StringVar(&index, "index", index, "the php entrypoint")
	pflag.StringSliceVarP(&headers, "header", "H", headers, "additional http headers are mapped to fcgi HTTP_X params")
	pflag.StringVarP(&method, "method", "X", method, "the method to use for the request")
	pflag.StringVarP(&body, "data", "d", body, "the request body")
	pflag.BoolVarP(&silent, "silent", "s", silent, "don't show any output, unless failed")
	pflag.IntVar(&statusMin, "status-min", statusMin, "fail if status is below")
	pflag.IntVar(&statusMax, "status-max", statusMax, "fail if status is above")
	pflag.Parse()

	if len(pflag.Args()) != 1 {
		log.Fatal("need exactly 1 argument")
	}

	u, err := url.Parse(pflag.Arg(0))
	if err != nil {
		return fmt.Errorf("invalid url: %w", err)
	}

	connFactory := gofast.SimpleConnFactory(u.Scheme, u.Host)
	fac := gofast.SimpleClientFactory(connFactory)

	doReq := gofast.Chain(gofast.BasicParamsMap, gofast.MapHeader, func(inner gofast.SessionHandler) gofast.SessionHandler {
		return func(client gofast.Client, req *gofast.Request) (*gofast.ResponsePipe, error) {
			req.Params["DOCUMENT_ROOT"] = docroot
			req.Params["SCRIPT_FILENAME"] = fmt.Sprintf("%s/%s", docroot, index)
			req.Params["REQUEST_URI"] = req.Raw.URL.Path
			req.Params["PATH_INFO"] = req.Raw.URL.Path
			return inner(client, req)
		}
	})(gofast.BasicSession)

	c, err := fac()
	if err != nil {
		return fmt.Errorf("fcgi client: %w", err)
	}

	defer c.Close()

	r, err := http.NewRequest(method, u.RequestURI(), strings.NewReader(body))
	if err != nil {
		return fmt.Errorf("http request: %w", err)
	}

	for _, rawH := range headers {
		parts := strings.SplitN(rawH, ":", 2)
		if len(parts) != 2 {
			return fmt.Errorf("malformed header: %s", rawH)
		}
		r.Header.Set(strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]))
	}

	pipe, err := doReq(c, gofast.NewRequest(r))
	if err != nil {
		return fmt.Errorf("fcgi request: %w", err)
	}

	defer pipe.Close()

	rec := httptest.NewRecorder()

	pipe.WriteTo(rec, os.Stderr)

	if !silent && rec.Body != nil {
		if _, err := io.Copy(os.Stdout, rec.Body); err != nil {
			return fmt.Errorf("copy body: %w", err)
		}
		fmt.Println()
	}

	if sc := rec.Result().StatusCode; sc < statusMin || sc > statusMax {
		return fmt.Errorf("bad status code: %d", rec.Result().StatusCode)
	}

	return nil
}

// DOCUMENT_ROOT=/var/www/html/public
// SCRIPT_FILENAME=/var/www/html/public/index.php
// REMOTE_ADDR=127.0.0.1
// REQUEST_METHOD=GET
// REQUEST_URI="$1"
// PATH_INFO="$1"
