default: build

VERSION ?= $(shell git describe --tags --always --dirty)

.PHONY: build
build:
	mkdir -p output/bin
	CGO_ENABLED=0 go build -ldflags="-X 'main.Version=$(VERSION)'" -tags containers_image_openpgp -o output/bin/tagbag ./cmd/tagbag

.PHONY: clean
clean:
	rm -rf output
