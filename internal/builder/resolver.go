package builder

import (
	"fmt"
	"sort"
	"strings"

	spec3 "github.com/shuttlefy/go-openapi3-spec"
	"github.com/shuttlefy/go-openapi3-swag/internal/parser"
)

// ── SchemaKey ──────────────────────────────────────────────────────────────────

// SchemaKey 唯一标识 Components.Schemas 中的一条条目。
// 格式：pkg.TypeName / pkg.FuncName.TypeName（局部 struct）/ pkg.Type[Args]（泛型）
type SchemaKey string

func NewSchemaKey(pkg, typeName string) SchemaKey {
	if pkg == "" {
		return SchemaKey(typeName)
	}
	return SchemaKey(pkg + "." + typeName)
}

// GenericSchemaKey 生成泛型实例化的 key，例如 "common.Resp[models.User]"。
func GenericSchemaKey(base SchemaKey, args ...SchemaKey) SchemaKey {
	parts := make([]string, len(args))
	for i, a := range args {
		parts[i] = string(a)
	}
	return SchemaKey(string(base) + "[" + strings.Join(parts, ",") + "]")
}

// CompositeSchemaKey 生成组合类型的 key，例如 "common.PageData{data=[]models.User}"。
func CompositeSchemaKey(base SchemaKey, overrides map[string]string) SchemaKey {
	if len(overrides) == 0 {
		return base
	}
	pairs := make([]string, 0, len(overrides))
	for k, v := range overrides {
		pairs = append(pairs, k+"="+v)
	}
	sort.Strings(pairs)
	return SchemaKey(string(base) + "{" + strings.Join(pairs, ",") + "}")
}

// Ref 返回该 key 对应的 $ref 路径。
func (k SchemaKey) Ref() string {
	return "#/components/schemas/" + string(k)
}

// ── Resolver ──────────────────────────────────────────────────────────────────

// PackageLoader 按需加载第三方包源文件。
// pkgName 是短包名（如 "gin"），srcFile 是引用该包的源文件（用于查 import path）。
type PackageLoader func(pkgName string, srcFile *parser.RawFile) []*parser.RawFile

// Resolver 管理 schema 注册、$ref 生成、循环引用检测和类型查找。
type Resolver struct {
	files      []*parser.RawFile
	components *spec3.Components
	registry   map[SchemaKey]struct{} // 已注册的 key
	inProgress map[SchemaKey]bool     // 循环引用保护
	sb         *SchemaBuilder         // 反向引用，由 NewSchemaBuilder 设置
	loader     PackageLoader
	loadedPkgs map[string]bool // key = import path，防止重复加载
	// localFuncCtx 记录当前正在构建 schema 的局部 struct 所属函数名。
	// 用于解析「局部 struct 的字段引用同函数内另一个局部 struct」的场景。
	localFuncCtx string
}

func NewResolver() *Resolver {
	return &Resolver{
		components: &spec3.Components{
			Schemas:         spec3.NewOrderedSchemas(),
			SecuritySchemes: spec3.NewOrderedSecuritySchemes(),
		},
		registry:   make(map[SchemaKey]struct{}),
		inProgress: make(map[SchemaKey]bool),
		loadedPkgs: make(map[string]bool),
	}
}

func (r *Resolver) SetFiles(files []*parser.RawFile) { r.files = files }

// SetLoader 注入第三方包懒加载器。
func (r *Resolver) SetLoader(loader PackageLoader) { r.loader = loader }

func (r *Resolver) Components() *spec3.Components { return r.components }

// register 将 schema 写入 Components.Schemas 并记录到 registry。
func (r *Resolver) register(key SchemaKey, schema *spec3.Schema) {
	r.components.Schemas.Set(string(key), schema)
	r.registry[key] = struct{}{}
}

// RefOf 返回只含 $ref 的 schema 对象。
func (r *Resolver) RefOf(key SchemaKey) *spec3.Schema {
	return &spec3.Schema{Reference: spec3.Reference{Ref: key.Ref()}}
}

// ── 内置类型 ──────────────────────────────────────────────────────────────────

