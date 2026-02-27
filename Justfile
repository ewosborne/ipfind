# justfile for ipfind

hello:
	go run . 1.2.3.4


bt: test build
	./ipfind 1.2.3.4 sample.txt -l

b: build

build: fmt
	go build -ldflags "-s -w" -o ipfind .

test:
	go test ./... -v

fmt:
	gofmt -w .

clean:
	rm -f ipfind

install: build
	mkdir -p $HOME/bin
	cp ipfind $HOME/bin/
	chmod 755 $HOME/bin/ipfind
	echo "installed ipfind to $HOME/bin/ipfind"
