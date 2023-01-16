package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"
)

func TestRequest(t *testing.T) {
	type params struct {
		name             string
		giveUrlFmtStr    string
		giveOptions      RequestOptions
		wantStatusCode   int
		wantContentRegex string
	}

	cases := []params{
		{
			name:             "index",
			giveUrlFmtStr:    "tcp4://localhost:%s/",
			giveOptions:      defaultOpts(),
			wantStatusCode:   200,
			wantContentRegex: `(?s).+<title>PHP 8.2.1 - phpinfo\(\)<\/title>.+`,
		},
		{
			name:             "ping",
			giveUrlFmtStr:    "tcp4://localhost:%s/fpm-ping",
			giveOptions:      defaultOpts(),
			wantStatusCode:   200,
			wantContentRegex: `^pong$`,
		},
		{
			name:             "status",
			giveUrlFmtStr:    "tcp4://localhost:%s/fpm-status",
			giveOptions:      defaultOpts(),
			wantStatusCode:   200,
			wantContentRegex: `(?s)^pool:\s+www.+`,
		},
		{
			name:             "status-json",
			giveUrlFmtStr:    "tcp4://localhost:%s/fpm-status?json",
			giveOptions:      defaultOpts(),
			wantStatusCode:   200,
			wantContentRegex: `^{"pool":"www",.+`,
		},
	}
	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			cid, port, err := startFpmContainer(ctx)
			if err != nil {
				t.Fatal(err)
			}
			defer func() {
				if err := stopFpmContainer(ctx, cid); err != nil {
					t.Error(err)
				}
			}()

			res, err := fpmRequest(fmt.Sprintf(tt.giveUrlFmtStr, port), tt.giveOptions)
			if err != nil {
				t.Fatal(err)
			}

			defer res.Body.Close()

			if res.StatusCode != tt.wantStatusCode {
				t.Errorf("bad status: got %d, want: %d", res.StatusCode, tt.wantStatusCode)
			}

			b := new(bytes.Buffer)
			if _, err := io.Copy(b, res.Body); err != nil {
				t.Fatal(err)
			}
			ok, err := regexp.Match(tt.wantContentRegex, b.Bytes())
			if err != nil {
				t.Error(err)
			}

			if !ok {
				t.Errorf("content does not match: regex: %q, content: %q", tt.wantContentRegex, b.String())
			}
		})
	}
}

func startFpmContainer(ctx context.Context) (id string, port string, err error) {
	testdata, err := filepath.Abs("./testdata")
	if err != nil {
		return "", "", err
	}

	cid, err := cmd(ctx,
		"docker", "run", "--rm", "-d",
		"-p", "127.0.0.1::9000",
		fmt.Sprintf("-v=%s:%s", filepath.Join(testdata, "php", "html"), "/var/www/html"),
		fmt.Sprintf("-v=%s:%s", filepath.Join(testdata, "php", "healthz.conf"), "/usr/local/etc/php-fpm.d/zz-status.conf"),
		"docker.io/library/php:8.2.1-fpm-alpine3.17",
	)

	if err != nil {
		return "", "", err
	}

	time.Sleep(time.Second)

	port, err = cmd(ctx,
		"docker", "container",
		"inspect", cid, "--format",
		`{{ (index (index .NetworkSettings.Ports "9000/tcp") 0).HostPort }}`,
	)
	if err != nil {
		return "", "", err
	}

	time.Sleep(time.Second)

	return cid, port, nil
}

func stopFpmContainer(ctx context.Context, id string) error {
	_, err := cmd(ctx, "docker", "stop", id)
	return err
}

func cmd(ctx context.Context, args ...string) (string, error) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	c := exec.CommandContext(ctx, args[0], args[1:]...)
	c.Stdout = stdout
	c.Stderr = stderr
	if err := c.Run(); err != nil {
		return "", fmt.Errorf("%w: %s", err, stderr.String())
	}
	return strings.TrimSpace(stdout.String()), nil
}
