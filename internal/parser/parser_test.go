package parser

import (
	"go/ast"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ── shouldSkipDir ─────────────────────────────────────────────────────────────

func TestShouldSkipDir(t *testing.T) {
	cases := []struct {
		name string
		want bool
	}{
		{".git", true},
		{".hidden", true},
		{"vendor", true},
		{"testdata", true},
		{"src", false},
		{"internal", false},
		{"pkg", false},
		{"", false},
	}
	for _, tc := range cases {
		if got := shouldSkipDir(tc.name); got != tc.want {
			t.Errorf("shouldSkipDir(%q) = %v, want %v", tc.name, got, tc.want)
		}
	}
}

// ── isGoSourceFile ────────────────────────────────────────────────────────────

func TestIsGoSourceFile(t *testing.T) {
	cases := []struct {
		name string
		want bool
	}{
		{"parser.go", true},
		{"model.go", true},
		{"parser_test.go", false},
		{"model_test.go", false},
		{"go.mod", false},
		{"README.md", false},
		{".go", true},
		{"", false},
	}
	for _, tc := range cases {
		if got := isGoSourceFile(tc.name); got != tc.want {
			t.Errorf("isGoSourceFile(%q) = %v, want %v", tc.name, got, tc.want)
		}
	}
}

// ── lastPathSegment ───────────────────────────────────────────────────────────

func TestLastPathSegment(t *testing.T) {
	cases := []struct {
		path string
		want string
	}{
		{"fmt", "fmt"},
		{"encoding/json", "json"},
		{"github.com/example/models", "models"},
		{"a/b/c/d", "d"},
		{"", ""},
	}
	for _, tc := range cases {
		if got := lastPathSegment(tc.path); got != tc.want {
			t.Errorf("lastPathSegment(%q) = %q, want %q", tc.path, got, tc.want)
		}
	}
}

// ── mergeComments ─────────────────────────────────────────────────────────────

func TestMergeComments(t *testing.T) {
	makeGroup := func(texts ...string) *ast.CommentGroup {
		var list []*ast.Comment
		for _, text := range texts {
			list = append(list, &ast.Comment{Text: text})
		}
		return &ast.CommentGroup{List: list}
	}

	t.Run("nil groups returns empty", func(t *testing.T) {
		got := mergeComments(nil, nil)
		if len(got) != 0 {
			t.Errorf("want empty, got %v", got)
		}
	})

	t.Run("line comment strips //", func(t *testing.T) {
		got := mergeComments(makeGroup("// hello world"))
		if len(got) != 1 || got[0] != "hello world" {
			t.Errorf("got %v", got)
		}
	})

	t.Run("block comment strips /* */", func(t *testing.T) {
		got := mergeComments(makeGroup("/* block comment */"))
		if len(got) != 1 || got[0] != "block comment" {
			t.Errorf("got %v", got)
		}
	})

	t.Run("multiple groups merged in order", func(t *testing.T) {
		got := mergeComments(makeGroup("// first"), makeGroup("// second"))
		if len(got) != 2 || got[0] != "first" || got[1] != "second" {
			t.Errorf("got %v", got)
		}
	})

	t.Run("whitespace-only comment is skipped", func(t *testing.T) {
		got := mergeComments(makeGroup("//   "))
		if len(got) != 0 {
			t.Errorf("expected empty, got %v", got)
		}
	})

	t.Run("multiple comments in one group", func(t *testing.T) {
		got := mergeComments(makeGroup("// line1", "// line2", "// line3"))
		if len(got) != 3 {
			t.Fatalf("want 3 lines, got %v", got)
		}
		if got[0] != "line1" || got[1] != "line2" || got[2] != "line3" {
			t.Errorf("got %v", got)
		}
	})
}

// ── typeExprToString ──────────────────────────────────────────────────────────

func TestTypeExprToString(t *testing.T) {
	cases := []struct {
		name string
		expr ast.Expr
		want string
	}{
		{
			"nil",
			nil,
			"",
		},
		{
			"ident",
			&ast.Ident{Name: "string"},
			"string",
		},
		{
			"pointer",
			&ast.StarExpr{X: &ast.Ident{Name: "string"}},
			"*string",
		},
		{
			"double pointer",
			&ast.StarExpr{X: &ast.StarExpr{X: &ast.Ident{Name: "string"}}},
			"**string",
		},
		{
			"slice",
			&ast.ArrayType{Elt: &ast.Ident{Name: "int"}},
			"[]int",
		},
		{
			"fixed array treated as slice",
			&ast.ArrayType{
				Len: &ast.BasicLit{Kind: token.INT, Value: "3"},
				Elt: &ast.Ident{Name: "string"},
			},
			"[]string",
		},
		{
			"map",
			&ast.MapType{
				Key:   &ast.Ident{Name: "string"},
				Value: &ast.Ident{Name: "int"},
			},
			"map[string]int",
		},
		{
			"selector expr",
			&ast.SelectorExpr{
				X:   &ast.Ident{Name: "time"},
				Sel: &ast.Ident{Name: "Time"},
			},
			"time.Time",
		},
		{
			"interface{}",
			&ast.InterfaceType{Methods: &ast.FieldList{}},
			"interface{}",
		},
		{
			"anonymous struct{}",
			&ast.StructType{Fields: &ast.FieldList{}},
			"struct{}",
		},
		{
			"func type",
			&ast.FuncType{},
			"func",
		},
		{
			"chan type",
			&ast.ChanType{Value: &ast.Ident{Name: "int"}},
			"chan",
		},
		{
			"ellipsis",
			&ast.Ellipsis{Elt: &ast.Ident{Name: "int"}},
			"...int",
		},
		{
			"paren expr",
			&ast.ParenExpr{X: &ast.Ident{Name: "int"}},
			"(int)",
		},
		{
			"basic lit",
			&ast.BasicLit{Kind: token.INT, Value: "42"},
			"42",
		},
		{
			"binary expr (union constraint)",
			&ast.BinaryExpr{
				X:  &ast.Ident{Name: "int"},
				Op: token.OR,
				Y:  &ast.Ident{Name: "string"},
			},
			"int|string",
		},
		{
			"generic single arg",
			&ast.IndexExpr{
				X:     &ast.Ident{Name: "Pair"},
				Index: &ast.Ident{Name: "string"},
			},
			"Pair[string]",
		},
		{
			"generic multi args",
			&ast.IndexListExpr{
				X:       &ast.Ident{Name: "KV"},
				Indices: []ast.Expr{&ast.Ident{Name: "string"}, &ast.Ident{Name: "int"}},
			},
			"KV[string,int]",
		},
		{
			"nested slice of map",
			&ast.ArrayType{
				Elt: &ast.MapType{
					Key:   &ast.Ident{Name: "string"},
					Value: &ast.Ident{Name: "int"},
				},
			},
			"[]map[string]int",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := typeExprToString(tc.expr)
			if got != tc.want {
				t.Errorf("typeExprToString(%T) = %q, want %q", tc.expr, got, tc.want)
			}
		})
	}
}

