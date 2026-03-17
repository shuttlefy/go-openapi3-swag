package annotations

import (
	"github.com/shuttlefy/go-openapi3-swag/testdata/annotations/common"
	"github.com/shuttlefy/go-openapi3-swag/testdata/annotations/models"
)

// ═══════════════════════════════════════════════════════════════════════════════
// User 相关接口（v2 新增）
// ═══════════════════════════════════════════════════════════════════════════════

// GetMe 获取当前登录用户资料。
// 返回值引用 models.User，确保 models 包不被工具自动移除。
//
// @Summary  Get current user profile
// @Tags     users
// @Produce  json
// @Success  200 {object} models.User "Current user"
// @Failure  401 {object} models.ErrorResponse "Not authenticated"
// @Security BearerAuth
// @Router   /users/me [get]
func GetMe() (*models.User, error) { return nil, nil }

// UpdateMe 更新当前用户资料（支持 ApiKey 或 Bearer 双重认证）。
//
// @Summary     Update user profile
// @Description Updates the authenticated user's profile. Partial update is supported.
// @ID          updateMe
// @Tags        users
// @Accept      json
// @Produce     json
// @Param       body  body  models.UpdateUserRequest  true  "Profile fields to update"
// @Success     200 {object} models.User "Updated profile"
// @Failure     400 {object} models.ErrorResponse "Validation error"
// @Failure     401 {object} models.ErrorResponse "Not authenticated"
// @Security    ApiKeyAuth
// @Security    BearerAuth
// @Router      /users/me [put]
func UpdateMe() {}

// Login 用户登录，成功时响应头携带 token 元数据。
//
// @Summary  User login
// @Tags     users
// @Accept   json
// @Produce  json
// @Param    body  body  models.LoginRequest  true  "Login credentials"
// @Success  200 {object} models.TokenResponse "Login successful"
// @Header   200 {string} X-Access-Token  "Short-lived access token"
// @Header   200 {string} X-Expires-In   "Token TTL in seconds"
// @Failure  401 {object} models.ErrorResponse "Invalid credentials"
// @Failure  429 {object} models.ErrorResponse "Too many attempts"
// @Router   /users/login [post]
func Login() {}

// Logout 登出（Header 参数示例）。
//
// @Summary  User logout
// @Tags     users
// @Param    X-Session-Token  header  string  true  "Session token to invalidate"
// @Success  204  "Logged out"
// @Failure  401  {object} models.ErrorResponse "Not authenticated"
// @Security BearerAuth
// @Router   /users/logout [post]
func Logout() {}

// ═══════════════════════════════════════════════════════════════════════════════
// Admin 接口（展示多 tag、复合响应、path+query 混合参数）
// ═══════════════════════════════════════════════════════════════════════════════

// AdminListUsers 管理员列出用户（多 tag、Cookie 认证、复合响应类型）。
// 返回值引用 common.PageData，确保 common 包不被工具自动移除。
//
// @Summary  Admin: list users
// @Tags     users, admin
// @Produce  json
// @Param    role    query   string   false "Filter by role" enums(admin,user,guest)
// @Param    active  query   boolean  false "Only active users"
// @Param    X-Admin-Token  header  string  true  "Admin token"
// @Success  200 {object} common.PageData{list=[]models.User} "User list"
// @Failure  403 {object} models.ErrorResponse "Forbidden"
// @Security ApiKeyAuth
// @Router   /admin/users [get]
func AdminListUsers() (*common.PageData, error) { return nil, nil }

// AdminGetUser 管理员按 ID 查看用户（3 段路径参数 + nested 组合响应）。
//
// @Summary  Admin: get user by ID
// @Tags     users, admin
// @Produce  json
// @Param    id    path  integer  true  int64 "User ID"
// @Success  200 {object} common.Response{data=models.User} "User detail"
// @Failure  404 {object} models.ErrorResponse "Not found"
// @Security ApiKeyAuth
// @Router   /admin/users/{id} [get]
func AdminGetUser() {}

// AdminDeleteUser 管理员删除用户（多重 failure，OAuth2 scope）。
//
// @Summary  Admin: delete user
// @Tags     users, admin
// @Param    id    path  integer  true  int64 "User ID"
// @Success  204   "Deleted"
// @Failure  400   {object} models.ErrorResponse "Bad request"
// @Failure  403   {object} models.ErrorResponse "Forbidden"
// @Failure  404   {object} models.ErrorResponse "Not found"
// @Failure  500   {object} models.ErrorResponse "Internal error"
// @Security OAuth2[write]
// @Router   /admin/users/{id} [delete]
func AdminDeleteUser() {}

// ═══════════════════════════════════════════════════════════════════════════════
// 原始类型响应 / 边界场景
// ═══════════════════════════════════════════════════════════════════════════════

// HealthCheck 健康检查（响应为原始字符串）。
//
// @Summary  Health check
// @Produce  plain
// @Success  200 {string} string "ok"
// @Router   /health [get]
func HealthCheck() {}

// GetStats 获取统计数据（响应为数字）。
//
// @Summary  Get stats
// @Produce  json
// @Success  200 {integer} integer "Total pet count"
// @Router   /stats [get]
func GetStats() {}

// SearchPets 全文搜索（Cookie 认证，复合嵌套组合类型）。
//
// @Summary  Search pets
// @Tags     pets
// @Accept   json
// @Produce  json, xml
// @Param    q      query   string   true  "Search keyword"
// @Param    token  cookie  string   false "Optional auth token"
// @Success  200 {object} common.Response{data=common.PageData{list=[]models.Pet}} "Nested composite"
// @Failure  400 {object} models.ErrorResponse "Missing keyword"
// @Router   /pets/search [get]
func SearchPets() {}
