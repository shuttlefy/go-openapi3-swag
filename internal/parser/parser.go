package parser

import (
	"fmt"
	"go/ast"
	goparser "go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"

	"github.com/shuttlefy/go-openapi3-swag/config"
)

type GoParser struct{}

func (p *GoParser) Parse(dirs config.StringSlice) (*RawAST, error) {
	result := &RawAST{}
	fset := token.NewFileSet()

	for _, root := range dirs {
		if err := p.walkDir(fset, root, result); err != nil {
			return nil, fmt.Errorf("parse %s: %w", root, err)
		}
	}

	return result, nil
}

func (p *GoParser) walkDir(fset *token.FileSet, root string, result *RawAST) error {
	// Pre-scan: read only root-level files (non-recursive) to determine the
	// primary package before the full walk begins.  filepath.Walk processes
	// entries in lexicographic order, so a sub-directory like "bo/" would be
	// visited before files starting with later letters (e.g. "handlers_…"),
	// causing result.Package to be set to "bo" instead of "main".
	// By pinning result.Package from the root level first we ensure that sub-
	// package types are correctly treated as non-primary.
	if result.Package == "" {
		if pkg := p.detectRootPackage(fset, root); pkg != "" {
			result.Package = pkg
			if !containsPkg(result.Packages, pkg) {
				result.Packages = append(result.Packages, pkg)
			}
		}
	}

	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			base := info.Name()
			if base != "." && (strings.HasPrefix(base, ".") || base == "vendor") {
				return filepath.SkipDir
			}
			return nil
		}
		if !isGoSourceFile(info.Name()) {
			return nil
		}

		file, parseErr := goparser.ParseFile(fset, path, nil, goparser.ParseComments)
		if parseErr != nil {
			return nil
		}

		pkgName := file.Name.Name
		if result.Package == "" {
			result.Package = pkgName
		}
		// Collect all distinct package names across all scanned files.
		if !containsPkg(result.Packages, pkgName) {
			result.Packages = append(result.Packages, pkgName)
		}
		p.extractFile(fset, path, file, result)
		return nil
	})
}

// detectRootPackage reads the first non-test Go file directly inside root
// (not in sub-directories) and returns its package name.
// This is used to pin the primary package before the recursive walk begins.
func (p *GoParser) detectRootPackage(fset *token.FileSet, root string) string {
	entries, err := os.ReadDir(root)
	if err != nil {
		return ""
	}
	for _, entry := range entries {
		if entry.IsDir() || !isGoSourceFile(entry.Name()) {
			continue
		}
		file, parseErr := goparser.ParseFile(fset, filepath.Join(root, entry.Name()), nil, 0)
		if parseErr == nil {
			return file.Name.Name
		}
	}
	return ""
}

func isGoSourceFile(name string) bool {
	return strings.HasSuffix(name, ".go") && !strings.HasSuffix(name, "_test.go")
}

func containsPkg(pkgs []string, name string) bool {
	for _, p := range pkgs {
		if p == name {
			return true
		}
	}
	return false
}

func (p *GoParser) extractFile(fset *token.FileSet, filePath string, file *ast.File, result *RawAST) {
	pkgName := file.Name.Name
	for _, decl := range file.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			result.Functions = append(result.Functions, extractFunc(fset, filePath, d))
			p.extractFuncBody(fset, filePath, pkgName, d, result)
		case *ast.GenDecl:
			switch d.Tok {
			case token.TYPE:
				p.extractTypes(fset, filePath, pkgName, "", d, result)
			case token.CONST:
				p.extractConsts(fset, filePath, pkgName, "", d, result)
			}
		}
	}
}

func (p *GoParser) extractFuncBody(fset *token.FileSet, filePath, pkgName string, fn *ast.FuncDecl, result *RawAST) {
	if fn.Body == nil {
		return
	}
	funcName := fn.Name.Name
	for _, stmt := range fn.Body.List {
		ds, ok := stmt.(*ast.DeclStmt)
		if !ok {
			continue
		}
		gd, ok := ds.Decl.(*ast.GenDecl)
		if !ok {
			continue
		}
		switch gd.Tok {
		case token.TYPE:
			p.extractTypes(fset, filePath, pkgName, funcName, gd, result)
		case token.CONST:
			p.extractConsts(fset, filePath, pkgName, funcName, gd, result)
		}
	}
}

