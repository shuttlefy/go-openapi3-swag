package extractor

import (
	"fmt"
	"strings"
)

// ── 内部类型 ──────────────────────────────────────────────────────────────────

// tagLine 表示一行解析后的注解。
type tagLine struct {
	name  string // lowercase，不含 "@"，如 "summary" / "param" / "router"
	value string // "@" 之后去掉 tagName 的剩余内容，已 TrimSpace
}

// ── 注释行解析 ────────────────────────────────────────────────────────────────

// parseCommentLines 将 RawFunc.Comments 分为注解行（tagLine）和普通文本行。
// 标签名不区分大小写，统一转为小写。
func parseCommentLines(lines []string) (tags []tagLine, plain []string) {
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "@") {
			if line != "" {
				plain = append(plain, line)
			}
			continue
		}
		// 取 "@" 之后的部分，按首个空格分割 tagName 和 value
		rest := line[1:]
		idx := strings.IndexAny(rest, " \t")
		if idx == -1 {
			tags = append(tags, tagLine{name: strings.ToLower(rest)})
		} else {
			tags = append(tags, tagLine{
				name:  strings.ToLower(rest[:idx]),
				value: strings.TrimSpace(rest[idx+1:]),
			})
		}
	}
	return
}

// ── 通用辅助 ──────────────────────────────────────────────────────────────────

// extractLastQuoted 从字符串末尾提取最后一个 "..." 引号对。
// 返回 (引号内文本, 引号对之前的文本)。若未找到则 quoted="" rest=s。
func extractLastQuoted(s string) (quoted, rest string) {
	end := strings.LastIndex(s, `"`)
	if end == -1 {
		return "", s
	}
	start := strings.LastIndex(s[:end], `"`)
	if start == -1 {
		return "", s
	}
	return s[start+1 : end], strings.TrimSpace(s[:start])
}

// mimeAliases 将简写映射为完整 MIME 类型。
var mimeAliases = map[string]string{
	"json":                  "application/json",
	"xml":                   "application/xml",
	"plain":                 "text/plain",
	"html":                  "text/html",
	"mpfd":                  "multipart/form-data",
	"x-www-form-urlencoded": "application/x-www-form-urlencoded",
	"json-api":              "application/vnd.api+json",
	"json-stream":           "application/x-json-stream",
	"octet-stream":          "application/octet-stream",
	"png":                   "image/png",
	"jpeg":                  "image/jpeg",
	"gif":                   "image/gif",
}

// parseMIMETypes 解析逗号分隔的 MIME 类型列表，自动展开已知别名。
func parseMIMETypes(value string) []string {
	var result []string
	for _, part := range strings.Split(value, ",") {
		mime := strings.TrimSpace(part)
		if mime == "" {
			continue
		}
		if expanded, ok := mimeAliases[mime]; ok {
			result = append(result, expanded)
		} else {
			result = append(result, mime)
		}
	}
	return result
}

// ── @Param ────────────────────────────────────────────────────────────────────

// parseParamTag 解析 @Param 的 value 部分。
//
// 格式：name in type required ["format"] "description"
// 描述必须用双引号括起（description 可选）。
func parseParamTag(value string) (ParamAnnotation, error) {
	desc, rest := extractLastQuoted(value)
	fields := strings.Fields(rest)
	if len(fields) < 4 {
		return ParamAnnotation{}, fmt.Errorf("@Param 至少需要 4 个字段（name in type required），got %q", value)
	}
	p := ParamAnnotation{
		Name:        fields[0],
		In:          strings.ToLower(fields[1]),
		TypeName:    fields[2],
		Required:    strings.ToLower(fields[3]) == "true",
		Description: desc,
	}
	if len(fields) >= 5 {
		p.Format = fields[4]
	}
	return p, nil
}

// ── @Success / @Failure ───────────────────────────────────────────────────────

// parseResponseTag 解析 @Success / @Failure 的 value 部分。
//
// 格式：
//
//	code {wrapType} typeName "description"
//	code "description"            （无 body 响应，如 204）
func parseResponseTag(value string) (ResponseAnnotation, error) {
	desc, rest := extractLastQuoted(value)
	fields := strings.Fields(rest)
	if len(fields) == 0 {
		return ResponseAnnotation{}, fmt.Errorf("@Success/@Failure 缺少状态码，got %q", value)
	}

	resp := ResponseAnnotation{
		Code:        fields[0],
		Description: desc,
	}
	if len(fields) == 1 {
		// 仅有状态码，无 body
		return resp, nil
	}

	// 解析 {wrapType}
	for i, f := range fields[1:] {
		if strings.HasPrefix(f, "{") && strings.HasSuffix(f, "}") {
			resp.WrapType = strings.ToLower(f[1 : len(f)-1])
			// 紧随其后的 token 是 typeName
			modelIdx := i + 2 // fields 偏移：fields[0]=code, [1+i]={wrap}, [2+i]=type
			if modelIdx < len(fields) {
				resp.TypeName = fields[modelIdx]
				resp.IsArray = strings.HasPrefix(resp.TypeName, "[]")
			}
			break
		}
	}
	return resp, nil
}