// ── embeddedFieldName ─────────────────────────────────────────────────────────

func TestEmbeddedFieldName(t *testing.T) {
	cases := []struct {
		name string
		expr ast.Expr
		want string
	}{
		{
			"ident",
			&ast.Ident{Name: "BaseModel"},
			"BaseModel",
		},
		{
			"selector",
			&ast.SelectorExpr{X: &ast.Ident{Name: "pkg"}, Sel: &ast.Ident{Name: "Base"}},
			"Base",
		},
		{
			"star ident",
			&ast.StarExpr{X: &ast.Ident{Name: "Auditable"}},
			"Auditable",
		},
		{
			"star selector",
			&ast.StarExpr{
				X: &ast.SelectorExpr{X: &ast.Ident{Name: "pkg"}, Sel: &ast.Ident{Name: "Base"}},
			},
			"Base",
		},
		{
			"unknown type returns empty",
			&ast.BasicLit{Value: "x"},
			"",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := embeddedFieldName(tc.expr)
			if got != tc.want {
				t.Errorf("embeddedFieldName(%T) = %q, want %q", tc.expr, got, tc.want)
			}
		})
	}
}

// ── constValueToString ────────────────────────────────────────────────────────

func TestConstValueToString(t *testing.T) {
	cases := []struct {
		name    string
		expr    ast.Expr
		iotaIdx int
		want    string
	}{
		{"nil iota 0", nil, 0, "0"},
		{"nil iota 3", nil, 3, "3"},
		{
			"string lit",
			&ast.BasicLit{Kind: token.STRING, Value: `"active"`},
			0, "active",
		},
		{
			"int lit",
			&ast.BasicLit{Kind: token.INT, Value: "42"},
			0, "42",
		},
		{
			"iota ident at index 2",
			&ast.Ident{Name: "iota"},
			2, "2",
		},
		{
			"non-iota ident",
			&ast.Ident{Name: "SomeConst"},
			0, "SomeConst",
		},
		{
			"unary minus",
			&ast.UnaryExpr{
				Op: token.SUB,
				X:  &ast.BasicLit{Kind: token.INT, Value: "1"},
			},
			0, "-1",
		},
		{
			"binary left shift",
			&ast.BinaryExpr{
				X:  &ast.BasicLit{Kind: token.INT, Value: "1"},
				Op: token.SHL,
				Y:  &ast.Ident{Name: "iota"},
			},
			2, "1<<2",
		},
		{
			"call expr",
			&ast.CallExpr{
				Fun:  &ast.Ident{Name: "Status"},
				Args: []ast.Expr{&ast.BasicLit{Kind: token.STRING, Value: `"ok"`}},
			},
			0, "Status(...)",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := constValueToString(tc.expr, tc.iotaIdx)
			if got != tc.want {
				t.Errorf("constValueToString() = %q, want %q", got, tc.want)
			}
		})
	}
}

// ── integration: parse testdata/types ────────────────────────────────────────

