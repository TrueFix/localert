.PHONY: build web test check release installers

VERSION ?= 0.3.1
RELEASE_BASE_URL ?=
UPDATE_FEED_URL ?=
LDFLAGS = -s -w -X main.version=$(VERSION) -X main.defaultUpdateFeed=$(UPDATE_FEED_URL)

web:
	npm --prefix web ci
	npm --prefix web run build

build: web
	CGO_ENABLED=0 go build -trimpath -ldflags "$(LDFLAGS)" -o bin/localert ./cmd/localert
	$(MAKE) installers

test:
	go test -race ./internal/...

check: test
	go vet ./...
	sh -n scripts/install.sh
	python -m unittest discover -s scripts -p 'test_*.py'

release: web
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags "$(LDFLAGS)" -o bin/localert-linux-amd64 ./cmd/localert
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -trimpath -ldflags "$(LDFLAGS)" -o bin/localert-linux-arm64 ./cmd/localert
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -trimpath -ldflags "$(LDFLAGS)" -o bin/localert-windows-amd64.exe ./cmd/localert
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -trimpath -ldflags "$(LDFLAGS)" -o bin/localert-macos-amd64 ./cmd/localert
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -trimpath -ldflags "$(LDFLAGS)" -o bin/localert-macos-arm64 ./cmd/localert
	$(MAKE) installers
	go run ./cmd/localert-release --version "$(VERSION)" --dir bin --base-url "$(RELEASE_BASE_URL)"

installers:
	mkdir -p bin
	cp scripts/install.sh scripts/install.ps1 bin/
	chmod 755 bin/install.sh
