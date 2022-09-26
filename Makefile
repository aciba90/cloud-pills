.PHONY: all
all: test lint build

.PHONY: test
test:
	go test ./...

.PHONY: lint
lint: vet fmt

.PHONY: vet
vet:
	go vet ./...

.PHONY: fmt
fmt:
	test -z $$(go list -f '{{.Dir}}' ./... | grep -v /vendor/ | xargs -L1 gofmt -l)

.PHONY: fmt_do
fmt_do:
	gofmt -w -s .

.PHONY: build
build: bin
	go build -o bin/cloud-clean ./cmd/cloud-clean

bin:
	@mkdir bin

.PHONY: clean
clean:
	rm -rf bin/