func TestGoParser_Parse_Types(t *testing.T) {
	p := &GoParser{MaxDepth: -1}
	files, err := p.Parse([]string{"../../testdata/types"})
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}
	rf := files[0]

	if rf.Package != "types" {
		t.Errorf("Package = %q, want %q", rf.Package, "types")
	}

	t.Run("imports", func(t *testing.T) {
		byPath := make(map[string]RawImport)
		for _, imp := range rf.Imports {
			byPath[imp.Path] = imp
		}

		for _, path := range []string{"cmp", "context", "time", "net/url"} {
			if _, ok := byPath[path]; !ok {
				t.Errorf("missing import %q", path)
			}
		}

		jsonImp, ok := byPath["encoding/json"]
		if !ok {
			t.Fatal("missing encoding/json import")
		}
		if jsonImp.Alias != "alias" {
			t.Errorf("encoding/json alias = %q, want %q", jsonImp.Alias, "alias")
		}
		if jsonImp.PkgName != "json" {
			t.Errorf("encoding/json PkgName = %q, want %q", jsonImp.PkgName, "json")
		}
	})

	t.Run("structs", func(t *testing.T) {
		byName := make(map[string]RawStruct)
		for _, s := range rf.Structs {
			byName[s.Name] = s
		}

		// Basic struct with all primitive fields
		prim, ok := byName["Primitives"]
		if !ok {
			t.Fatal("missing Primitives struct")
		}
		if len(prim.Fields) == 0 {
			t.Error("Primitives: no fields")
		}
		if prim.Fields[0].Name != "B" || prim.Fields[0].TypeName != "bool" {
			t.Errorf("Primitives.Fields[0] = {%q, %q}, want {B, bool}",
				prim.Fields[0].Name, prim.Fields[0].TypeName)
		}
		if len(prim.Comments) == 0 || !strings.Contains(prim.Comments[0], "Primitives") {
			t.Errorf("Primitives.Comments = %v, expected comment about Primitives", prim.Comments)
		}

		// Embedded fields (anonymous fields)
		emb, ok := byName["Embedded"]
		if !ok {
			t.Fatal("missing Embedded struct")
		}
		fieldByName := make(map[string]RawField)
		for _, f := range emb.Fields {
			fieldByName[f.Name] = f
		}
		bm, ok := fieldByName["BaseModel"]
		if !ok {
			t.Fatal("Embedded missing BaseModel embedded field")
		}
		if !bm.Embedded {
			t.Error("BaseModel: Embedded should be true")
		}
		aud, ok := fieldByName["Auditable"]
		if !ok {
			t.Fatal("Embedded missing Auditable embedded field")
		}
		if !aud.Embedded {
			t.Error("Auditable: Embedded should be true")
		}
		if aud.TypeName != "*Auditable" {
			t.Errorf("Auditable.TypeName = %q, want %q", aud.TypeName, "*Auditable")
		}

		// Multi-name field declaration: X, Y, Z float64 → 3 fields
		mn, ok := byName["MultiName"]
		if !ok {
			t.Fatal("missing MultiName struct")
		}
		if len(mn.Fields) != 7 { // X Y Z + Min Max + A B
			t.Errorf("MultiName fields = %d, want 7", len(mn.Fields))
		}

		// Generic struct: single type parameter
		pair, ok := byName["Pair"]
		if !ok {
			t.Fatal("missing Pair struct")
		}
		if len(pair.TypeParams) != 1 {
			t.Fatalf("Pair TypeParams = %d, want 1", len(pair.TypeParams))
		}
		if pair.TypeParams[0].Name != "T" || pair.TypeParams[0].Constraint != "any" {
			t.Errorf("Pair TypeParams[0] = %+v, want {T, any}", pair.TypeParams[0])
		}

		// Generic struct: two type parameters
		kv, ok := byName["KV"]
		if !ok {
			t.Fatal("missing KV struct")
		}
		if len(kv.TypeParams) != 2 {
			t.Fatalf("KV TypeParams = %d, want 2", len(kv.TypeParams))
		}
		if kv.TypeParams[0].Name != "K" || kv.TypeParams[1].Name != "V" {
			t.Errorf("KV TypeParams = %+v", kv.TypeParams)
		}

		// Generic struct: external-package constraint (cmp.Ordered)
		grayList, ok := byName["GrayList"]
		if !ok {
			t.Fatal("missing GrayList struct")
		}
		if len(grayList.TypeParams) != 1 {
			t.Fatalf("GrayList TypeParams = %d, want 1", len(grayList.TypeParams))
		}
		if grayList.TypeParams[0].Name != "T" || grayList.TypeParams[0].Constraint != "cmp.Ordered" {
			t.Errorf("GrayList TypeParams[0] = %+v, want {T, cmp.Ordered}", grayList.TypeParams[0])
		}
		glFields := make(map[string]RawField)
		for _, f := range grayList.Fields {
			glFields[f.Name] = f
		}
		if bl, ok := glFields["Blacklist"]; !ok {
			t.Error("GrayList missing Blacklist field")
		} else if bl.TypeName != "[]T" {
			t.Errorf("GrayList.Blacklist.TypeName = %q, want %q", bl.TypeName, "[]T")
		}
		if wl, ok := glFields["Whitelist"]; !ok {
			t.Error("GrayList missing Whitelist field")
		} else if wl.TypeName != "[]T" {
			t.Errorf("GrayList.Whitelist.TypeName = %q, want %q", wl.TypeName, "[]T")
		}

		// Instantiated generic fields: GrayList[string] and GrayList[int]
		ivConf, ok := byName["InitVersionGrayScaleConfig"]
		if !ok {
			t.Fatal("missing InitVersionGrayScaleConfig struct")
		}
		ivFields := make(map[string]RawField)
		for _, f := range ivConf.Fields {
			ivFields[f.Name] = f
		}
		if ua, ok := ivFields["UserAlias"]; !ok {
			t.Error("InitVersionGrayScaleConfig missing UserAlias field")
		} else if ua.TypeName != "GrayList[string]" {
			t.Errorf("UserAlias.TypeName = %q, want %q", ua.TypeName, "GrayList[string]")
		}
		if did, ok := ivFields["DepartID"]; !ok {
			t.Error("InitVersionGrayScaleConfig missing DepartID field")
		} else if did.TypeName != "GrayList[int]" {
			t.Errorf("DepartID.TypeName = %q, want %q", did.TypeName, "GrayList[int]")
		}

		// Nested struct wrapping generic instantiation
		gsConf, ok := byName["GrayscaleConf"]
		if !ok {
			t.Fatal("missing GrayscaleConf struct")
		}
		if len(gsConf.Fields) != 1 {
			t.Fatalf("GrayscaleConf fields = %d, want 1", len(gsConf.Fields))
		}
		if gsConf.Fields[0].Name != "InitVersionConf" || gsConf.Fields[0].TypeName != "InitVersionGrayScaleConfig" {
			t.Errorf("GrayscaleConf.Fields[0] = {%q, %q}, want {InitVersionConf, InitVersionGrayScaleConfig}",
				gsConf.Fields[0].Name, gsConf.Fields[0].TypeName)
		}

		// Anonymous struct slice fields ([]struct{...})
		was, ok := byName["WithAnonSlice"]
		if !ok {
			t.Fatal("missing WithAnonSlice struct")
		}
		wasFields := make(map[string]RawField)
		for _, f := range was.Fields {
			wasFields[f.Name] = f
		}
		if f, ok := wasFields["ComputingArchitecture"]; !ok {
			t.Error("WithAnonSlice missing ComputingArchitecture field")
		} else if f.TypeName != "[]WithAnonSlice_ComputingArchitecture" {
			t.Errorf("ComputingArchitecture.TypeName = %q, want %q", f.TypeName, "[]WithAnonSlice_ComputingArchitecture")
		}
		if f, ok := wasFields["CustomizedFamily"]; !ok {
			t.Error("WithAnonSlice missing CustomizedFamily field")
		} else if f.TypeName != "[]WithAnonSlice_CustomizedFamily" {
			t.Errorf("CustomizedFamily.TypeName = %q, want %q", f.TypeName, "[]WithAnonSlice_CustomizedFamily")
		}
		// Synthetic structs must exist and carry their fields
		for _, synName := range []string{"WithAnonSlice_ComputingArchitecture", "WithAnonSlice_CustomizedFamily"} {
			syn, ok := byName[synName]
			if !ok {
				t.Errorf("missing synthetic struct %q", synName)
				continue
			}
			if len(syn.Fields) != 2 {
				t.Errorf("%s: want 2 fields, got %d", synName, len(syn.Fields))
			}
		}

		// Anonymous struct pointer field (*struct{...})
		wap, ok := byName["WithAnonPtr"]
		if !ok {
			t.Fatal("missing WithAnonPtr struct")
		}
		wapFields := make(map[string]RawField)
		for _, f := range wap.Fields {
			wapFields[f.Name] = f
		}
		if f, ok := wapFields["Header"]; !ok {
			t.Error("WithAnonPtr missing Header field")
		} else if f.TypeName != "*WithAnonPtr_Header" {
			t.Errorf("Header.TypeName = %q, want %q", f.TypeName, "*WithAnonPtr_Header")
		}
		if syn, ok := byName["WithAnonPtr_Header"]; !ok {
			t.Error("missing synthetic struct WithAnonPtr_Header")
		} else if len(syn.Fields) != 2 {
			t.Errorf("WithAnonPtr_Header: want 2 fields, got %d", len(syn.Fields))
		}

		// Nested anonymous struct (anon struct containing []struct{...})
		wna, ok := byName["WithNestedAnon"]
		if !ok {
			t.Fatal("missing WithNestedAnon struct")
		}
		wnaFields := make(map[string]RawField)
		for _, f := range wna.Fields {
			wnaFields[f.Name] = f
		}
		if f, ok := wnaFields["Result"]; !ok {
			t.Error("WithNestedAnon missing Result field")
		} else if f.TypeName != "WithNestedAnon_Result" {
			t.Errorf("Result.TypeName = %q, want %q", f.TypeName, "WithNestedAnon_Result")
		}
		// WithNestedAnon_Result must exist and have an Items field pointing to the inner synthetic struct
		wnaResult, ok := byName["WithNestedAnon_Result"]
		if !ok {
			t.Fatal("missing synthetic struct WithNestedAnon_Result")
		}
		wnaResultFields := make(map[string]RawField)
		for _, f := range wnaResult.Fields {
			wnaResultFields[f.Name] = f
		}
		if f, ok := wnaResultFields["Items"]; !ok {
			t.Error("WithNestedAnon_Result missing Items field")
		} else if f.TypeName != "[]WithNestedAnon_Result_Items" {
			t.Errorf("Items.TypeName = %q, want %q", f.TypeName, "[]WithNestedAnon_Result_Items")
		}
		if syn, ok := byName["WithNestedAnon_Result_Items"]; !ok {
			t.Error("missing synthetic struct WithNestedAnon_Result_Items")
		} else if len(syn.Fields) != 2 {
			t.Errorf("WithNestedAnon_Result_Items: want 2 fields, got %d", len(syn.Fields))
		}

		// Struct tags
		tv, ok := byName["TagVariants"]
		if !ok {
			t.Fatal("missing TagVariants struct")
		}
		tvFields := make(map[string]RawField)
		for _, f := range tv.Fields {
			tvFields[f.Name] = f
		}
		if exported, ok := tvFields["Exported"]; !ok {
			t.Error("missing Exported field")
		} else if exported.Tag != `json:"exported"` {
			t.Errorf("Exported.Tag = %q, want %q", exported.Tag, `json:"exported"`)
		}
		if noTag, ok := tvFields["NoTag"]; !ok {
			t.Error("missing NoTag field")
		} else if noTag.Tag != "" {
			t.Errorf("NoTag.Tag = %q, want empty", noTag.Tag)
		}
	})

	t.Run("type aliases", func(t *testing.T) {
		byName := make(map[string]RawTypeAlias)
		for _, a := range rf.TypeAliases {
			byName[a.Name] = a
		}

		cases := []struct{ name, typeName string }{
			{"StringAlias", "string"},
			{"IntAlias", "int"},
			{"TimeAlias", "time.Time"},
			{"PrimitivesAlias", "Primitives"},
		}
		for _, tc := range cases {
			a, ok := byName[tc.name]
			if !ok {
				t.Errorf("missing type alias %q", tc.name)
				continue
			}
			if a.TypeName != tc.typeName {
				t.Errorf("alias %q TypeName = %q, want %q", tc.name, a.TypeName, tc.typeName)
			}
		}
	})

	t.Run("consts", func(t *testing.T) {
		byName := make(map[string]RawConst)
		for _, c := range rf.Consts {
			byName[c.Name] = c
		}

		// String enum
		for _, tc := range []struct{ name, typeName, value string }{
			{"StatusActive", "Status", "active"},
			{"StatusInactive", "Status", "inactive"},
			{"StatusDeleted", "Status", "deleted"},
		} {
			c, ok := byName[tc.name]
			if !ok {
				t.Errorf("missing const %q", tc.name)
				continue
			}
			if c.TypeName != tc.typeName || c.Value != tc.value {
				t.Errorf("const %q = {%q, %q}, want {%q, %q}",
					tc.name, c.TypeName, c.Value, tc.typeName, tc.value)
			}
		}

		// Int iota enum (DirectionNorth=0, DirectionSouth=1)
		if north, ok := byName["DirectionNorth"]; !ok {
			t.Error("missing DirectionNorth")
		} else if north.Value != "0" {
			t.Errorf("DirectionNorth.Value = %q, want %q", north.Value, "0")
		}
		if south, ok := byName["DirectionSouth"]; !ok {
			t.Error("missing DirectionSouth")
		} else if south.Value != "1" {
			t.Errorf("DirectionSouth.Value = %q, want %q", south.Value, "1")
		}

		// All four direction consts must be present
		for _, name := range []string{"DirectionNorth", "DirectionSouth", "DirectionEast", "DirectionWest"} {
			if _, ok := byName[name]; !ok {
				t.Errorf("missing const %q", name)
			}
		}

		// Priority consts (bit shift iota)
		for _, name := range []string{"PriorityLow", "PriorityMedium", "PriorityHigh", "PriorityCritical"} {
			if _, ok := byName[name]; !ok {
				t.Errorf("missing const %q", name)
			}
		}
	})

	t.Run("functions", func(t *testing.T) {
		byName := make(map[string]RawFunc)
		for _, fn := range rf.Functions {
			byName[fn.Name] = fn
		}

		// Plain(): no params, no results, no receiver
		plain, ok := byName["Plain"]
		if !ok {
			t.Fatal("missing Plain function")
		}
		if len(plain.Params) != 0 || len(plain.Results) != 0 {
			t.Errorf("Plain: params=%d results=%d, want 0/0", len(plain.Params), len(plain.Results))
		}
		if plain.Receiver != "" {
			t.Errorf("Plain.Receiver = %q, want empty", plain.Receiver)
		}

		// MultiReturn(a int, b string) (int, error)
		mr, ok := byName["MultiReturn"]
		if !ok {
			t.Fatal("missing MultiReturn")
		}
		if len(mr.Params) != 2 {
			t.Errorf("MultiReturn params = %d, want 2", len(mr.Params))
		}
		if len(mr.Results) != 2 {
			t.Errorf("MultiReturn results = %d, want 2", len(mr.Results))
		}

		// NamedReturn(n int) (result string, err error) — named results
		nr, ok := byName["NamedReturn"]
		if !ok {
			t.Fatal("missing NamedReturn")
		}
		if len(nr.Results) != 2 {
			t.Fatalf("NamedReturn results = %d, want 2", len(nr.Results))
		}
		if nr.Results[0].Name != "result" || nr.Results[0].TypeName != "string" {
			t.Errorf("NamedReturn results[0] = %+v, want {result, string}", nr.Results[0])
		}

		// Variadic(prefix string, values ...int) []int
		vari, ok := byName["Variadic"]
		if !ok {
			t.Fatal("missing Variadic")
		}
		if len(vari.Params) != 2 {
			t.Fatalf("Variadic params = %d, want 2", len(vari.Params))
		}
		if vari.Params[1].TypeName != "...int" {
			t.Errorf("Variadic params[1].TypeName = %q, want %q", vari.Params[1].TypeName, "...int")
		}

		// DisplayName: value receiver
		dn, ok := byName["DisplayName"]
		if !ok {
			t.Fatal("missing DisplayName")
		}
		if dn.Receiver != "User" {
			t.Errorf("DisplayName.Receiver = %q, want %q", dn.Receiver, "User")
		}

		// SetStatus: pointer receiver
		ss, ok := byName["SetStatus"]
		if !ok {
			t.Fatal("missing SetStatus")
		}
		if ss.Receiver != "*User" {
			t.Errorf("SetStatus.Receiver = %q, want %q", ss.Receiver, "*User")
		}

		// FuncWithLocalStruct: 2 local structs
		floc, ok := byName["FuncWithLocalStruct"]
		if !ok {
			t.Fatal("missing FuncWithLocalStruct")
		}
		if len(floc.LocalStructs) != 2 {
			t.Fatalf("FuncWithLocalStruct.LocalStructs = %d, want 2", len(floc.LocalStructs))
		}
		localByName := make(map[string]bool)
		for _, ls := range floc.LocalStructs {
			localByName[ls.Name] = true
		}
		for _, name := range []string{"LocalReq", "LocalResp"} {
			if !localByName[name] {
				t.Errorf("missing local struct %q", name)
			}
		}

		// FuncWithLocalTypeDef: 2 local type defs, 0 local structs
		ftd, ok := byName["FuncWithLocalTypeDef"]
		if !ok {
			t.Fatal("missing FuncWithLocalTypeDef")
		}
		if len(ftd.LocalStructs) != 0 {
			t.Errorf("FuncWithLocalTypeDef.LocalStructs = %d, want 0", len(ftd.LocalStructs))
		}
		if len(ftd.LocalTypeDefs) != 2 {
			t.Fatalf("FuncWithLocalTypeDef.LocalTypeDefs = %d, want 2", len(ftd.LocalTypeDefs))
		}
		tdByName := make(map[string]RawTypeDef)
		for _, td := range ftd.LocalTypeDefs {
			tdByName[td.Name] = td
		}
		if view, ok := tdByName["View"]; !ok {
			t.Error("missing local type def View")
		} else if view.TypeName != "BaseModel" {
			t.Errorf("View.TypeName = %q, want %q", view.TypeName, "BaseModel")
		}
		if vl, ok := tdByName["ViewList"]; !ok {
			t.Error("missing local type def ViewList")
		} else if vl.TypeName != "[]string" {
			t.Errorf("ViewList.TypeName = %q, want %q", vl.TypeName, "[]string")
		}
	})
}

