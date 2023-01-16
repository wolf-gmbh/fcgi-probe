# fcqi-probe

Probe fcgi applications and fail on bad status.

> **Note** this is currently geared towards fpm. It probably wont work with other types of fcgi apps.

## Synopsis

```console
Usage of fcgi-probe:

  fcgi-probe [OPTIONS]... [URL]

OPTIONS:
      --document-root string   the fpm document root (default "/var/www/html/public")
      --index string           the php entrypoint (default "index.php")
  -H, --header strings         additional http headers are mapped to fcgi HTTP_X params
  -X, --method string          the method to use for the request (default "GET")
  -d, --data string            the request body
  -s, --silent                 don't show any output, unless failed
      --status-min int         fail if status is below (default 200)
      --status-max int         fail if status is above (default 299)
```

The fcqi `REQUEST_URI` and `PATH_INFO` are derived from the request path.

## Example

```bash
fcgi-probe tcp4://localhost:9000/livez
```

## Installation

## Binary Release

```bash
curl -fsSLO github.com/wolf-gmbh/fcgi-probe/releases/latest/downloads/fcgi-probe-linux-amd64-static.tar.gz \
  | tar -C /usr/local/bin/ -xzf -
```

### From Source

```
go install github.com/wolf-gmbh/fcgi-probe@latest
```
