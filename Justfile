# justfile for ipfind

build:
	go build -o ipfind .

test:
	go test ./...

fmt:
	gofmt -w .

clean:
	rm -f ipfind