// ── integration: MaxDepth ─────────────────────────────────────────────────────

// setupDepthFixture creates a two-level directory fixture for depth tests:
//
//	root/
//	  root.go   (package root, type Root struct{})
//	  sub/
//	    sub.go  (package sub,  type Sub  struct{})
func setupDepthFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	sub := filepath.Join(root, "sub")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatalf("mkdir sub: %v", err)
	}
	write := func(path, src string) {
		if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}
	write(filepath.Join(root, "root.go"), "package root\n\ntype Root struct{}\n")
	write(filepath.Join(sub, "sub.go"), "package sub\n\ntype Sub struct{}\n")
	return root
}

func TestGoParser_Parse_Depth(t *testing.T) {
	root := setupDepthFixture(t)

	t.Run("MaxDepth=0 excludes subdirectories", func(t *testing.T) {
		p := &GoParser{MaxDepth: 0}
		files, err := p.Parse([]string{root})
		if err != nil {
			t.Fatalf("Parse: %v", err)
		}
		for _, f := range files {
			if strings.Contains(f.FilePath, "sub") {
				t.Errorf("MaxDepth=0 must not include sub dir, got %q", f.FilePath)
			}
		}
		// root.go should still be parsed
		if len(files) == 0 {
			t.Error("MaxDepth=0 should still parse root-level files")
		}
	})

	t.Run("MaxDepth=1 includes one level of subdirectories", func(t *testing.T) {
		p := &GoParser{MaxDepth: 1}
		files, err := p.Parse([]string{root})
		if err != nil {
			t.Fatalf("Parse: %v", err)
		}
		hasSub := false
		for _, f := range files {
			if strings.Contains(f.FilePath, "sub") {
				hasSub = true
				break
			}
		}
		if !hasSub {
			t.Error("MaxDepth=1 should include sub dir files")
		}
	})

	t.Run("MaxDepth=-1 includes all subdirectories", func(t *testing.T) {
		p := &GoParser{MaxDepth: -1}
		files, err := p.Parse([]string{root})
		if err != nil {
			t.Fatalf("Parse: %v", err)
		}
		byPkg := make(map[string]bool)
		for _, f := range files {
			byPkg[f.Package] = true
		}
		if !byPkg["root"] || !byPkg["sub"] {
			t.Errorf("MaxDepth=-1 should find both packages, got %v", byPkg)
		}
	})

	t.Run("MaxDepth=0 finds Root struct but not Sub", func(t *testing.T) {
		p := &GoParser{MaxDepth: 0}
		files, err := p.Parse([]string{root})
		if err != nil {
			t.Fatalf("Parse: %v", err)
		}
		var hasRoot, hasSub bool
		for _, f := range files {
			for _, s := range f.Structs {
				switch s.Name {
				case "Root":
					hasRoot = true
				case "Sub":
					hasSub = true
				}
			}
		}
		if !hasRoot {
			t.Error("MaxDepth=0 should find Root struct")
		}
		if hasSub {
			t.Error("MaxDepth=0 must not find Sub struct from subdir")
		}
	})
}

