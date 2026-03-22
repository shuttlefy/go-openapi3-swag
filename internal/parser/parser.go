package parser

import (
	"fmt"
	"go/ast"
	goparser "go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

// GoParser 递归扫描目录，将每个 .go 源文件解析为 RawFile。
//
// MaxDepth 控制子目录递归深度：
//   - MaxDepth == -1 : 无限递归
//   - MaxDepth == 0  : 只扫描 dirs 本身，不进入子目录
//   - MaxDepth == N  : 最多向下 N 层子目录
type GoParser struct {
	MaxDepth int
}

// Parse 解析给定目录列表，返回所有解析到的源文件。
// 每个目录按 MaxDepth 规则递归展开子目录。
func (p *GoParser) Parse(dirs []string) ([]*RawFile, error) {
	var result []*RawFile
	for _, dir := range dirs {
		abs, err := filepath.Abs(dir)
		if err != nil {
			return nil, fmt.Errorf("resolve dir %q: %w", dir, err)
		}
		files, err := p.walkDir(abs, 0)
		if err != nil {
			return nil, err
		}
		result = append(result, files...)
	}
	return result, nil
}

// ParseDir 解析单个目录（不递归），用于模块缓存懒加载场景。
// 只扫描 dir 直接下一层的非 _test.go 源文件。
func (p *GoParser) ParseDir(dir string) ([]*RawFile, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read dir %q: %w", dir, err)
	}
	var result []*RawFile
	for _, entry := range entries {
		if entry.IsDir() || !isGoSourceFile(entry.Name()) {
			continue
		}
		rf, err := p.parseFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			return nil, err
		}
		if rf != nil {
			result = append(result, rf)
		}
	}
	return result, nil
}

// walkDir 递归扫描单个目录。
func (p *GoParser) walkDir(dir string, depth int) ([]*RawFile, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read dir %q: %w", dir, err)
	}

	var result []*RawFile
	for _, entry := range entries {
		name := entry.Name()
		fullPath := filepath.Join(dir, name)

		if entry.IsDir() {
			if shouldSkipDir(name) {
				continue
			}
			// 深度检查：MaxDepth < 0 表示无限；否则检查是否超限
			if p.MaxDepth >= 0 && depth >= p.MaxDepth {
				continue
			}
			sub, err := p.walkDir(fullPath, depth+1)
			if err != nil {
				return nil, err
			}
			result = append(result, sub...)
		} else {
			if !isGoSourceFile(name) {
				continue
			}
			rf, err := p.parseFile(fullPath)
			if err != nil {
				return nil, err
			}
			if rf != nil {
				result = append(result, rf)
			}
		}
	}
	return result, nil
}

// shouldSkipDir 判断是否跳过该目录。
func shouldSkipDir(name string) bool {
	return strings.HasPrefix(name, ".") || name == "vendor" || name == "testdata"
}

// isGoSourceFile 判断是否为需要解析的 Go 源文件（跳过 _test.go）。
func isGoSourceFile(name string) bool {
	return strings.HasSuffix(name, ".go") && !strings.HasSuffix(name, "_test.go")
}

// parseFile 解析单个 .go 文件，返回 RawFile。
func (p *GoParser) parseFile(path string) (*RawFile, error) {
	fset := token.NewFileSet()
	astFile, err := goparser.ParseFile(fset, path, nil, goparser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("parse %q: %w", path, err)
	}

	rf := &RawFile{
		Package:  astFile.Name.Name,
		FilePath: path,
		Imports:  extractImports(astFile),
	}

	for _, decl := range astFile.Decls {
		switch d := decl.(type) {
		case *ast.GenDecl:
			extractGenDecl(d, rf)
		case *ast.FuncDecl:
			extractFuncDecl(d, fset, rf)
		}
	}

	return rf, nil
}

// ── imports ───────────────────────────────────────────────────────────────────

