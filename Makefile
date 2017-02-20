default: vet test

vet:
	go tool vet .

test:
	go test ./...

.PHONY: default test vet