// ── integration: multiple dirs ────────────────────────────────────────────────

func TestGoParser_Parse_MultipleDirs(t *testing.T) {
	root := setupDepthFixture(t)

	p := &GoParser{MaxDepth: 0}
	// Combine temp root dir with a real testdata dir
	files, err := p.Parse([]string{root, "../../testdata/types"})
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	packages := make(map[string]bool)
	for _, f := range files {
		packages[f.Package] = true
	}
	if !packages["root"] {
		t.Error("expected root package from temp fixture")
	}
	if !packages["types"] {
		t.Error("expected types package from testdata/types")
	}
}

// ── integration: cross-file type references ───────────────────────────────────

// structsIndex 将多个 RawFile 中所有 struct 合并为按名索引的 map。
func structsIndex(files []*RawFile) map[string]RawStruct {
	m := make(map[string]RawStruct)
	for _, rf := range files {
		for _, s := range rf.Structs {
			m[s.Name] = s
		}
	}
	return m
}

// fieldsIndex 将 RawStruct 的字段列表转为按名索引的 map。
func fieldsIndex(s RawStruct) map[string]RawField {
	m := make(map[string]RawField, len(s.Fields))
	for _, f := range s.Fields {
		m[f.Name] = f
	}
	return m
}

// setupChainFixture 在临时目录中创建一个 4 文件的 shop 包：
//
//	order.go    Order  → *Pet (跨文件), time.Time (外部包)
//	pet.go      Pet    → *Category (跨文件), []Tag (跨文件)
//	category.go Category → *Category (自引用递归), time.Time (外部包)
//	tag.go      Tag    → 叶节点（仅基础类型）
//	item.go     Item   → 复合外部类型：[]time.Duration, map[string]url.URL, *url.URL
func setupChainFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	write := func(name, src string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(root, name), []byte(src), 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}

	write("order.go", `package shop

import "time"

// Order 订单，引用跨文件类型 Pet 和外部包类型 time.Time。
type Order struct {
	ID        int64
	Pet       *Pet
	Quantity  int
	CreatedAt time.Time
}
`)

	write("pet.go", `package shop

// Pet 宠物，引用跨文件类型 Category 和 Tag 切片。
type Pet struct {
	ID       int64
	Name     string
	Category *Category
	Tags     []Tag
}
`)

	write("category.go", `package shop

import "time"

// Category 分类，自引用（递归类型）并引用外部包类型。
type Category struct {
	ID        int64
	Name      string
	Parent    *Category
	UpdatedAt time.Time
}
`)

	write("tag.go", `package shop

// Tag 叶节点：仅包含基础类型字段，无外部依赖。
type Tag struct {
	Key   string
	Value string
}
`)

	write("item.go", `package shop

import (
	"net/url"
	"time"
)

// Item 使用复合外部类型：切片、map、指针均指向外部包类型。
type Item struct {
	Durations []time.Duration
	URLMap    map[string]url.URL
	PtrURL    *url.URL
	Matrix    [][]time.Duration
}
`)

	return root
}

