// Package main provides a static analysis tool that combines multiple analyzers
// including standard go vet analyzers, staticcheck analyzers, and custom analyzers.
// The multichecker performs comprehensive static analysis of Go code including:
// - Standard go vet checks
// - All SA-class analyzers from staticcheck
// - Additional staticcheck analyzers
// - Custom analyzers including an exit check analyzer
//
// Usage:
//
//	go run cmd/staticlint/main.go [flags] [packages]
//
// Flags:
//
//	-exitcheck
//	      Enable custom exit check analyzer (enabled by default)
//	-all
//	      Run all available analyzers including optional ones
//
// Examples:
//
//		# Analyze current package
//	 go run ./cmd/staticlint ./...
//		go run cmd/staticlint/main.go .
//
//		# Analyze specific package
//		go run cmd/staticlint/main.go ./...
//
//		# Analyze with specific checks
//		go run cmd/staticlint/main.go -exitcheck ./...
package main

import (
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/multichecker"
	"golang.org/x/tools/go/analysis/passes/assign"
	"golang.org/x/tools/go/analysis/passes/atomic"
	"golang.org/x/tools/go/analysis/passes/bools"
	"golang.org/x/tools/go/analysis/passes/buildtag"
	"golang.org/x/tools/go/analysis/passes/composite"
	"golang.org/x/tools/go/analysis/passes/copylock"
	"golang.org/x/tools/go/analysis/passes/errorsas"
	"golang.org/x/tools/go/analysis/passes/httpresponse"
	"golang.org/x/tools/go/analysis/passes/loopclosure"
	"golang.org/x/tools/go/analysis/passes/lostcancel"
	"golang.org/x/tools/go/analysis/passes/nilfunc"
	"golang.org/x/tools/go/analysis/passes/printf"
	"golang.org/x/tools/go/analysis/passes/shadow"
	"golang.org/x/tools/go/analysis/passes/shift"
	"golang.org/x/tools/go/analysis/passes/stdmethods"
	"golang.org/x/tools/go/analysis/passes/structtag"
	"golang.org/x/tools/go/analysis/passes/tests"
	"golang.org/x/tools/go/analysis/passes/unreachable"
	"golang.org/x/tools/go/analysis/passes/unsafeptr"

	"honnef.co/go/tools/quickfix"
	"honnef.co/go/tools/simple"
	"honnef.co/go/tools/staticcheck"
	"honnef.co/go/tools/stylecheck"

	"github.com/alisaviation/monitoring/cmd/staticlint/exitcheck"
)

func main() {
	analyzers := []*analysis.Analyzer{
		assign.Analyzer,
		atomic.Analyzer,
		bools.Analyzer,
		buildtag.Analyzer,
		composite.Analyzer,
		copylock.Analyzer,
		errorsas.Analyzer,
		httpresponse.Analyzer,
		loopclosure.Analyzer,
		lostcancel.Analyzer,
		nilfunc.Analyzer,
		printf.Analyzer,
		shadow.Analyzer,
		shift.Analyzer,
		stdmethods.Analyzer,
		structtag.Analyzer,
		tests.Analyzer,
		unreachable.Analyzer,
		unsafeptr.Analyzer,
		exitcheck.Analyzer,
	}

	for _, v := range staticcheck.Analyzers {
		if len(v.Analyzer.Name) >= 2 && v.Analyzer.Name[0:2] == "SA" {
			switch v.Analyzer.Name {
			case "SA1000", "SA1001", "SA1012", "SA1019", "SA4000", "SA4001", "SA5000", "SA5001":
				analyzers = append(analyzers, v.Analyzer)
			}
		}
	}

	analyzers = append(analyzers,
		quickfix.Analyzers[0].Analyzer,
		simple.Analyzers[0].Analyzer,
		stylecheck.Analyzers[0].Analyzer,
	)

	multichecker.Main(analyzers...)
}
