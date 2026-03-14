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
)

type GoParser struct{}

func (p *GoParser) Parse(dirs []string) (*RawAST, error) {
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

		if result.Package == "" {
			result.Package = file.Name.Name
		}
		p.extractFile(fset, path, file, result)
		return nil
	})
}

func isGoSourceFile(name string) bool {
	return strings.HasSuffix(name, ".go") && !strings.HasSuffix(name, "_test.go")
}

func (p *GoParser) extractFile(fset *token.FileSet, filePath string, file *ast.File, result *RawAST) {
	for _, decl := range file.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			result.Functions = append(result.Functions, extractFunc(fset, filePath, d))
			p.extractFuncBody(fset, filePath, d, result)
		case *ast.GenDecl:
			switch d.Tok {
			case token.TYPE:
				p.extractTypes(fset, filePath, "", d, result)
			case token.CONST:
				p.extractConsts(fset, filePath, "", d, result)
			}
		}
	}
}

func (p *GoParser) extractFuncBody(fset *token.FileSet, filePath string, fn *ast.FuncDecl, result *RawAST) {
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
			p.extractTypes(fset, filePath, funcName, gd, result)
		case token.CONST:
			p.extractConsts(fset, filePath, funcName, gd, result)
		}
	}
}

func (p *GoParser) extractTypes(fset *token.FileSet, filePath, funcScope string, gd *ast.GenDecl, result *RawAST) {
	for _, spec := range gd.Specs {
		ts, ok := spec.(*ast.TypeSpec)
		if !ok {
			continue
		}
		if st, ok := ts.Type.(*ast.StructType); ok {
			s := extractStruct(fset, filePath, ts, st, gd)
			s.FuncScope = funcScope
			result.Structs = append(result.Structs, s)
			continue
		}
		a := extractTypeAlias(filePath, ts, gd)
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

func (p *GoParser) extractConsts(fset *token.FileSet, filePath, funcScope string, gd *ast.GenDecl, result *RawAST) {
	var lastType string
	for _, spec := range gd.Specs {
		vs, ok := spec.(*ast.ValueSpec)
		if !ok {
			continue
		}

		typeName := lastType
		if vs.Type != nil {
			typeName = typeExpr(vs.Type)
			lastType = typeName
		}

		var value string
		if len(vs.Values) > 0 {
			value = exprToString(fset, vs.Values[0])
		}

		for _, name := range vs.Names {
			c := RawConst{
				Name:      name.Name,
				TypeName:  typeName,
				Value:     value,
				FilePath:  filePath,
				FuncScope: funcScope,
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

func exprToString(fset *token.FileSet, expr ast.Expr) string {
	switch v := expr.(type) {
	case *ast.BasicLit:
		return v.Value
	case *ast.Ident:
		return v.Name
	case *ast.BinaryExpr:
		return exprToString(fset, v.X) + " " + v.Op.String() + " " + exprToString(fset, v.Y)
	case *ast.CallExpr:
		return typeExpr(v.Fun) + "(...)"
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