func (p *GoParser) extractTypes(fset *token.FileSet, filePath, pkgName, funcScope string, gd *ast.GenDecl, result *RawAST) {
	for _, spec := range gd.Specs {
		ts, ok := spec.(*ast.TypeSpec)
		if !ok {
			continue
		}
		if st, ok := ts.Type.(*ast.StructType); ok {
			s := extractStruct(fset, filePath, ts, st, gd)
			s.PackageName = pkgName
			s.FuncScope = funcScope
			result.Structs = append(result.Structs, s)
			continue
		}
		a := extractTypeAlias(filePath, ts, gd)
		a.PackageName = pkgName
		a.FuncScope = funcScope
		result.TypeAliases = append(result.TypeAliases, a)
	}
}

func extractTypeAlias(filePath string, ts *ast.TypeSpec, gd *ast.GenDecl) RawTypeAlias {
	a := RawTypeAlias{
		Name:       ts.Name.Name,
		Underlying: typeExpr(ts.Type),
		FilePath:   filePath,
	}
	if ts.Doc != nil {
		for _, c := range ts.Doc.List {
			a.Comments = append(a.Comments, c.Text)
		}
	} else if gd.Doc != nil {
		for _, c := range gd.Doc.List {
			a.Comments = append(a.Comments, c.Text)
		}
	}
	return a
}

func (p *GoParser) extractConsts(fset *token.FileSet, filePath, pkgName, funcScope string, gd *ast.GenDecl, result *RawAST) {
	var lastType string
	var lastExpr ast.Expr // expression template; repeated implicitly when a spec has no values

	for iotaIdx, s := range gd.Specs {
		vs, ok := s.(*ast.ValueSpec)
		if !ok {
			continue
		}

		typeName := lastType
		if vs.Type != nil {
			typeName = typeExpr(vs.Type)
			lastType = typeName
		}

		// If this spec has explicit values update the template; otherwise reuse the
		// last one (Go iota inheritance: the expression is copied verbatim, only
		// the iota counter advances).
		var exprToEval ast.Expr
		if len(vs.Values) > 0 {
			exprToEval = vs.Values[0]
			lastExpr = exprToEval
		} else {
			exprToEval = lastExpr
		}

		value := evaluateConstExpr(exprToEval, iotaIdx)

		for _, name := range vs.Names {
			c := RawConst{
				Name:        name.Name,
				PackageName: pkgName,
				TypeName:    typeName,
				Value:       value,
				FilePath:    filePath,
				FuncScope:   funcScope,
			}
			if vs.Doc != nil {
				for _, cm := range vs.Doc.List {
					c.Comments = append(c.Comments, cm.Text)
				}
			}
			result.Consts = append(result.Consts, c)
		}
	}
}

// evaluateConstExpr resolves a const initialiser expression to its string
// representation, substituting the iota counter where needed.
//
// Handles:
//   - basic literals ("hello", 42, 3.14)
//   - iota identifier → decimal string of iotaVal
//   - binary expressions where both sides are resolvable (iota+1, iota<<2, …)
//   - unary minus (-iota, -1)
//   - parenthesised expressions
//
// Falls back to the raw source text for anything more exotic.
func evaluateConstExpr(expr ast.Expr, iotaVal int) string {
	if expr == nil {
		return ""
	}
	switch e := expr.(type) {
	case *ast.BasicLit:
		return e.Value

	case *ast.Ident:
		if e.Name == "iota" {
			return strconv.Itoa(iotaVal)
		}
		return e.Name

	case *ast.ParenExpr:
		return evaluateConstExpr(e.X, iotaVal)

	case *ast.UnaryExpr:
		operand := evaluateConstExpr(e.X, iotaVal)
		if e.Op == token.SUB {
			if v, err := strconv.ParseInt(operand, 0, 64); err == nil {
				return strconv.FormatInt(-v, 10)
			}
		}
		return e.Op.String() + operand

	case *ast.BinaryExpr:
		left := evaluateConstExpr(e.X, iotaVal)
		right := evaluateConstExpr(e.Y, iotaVal)
		lv, le := strconv.ParseInt(left, 0, 64)
		rv, re := strconv.ParseInt(right, 0, 64)
		if le == nil && re == nil {
			switch e.Op {
			case token.ADD:
				return strconv.FormatInt(lv+rv, 10)
			case token.SUB:
				return strconv.FormatInt(lv-rv, 10)
			case token.MUL:
				return strconv.FormatInt(lv*rv, 10)
			case token.QUO:
				if rv != 0 {
					return strconv.FormatInt(lv/rv, 10)
				}
			case token.SHL:
				return strconv.FormatInt(lv<<uint(rv), 10)
			case token.SHR:
				return strconv.FormatInt(lv>>uint(rv), 10)
			case token.OR:
				return strconv.FormatInt(lv|rv, 10)
			case token.AND:
				return strconv.FormatInt(lv&rv, 10)
			case token.XOR:
				return strconv.FormatInt(lv^rv, 10)
			case token.REM:
				if rv != 0 {
					return strconv.FormatInt(lv%rv, 10)
				}
			}
		}
		return left + " " + e.Op.String() + " " + right

	case *ast.CallExpr:
		// e.g. Type(iota) — evaluate the argument and wrap in the type call.
		if len(e.Args) == 1 {
			inner := evaluateConstExpr(e.Args[0], iotaVal)
			return inner
		}
		return typeExpr(e.Fun) + "(...)"

	default:
		return fmt.Sprintf("%T", expr)
	}
}

