package analyzer_test

import (
	"go/ast"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/analysistest"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

func TestAll(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get wd: %s", err)
	}

	//testdata := filepath.Dir(filepath.Dir(wd)) + "/testdata/test"

	testdata := filepath.Join(filepath.Dir(filepath.Dir(wd)), "testdata")
	//analysistest.Run(t, testdata, analyzer.Analyzer)

	analysistest.Run(t, testdata, Analyzer)
}

var Analyzer = &analysis.Analyzer{
	Name:     "ifshort",
	Doc:      "Checks that your code uses short syntax for if-statements whenever possible.",
	Run:      run,
	Requires: []*analysis.Analyzer{inspect.Analyzer},
}

func run(pass *analysis.Pass) (interface{}, error) {
	insp := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	nodeFilter := []ast.Node{
		(*ast.FuncDecl)(nil),
	}

	insp.Preorder(nodeFilter, func(node ast.Node) {
		v, ok := node.(*ast.FuncDecl)
		if !ok {
			return
		}

		process(pass, v)
	})


	return nil, nil
}

func process(pass *analysis.Pass, node *ast.FuncDecl) {
	varNameToRefs := map[string][]*ast.Ident{}

	var ifs []*ast.IfStmt
	updateVarRefs(varNameToRefs, &ifs, node)
	for _, ifStmt := range ifs {
		candidate := checkCondition(ifStmt, ifStmt.Cond, varNameToRefs)
		if candidate != nil {
			position := pass.Fset.Position(ifStmt.Pos())
			pass.Reportf(candidate.Pos(), "variable '%s' is only used in the if-statement (%s); consider using short syntax", candidate.Name, position)
		}
	}
}

func checkCondition(ifStmt *ast.IfStmt, node ast.Node, varNameToRefs map[string][]*ast.Ident) *ast.Ident {
	switch n := node.(type) {
	case *ast.Ident:
		refs, ok := varNameToRefs[n.Name]
		if !ok {
			// Declared outside of function?
			return nil
		}
		beforeCount := 0
		for _, ref := range refs {
			if ref.Pos() < ifStmt.Pos() {
				beforeCount++
			} else if ref.Pos() > ifStmt.End() {
				// Variable is referenced outside of if statement.
				return nil
			}
		}
		// Only one reference before condition: the initial assignment of the variable.
		if beforeCount == 1 {
			return refs[0]
		}
		return nil
	case *ast.BinaryExpr:
		if res := checkCondition(ifStmt, n.X, varNameToRefs); res != nil {
			return res
		}
		return checkCondition(ifStmt, n.Y, varNameToRefs)
	case *ast.CallExpr:
		for _, arg := range n.Args {
			if res := checkCondition(ifStmt, arg, varNameToRefs); res != nil {
				return res
			}
		}
	}
	return nil
}

func updateVarRefs(m map[string][]*ast.Ident, ifs *[]*ast.IfStmt, node ast.Node) {
	switch v := node.(type) {
	case *ast.Ident:
		m[v.Name] = append(m[v.Name], v)
	case *ast.FuncDecl:
		updateVarRefs(m, ifs, v.Body)
	case *ast.AssignStmt:
		for _, o := range append(v.Lhs, v.Rhs...) {
			updateVarRefs(m, ifs, o)
		}
	case *ast.BlockStmt:
		for _, line := range v.List {
			updateVarRefs(m, ifs, line)
		}
	case *ast.IfStmt:
		updateVarRefs(m, ifs, v.Cond)
		updateVarRefs(m, ifs, v.Body)
		*ifs = append(*ifs, v)
	case *ast.BinaryExpr:
		updateVarRefs(m, ifs, v.X)
		updateVarRefs(m, ifs, v.Y)
	case *ast.ExprStmt:
		updateVarRefs(m, ifs, v.X)
	case *ast.CallExpr:
		for _, arg := range v.Args {
			updateVarRefs(m, ifs, arg)
		}
	case *ast.ReturnStmt:
		for _, expr := range v.Results {
			updateVarRefs(m, ifs, expr)
		}
	}
}
