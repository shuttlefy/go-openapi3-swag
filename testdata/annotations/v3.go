package annotations

import (
	"github.com/gin-gonic/gin"
)

// ═══════════════════════════════════════════════════════════════════════════════
// 第三方包类型引用示例（gin）
//
// 本文件用于测试 resolver 能否正确解析来自第三方包的类型引用，例如
// github.com/gin-gonic/gin 中的 gin.H（map[string]any）与 gin.Error。
// 函数签名中直接引用这两个类型，确保 gin import 不被工具自动移除。
// ═══════════════════════════════════════════════════════════════════════════════

// GetVersion 返回服务版本信息（响应为 gin.H 灵活 map）。
//
// @Summary  Get service version
// @Tags     meta
// @Produce  json
// @Success  200 {object} gin.H "Version info, e.g. {\"version\":\"1.0.0\",\"env\":\"prod\"}"
// @Router   /version [get]
func GetVersion() gin.H { return gin.H{"version": "1.0.0"} }

// GetDebugInfo 返回调试信息（带错误明细，使用 gin.Error 类型）。
//
// @Summary  Get debug info
// @Tags     meta
// @Produce  json
// @Success  200 {object} gin.H        "Debug key-value map"
// @Failure  500 {object} gin.Error    "Internal error detail"
// @Security ApiKeyAuth
// @Router   /debug [get]
func GetDebugInfo() (*gin.H, *gin.Error) { return nil, nil }

// Echo 回显请求体（body 和响应均为 gin.H）。
//
// @Summary  Echo request body
// @Tags     meta
// @Accept   json
// @Produce  json
// @Param    body  body  gin.H  true  "Any JSON object to echo back"
// @Success  200 {object} gin.H    "Same object echoed"
// @Failure  400 {object} gin.Error "Parse error"
// @Router   /echo [post]
func Echo() {}

// ListErrors 列出最近记录的错误（返回 gin.Error 数组）。
//
// @Summary  List recent errors
// @Tags     meta
// @Produce  json
// @Param    limit  query  integer  false int32 "Max items"
// @Success  200 {array}  gin.Error         "Error list"
// @Failure  403 {object} gin.Error "Forbidden"
// @Security ApiKeyAuth
// @Router   /errors [get]
func ListErrors() ([]*gin.Error, error) { return nil, nil }
