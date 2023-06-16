package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"

	"github.com/spf13/pflag"
	"github.com/yookoala/gofast"
)

type RequestOptions struct {
	docroot   string
	index     string
	method    string
	body      string
	statusMin int
	statusMax int
	silent    bool
	headers   []string
	debug     bool
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "\nERROR: %v\n", err)
		os.Exit(1)
	}
}

func defaultOpts() RequestOptions {
	return RequestOptions{
		docroot:   "/var/www/html",
		index:     "index.php",
		method:    http.MethodGet,
		statusMin: 200,
		statusMax: 299,
	}
}

func run() error {
	opts := defaultOpts()

	pflag.CommandLine.SortFlags = false
	pflag.StringVar(&opts.docroot, "document-root", opts.docroot, "the fpm document root")
	pflag.StringVar(&opts.index, "index", opts.index, "the php entrypoint")
	pflag.StringSliceVarP(&opts.headers, "header", "H", opts.headers, "additional http headers are mapped to fcgi HTTP_X params")
	pflag.StringVarP(&opts.method, "method", "X", opts.method, "the method to use for the request")
	pflag.StringVarP(&opts.body, "data", "d", opts.body, "the request body")
	pflag.BoolVarP(&opts.silent, "silent", "s", opts.silent, "don't show any output, unless failed")
	pflag.IntVar(&opts.statusMin, "status-min", opts.statusMin, "fail if status is below")
	pflag.IntVar(&opts.statusMax, "status-max", opts.statusMax, "fail if status is above")
	pflag.BoolVar(&opts.debug, "debug", opts.debug, "dump the fcgi params")
	pflag.Parse()

	if len(pflag.Args()) != 1 {
		return errors.New("need exactly 1 argument")
	}

	res, err := fpmRequest(pflag.Arg(0), opts)
	if err != nil {
		return err
	}

	if !opts.silent && res.Body != nil {
		if _, err := io.Copy(os.Stdout, res.Body); err != nil {
			return fmt.Errorf("copy body: %w", err)
		}
		fmt.Println()
	}

	if sc := res.StatusCode; sc < opts.statusMin || sc > opts.statusMax {
		return fmt.Errorf("bad status code: %d", res.StatusCode)
	}

	return nil

}

func fpmRequest(rawUrl string, opts RequestOptions) (*http.Response, error) {
	u, err := url.Parse(rawUrl)
	if err != nil {
		return nil, fmt.Errorf("invalid url: %w", err)
	}

	httpReq, err := http.NewRequest(opts.method, u.String(), strings.NewReader(opts.body))
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}

	err = parseHeader(opts.headers, httpReq.Header)
	if err != nil {
		return nil, err
	}

	connFactory := gofast.SimpleConnFactory(u.Scheme, u.Host)
	newFcgiClient := gofast.SimpleClientFactory(connFactory)

	client, err := newFcgiClient()
	if err != nil {
		return nil, fmt.Errorf("fcgi client: %w", err)
	}

	defer client.Close()

	req := gofast.NewRequest(httpReq)

	pipe, err := fpmHandler(opts.docroot, opts.index, opts.debug)(client, req)
	if err != nil {
		return nil, fmt.Errorf("fcgi request: %w", err)
	}

	defer pipe.Close()

	rec := httptest.NewRecorder()

	ew := new(bytes.Buffer)
	if err := pipe.WriteTo(rec, ew); err != nil {
		return nil, fmt.Errorf("recorder: %s: %w", ew.String(), err)
	}

	return rec.Result(), nil
}

func fpmHandler(docroot, index string, debug bool) gofast.SessionHandler {
	return gofast.Chain(gofast.MapHeader, func(inner gofast.SessionHandler) gofast.SessionHandler {
		return func(client gofast.Client, req *gofast.Request) (*gofast.ResponsePipe, error) {
			if req.Raw.URL.Path == "" {
				req.Raw.URL.Path = "/"
			}

			req.Params["QUERY_STRING"] = req.Raw.URL.RawQuery
			req.Params["REQUEST_METHOD"] = req.Raw.Method
			req.Params["CONTENT_TYPE"] = req.Raw.Header.Get("content-type")
			req.Params["CONTENT_LENGTH"] = fmt.Sprintf("%d", req.Raw.ContentLength)

			req.Params["SCRIPT_FILENAME"] = fmt.Sprintf("%s/%s", docroot, index)
			req.Params["DOCUMENT_ROOT"] = docroot
			req.Params["DOCUMENT_URI"] = "/" + index
			req.Params["REQUEST_URI"] = req.Raw.URL.Path
			req.Params["SCRIPT_NAME"] = req.Raw.URL.Path

			if debug {
				if b, err := json.MarshalIndent(req.Params, "", "  "); err == nil {
					fmt.Println(string(b))
				}
			}

			return inner(client, req)
		}
	})(gofast.BasicSession)
}

func parseHeader(raw []string, dest http.Header) error {
	for _, r := range raw {
		key, value, ok := strings.Cut(r, ":")
		if !ok {
			return fmt.Errorf("malformed header: %s", r)
		}
		dest.Set(strings.TrimSpace(key), strings.TrimSpace(value))
	}
	return nil
}