func extractImports(f *ast.File) []RawImport {
	var imports []RawImport
	for _, imp := range f.Imports {
		path := strings.Trim(imp.Path.Value, `"`)
		pkgName := lastPathSegment(path)

		var alias string
		if imp.Name != nil {
			alias = imp.Name.Name
			if alias == "_" {
				continue // 跳过空白 import
			}
			// "." dot-import 保留
		}

		imports = append(imports, RawImport{
			Alias:   alias,
			Path:    path,
			PkgName: pkgName,
		})
	}
	return imports
}

func lastPathSegment(path string) string {
	parts := strings.Split(path, "/")
	return parts[len(parts)-1]
}

// ── generic declarations ──────────────────────────────────────────────────────

func extractGenDecl(decl *ast.GenDecl, rf *RawFile) {
	switch decl.Tok {
	case token.TYPE:
		for _, spec := range decl.Specs {
			ts := spec.(*ast.TypeSpec)
			switch t := ts.Type.(type) {
			case *ast.StructType:
				s := buildStruct(ts, t, decl.Doc, rf.FilePath)
				rf.Structs = append(rf.Structs, s)
			default:
				if ts.Assign.IsValid() {
					// type Foo = Bar  (透明别名)
					rf.TypeAliases = append(rf.TypeAliases, RawTypeAlias{
						Name:     ts.Name.Name,
						TypeName: typeExprToString(ts.Type),
						FilePath: rf.FilePath,
						Comments: mergeComments(decl.Doc, ts.Comment),
					})
				} else {
					// type Foo Bar (非 struct 新类型定义，如 type H map[string]any)
					rf.TypeDefs = append(rf.TypeDefs, RawTypeDef{
						Name:     ts.Name.Name,
						TypeName: typeExprToString(ts.Type),
						FilePath: rf.FilePath,
						Comments: mergeComments(decl.Doc, ts.Comment),
					})
				}
			}
		}
	case token.CONST:
		extractConsts(decl, rf)
	}
}

// ── structs ───────────────────────────────────────────────────────────────────

func buildStruct(ts *ast.TypeSpec, st *ast.StructType, groupDoc *ast.CommentGroup, filePath string) RawStruct {
	s := RawStruct{
		Name:     ts.Name.Name,
		FilePath: filePath,
		Comments: mergeComments(groupDoc, ts.Comment),
	}

	// 泛型类型参数（Go 1.18+）
	if ts.TypeParams != nil {
		for _, field := range ts.TypeParams.List {
			constraint := typeExprToString(field.Type)
			for _, name := range field.Names {
				s.TypeParams = append(s.TypeParams, RawTypeParam{
					Name:       name.Name,
					Constraint: constraint,
				})
			}
		}
	}

	// 字段
	for _, field := range st.Fields.List {
		s.Fields = append(s.Fields, extractFields(field)...)
	}

	return s
}

func extractFields(field *ast.Field) []RawField {
	typeName := typeExprToString(field.Type)
	tag := ""
	if field.Tag != nil {
		tag = strings.Trim(field.Tag.Value, "`")
	}
	comments := mergeComments(field.Doc, field.Comment)

	// 嵌入字段（匿名）
	if len(field.Names) == 0 {
		return []RawField{{
			Name:     embeddedFieldName(field.Type),
			TypeName: typeName,
			Tag:      tag,
			Comments: comments,
			Embedded: true,
		}}
	}

	// 具名字段（多个名共享同一类型时展开）
	result := make([]RawField, 0, len(field.Names))
	for _, ident := range field.Names {
		// 小写开头的字段为未导出字段，不计入 schema
		if len(ident.Name) > 0 && ident.Name[0] >= 'a' && ident.Name[0] <= 'z' {
			continue
		}
		result = append(result, RawField{
			Name:     ident.Name,
			TypeName: typeName,
			Tag:      tag,
			Comments: comments,
		})
	}
	return result
}

