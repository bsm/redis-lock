default: vet test

vet:
	go vet .

test:
	go test ./...

.PHONY: default test vet