// primitiveSchema 将 Go 基础类型名映射为对应 OpenAPI schema。
func primitiveSchema(typeName string) (*spec3.Schema, bool) {
	switch typeName {
	case "string":
		return &spec3.Schema{Type: "string"}, true
	case "bool", "boolean":
		return &spec3.Schema{Type: "boolean"}, true
	case "int", "int8", "int16", "int32",
		"uint", "uint8", "uint16", "uint32", "byte", "rune":
		return &spec3.Schema{Type: "integer", Format: "int32"}, true
	case "int64", "uint64", "uintptr":
		return &spec3.Schema{Type: "integer", Format: "int64"}, true
	case "float32":
		return &spec3.Schema{Type: "number", Format: "float"}, true
	case "float64":
		return &spec3.Schema{Type: "number", Format: "double"}, true
	case "integer":
		return &spec3.Schema{Type: "integer"}, true
	case "number":
		return &spec3.Schema{Type: "number"}, true
	case "file":
		return &spec3.Schema{Type: "string", Format: "binary"}, true
	case "interface{}", "any", "object":
		return &spec3.Schema{}, true
	case "unsafe.Pointer":
		return nil, false // skip
	}
	return nil, false
}

// builtinSchema 将外部包的已知类型映射为对应 OpenAPI schema。
func builtinSchema(pkg, typeName string) (*spec3.Schema, bool) {
	switch pkg + "." + typeName {
	case "time.Time":
		return &spec3.Schema{Type: "string", Format: "date-time"}, true
	case "time.Duration":
		return &spec3.Schema{Type: "integer", Format: "int64"}, true
	case "uuid.UUID":
		return &spec3.Schema{Type: "string", Format: "uuid"}, true
	case "decimal.Decimal":
		return &spec3.Schema{Type: "string", Format: "decimal"}, true
	case "json.RawMessage":
		return &spec3.Schema{}, true
	case "net.IP":
		return &spec3.Schema{Type: "string", Format: "ipv4"}, true
	case "url.URL":
		return &spec3.Schema{Type: "string", Format: "uri"}, true
	}
	return nil, false
}

// ── 主解析入口 ─────────────────────────────────────────────────────────────────

// Resolve 将类型字符串解析为 spec3.Schema。
// 复杂类型（struct/enum）注册到 Components.Schemas 后返回 $ref schema；
// 原始类型和组合类型直接返回内联 schema。
func (r *Resolver) Resolve(typeStr string, file *parser.RawFile) *spec3.Schema {
	typeStr = strings.TrimSpace(typeStr)
	if typeStr == "" {
		return nil
	}

	// [] 前缀 → array
	if strings.HasPrefix(typeStr, "[]") {
		elem := r.Resolve(typeStr[2:], file)
		if elem == nil {
			return nil
		}
		return &spec3.Schema{Type: "array", Items: elem}
	}

	// * 前缀 → nullable
	if strings.HasPrefix(typeStr, "*") {
		base := r.Resolve(typeStr[1:], file)
		if base == nil {
			return nil
		}
		cp := *base
		cp.Nullable = true
		return &cp
	}

	// map[K]V
	if strings.HasPrefix(typeStr, "map[") {
		return r.resolveMapType(typeStr, file)
	}

	// 跳过无法解析的特殊类型
	switch typeStr {
	case "struct{}", "func", "chan":
		return nil
	case "interface{}", "any", "error":
		return &spec3.Schema{} // error 是内置接口，映射为空 schema
	}

	// 组合类型：Base{field=Type,...}
	if i := indexCompositeOpen(typeStr); i != -1 {
		return r.resolveCompositeType(typeStr, i, file)
	}

	// 泛型实例化：Base[Args...]
	if i := strings.Index(typeStr, "["); i > 0 {
		return r.resolveGenericType(typeStr, i, file)
	}

	return r.resolveSimpleType(typeStr, file)
}

// ── 简单类型解析 ──────────────────────────────────────────────────────────────

