all: clean build

PLAT ?= darwin
ARCH ?= amd64

ifeq ($(PLAT),windows)
binary_ext = .exe
endif

# locations
repo_root := $(shell dirname $(realpath $(firstword $(MAKEFILE_LIST))))
bin_root   = $(repo_root)/bin
bin_dir    = $(bin_root)/$(PLAT)/$(ARCH)
dist_dir   = $(repo_root)/dist

# build and package outputs
clic_bin     = $(bin_dir)/clic$(binary_ext)
clic_tarball = $(dist_dir)/clic-$(VERSION)-$(PLAT)-$(ARCH).tar.gz

# files
source_files     = *.go */*.go */*/*.go
coverage_profile = coverage.out
coverage_report  = coverage.report.out

# versioning
ifndef VERSION
git_rev := $(shell git rev-parse --short HEAD)
git_tag := $(shell git describe --tags --match "v*.*.*" --abbrev=0 HEAD 2>/dev/null)
ifeq ($(git_tag),)
git_tag := v0.0.0
endif
timestamp := $(shell date -u +%Y%m%d%H%M%S)
VERSION   := $(git_tag)-$(git_rev)-$(timestamp)
endif

clean:
	go clean -i -testcache ./...
	rm -rf $(bin_root)
	rm -rf $(dist_dir)
	rm -f $(coverage_profile) $(coverage_report)

build: $(clic_bin)

package: $(clic_tarball)

test: $(coverage_profile)

coverage: $(coverage_repkort)
	cat $(coverage_report)

coverage-html: $(coverage_profile)
	go tool cover -html=$(coverage_profile)

$(clic_bin): $(source_files)
	mkdir -p $(bin_dir)
	GOOS=$(PLAT) GOARCH=$(ARCH) \
	go build \
	-ldflags "-X 'main.Version=$(VERSION)'" \
	-trimpath \
	-o $(clic_bin)

$(clic_tarball): $(clic_bin)
	mkdir -p $(dist_dir)
	tar -czf $(clic_tarball) -C $(bin_dir) .

$(coverage_report): $(coverage_profile)
	go tool cover -func=$(coverage_profile) > $(coverage_report)

$(coverage_profile): $(source_files)
	go test -tags test -coverprofile=$(coverage_profile) -race ./...
