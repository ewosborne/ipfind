# justfile for ipfind

build: clean test
	go build -ldflags "-s -w" -o ipfind .

test:
	go test ./... -v

fmt:
	gofmt -w .

clean:
	rm -f ipfind

install: clean build
	mkdir -p $HOME/bin
	cp ipfind $HOME/bin/
	chmod 755 $HOME/bin/ipfind
	echo "installed ipfind to $HOME/bin/ipfind"
