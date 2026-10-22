PLATFORMS := darwin/amd64 darwin/arm64 linux/amd64 linux/arm64
BINARIES  := $(foreach p,$(PLATFORMS),bin/pstack-setup-$(subst /,-,$(p)))

.PHONY: all build test grab-original clean

all: build

# Build one binary per platform into bin/. Run after changing setup/, then commit bin/.
build: $(BINARIES)

bin/pstack-setup-%: setup/*.go go.mod
	@mkdir -p bin
	CGO_ENABLED=0 GOOS=$(word 1,$(subst -, ,$*)) GOARCH=$(word 2,$(subst -, ,$*)) \
		go build -trimpath -ldflags="-s -w" -o $@ ./setup
	@echo "built $@"

# Run the Go tests for the setup command.
test:
	go vet ./setup/
	go test ./setup/ -count=1

# Clone the original cursor plugins repo next to this project for reference.
grab-original:
	git clone org-126759922@github.com:cursor/plugins.git cursor-plugins

clean:
	rm -f $(BINARIES)