func (r *Resolver) resolveSimpleType(typeStr string, file *parser.RawFile) *spec3.Schema {
	if s, ok := primitiveSchema(typeStr); ok {
		return s
	}

	parts := strings.SplitN(typeStr, ".", 3)
	switch len(parts) {
	case 1:
		// 无限定符：使用当前文件包名
		if file != nil {
			return r.lookupAndBuild(NewSchemaKey(file.Package, parts[0]), file.Package, parts[0], "", file)
		}
		return nil
	case 2:
		pkg := r.resolveQualifier(parts[0], file)
		if pkg == "" {
			return nil // qualifier 未在当前文件 import 中声明
		}
		if s, ok := builtinSchema(pkg, parts[1]); ok {
			return s
		}
		return r.lookupAndBuild(NewSchemaKey(pkg, parts[1]), pkg, parts[1], "", file)
	case 3:
		pkg := r.resolveQualifier(parts[0], file)
		if pkg == "" {
			return nil // qualifier 未在当前文件 import 中声明
		}
		key := SchemaKey(fmt.Sprintf("%s.%s.%s", pkg, parts[1], parts[2]))
		return r.lookupAndBuild(key, pkg, parts[2], parts[1], file)
	}
	return nil
}

// resolveQualifier 将包限定符解析为包名（支持 import alias）。
//
// 当 file 为 nil 时宽松处理（直接返回 qualifier），适用于编程构造 RawFile 的测试场景。
// 当 file 非 nil 时严格按 imports 查找，优先级：
//  1. qualifier 等于当前文件包名 → 当前包
//  2. qualifier 命中某条 import 的 Alias → 返回该 import 的 PkgName（别名优先）
//  3. qualifier 命中某条 import 的 PkgName → 返回 PkgName
//     兼容「用别名/dot-import 引入但注解中写了包本名」的场景，例如：
//     import rsp "github.com/foo/resourcemgrmessage"
//     注解：response.baseResponse{data=resourcemgrmessage.QuerySpecificDbAccountRsp}
//  4. 均未命中 → 返回 ""
func (r *Resolver) resolveQualifier(qualifier string, file *parser.RawFile) string {
	if file == nil {
		return qualifier // 无文件上下文时宽松处理
	}
	if qualifier == file.Package {
		return qualifier
	}
	for _, imp := range file.Imports {
		// 优先：别名精确匹配
		if imp.Alias == qualifier {
			return imp.PkgName
		}
		// 兼容：用包本名引用（无论该 import 是否设置了别名，包括 dot-import）
		if imp.PkgName == qualifier {
			return imp.PkgName
		}
	}
	// 兜底：当前文件未 import 该包，但 qualifier 可能直接就是已加载文件的包名
	for _, rf := range r.files {
		if rf.Package == qualifier {
			return qualifier
		}
	}
	return "" // qualifier 未出现在该文件的 import 声明中，且已加载文件中也无此包名
}

// ── 类型查找与 schema 构建 ────────────────────────────────────────────────────

