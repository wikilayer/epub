.DEFAULT_GOAL := build

STATICCHECK_VERSION ?= v0.8.1

# The Go that CI installs rather than whatever Homebrew last upgraded to: a
# newer local Go can write export data the pinned staticcheck cannot read, and
# lint then fails here only.
GOTOOLCHAIN ?= go1.27.1
export GOTOOLCHAIN

.PHONY: install-tools install-epubcheck format lint comments test-build test build

install-tools:
	go install honnef.co/go/tools/cmd/staticcheck@$(STATICCHECK_VERSION)
	python3 -m pip install --quiet --upgrade git+https://github.com/botforge-pro/commentcensor.git

# The tests hand every book they make to epubcheck and fail without it. CI runs
# on Ubuntu; on a Mac, `brew install epubcheck`.
install-epubcheck:
	sudo apt-get update -qq && sudo apt-get install -y -qq epubcheck

format:
	gofmt -w .

comments:
	commentcensor .

lint: comments
	go vet ./...
	gofmt -l . | (! grep .)
	$(shell go env GOPATH)/bin/staticcheck ./...

test-build:
	go build ./...
	go test -run '^$$' ./...

test:
	go test -count=1 ./...

build: lint test-build test
	go build ./...