// TestGoParser_Parse_CrossFileTypeChain 验证跨文件类型引用链：
//
//	Order → *Pet → *Category → *Category（递归）
//	每一层均检查 TypeName 字符串，并通过 structsIndex 验证可在解析结果中找到对应类型。
func TestGoParser_Parse_CrossFileTypeChain(t *testing.T) {
	root := setupChainFixture(t)
	p := &GoParser{MaxDepth: 0}
	files, err := p.Parse([]string{root})
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	idx := structsIndex(files)

	// 所有 struct 必须被解析到
	for _, name := range []string{"Order", "Pet", "Category", "Tag", "Item"} {
		if _, ok := idx[name]; !ok {
			t.Errorf("missing struct %q", name)
		}
	}

	// ── Order ────────────────────────────────────────────────────────────────
	orderF := fieldsIndex(idx["Order"])

	// Order.Pet → 跨文件引用，TypeName 记为 "*Pet"
	if f, ok := orderF["Pet"]; !ok {
		t.Error("Order missing Pet field")
	} else if f.TypeName != "*Pet" {
		t.Errorf("Order.Pet.TypeName = %q, want %q", f.TypeName, "*Pet")
	}

	// Order.CreatedAt → 外部包类型，TypeName 记为 "time.Time"
	if f, ok := orderF["CreatedAt"]; !ok {
		t.Error("Order missing CreatedAt field")
	} else if f.TypeName != "time.Time" {
		t.Errorf("Order.CreatedAt.TypeName = %q, want %q", f.TypeName, "time.Time")
	}

	// ── Pet ──────────────────────────────────────────────────────────────────
	petF := fieldsIndex(idx["Pet"])

	// Pet.Category → 跨文件引用（第 2 层）
	if f, ok := petF["Category"]; !ok {
		t.Error("Pet missing Category field")
	} else if f.TypeName != "*Category" {
		t.Errorf("Pet.Category.TypeName = %q, want %q", f.TypeName, "*Category")
	}

	// Pet.Tags → 跨文件类型的切片
	if f, ok := petF["Tags"]; !ok {
		t.Error("Pet missing Tags field")
	} else if f.TypeName != "[]Tag" {
		t.Errorf("Pet.Tags.TypeName = %q, want %q", f.TypeName, "[]Tag")
	}

	// ── Category ─────────────────────────────────────────────────────────────
	catF := fieldsIndex(idx["Category"])

	// Category.Parent → 自引用递归
	if f, ok := catF["Parent"]; !ok {
		t.Error("Category missing Parent field")
	} else if f.TypeName != "*Category" {
		t.Errorf("Category.Parent.TypeName = %q, want %q", f.TypeName, "*Category")
	}

	// Category.UpdatedAt → 外部包类型（与 Order 同一外部包，跨文件各自 import）
	if f, ok := catF["UpdatedAt"]; !ok {
		t.Error("Category missing UpdatedAt field")
	} else if f.TypeName != "time.Time" {
		t.Errorf("Category.UpdatedAt.TypeName = %q, want %q", f.TypeName, "time.Time")
	}

	// ── 引用链可达性验证 ──────────────────────────────────────────────────────
	//
	// 从 Order.Pet 出发沿链追踪：Order→Pet→Category→Category（自递归）
	// 每步去掉 "*" / "[]" 前缀，从 structsIndex 中查找对应 struct。
	chain := []struct{ from, field, wantType string }{
		{"Order", "Pet", "*Pet"},
		{"Pet", "Category", "*Category"},
		{"Category", "Parent", "*Category"},
	}
	for _, step := range chain {
		s, ok := idx[step.from]
		if !ok {
			t.Errorf("chain: struct %q not found", step.from)
			continue
		}
		f, ok := fieldsIndex(s)[step.field]
		if !ok {
			t.Errorf("chain: %s.%s not found", step.from, step.field)
			continue
		}
		if f.TypeName != step.wantType {
			t.Errorf("chain: %s.%s.TypeName = %q, want %q", step.from, step.field, f.TypeName, step.wantType)
			continue
		}
		// 去掉前缀后，目标类型须在解析结果中存在
		target := strings.TrimPrefix(f.TypeName, "*")
		target = strings.TrimPrefix(target, "[]")
		if _, ok := idx[target]; !ok {
			t.Errorf("chain: type %q (from %s.%s) not found in parsed structs", target, step.from, step.field)
		}
	}
}