func (r *Resolver) lookupAndBuild(key SchemaKey, pkg, typeName, funcName string, currentFile *parser.RawFile) *spec3.Schema {
	if _, ok := r.registry[key]; ok {
		return r.RefOf(key)
	}
	if r.inProgress[key] {
		return r.RefOf(key)
	}
	r.inProgress[key] = true
	defer delete(r.inProgress, key)

	for _, rf := range r.files {
		if rf.Package != pkg {
			continue
		}

		// 函数局部 struct
		if funcName != "" {
			for _, fn := range rf.Functions {
				if fn.Name != funcName {
					continue
				}
				for _, ls := range fn.LocalStructs {
					if ls.Name == typeName {
						// 设置局部函数上下文，使字段解析时能找到同函数内的其他局部 struct
						prev := r.localFuncCtx
						r.localFuncCtx = funcName
						s := r.sb.buildStructSchema(ls, rf)
						r.localFuncCtx = prev
						r.register(key, s)
						return r.RefOf(key)
					}
				}
			}
			continue
		}

		// 无限定符的简单类型：若当前处于某函数的局部 struct 构建上下文中，
		// 优先在该函数的局部 struct 中查找（解决同函数内 struct 互相引用的问题）
		if funcName == "" && r.localFuncCtx != "" {
			for _, fn := range rf.Functions {
				if fn.Name != r.localFuncCtx {
					continue
				}
				for _, ls := range fn.LocalStructs {
					if ls.Name == typeName {
						// 用包含函数名的完整 key 注册，避免不同函数同名局部 struct 冲突
						localKey := SchemaKey(fmt.Sprintf("%s.%s.%s", pkg, r.localFuncCtx, typeName))
						if _, ok := r.registry[localKey]; ok {
							return r.RefOf(localKey)
						}
						if r.inProgress[localKey] {
							return r.RefOf(localKey)
						}
						r.inProgress[localKey] = true
						s := r.sb.buildStructSchema(ls, rf)
						delete(r.inProgress, localKey)
						r.register(localKey, s)
						return r.RefOf(localKey)
					}
				}
			}
		}

		// 包级 struct
		for _, s := range rf.Structs {
			if s.Name == typeName {
				schema := r.sb.buildStructSchema(s, rf)
				r.register(key, schema)
				return r.RefOf(key)
			}
		}

		// 类型别名穿透
		for _, a := range rf.TypeAliases {
			if a.Name == typeName {
				return r.Resolve(a.TypeName, rf)
			}
		}

		// 常量 enum
		var consts []parser.RawConst
		for _, c := range rf.Consts {
			if c.TypeName == typeName {
				consts = append(consts, c)
			}
		}
		if len(consts) > 0 {
			schema := buildEnumSchema(consts)
			r.register(key, schema)
			return r.RefOf(key)
		}

		// 非 struct 类型定义（type H map[string]any 等）——透传底层类型，不注册独立 key
		for _, td := range rf.TypeDefs {
			if td.Name == typeName {
				return r.Resolve(td.TypeName, rf)
			}
		}
	}

	// 在已知文件中找不到 → 尝试通过 PackageLoader 懒加载
	if r.loader != nil && currentFile != nil {
		importPath := findImportPath(pkg, currentFile)
		cacheKey := importPath
		if cacheKey == "" {
			cacheKey = pkg
		}
		if !r.loadedPkgs[cacheKey] {
			r.loadedPkgs[cacheKey] = true
			if loaded := r.loader(pkg, currentFile); len(loaded) > 0 {
				r.files = append(r.files, loaded...)
				// 重试前必须清除 inProgress 标记，否则递归调用会因为 inProgress[key]==true
				// 直接返回悬空 $ref（schema 未注册），导致 components 中缺失该类型的定义。
				delete(r.inProgress, key)
				return r.lookupAndBuild(key, pkg, typeName, funcName, currentFile)
			}
		}
	}
	return nil
}

// FindRawStruct 按类型字符串在已加载文件中查找 RawStruct。
// 返回 (struct, 所在文件)；未找到时两者均为 nil。
// 支持三种格式：
//   - "TypeName"               → 使用 file 的包名，查包级 struct
//   - "pkg.TypeName"           → 查包级 struct
//   - "pkg.FuncName.TypeName"  → 查函数局部 struct
func (r *Resolver) FindRawStruct(typeStr string, file *parser.RawFile) (*parser.RawStruct, *parser.RawFile) {
	typeStr = strings.TrimSpace(typeStr)
	if typeStr == "" {
		return nil, nil
	}

	parts := strings.SplitN(typeStr, ".", 3)
	switch len(parts) {
	case 1:
		// "TypeName" — 当前包
		if file == nil {
			return nil, nil
		}
		return r.findPackageStruct(file.Package, parts[0])

	case 2:
		// "pkg.TypeName" — 包级 struct
		pkg := r.resolveQualifier(parts[0], file)
		if pkg == "" {
			return nil, nil
		}
		return r.findPackageStruct(pkg, parts[1])

	case 3:
		// "pkg.FuncName.TypeName" — 函数局部 struct
		pkg := r.resolveQualifier(parts[0], file)
		if pkg == "" {
			return nil, nil
		}
		funcName, typeName := parts[1], parts[2]
		for i := range r.files {
			rf := r.files[i]
			if rf.Package != pkg {
				continue
			}
			for _, fn := range rf.Functions {
				if fn.Name != funcName {
					continue
				}
				for j := range fn.LocalStructs {
					if fn.LocalStructs[j].Name == typeName {
						return &fn.LocalStructs[j], rf
					}
				}
			}
		}
		return nil, nil
	}
	return nil, nil
}

