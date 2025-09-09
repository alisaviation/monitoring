// Package exitcheck provides an analyzer that detects direct calls to os.Exit
// in the main function of the main package.
//
// This analyzer helps ensure proper application shutdown by preventing
// direct calls to os.Exit, which can skip deferred functions and other
// cleanup operations. Instead, applications should return errors from main
// or use proper shutdown mechanisms.
package exitcheck

import (
	"go/ast"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// Analyzer is the exit check analyzer that detects direct os.Exit calls
// in main functions of main packages.
var Analyzer = &analysis.Analyzer{
	Name:     "exitcheck",
	Doc:      "detect direct calls to os.Exit in main function of main package",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

func run(pass *analysis.Pass) (interface{}, error) {
	inspect := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	if pass.Pkg.Name() != "main" {
		return nil, nil
	}

	nodeFilter := []ast.Node{
		(*ast.FuncDecl)(nil),
		(*ast.CallExpr)(nil),
	}

	inspect.WithStack(nodeFilter, func(n ast.Node, push bool, stack []ast.Node) bool {
		if !push {
			return true
		}

		var inMainFunc bool
		for _, node := range stack {
			if fd, ok := node.(*ast.FuncDecl); ok && fd.Name.Name == "main" {
				inMainFunc = true
				break
			}
		}

		if !inMainFunc {
			return true
		}

		callExpr, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}

		selExpr, ok := callExpr.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}

		ident, ok := selExpr.X.(*ast.Ident)
		if !ok {
			return true
		}

		if ident.Name == "os" && selExpr.Sel.Name == "Exit" {
			pass.Reportf(callExpr.Pos(), "direct call to os.Exit in main function is not allowed")
		}

		return true
	})

	return nil, nil
}
