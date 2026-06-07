package main

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/jefflinse/clic"
	"github.com/jefflinse/clic/provider"
	"github.com/jefflinse/clic/spec"
	"github.com/spf13/cobra"
)

func testCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "test <suite>",
		Short: "run a contract-test suite against an API",
		Long: "Run a suite of named request cases against an API, asserting each " +
			"response's status, conformance to the OpenAPI schema, and optional " +
			"gojq body assertions. Exits non-zero if any case fails.",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTests(cmd, args[0])
		},
	}

	cmd.Flags().Bool("json", false, "emit a machine-readable JSON report")
	cmd.Flags().String("junit", "", "write a JUnit XML report to the given file")
	cmd.Flags().String("spec", "", "override the spec referenced by the suite")
	return cmd
}

// caseResult is the outcome of running a single test case.
type caseResult struct {
	Name     string   `json:"name"`
	Status   int      `json:"status"`
	Failures []string `json:"failures,omitempty"`
}

func (cr *caseResult) fail(msg string) { cr.Failures = append(cr.Failures, msg) }
func (cr *caseResult) passed() bool    { return len(cr.Failures) == 0 }

func runTests(cmd *cobra.Command, suitePath string) error {
	suite, err := loadTestSuite(suitePath)
	if err != nil {
		return err
	}

	specRef := suite.Spec
	if override, _ := cmd.Flags().GetString("spec"); override != "" {
		specRef = override
	}
	if specRef == "" {
		return fmt.Errorf("no spec given: set 'spec:' in the suite or pass --spec")
	}

	appSpec, err := clic.LoadSpec(resolveLocation(specRef), spec.FormatUnknown)
	if err != nil {
		return err
	}

	opts := provider.ResolveOptions(cmd.Flags())
	if opts.Server == "" {
		opts.Server = suite.Server
	}
	if err := resolveOAuth(cmd.Context(), appSpec.Auth, opts); err != nil {
		return err
	}

	results := make([]caseResult, 0, len(suite.Cases))
	for _, c := range suite.Cases {
		results = append(results, runCase(cmd.Context(), appSpec, opts, c))
	}

	if path, _ := cmd.Flags().GetString("junit"); path != "" {
		if err := writeJUnit(path, results); err != nil {
			return err
		}
	}

	if jsonOut, _ := cmd.Flags().GetBool("json"); jsonOut {
		reportJSON(results)
	} else {
		reportText(results)
	}

	for _, r := range results {
		if !r.passed() {
			os.Exit(1)
		}
	}
	return nil
}

// runCase builds a fresh app (so each case gets clean flag state), runs the
// command, captures the structured result via a sink, and evaluates the case's
// expectations against it.
func runCase(base context.Context, appSpec *spec.App, opts *provider.Options, c testCase) caseResult {
	cr := caseResult{Name: c.Name}

	app, err := clic.NewAppFromSpec(appSpec)
	if err != nil {
		cr.fail("build app: " + err.Error())
		return cr
	}

	args, err := shellwords(c.Cmd)
	if err != nil {
		cr.fail("parse cmd: " + err.Error())
		return cr
	}

	sink := &provider.ResultSink{}
	ctx := provider.WithResultSink(provider.WithOptions(base, opts), sink)
	if err := app.RunContext(ctx, args); err != nil {
		cr.fail("run: " + err.Error())
		return cr
	}
	if sink.Result == nil {
		cr.fail("command produced no HTTP result")
		return cr
	}

	evaluateCase(&cr, c.Expect, sink.Result)
	return cr
}

// evaluateCase checks a response against a case's expectations, recording a
// failure message for each unmet expectation.
func evaluateCase(cr *caseResult, e expectation, res *provider.Result) {
	cr.Status = res.Status

	if want := wantStatuses(e.Status); len(want) > 0 && !slices.Contains(want, res.Status) {
		cr.fail(fmt.Sprintf("status: got %d, want %v", res.Status, want))
	}

	evaluateContract(cr, e.Contract, res)

	for _, a := range e.Assert {
		if err := evalAssertion(a, res.Body); err != nil {
			cr.fail(fmt.Sprintf("assert %s: %v", a.JQ, err))
		}
	}
}

// evaluateContract applies the contract expectation: skipped when explicitly
// false; otherwise any violations fail the case. When explicitly true, the
// absence of a declared schema is itself a failure.
func evaluateContract(cr *caseResult, want *bool, res *provider.Result) {
	if want != nil && !*want {
		return
	}
	explicit := want != nil && *want

	if res.Contract == nil || !res.Contract.Checked {
		if explicit {
			cr.fail("contract: no response schema to validate against")
		}
		return
	}
	for _, v := range res.Contract.Violations {
		cr.fail("contract: " + v)
	}
}

func reportText(results []caseResult) {
	passed := 0
	for _, r := range results {
		if r.passed() {
			passed++
			fmt.Printf("✓ %s (%d)\n", r.Name, r.Status)
			continue
		}
		fmt.Printf("✗ %s (%d)\n", r.Name, r.Status)
		for _, f := range r.Failures {
			fmt.Printf("    %s\n", f)
		}
	}
	fmt.Printf("\n%d passed, %d failed\n", passed, len(results)-passed)
}

func reportJSON(results []caseResult) {
	passed := 0
	for _, r := range results {
		if r.passed() {
			passed++
		}
	}
	report := struct {
		Total  int          `json:"total"`
		Passed int          `json:"passed"`
		Failed int          `json:"failed"`
		Cases  []caseResult `json:"cases"`
	}{
		Total:  len(results),
		Passed: passed,
		Failed: len(results) - passed,
		Cases:  results,
	}
	b, _ := json.MarshalIndent(report, "", "  ")
	fmt.Println(string(b))
}

// writeJUnit writes the results as a JUnit XML report for CI consumption.
func writeJUnit(path string, results []caseResult) error {
	type failure struct {
		Message string `xml:"message,attr"`
	}
	type tc struct {
		Name    string   `xml:"name,attr"`
		Failure *failure `xml:"failure,omitempty"`
	}
	type ts struct {
		XMLName  xml.Name `xml:"testsuite"`
		Name     string   `xml:"name,attr"`
		Tests    int      `xml:"tests,attr"`
		Failures int      `xml:"failures,attr"`
		Cases    []tc     `xml:"testcase"`
	}

	suite := ts{Name: "clic", Tests: len(results)}
	for _, r := range results {
		c := tc{Name: r.Name}
		if !r.passed() {
			suite.Failures++
			c.Failure = &failure{Message: strings.Join(r.Failures, "; ")}
		}
		suite.Cases = append(suite.Cases, c)
	}

	out, err := xml.MarshalIndent(suite, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append([]byte(xml.Header), out...), 0o644)
}
