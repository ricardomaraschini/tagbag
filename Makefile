default: build

.PHONY: build
build:
	mkdir -p output/bin
	go build -o output/bin/tagbag ./cmd/tagbag

.PHONY: clean
clean:
	rm -rf output
