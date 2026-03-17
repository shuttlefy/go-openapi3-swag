package extractor

import (
	"fmt"
	"strings"

	"github.com/shuttlefy/go-openapi3-swag/internal/parser"
)

// GoExtractor 从 []*parser.RawFile 中提取结构化 API 注解。
type GoExtractor struct{}

// Extract 遍历所有 RawFile，提取全局注解和操作注解。
//
// 全局注解：包含 @title / @version 等全局标签的函数（通常是 main）。
// 操作注解：包含 @Router 标签的函数，缺少 @Router 则忽略。
func (e *GoExtractor) Extract(files []*parser.RawFile) (*ExtractResult, error) {
	result := &ExtractResult{}
	secCtx := newSecDefCtx()

	for _, rf := range files {
		for _, fn := range rf.Functions {
			if len(fn.Comments) == 0 {
				continue
			}
			tags, plain := parseCommentLines(fn.Comments)

			// 仅在非操作函数（无 @Router）时应用全局标签，避免操作级 @description 污染 info.description
			if route, ok := findRouterTag(tags); ok {
				op, err := buildOperation(fn, tags, plain, route)
				if err != nil {
					return nil, fmt.Errorf("%s:%d: %w", rf.FilePath, fn.Line, err)
				}
				result.Operations = append(result.Operations, op)
			} else {
				applyGlobalTags(tags, &result.Global, secCtx)
			}
		}
	}

	// 将构建好的 security defs 写回 Global
	result.Global.SecurityDefs = secCtx.defs
	return result, nil
}

// ── 全局注解 ──────────────────────────────────────────────────────────────────

// applyGlobalTags 将 global 级别的 tagLine 应用到 GlobalAnnotation。
func applyGlobalTags(tags []tagLine, global *GlobalAnnotation, secCtx *secDefCtx) {
	for _, tag := range tags {
		switch tag.name {
		case "title":
			global.Title = tag.value
		case "version":
			global.Version = tag.value
		case "description":
			if global.Description == "" {
				global.Description = tag.value
			} else {
				global.Description += "\n" + tag.value
			}
		case "termsofservice":
			global.TermsOfService = tag.value

		// contact.*
		case "contact.name":
			global.Contact.Name = tag.value
		case "contact.url":
			global.Contact.URL = tag.value
		case "contact.email":
			global.Contact.Email = tag.value

		// license.*
		case "license.name":
			global.License.Name = tag.value
		case "license.url":
			global.License.URL = tag.value

		// externalDocs.*
		case "externaldocs.url":
			global.ExternalDocs.URL = tag.value
		case "externaldocs.description":
			global.ExternalDocs.Description = tag.value

		// @server
		case "server":
			global.Servers = append(global.Servers, parseServerTag(tag.value))

		// swaggo 兼容
		case "host":
			global.Host = tag.value
		case "basepath":
			global.BasePath = tag.value
		case "schemes":
			for _, s := range strings.Fields(tag.value) {
				global.Schemes = append(global.Schemes, s)
			}

		// @tag 全局标签声明
		case "tag":
			global.Tags = append(global.Tags, parseTagDecl(tag.value))

		// @securityDefinitions.*
		default:
			if strings.HasPrefix(tag.name, "securitydefinitions.") {
				rest := tag.name[len("securitydefinitions."):]
				secCtx.applyTag(rest, tag.value)
			}
		}
	}
}

// ── 操作注解 ──────────────────────────────────────────────────────────────────

// findRouterTag 在 tags 中寻找 @Router，找到则返回 (RouteInfo, true)。
func findRouterTag(tags []tagLine) (RouteInfo, bool) {
	for _, tag := range tags {
		if tag.name == "router" {
			route, err := parseRouterTag(tag.value)
			if err == nil {
				return route, true
			}
		}
	}
	return RouteInfo{}, false
}

// buildOperation 将函数信息和已解析的 tags 组装为 OperationAnnotation。
func buildOperation(
	fn parser.RawFunc,
	tags []tagLine,
	plainLines []string,
	route RouteInfo,
) (OperationAnnotation, error) {
	op := OperationAnnotation{
		FuncName: fn.Name,
		FilePath: fn.FilePath,
		Line:     fn.Line,
		Route:    route,
	}

	var descLines []string

	for _, tag := range tags {
		switch tag.name {
		case "summary":
			op.Summary = tag.value
		case "description":
			descLines = append(descLines, tag.value)
		case "id":
			op.OperationID = tag.value
		case "tags":
			op.Tags = splitTags(tag.value)
		case "accept":
			op.Accept = parseMIMETypes(tag.value)
		case "produce":
			op.Produce = parseMIMETypes(tag.value)
		case "deprecated":
			op.Deprecated = true

		case "param":
			p, err := parseParamTag(tag.value)
			if err != nil {
				return op, fmt.Errorf("@Param %w", err)
			}
			op.Params = append(op.Params, p)

		case "success", "failure":
			resp, err := parseResponseTag(tag.value)
			if err != nil {
				return op, fmt.Errorf("@%s %w", tag.name, err)
			}
			op.Responses = append(op.Responses, resp)

		case "header":
			h, err := parseHeaderTag(tag.value)
			if err != nil {
				return op, fmt.Errorf("@Header %w", err)
			}
			op.Headers = append(op.Headers, h)

		case "security":
			op.Security = append(op.Security, parseSecurityTag(tag.value))
		}
	}

	// Description 优先用 @Description；无则回退到普通注释行
	if len(descLines) > 0 {
		op.Description = strings.Join(descLines, "\n")
	} else if len(plainLines) > 0 {
		op.Description = strings.Join(plainLines, "\n")
	}

	return op, nil
}

// splitTags 拆分逗号分隔的标签列表，去除空白。
func splitTags(value string) []string {
	var result []string
	for _, s := range strings.Split(value, ",") {
		s = strings.TrimSpace(s)
		if s != "" {
			result = append(result, s)
		}
	}
	return result
}