// embeddedFieldName 从嵌入字段的类型表达式中提取基名。
// 例：*pkg.Base → "Base"，*Base → "Base"，pkg.Base → "Base"
func embeddedFieldName(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.Ident:
		return e.Name
	case *ast.SelectorExpr:
		return e.Sel.Name
	case *ast.StarExpr:
		return embeddedFieldName(e.X)
	default:
		return ""
	}
}

// ── constants ─────────────────────────────────────────────────────────────────

func extractConsts(decl *ast.GenDecl, rf *RawFile) {
	var lastType string // 用于 iota 块中继承类型
	for iotaIdx, spec := range decl.Specs {
		vs := spec.(*ast.ValueSpec)

		// 类型继承
		typeName := lastType
		if vs.Type != nil {
			typeName = typeExprToString(vs.Type)
			lastType = typeName
		}
		if typeName == "" {
			continue // 无类型常量，跳过
		}

		comments := mergeComments(decl.Doc, vs.Comment)

		for i, nameIdent := range vs.Names {
			var valExpr ast.Expr
			if i < len(vs.Values) {
				valExpr = vs.Values[i]
			}
			value := constValueToString(valExpr, iotaIdx)
			rf.Consts = append(rf.Consts, RawConst{
				Name:     nameIdent.Name,
				TypeName: typeName,
				Value:    value,
				FilePath: rf.FilePath,
				Comments: comments,
			})
		}
	}
}

// constValueToString 将常量值表达式转为字符串。
// iotaIdx 是该 spec 在 GenDecl.Specs 中的下标，用于 iota 计算。
func constValueToString(expr ast.Expr, iotaIdx int) string {
	if expr == nil {
		return fmt.Sprintf("%d", iotaIdx)
	}
	switch e := expr.(type) {
	case *ast.BasicLit:
		if e.Kind == token.STRING {
			return strings.Trim(e.Value, `"`)
		}
		return e.Value
	case *ast.Ident:
		if e.Name == "iota" {
			return fmt.Sprintf("%d", iotaIdx)
		}
		return e.Name
	case *ast.UnaryExpr:
		return e.Op.String() + constValueToString(e.X, iotaIdx)
	case *ast.BinaryExpr:
		return constValueToString(e.X, iotaIdx) + e.Op.String() + constValueToString(e.Y, iotaIdx)
	case *ast.CallExpr:
		return typeExprToString(e.Fun) + "(...)"
	default:
		return fmt.Sprintf("%v", expr)
	}
}

// ── functions ─────────────────────────────────────────────────────────────────

func extractFuncDecl(decl *ast.FuncDecl, fset *token.FileSet, rf *RawFile) {
	if decl.Name == nil {
		return
	}

	pos := fset.Position(decl.Pos())
	fn := RawFunc{
		Name:     decl.Name.Name,
		FilePath: rf.FilePath,
		Line:     pos.Line,
		Comments: mergeComments(decl.Doc),
	}

	// 接收者
	if decl.Recv != nil && len(decl.Recv.List) > 0 {
		fn.Receiver = typeExprToString(decl.Recv.List[0].Type)
	}

	// 参数
	if decl.Type.Params != nil {
		for _, field := range decl.Type.Params.List {
			typeName := typeExprToString(field.Type)
			if len(field.Names) == 0 {
				fn.Params = append(fn.Params, RawParam{TypeName: typeName})
			} else {
				for _, n := range field.Names {
					fn.Params = append(fn.Params, RawParam{Name: n.Name, TypeName: typeName})
				}
			}
		}
	}

	// 返回值
	if decl.Type.Results != nil {
		for _, field := range decl.Type.Results.List {
			typeName := typeExprToString(field.Type)
			if len(field.Names) == 0 {
				fn.Results = append(fn.Results, RawParam{TypeName: typeName})
			} else {
				for _, n := range field.Names {
					fn.Results = append(fn.Results, RawParam{Name: n.Name, TypeName: typeName})
				}
			}
		}
	}

	// 函数体内的局部 struct
	if decl.Body != nil {
		fn.LocalStructs = extractLocalStructs(decl.Body, rf.FilePath)
	}

	rf.Functions = append(rf.Functions, fn)
}

