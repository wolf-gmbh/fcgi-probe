# fcqi-probe

Probe fcgi applications and fail on bad status.

> **Note** this is currently geared towards fpm. It probably wont work with other types of fcgi apps.

## Synopsis

```console
Usage of fcgi-probe:

  fcgi-probe [OPTIONS]... [URL]

OPTIONS:
      --document-root string   the fpm document root (default "/var/www/html")
      --index string           the php entrypoint (default "index.php")
  -H, --header strings         additional http headers are mapped to fcgi HTTP_X params
  -X, --method string          the method to use for the request (default "GET")
  -d, --data string            the request body
  -s, --silent                 don't show any output, unless failed
      --status-min int         fail if status is below (default 200)
      --status-max int         fail if status is above (default 299)
      --debug                  dump the fcgi params
```

The fcqi `REQUEST_URI` and `PATH_INFO` are derived from the request path.

## Example

```bash
fcgi-probe tcp4://localhost:9000/livez
```

> **Note** Make sure your document root and index parameter match with the fcgi app

## Installation

## Binary Release

```bash
curl -fsSL github.com/wolf-gmbh/fcgi-probe/releases/latest/download/fcgi-probe
chmod +x fcgi-probe
```

### From Source

```bash
go install github.com/wolf-gmbh/fcgi-probe@latest
```