// TestGoParser_Parse_ExternalTypeExpressions 验证复合外部类型表达式的 TypeName。
//
//	[]time.Duration        → 外部类型的切片
//	map[string]url.URL     → 外部类型作为 map value
//	*url.URL               → 外部类型的指针
//	[][]time.Duration      → 二维切片（嵌套切片）
func TestGoParser_Parse_ExternalTypeExpressions(t *testing.T) {
	root := setupChainFixture(t)
	p := &GoParser{MaxDepth: 0}
	files, err := p.Parse([]string{root})
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	idx := structsIndex(files)
	item, ok := idx["Item"]
	if !ok {
		t.Fatal("missing Item struct")
	}
	itemF := fieldsIndex(item)

	cases := []struct{ field, want string }{
		{"Durations", "[]time.Duration"},
		{"URLMap", "map[string]url.URL"},
		{"PtrURL", "*url.URL"},
		{"Matrix", "[][]time.Duration"},
	}
	for _, tc := range cases {
		f, ok := itemF[tc.field]
		if !ok {
			t.Errorf("Item missing field %q", tc.field)
			continue
		}
		if f.TypeName != tc.want {
			t.Errorf("Item.%s.TypeName = %q, want %q", tc.field, f.TypeName, tc.want)
		}
	}
}

