all: build

source_files = *.go */*.go */*/*.go
coverage_profile = coverage.out
coverage_report = coverage.report.out
hm_bin = hm/hm

clean:
	cd hm && go clean -i -testcache ./...
	rm -f $(coverage_profile) $(coverage_report)

build: $(hm_bin)

test: $(coverage_profile)

coverage: $(coverage_report)
	cat $(coverage_report)

coverage-html: $(coverage_profile)
	go tool cover -html=$(coverage_profile)

$(hm_bin): $(source_files)
	cd hm && go build

$(coverage_report): $(coverage_profile)
	go tool cover -func=$(coverage_profile) > $(coverage_report)

$(coverage_profile): $(source_files)
	go test -tags test -coverprofile=$(coverage_profile) -race ./...


