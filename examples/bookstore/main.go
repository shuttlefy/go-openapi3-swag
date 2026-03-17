// Package main 书店 API 示例。
//
// 演示如何将 swag3 生成的 OpenAPI 规范通过 pkg/swaggin 挂载到 Gin 路由，
// 同时提供 Swagger UI 和 Redoc 两种文档界面。
//
// # 生成 OpenAPI 规范
//
//	swag3 -dirs . -output docs/openapi.json
//
// # 启动服务
//
//	go run .
//
// 启动后访问：
//   - Swagger UI : http://localhost:8080/docs
//   - Redoc      : http://localhost:8080/redoc
//   - Raw JSON   : http://localhost:8080/openapi.json
package main

import (
	"github.com/gin-gonic/gin"
	"github.com/shuttlefy/go-openapi3-swag/examples/bookstore/common"
	"github.com/shuttlefy/go-openapi3-swag/examples/bookstore/models"
	"github.com/shuttlefy/go-openapi3-swag/pkg/swaggin"
)

// ── 全局注解 ──────────────────────────────────────────────────────────────────

// @title           Bookstore API
// @version         1.0.0
// @description     A simple bookstore management API demonstrating swag3 + swaggin integration.
//
// @contact.name    Bookstore Support
// @contact.email   support@bookstore.example.com
//
// @license.name    MIT
// @license.url     https://opensource.org/licenses/MIT
//
// @server http://localhost:9999 "Local development"
// @server https://api.bookstore.example.com "Production"
//
// @tag books  "书籍管理接口"
//
// @securityDefinitions.apikey ApiKeyAuth
// @securityDefinitions.apikey.in     header
// @securityDefinitions.apikey.name   X-API-Key
// @securityDefinitions.apikey.description API key 认证，写入请求头 X-API-Key
//
// @securityDefinitions.bearer BearerAuth
// @securityDefinitions.bearer.bearerFormat JWT
// @securityDefinitions.bearer.description JWT Bearer Token 认证
func main() {
	r := gin.Default()

	// ── 业务路由 ──────────────────────────────────────────────────────────────
	v1 := r.Group("/books")
	{
		v1.GET("", listBooks)
		v1.POST("", createBook)
		v1.GET("/:id", getBook)
		v1.PUT("/:id", updateBook)
		v1.DELETE("/:id", deleteBook)
	}

	// ── OpenAPI 文档路由 ──────────────────────────────────────────────────────
	// swaggin.Register 挂载以下路由：
	//   GET /openapi.json  →  原始 JSON spec（由 swag3 生成）
	//   GET /docs          →  Swagger UI
	//   GET /redoc         →  Redoc（同时注册两种 UI）
	swaggin.Register(r, swaggin.Options{
		SpecFile:  "docs/openapi.json",
		Title:     "Bookstore API",
		JSONPath:  "/openapi.json",
		UIPath:    "/docs",
		RedocPath: "/redoc",
		AllowCORS: true,
	})

	r.Run(":9999")
}

// ── 确保 import 不被移除（swag3 通过文件 import 定位注解中的类型引用） ──────────
var _ common.PageData
var _ models.Book
