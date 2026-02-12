# justfile for ipfind

build:
	go build -o ipfind .

test:
	go test ./...

fmt:
	gofmt -w .

clean:
	rm -f ipfind

install:
	just build
	mkdir -p $HOME/bin
	cp ipfind $HOME/bin/
	chmod 755 $HOME/bin/ipfind
	echo "installed ipfind to $HOME/bin/ipfind"