func extractFunc(fset *token.FileSet, filePath string, fn *ast.FuncDecl) RawFunc {
	f := RawFunc{
		Name:     fn.Name.Name,
		FilePath: filePath,
		Line:     fset.Position(fn.Pos()).Line,
	}

	if fn.Doc != nil {
		for _, c := range fn.Doc.List {
			f.Comments = append(f.Comments, c.Text)
		}
	}

	if fn.Recv != nil && len(fn.Recv.List) > 0 {
		f.Receiver = typeExpr(fn.Recv.List[0].Type)
	}

	if fn.Type.Params != nil {
		f.Params = extractFieldList(fn.Type.Params)
	}
	if fn.Type.Results != nil {
		f.Results = extractFieldList(fn.Type.Results)
	}

	return f
}

func extractFieldList(fl *ast.FieldList) []RawParam {
	var params []RawParam
	for _, field := range fl.List {
		typeName := typeExpr(field.Type)
		if len(field.Names) == 0 {
			params = append(params, RawParam{TypeName: typeName})
			continue
		}
		for _, name := range field.Names {
			params = append(params, RawParam{Name: name.Name, TypeName: typeName})
		}
	}
	return params
}

func extractStruct(fset *token.FileSet, filePath string, ts *ast.TypeSpec, st *ast.StructType, gd *ast.GenDecl) RawStruct {
	s := RawStruct{
		Name:     ts.Name.Name,
		FilePath: filePath,
	}

	if ts.Doc != nil {
		for _, c := range ts.Doc.List {
			s.Comments = append(s.Comments, c.Text)
		}
	} else if gd.Doc != nil {
		for _, c := range gd.Doc.List {
			s.Comments = append(s.Comments, c.Text)
		}
	}

	for _, field := range st.Fields.List {
		rf := RawField{
			TypeName: typeExpr(field.Type),
		}
		if field.Tag != nil {
			rf.Tag = field.Tag.Value
			parseFieldTag(&rf)
		}
		if field.Doc != nil {
			for _, c := range field.Doc.List {
				rf.Comments = append(rf.Comments, c.Text)
			}
		}

		if len(field.Names) == 0 {
			s.Fields = append(s.Fields, rf)
			continue
		}
		for _, name := range field.Names {
			named := rf
			named.Name = name.Name
			s.Fields = append(s.Fields, named)
		}
	}

	return s
}