// findPackageStruct 在已加载文件中按包名和类型名查找包级 struct。
func (r *Resolver) findPackageStruct(pkg, typeName string) (*parser.RawStruct, *parser.RawFile) {
	for i := range r.files {
		rf := r.files[i]
		if rf.Package != pkg {
			continue
		}
		for j := range rf.Structs {
			if rf.Structs[j].Name == typeName {
				return &rf.Structs[j], rf
			}
		}
	}
	return nil, nil
}

// findImportPath 从文件的 import 声明中根据包名找到完整 import path。
// 查找优先级与 resolveQualifier 一致：
//  1. Alias 精确匹配（别名优先）
//  2. PkgName 匹配（兼容别名引入、dot-import、无别名引入）
func findImportPath(pkgName string, file *parser.RawFile) string {
	for _, imp := range file.Imports {
		if imp.Alias == pkgName {
			return imp.Path
		}
		if imp.PkgName == pkgName {
			return imp.Path
		}
	}
	return ""
}

// ── map 类型 ──────────────────────────────────────────────────────────────────

func (r *Resolver) resolveMapType(typeStr string, file *parser.RawFile) *spec3.Schema {
	inner := typeStr[4:] // 去掉 "map["
	depth := 0
	for i, ch := range inner {
		switch ch {
		case '[':
			depth++
		case ']':
			if depth == 0 {
				valueSchema := r.Resolve(strings.TrimSpace(inner[i+1:]), file)
				s := &spec3.Schema{Type: "object"}
				if valueSchema != nil {
					s.AdditionalProperties = valueSchema
				}
				return s
			}
			depth--
		}
	}
	return &spec3.Schema{Type: "object"}
}

// ── 泛型实例化 ────────────────────────────────────────────────────────────────

func (r *Resolver) resolveGenericType(typeStr string, bracketIdx int, file *parser.RawFile) *spec3.Schema {
	baseStr := typeStr[:bracketIdx]
	argsStr := typeStr[bracketIdx+1 : len(typeStr)-1]
	typeArgs := splitTypeArgs(argsStr)

	baseParts := strings.SplitN(baseStr, ".", 2)
	var pkg, typeName string
	if len(baseParts) == 2 {
		pkg = r.resolveQualifier(baseParts[0], file)
		if pkg == "" {
			return nil // 泛型基类型的 qualifier 未在当前文件 import 中声明
		}
		typeName = baseParts[1]
	} else {
		if file != nil {
			pkg = file.Package
		}
		typeName = baseParts[0]
	}

	for _, rf := range r.files {
		if rf.Package != pkg {
			continue
		}
		for _, s := range rf.Structs {
			if s.Name != typeName || len(s.TypeParams) == 0 {
				continue
			}
			argKeys := make([]SchemaKey, len(typeArgs))
			argMap := make(map[string]string, len(s.TypeParams))
			for i, arg := range typeArgs {
				argKeys[i] = SchemaKey(arg)
				if i < len(s.TypeParams) {
					argMap[s.TypeParams[i].Name] = arg
				}
			}
			key := GenericSchemaKey(NewSchemaKey(pkg, typeName), argKeys...)

			if _, ok := r.registry[key]; ok {
				return r.RefOf(key)
			}
			if r.inProgress[key] {
				return r.RefOf(key)
			}
			r.inProgress[key] = true
			defer delete(r.inProgress, key)

			schema := r.sb.buildStructSchemaWithSubst(s, rf, argMap)
			r.register(key, schema)
			return r.RefOf(key)
		}
	}
	// fallback: 按非泛型处理基础类型
	return r.Resolve(baseStr, file)
}

// ── 组合类型 ─────────────────────────────────────────────────────────────────

