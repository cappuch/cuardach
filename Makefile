VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE    ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS  = -s -w \
	-X 'github.com/cappuch/cuardach/src/cmd.Version=$(VERSION)' \
	-X 'github.com/cappuch/cuardach/src/cmd.CommitHash=$(COMMIT)' \
	-X 'github.com/cappuch/cuardach/src/cmd.BuildDate=$(DATE)'

.PHONY: build clean test install

build:
	go build -ldflags "$(LDFLAGS)" -o cuardach ./src/

install:
	go install -ldflags "$(LDFLAGS)" ./src/

test:
	go test ./src/...

clean:
	rm -f cuardach
