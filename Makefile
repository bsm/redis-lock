default: vet test

deps:
	go get -t ./...

vet:
	go tool vet .

test: deps
	go test ./...

.PHONY: default deps test