// ── @Header ───────────────────────────────────────────────────────────────────

// parseHeaderTag 解析 @Header 的 value 部分。
//
// 格式：code {type} headerName "description"
func parseHeaderTag(value string) (HeaderAnnotation, error) {
	desc, rest := extractLastQuoted(value)
	fields := strings.Fields(rest)
	if len(fields) < 3 {
		return HeaderAnnotation{}, fmt.Errorf("@Header 至少需要 3 个字段（code {type} name），got %q", value)
	}
	h := HeaderAnnotation{
		Code:        fields[0],
		Description: desc,
		Name:        fields[2],
	}
	if strings.HasPrefix(fields[1], "{") && strings.HasSuffix(fields[1], "}") {
		h.TypeName = fields[1][1 : len(fields[1])-1]
	}
	return h, nil
}

// ── @Router ───────────────────────────────────────────────────────────────────

// parseRouterTag 解析 @Router 的 value 部分。
//
// 格式：/path [method]
func parseRouterTag(value string) (RouteInfo, error) {
	lb := strings.LastIndex(value, "[")
	rb := strings.LastIndex(value, "]")
	if lb == -1 || rb == -1 || lb > rb {
		return RouteInfo{}, fmt.Errorf("@Router 格式错误，期望 `/path [method]`，got %q", value)
	}
	return RouteInfo{
		Path:   strings.TrimSpace(value[:lb]),
		Method: strings.ToUpper(strings.TrimSpace(value[lb+1 : rb])),
	}, nil
}

// ── @Security（操作级别） ──────────────────────────────────────────────────────

// parseSecurityTag 解析操作级别 @Security 的 value 部分。
//
// 格式：SchemeName 或 SchemeName[scope1, scope2]
func parseSecurityTag(value string) SecurityRequirement {
	value = strings.TrimSpace(value)
	lb := strings.Index(value, "[")
	if lb == -1 {
		return SecurityRequirement{Name: value}
	}
	rb := strings.Index(value[lb:], "]")
	if rb == -1 {
		return SecurityRequirement{Name: value[:lb]}
	}
	rb += lb
	name := strings.TrimSpace(value[:lb])
	var scopes []string
	for _, s := range strings.Split(value[lb+1:rb], ",") {
		s = strings.TrimSpace(s)
		if s != "" {
			scopes = append(scopes, s)
		}
	}
	return SecurityRequirement{Name: name, Scopes: scopes}
}

// ── @tag（全局标签声明） ───────────────────────────────────────────────────────

// parseTagDecl 解析全局 @tag 声明。
//
// 格式：tagName "description"
func parseTagDecl(value string) TagAnnotation {
	desc, rest := extractLastQuoted(value)
	return TagAnnotation{Name: strings.TrimSpace(rest), Description: desc}
}

// ── @server ───────────────────────────────────────────────────────────────────

// parseServerTag 解析 @server 声明。
//
// 格式：URL "description" 或 URL（无描述）
func parseServerTag(value string) ServerAnnotation {
	desc, rest := extractLastQuoted(value)
	url := strings.TrimSpace(rest)
	if url == "" {
		// 没有引号描述，整个 value 是 URL
		url = strings.TrimSpace(value)
		desc = ""
	}
	return ServerAnnotation{URL: url, Description: desc}
}

// ── @securityDefinitions.* ────────────────────────────────────────────────────

// secDefCtx 是 security definition 的构建上下文。
type secDefCtx struct {
	defs        []SecurityDefAnnotation
	lastIdx     map[string]int // schemeKey → index in defs
	lastFlowIdx map[string]int // schemeKey → flow index in defs[lastIdx].Flows
}

func newSecDefCtx() *secDefCtx {
	return &secDefCtx{
		lastIdx:     make(map[string]int),
		lastFlowIdx: make(map[string]int),
	}
}

