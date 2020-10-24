all: build

source_files = *.go */*.go */*/*.go
coverage_profile = coverage.out
coverage_report = coverage.report.out
hm_bim = hm/hm
build_bin = tools/build/build
registry_bin = tools/registry/registry
validate_bin = tools/validate/validate

clean:
	cd hm && go clean -i -testcache ./...
	cd tools/build && go clean -i -testcache ./...
	cd tools/registry && go clean -i -testcache ./...
	cd tools/validate && go clean -i -testcache ./...
	rm -f $(coverage_profile) $(coverage_report)

build: $(build_bin) $(registry_bin) $(run_bin) $(validate_bin)

test: $(coverage_profile)

coverage: $(coverage_report)
	cat $(coverage_report)

coverage-html: $(coverage_profile)
	go tool cover -html=$(coverage_profile)

$(build_bin): $(source_files)
	cd tools/build && go build

$(registry_bin): $(source_files)
	cd tools/registry && go build

$(validate_bin): $(source_files)
	cd tools/validate && go build

$(coverage_report): $(coverage_profile)
	go tool cover -func=$(coverage_profile) > $(coverage_report)

$(coverage_profile): $(source_files)
	go test -tags test -coverprofile=$(coverage_profile) -race ./...


