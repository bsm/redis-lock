default: test

deps:
	go get -t ./...

test: deps
	go test ./...

.PHONY: deault deps test