// extractLocalStructs 从函数体中提取所有局部 struct 定义。
func extractLocalStructs(body *ast.BlockStmt, filePath string) []RawStruct {
	var result []RawStruct
	ast.Inspect(body, func(n ast.Node) bool {
		decl, ok := n.(*ast.GenDecl)
		if !ok || decl.Tok != token.TYPE {
			return true
		}
		for _, spec := range decl.Specs {
			ts := spec.(*ast.TypeSpec)
			st, ok := ts.Type.(*ast.StructType)
			if !ok {
				continue
			}
			result = append(result, buildStruct(ts, st, decl.Doc, filePath))
		}
		return true
	})
	return result
}

// ── type expression → string ──────────────────────────────────────────────────

// typeExprToString 将 ast.Expr 转换为规范化的类型字符串。
//
// 特殊类型映射：
//   - func/chan/interface{}/struct{}  → 保留为标记字符串，builder 阶段跳过或特殊处理
//   - 固定长度数组 [N]T              → 统一视为 []T（JSON schema 层面没有定长数组）
func typeExprToString(expr ast.Expr) string {
	if expr == nil {
		return ""
	}
	switch e := expr.(type) {
	case *ast.Ident:
		return e.Name

	case *ast.StarExpr:
		return "*" + typeExprToString(e.X)

	case *ast.ArrayType:
		// 固定数组和切片统一为 []T
		return "[]" + typeExprToString(e.Elt)

	case *ast.MapType:
		return "map[" + typeExprToString(e.Key) + "]" + typeExprToString(e.Value)

	case *ast.SelectorExpr:
		return typeExprToString(e.X) + "." + e.Sel.Name

	case *ast.IndexExpr: // 泛型单参数实例化：T[A]
		return typeExprToString(e.X) + "[" + typeExprToString(e.Index) + "]"

	case *ast.IndexListExpr: // 泛型多参数实例化：T[A, B]
		args := make([]string, len(e.Indices))
		for i, idx := range e.Indices {
			args[i] = typeExprToString(idx)
		}
		return typeExprToString(e.X) + "[" + strings.Join(args, ",") + "]"

	case *ast.InterfaceType:
		return "interface{}"

	case *ast.StructType:
		return "struct{}" // 匿名 struct

	case *ast.FuncType:
		return "func"

	case *ast.ChanType:
		return "chan"

	case *ast.Ellipsis:
		return "..." + typeExprToString(e.Elt)

	case *ast.BinaryExpr: // 泛型约束中的联合类型：int | string
		return typeExprToString(e.X) + "|" + typeExprToString(e.Y)

	case *ast.BasicLit:
		return e.Value

	case *ast.ParenExpr:
		return "(" + typeExprToString(e.X) + ")"

	default:
		return fmt.Sprintf("unknown(%T)", expr)
	}
}

// ── comment helpers ───────────────────────────────────────────────────────────

// mergeComments 合并多个注释组，返回去掉 `//` 前缀后的纯文本行。
// 空行（`//` 无内容）在两段内容之间保留为 ""，以支持 Markdown 段落分隔；
// 前导和尾随空行忽略，多个连续空行折叠为一个。
func mergeComments(groups ...*ast.CommentGroup) []string {
	var result []string
	pendingBlank := false
	for _, g := range groups {
		if g == nil {
			continue
		}
		for _, c := range g.List {
			text := c.Text
			if strings.HasPrefix(text, "//") {
				text = strings.TrimSpace(text[2:])
			} else if strings.HasPrefix(text, "/*") {
				text = strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(text, "/*"), "*/"))
			}
			if text == "" {
				if len(result) > 0 {
					pendingBlank = true
				}
			} else {
				if pendingBlank {
					result = append(result, "")
					pendingBlank = false
				}
				result = append(result, text)
			}
		}
	}
	return result
}
