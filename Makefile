all: build

source_files = *.go */*.go */*/*.go
coverage_profile = coverage.out
coverage_report = coverage.report.out
build_bin = tools/compiler/compiler
run_bin = tools/runner/runner
validate_bin = tools/validator/validator

clean:
	cd tools/validate && go clean -i -testcache ./...
	cd tools/run && go clean -i -testcache ./...
	cd tools/build && go clean -i -testcache ./...
	rm -f $(coverage_profile) $(coverage_report)

build: $(validate_bin) $(run_bin) $(build_bin)

test: $(coverage_profile)

coverage: $(coverage_report)
	cat $(coverage_report)

coverage-html: $(coverage_profile)
	go tool cover -html=$(coverage_profile)

$(build_bin): $(source_files)
	cd tools/build && go build

$(run_bin): $(source_files)
	cd tools/run && go build

$(validate_bin): $(source_files)
	cd tools/validate && go build

$(coverage_report): $(coverage_profile)
	go tool cover -func=$(coverage_profile) > $(coverage_report)

$(coverage_profile): $(source_files)
	go test -tags test -coverprofile=$(coverage_profile) -race ./...