func typeExpr(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return "*" + typeExpr(t.X)
	case *ast.ArrayType:
		return "[]" + typeExpr(t.Elt)
	case *ast.SelectorExpr:
		return typeExpr(t.X) + "." + t.Sel.Name
	case *ast.MapType:
		return "map[" + typeExpr(t.Key) + "]" + typeExpr(t.Value)
	case *ast.InterfaceType:
		return "interface{}"
	case *ast.Ellipsis:
		return "..." + typeExpr(t.Elt)
	case *ast.FuncType:
		return funcTypeExpr(t)
	case *ast.ChanType:
		return chanTypeExpr(t)
	case *ast.IndexExpr:
		return typeExpr(t.X) + "[" + typeExpr(t.Index) + "]"
	case *ast.IndexListExpr:
		parts := make([]string, len(t.Indices))
		for i, idx := range t.Indices {
			parts[i] = typeExpr(idx)
		}
		return typeExpr(t.X) + "[" + strings.Join(parts, ", ") + "]"
	case *ast.StructType:
		return "struct{}"
	case *ast.ParenExpr:
		return "(" + typeExpr(t.X) + ")"
	default:
		return fmt.Sprintf("%T", expr)
	}
}

func funcTypeExpr(ft *ast.FuncType) string {
	var b strings.Builder
	b.WriteString("func(")
	if ft.Params != nil {
		for i, field := range ft.Params.List {
			if i > 0 {
				b.WriteString(", ")
			}
			b.WriteString(typeExpr(field.Type))
		}
	}
	b.WriteString(")")
	if ft.Results != nil {
		types := ft.Results.List
		if len(types) == 1 && len(types[0].Names) == 0 {
			b.WriteString(" ")
			b.WriteString(typeExpr(types[0].Type))
		} else if len(types) > 0 {
			b.WriteString(" (")
			for i, field := range types {
				if i > 0 {
					b.WriteString(", ")
				}
				b.WriteString(typeExpr(field.Type))
			}
			b.WriteString(")")
		}
	}
	return b.String()
}

func chanTypeExpr(ct *ast.ChanType) string {
	switch ct.Dir {
	case ast.SEND:
		return "chan<- " + typeExpr(ct.Value)
	case ast.RECV:
		return "<-chan " + typeExpr(ct.Value)
	default:
		return "chan " + typeExpr(ct.Value)
	}
}

// parseFieldTag extracts structured info from a raw struct tag string.
func parseFieldTag(rf *RawField) {
	raw := strings.Trim(rf.Tag, "`")
	tag := reflect.StructTag(raw)

	// json tag
	if jsonTag := tag.Get("json"); jsonTag != "" {
		parts := strings.Split(jsonTag, ",")
		rf.JSONName = parts[0]
		for _, opt := range parts[1:] {
			if opt == "omitempty" {
				rf.Omitempty = true
			}
		}
	}

	// required from binding/validate
	if binding := tag.Get("binding"); binding != "" {
		for _, rule := range strings.Split(binding, ",") {
			if rule == "required" {
				rf.Required = true
				break
			}
		}
	}
	if !rf.Required {
		if validate := tag.Get("validate"); validate != "" {
			for _, rule := range strings.Split(validate, ",") {
				if rule == "required" {
					rf.Required = true
					break
				}
			}
		}
	}

	// value tags
	rf.Example = tag.Get("example")
	rf.Format = tag.Get("format")
	rf.Default = tag.Get("default")
	rf.Description = tag.Get("description")
	rf.Pattern = tag.Get("pattern")

	// boolean tags
	rf.ReadOnly = tag.Get("readonly") == "true"
	rf.WriteOnly = tag.Get("writeonly") == "true"
	rf.Deprecated = tag.Get("deprecated") == "true"
	rf.UniqueItems = tag.Get("uniqueItems") == "true"

	// numeric constraint tags
	rf.Minimum = parseOptionalFloat(tag.Get("minimum"))
	rf.Maximum = parseOptionalFloat(tag.Get("maximum"))
	rf.MinLength = parseOptionalInt(tag.Get("minLength"))
	rf.MaxLength = parseOptionalInt(tag.Get("maxLength"))
	rf.MinItems = parseOptionalInt(tag.Get("minItems"))
	rf.MaxItems = parseOptionalInt(tag.Get("maxItems"))

	// enums
	if enums := tag.Get("enums"); enums != "" {
		for _, v := range strings.Split(enums, ",") {
			v = strings.TrimSpace(v)
			if v != "" {
				rf.Enums = append(rf.Enums, v)
			}
		}
	}
}

func parseOptionalFloat(s string) *float64 {
	if s == "" {
		return nil
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return nil
	}
	return &v
}

func parseOptionalInt(s string) *int64 {
	if s == "" {
		return nil
	}
	v, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return nil
	}
	return &v
}
