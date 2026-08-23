package bot

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRepliesAreEmbeds(t *testing.T) {
	files, err := filepath.Glob("*.go")

	if err != nil {
		t.Fatal(err)
	}

	const allowed = "permeditor_apply.go"

	for _, path := range files {
		if strings.HasSuffix(path, "_test.go") || path == allowed {
			continue
		}

		src, err := os.ReadFile(path)

		if err != nil {
			t.Fatal(err)
		}

		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, path, src, 0)

		if err != nil {
			t.Fatalf("parsing %s: %v", path, err)
		}

		ast.Inspect(file, func(n ast.Node) bool {
			lit, ok := n.(*ast.CompositeLit)

			if !ok {
				return true
			}

			sel, ok := lit.Type.(*ast.SelectorExpr)

			if !ok || sel.Sel.Name != "MessageCreate" {
				return true
			}

			for _, elt := range lit.Elts {
				kv, ok := elt.(*ast.KeyValueExpr)

				if !ok {
					continue
				}

				if key, ok := kv.Key.(*ast.Ident); ok && key.Name == "Content" {
					t.Errorf("%s: MessageCreate sets Content — replies must be embeds, use Ctx.Say/Ok/Fail",
						fset.Position(kv.Pos()))
				}
			}

			return true
		})
	}
}
