package reporting

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strings"
	"testing"
)

// Keep terminal ownership a repository rule for future processors too.
func TestTerminalInteractionsUseSharedPopupAPIs(t *testing.T) {
	headlessFallbacks := map[string]bool{
		"reporting/form.go":  true,
		"system/commands.go": true,
		"system/files.go":    true,
		"system/prompt.go":   true,
	}
	root := filepath.Join("..", "..", "internal")
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			return err
		}
		ast.Inspect(file, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			selector, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			owner, ok := selector.X.(*ast.Ident)
			if !ok {
				return true
			}
			if owner.Name == "tea" && (selector.Sel.Name == "Exec" || selector.Sel.Name == "ExecProcess") {
				t.Errorf("%s: terminal handoff bypasses the popup contract", fset.Position(call.Pos()))
			}
			if owner.Name == "reporting" && selector.Sel.Name == "WithTerminal" && !headlessFallbacks[rel] {
				t.Errorf("%s: use RunForm or RunInteractiveCommand", fset.Position(call.Pos()))
			}
			if owner.Name == "form" && selector.Sel.Name == "Run" && rel != "reporting/form.go" {
				t.Errorf("%s: standalone form bypasses RunForm", fset.Position(call.Pos()))
			}
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
