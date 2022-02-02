package analyzer_test

import (
	"go/ast"
	"go/token"
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
	Run:      run2,
	Requires: []*analysis.Analyzer{inspect.Analyzer},
}

type blockNode struct {
	count    int
	level    int
	children []blockNode
}

func run2(pass *analysis.Pass) (interface{}, error) {
	insp := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	nodeFilter := []ast.Node{
		(*ast.FuncDecl)(nil),
	}
	insp.Preorder(nodeFilter, func(node ast.Node) {
		v, ok := node.(*ast.FuncDecl)
		if !ok {
			return
		}

		process2(pass, v)
	})
	return nil, nil
}

type scope struct {
	start   token.Pos
	end     token.Pos
	parent  *scope
	flipped bool
	ref     ast.Node
	decl    ast.Node
}

func process2(pass *analysis.Pass, node *ast.FuncDecl) {
	varToSmallestScope := map[string]scope{}
	decl := map[string]ast.Node{}
	currentScope := scope{start: token.NoPos, end: token.NoPos, ref: node}
	for _, line := range node.Body.List {
		updateScope(varToSmallestScope, decl, currentScope, line)
	}

	for k, v := range varToSmallestScope {
		if v.start != token.NoPos && v.end != token.NoPos {
			p := decl[k]
			position := pass.Fset.Position(v.start)
			pass.Reportf(p.Pos(), "variable '%s' is only used in the if-statement (%s); consider using short syntax", k, position)
		}

	}

	//if candidate != nil {
	//	position := pass.Fset.Position(ifStmt.Pos())
	//	pass.Reportf(candidate.Pos(), "variable '%s' is only used in the if-statement (%s); consider using short syntax", candidate.Name, position)
	//}
	//fmt.Println("done")
}

/*
If second encounter and current scope is same, make smallest scope the parent.


*/

func updateScope(m map[string]scope, decl map[string]ast.Node, currentScope scope, node ast.Node) {
	switch v := node.(type) {
	case *ast.Ident:
		if token.IsIdentifier(v.Name) && v.Obj != nil {
			if asst, ok := v.Obj.Decl.(*ast.AssignStmt); ok {
				for _, lh := range asst.Lhs {
					if lh.Pos() == v.Pos() {
						decl[v.Name] = v
						return
					}
				}
			}
			smallest, ok := m[v.Name]
			// First reference to variable since defining
			if !ok {
				m[v.Name] = currentScope
				return
			}

			// Sub scope
			if !currentScope.flipped && currentScope.end < smallest.end && currentScope.start > smallest.start {
				m[v.Name] = currentScope
				return
			}

			// Same scope; set smallest to parent.
			//if currentScope.start == smallest.start && currentScope.end == smallest.end {
			//	var parent scope
			//	if smallest.parent != nil {
			//		parent = *smallest.parent
			//		parent.flipped = true
			//	}
			//	m[v.Name] = parent
			//}

			// Outside scope; set smallest to parent.
			if currentScope.start > smallest.start && currentScope.end > smallest.end {
				var parent scope
				if smallest.parent != nil {
					parent = *smallest.parent
					parent.flipped = true
				}
				m[v.Name] = parent
			}
		}
	//case *ast.FuncDecl:
	//	updateScope(m, ifs, v.Body)
	case *ast.AssignStmt:
		for _, o := range append(v.Lhs, v.Rhs...) {
			updateScope(m, decl, currentScope, o)
		}
	case *ast.BlockStmt:
		for _, line := range v.List {
			updateScope(m, decl, currentScope, line)
		}
	case *ast.IfStmt:
		//updateScope(m, scope{start: v.Pos(), end: v.End(), parent: &currentScope}, v.Cond)
		updateScope(m, decl, currentScope, v.Cond)
		updateScope(m, decl, scope{start: v.Pos(), end: v.End(), parent: &currentScope, ref: v}, v.Body)
	case *ast.BinaryExpr:
		updateScope(m, decl, currentScope, v.X)
		updateScope(m, decl, currentScope, v.Y)
	case *ast.ExprStmt:
		updateScope(m, decl, currentScope, v.X)
	case *ast.CallExpr:
		for _, arg := range v.Args {
			updateScope(m, decl, currentScope, arg)
		}
	case *ast.ReturnStmt:
		for _, expr := range v.Results {
			updateScope(m, decl, currentScope, expr)
		}
	}
}

//func check(block *ast.BlockStmt) {
//	for _, line := range block.List {
//
//	}
//
//}
//
func thing(node blockNode, n ast.Node) {
	switch v := n.(type) {
	case *ast.Ident:
		if decl, ok := v.Obj.Decl.(*ast.AssignStmt); ok {
			for _, lhs := range decl.Lhs {
				if lhs.Pos() == v.Pos() {
					return
				}
			}
		}

	}

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
		if token.IsIdentifier(v.Name) && v.Obj != nil {
			m[v.Name] = append(m[v.Name], v)
		}
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
