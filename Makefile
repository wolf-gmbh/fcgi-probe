dist:
	CGO_ENABLED=0 go build -o dist/fcgi-probe ./cmd/fpm/
	tar -czf dist/fcgi-probe-linux-amd64-static.tar.gz dist/fcgi-probe

clean:
	rm -rf dist
