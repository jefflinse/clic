all: clean build

source_files = *.go */*.go */*/*.go
bin_dir = hm/bin
coverage_profile = coverage.out
coverage_report = coverage.report.out
hm_bin = $(bin_dir)/hm

git_rev := $(shell git rev-parse --short HEAD)
git_tag := $(shell git describe --tags --match "v*.*.*" --abbrev=0 HEAD 2>/dev/null)
timestamp := $(shell date -u +%Y%m%d%H%M%S)

ifeq ($(git_tag),)
git_tag := v0.0.0
endif

ifndef VERSION
VERSION := $(git_tag)-$(git_rev)-$(timestamp)
endif

clean:
	go clean -i -testcache ./...
	rm -rf $(bin_dir)
	rm -f $(coverage_profile) $(coverage_report)

build: $(hm_bin)

package: $(hm_tarball)

test: $(coverage_profile)

coverage: $(coverage_repkort)
	cat $(coverage_report)

coverage-html: $(coverage_profile)
	go tool cover -html=$(coverage_profile)

$(hm_bin): $(source_files)
	mkdir -p $(bin_dir)
	cd hm && go build -ldflags "-X 'main.Version=$(VERSION)'" -o bin/hm

$(hm_tarball): $(hm_bin)
	tar -zcvf handyman-darwin-amd64-$(VERSION).tar.gz $(bin_dir)

$(coverage_report): $(coverage_profile)
	go tool cover -func=$(coverage_profile) > $(coverage_report)

$(coverage_profile): $(source_files)
	go test -tags test -coverprofile=$(coverage_profile) -race ./...


