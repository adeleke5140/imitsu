VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BINARY := itui
BUILD_DIR := dist

PLATFORMS := \
	darwin/arm64 \
	darwin/amd64 \
	linux/arm64 \
	linux/amd64

LDFLAGS := -s -w -X main.version=$(VERSION)

.PHONY: build release clean

build:
	go build -ldflags "$(LDFLAGS)" -o $(BINARY) .

release: clean
	@mkdir -p $(BUILD_DIR)
	@for platform in $(PLATFORMS); do \
		os=$${platform%/*}; \
		arch=$${platform#*/}; \
		output=$(BUILD_DIR)/$(BINARY)-$${os}-$${arch}; \
		echo "Building $${os}/$${arch}..."; \
		GOOS=$${os} GOARCH=$${arch} go build -ldflags "$(LDFLAGS)" -o $${output} . ; \
	done
	@echo "Done. Binaries in $(BUILD_DIR)/"
	@ls -lh $(BUILD_DIR)/

clean:
	rm -rf $(BUILD_DIR) $(BINARY)