// indexCompositeOpen 找到最外层 '{' 的位置（跳过 '[...]' 内的 '{'）。
func indexCompositeOpen(s string) int {
	depth := 0
	for i, ch := range s {
		switch ch {
		case '[':
			depth++
		case ']':
			depth--
		case '{':
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

func (r *Resolver) resolveCompositeType(typeStr string, braceIdx int, file *parser.RawFile) *spec3.Schema {
	baseStr := strings.TrimSpace(typeStr[:braceIdx])
	overrideStr := typeStr[braceIdx+1 : len(typeStr)-1]
	overrides := parseOverrides(overrideStr)

	// 先验证 base qualifier 是否可解析，避免生成孤立的 override schema
	baseParts := strings.SplitN(baseStr, ".", 2)
	if len(baseParts) == 2 {
		if pkg := r.resolveQualifier(baseParts[0], file); pkg == "" {
			return nil // base 的 qualifier 未在当前文件 import 中声明
		}
	}

	baseSchema := r.Resolve(baseStr, file)
	if baseSchema == nil {
		return nil
	}

	// 先解析所有 override 字段
	props := spec3.NewOrderedSchemas()
	for fieldName, typeExpr := range overrides {
		if fs := r.Resolve(typeExpr, file); fs != nil {
			props.Set(fieldName, fs)
		}
	}

	// 将覆盖字段作为第二个 allOf 元素，直接返回内联 schema，不注册到 Components.Schemas。
	// 生成结构：
	//   allOf:
	//     - $ref: baseSchema
	//     - type: object
	//       properties:
	//         fieldName: overrideSchema
	composite := &spec3.Schema{}
	if baseSchema != nil && len(props.Keys()) > 0 {
		overridePart := &spec3.Schema{Type: "object", Properties: &props}
		composite.AllOf = []*spec3.Schema{baseSchema, overridePart}
	} else if baseSchema != nil {
		composite.AllOf = []*spec3.Schema{baseSchema}
	} else if len(props.Keys()) > 0 {
		composite.Properties = &props
	}
	return composite
}

// ── enum schema ───────────────────────────────────────────────────────────────

func buildEnumSchema(consts []parser.RawConst) *spec3.Schema {
	schema := &spec3.Schema{}
	if len(consts) > 0 && isNumericStr(consts[0].Value) {
		schema.Type = "integer"
	} else {
		schema.Type = "string"
	}

	varnames := make([]string, 0, len(consts))
	descriptions := make([]string, 0, len(consts))
	hasDesc := false

	for _, c := range consts {
		schema.Enum = append(schema.Enum, c.Value)
		varnames = append(varnames, c.Name)
		desc := strings.Join(c.Comments, " ")
		descriptions = append(descriptions, desc)
		if desc != "" {
			hasDesc = true
		}
	}

	schema.Extensions.Set("x-enum-varnames", varnames)
	if hasDesc {
		schema.Extensions.Set("x-enumdescriptions", descriptions)
	}

	return schema
}

func isNumericStr(s string) bool {
	if s == "" {
		return false
	}
	for i, ch := range s {
		if ch == '-' && i == 0 {
			continue
		}
		if ch < '0' || ch > '9' {
			return false
		}
	}
	return true
}

// ── 辅助函数 ──────────────────────────────────────────────────────────────────

// splitTypeArgs 按顶层逗号拆分泛型参数或 override 列表。
func splitTypeArgs(s string) []string {
	var args []string
	depth, start := 0, 0
	for i, ch := range s {
		switch ch {
		case '[', '{':
			depth++
		case ']', '}':
			depth--
		case ',':
			if depth == 0 {
				if part := strings.TrimSpace(s[start:i]); part != "" {
					args = append(args, part)
				}
				start = i + 1
			}
		}
	}
	if part := strings.TrimSpace(s[start:]); part != "" {
		args = append(args, part)
	}
	return args
}

// parseOverrides 解析 "field1=Type1,field2=Type2" 为 map。
func parseOverrides(s string) map[string]string {
	result := make(map[string]string)
	for _, part := range splitTypeArgs(s) {
		if idx := strings.Index(part, "="); idx != -1 {
			result[strings.TrimSpace(part[:idx])] = strings.TrimSpace(part[idx+1:])
		}
	}
	return result
}

// strPtr 返回非空字符串的指针，空字符串返回 nil。
func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