// applySecurityDefTag 处理一条 securityDefinitions 标签。
// tagRest 是 "securitydefinitions." 之后的部分（已小写）。
func (ctx *secDefCtx) applyTag(tagRest, value string) {
	schemeKey, attr := parseSecurityDefKey(tagRest)
	if schemeKey == "" {
		return
	}

	if attr == "" {
		// 创建新的安全方案定义
		ctx.createDef(schemeKey, value)
	} else {
		// 更新已有安全方案定义的属性
		ctx.applyAttr(schemeKey, attr, value)
	}
}

// parseSecurityDefKey 解析 schemeKey 和 attr。
// 例："apikey" → ("apikey", "")，"apikey.in" → ("apikey", "in")
// "oauth2.authorizationcode" → ("oauth2.authorizationcode", "")
// "oauth2.authorizationcode.authorizationurl" → ("oauth2.authorizationcode", "authorizationurl")
// "oauth2.authorizationcode.scope.read" → ("oauth2.authorizationcode", "scope.read")
func parseSecurityDefKey(rest string) (schemeKey, attr string) {
	if strings.HasPrefix(rest, "oauth2.") {
		after := rest[len("oauth2."):]
		for _, flow := range []string{"implicit", "password", "clientcredentials", "authorizationcode"} {
			if after == flow {
				return "oauth2." + flow, ""
			}
			if strings.HasPrefix(after, flow+".") {
				return "oauth2." + flow, after[len(flow)+1:]
			}
		}
		return "", "" // 未知 oauth2 flow
	}

	parts := strings.SplitN(rest, ".", 2)
	schemeKey = parts[0]
	if len(parts) == 2 {
		attr = parts[1]
	}
	return
}

func (ctx *secDefCtx) createDef(schemeKey, name string) {
	var def SecurityDefAnnotation
	switch schemeKey {
	case "apikey":
		def = SecurityDefAnnotation{Name: name, Type: "apiKey"}
	case "basic":
		def = SecurityDefAnnotation{Name: name, Type: "http", Scheme: "basic"}
	case "bearer":
		def = SecurityDefAnnotation{Name: name, Type: "http", Scheme: "bearer"}
	case "openidconnect":
		def = SecurityDefAnnotation{Name: name, Type: "openIdConnect"}
	default:
		if strings.HasPrefix(schemeKey, "oauth2.") {
			flow := oauth2FlowType(strings.TrimPrefix(schemeKey, "oauth2."))
			def = SecurityDefAnnotation{
				Name:  name,
				Type:  "oauth2",
				Flows: []OAuthFlowAnnotation{{Type: flow, Scopes: map[string]string{}}},
			}
			ctx.defs = append(ctx.defs, def)
			idx := len(ctx.defs) - 1
			ctx.lastIdx[schemeKey] = idx
			ctx.lastFlowIdx[schemeKey] = 0
			return
		}
		return // 未知 scheme
	}
	ctx.defs = append(ctx.defs, def)
	ctx.lastIdx[schemeKey] = len(ctx.defs) - 1
}

func (ctx *secDefCtx) applyAttr(schemeKey, attr, value string) {
	idx, ok := ctx.lastIdx[schemeKey]
	if !ok || idx >= len(ctx.defs) {
		return
	}
	def := &ctx.defs[idx]

	switch schemeKey {
	case "apikey":
		switch attr {
		case "in":
			def.In = value
		case "name":
			def.KeyName = value
		case "description":
			def.Description = value
		}
	case "basic", "bearer":
		switch attr {
		case "bearerformat":
			def.BearerFormat = value
		case "description":
			def.Description = value
		}
	case "openidconnect":
		switch attr {
		case "openidconnecturl":
			def.OpenIDConnectURL = value
		case "description":
			def.Description = value
		}
	default:
		if !strings.HasPrefix(schemeKey, "oauth2.") {
			return
		}
		flowIdx, ok := ctx.lastFlowIdx[schemeKey]
		if !ok || flowIdx >= len(def.Flows) {
			return
		}
		flow := &def.Flows[flowIdx]
		switch {
		case strings.HasPrefix(attr, "scope."):
			scopeName := attr[len("scope."):]
			if flow.Scopes == nil {
				flow.Scopes = map[string]string{}
			}
			flow.Scopes[scopeName] = strings.Trim(value, `"`)
		case attr == "authorizationurl":
			flow.AuthorizationURL = value
		case attr == "tokenurl":
			flow.TokenURL = value
		case attr == "description":
			def.Description = value
		}
	}
}

// oauth2FlowType 将小写 key 转为规范的 OAuth2 flow 类型字符串。
func oauth2FlowType(key string) string {
	switch key {
	case "implicit":
		return "implicit"
	case "password":
		return "password"
	case "clientcredentials":
		return "clientCredentials"
	case "authorizationcode":
		return "authorizationCode"
	default:
		return key
	}
}