// TestGoParser_Parse_SamePackageMultiFile 验证多文件同包场景下：
// 解析结果按文件拆分（每个 RawFile 对应一个源文件），且各文件的 Package 名相同。
func TestGoParser_Parse_SamePackageMultiFile(t *testing.T) {
	root := setupChainFixture(t)
	p := &GoParser{MaxDepth: 0}
	files, err := p.Parse([]string{root})
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	if len(files) != 5 {
		t.Fatalf("expected 5 files (one per .go), got %d", len(files))
	}
	for _, rf := range files {
		if rf.Package != "shop" {
			t.Errorf("file %q: Package = %q, want %q", rf.FilePath, rf.Package, "shop")
		}
	}

	// 每个源文件只含自己定义的 struct
	fileStructs := make(map[string][]string) // basename → struct names
	for _, rf := range files {
		base := filepath.Base(rf.FilePath)
		for _, s := range rf.Structs {
			fileStructs[base] = append(fileStructs[base], s.Name)
		}
	}
	expects := map[string]string{
		"order.go":    "Order",
		"pet.go":      "Pet",
		"category.go": "Category",
		"tag.go":      "Tag",
		"item.go":     "Item",
	}
	for file, wantStruct := range expects {
		names := fileStructs[file]
		found := false
		for _, n := range names {
			if n == wantStruct {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("file %q should contain struct %q, got %v", file, wantStruct, names)
		}
	}
}

// ── integration: error cases ──────────────────────────────────────────────────

func TestGoParser_Parse_NonExistentDir(t *testing.T) {
	p := &GoParser{MaxDepth: -1}
	_, err := p.Parse([]string{"/nonexistent/path/does/not/exist"})
	if err == nil {
		t.Error("expected error for non-existent directory, got nil")
	}
}

func TestGoParser_Parse_EmptyDirList(t *testing.T) {
	p := &GoParser{MaxDepth: -1}
	files, err := p.Parse([]string{})
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(files) != 0 {
		t.Errorf("expected 0 files for empty dir list, got %d", len(files))
	}
}

func TestGoParser_ParseDir(t *testing.T) {
	p := &GoParser{}
	// ParseDir 只扫描单层目录，不递归
	files, err := p.ParseDir("../../testdata/annotations")
	if err != nil {
		t.Fatalf("ParseDir: %v", err)
	}
	// testdata/annotations 根目录有 v1.go / v2.go / v3.go，非 _test.go 文件
	if len(files) == 0 {
		t.Error("expected at least one file from testdata/annotations, got 0")
	}
	// 子目录 common/ 和 models/ 不被递归进入
	for _, f := range files {
		if f.Package == "common" || f.Package == "models" {
			t.Errorf("ParseDir should not recurse into subdirectory, got package %q from %s", f.Package, f.FilePath)
		}
	}
}
