// Package annotations 提供完整的 swag3 注解示例，覆盖所有标签场景。
//
// Resolver 通过当前文件的 import 声明定位注解中的类型引用，
// 因此注解中凡涉及 common.XXX / models.XXX 的文件必须显式 import 对应包。
// 函数签名中直接引用了这两个包的类型，确保 import 不被工具自动移除。
package annotations

import (
	"github.com/shuttlefy/go-openapi3-swag/testdata/annotations/common"
	"github.com/shuttlefy/go-openapi3-swag/testdata/annotations/models"
)

// ── 全局注解（写在 main 函数上） ──────────────────────────────────────────────

// @title           Pet Store API
// @version         1.0.0
// @description     A comprehensive pet store API demonstrating all swag3 annotation features.
// @termsOfService  https://example.com/terms
//
// @contact.name    API Support Team
// @contact.email   support@petstore.example.com
// @contact.url     https://www.petstore.example.com/support
//
// @license.name    Apache 2.0
// @license.url     https://www.apache.org/licenses/LICENSE-2.0.html
//
// @server https://api.petstore.example.com "Production"
// @server https://staging.petstore.example.com "Staging"
// @server http://localhost:8080
//
// @externalDocs.url         https://example.com/docs
// @externalDocs.description Find out more about the Pet Store API
//
// @tag pets    "Everything about your Pets"
// @tag orders  "Access to Pet Store Orders"
// @tag users   "Operations about user"
//
// @securityDefinitions.apikey ApiKeyAuth
// @securityDefinitions.apikey.in header
// @securityDefinitions.apikey.name X-API-Key
// @securityDefinitions.apikey.description API key for protected endpoints
//
// @securityDefinitions.bearer BearerAuth
// @securityDefinitions.bearer.bearerFormat JWT
// @securityDefinitions.bearer.description JWT bearer token
//
// @securityDefinitions.oauth2.authorizationCode OAuth2
// @securityDefinitions.oauth2.authorizationCode.authorizationUrl https://petstore.example.com/oauth/authorize
// @securityDefinitions.oauth2.authorizationCode.tokenUrl https://petstore.example.com/oauth/token
// @securityDefinitions.oauth2.authorizationCode.scope.read "Read access to protected resources"
// @securityDefinitions.oauth2.authorizationCode.scope.write "Write access to protected resources"
func main() {}

// ═══════════════════════════════════════════════════════════════════════════════
// Pet 相关接口
// ═══════════════════════════════════════════════════════════════════════════════

// ListPets 返回分页宠物列表。
//
// 非标签注释行：此函数列举所有满足条件的宠物记录。
//
// @Summary  List all pets
// @Tags     pets
// @Accept   json
// @Produce  json
// @Param    status  query   string  false "Filter by pet status" enums(available,pending,sold)
// @Param    limit   query   integer false int32 "Max records per page"
// @Param    offset  query   integer false int32 "Pagination offset"
// @Success  200 {object} common.PageData{list=[]models.Pet} "Paginated pet list"
// @Header   200 {string} X-Total-Count "Total number of pets"
// @Header   200 {string} X-Request-Id  "Request tracking ID"
// @Failure  400 {object} models.ErrorResponse "Invalid query parameters"
// @Failure  500 {object} models.ErrorResponse "Internal server error"
// @Security ApiKeyAuth
// @Router   /pets [get]
func ListPets() (*common.PageData, error) { return nil, nil }

// CreatePet 创建新宠物。
//
// @Summary     Create a new pet
// @Description Creates a pet with the provided information. Name is required.
// @ID          createPet
// @Tags        pets
// @Accept      json
// @Produce     json
// @Param       body body models.CreatePetRequest true "Pet to create"
// @Success     201 {object} models.Pet "Pet created successfully"
// @Failure     400 {object} models.ErrorResponse "Validation error"
// @Failure     401 {object} models.ErrorResponse "Unauthorized"
// @Security    BearerAuth
// @Security    OAuth2[write]
// @Router      /pets [post]
func CreatePet(req models.CreatePetRequest) (*models.Pet, error) { return nil, nil }

// GetPet 根据 ID 获取单只宠物。
//
// @Summary  Get pet by ID
// @Tags     pets
// @Produce  json
// @Param    id   path  integer  true  int64 "Pet ID"
// @Success  200  {object} models.Pet "Found"
// @Failure  404  {object} models.ErrorResponse "Pet not found"
// @Failure  400  {object} models.ErrorResponse "Invalid ID format"
// @Router   /pets/{id} [get]
func GetPet() {}

// UpdatePet 更新宠物信息。
//
// @Summary  Update pet
// @Tags     pets
// @Accept   json
// @Produce  json
// @Param    id    path  integer               true  int64 "Pet ID"
// @Param    body  body  models.UpdatePetRequest true  "Fields to update"
// @Success  200 {object} models.Pet "Updated pet"
// @Failure  400 {object} models.ErrorResponse "Validation error"
// @Failure  404 {object} models.ErrorResponse "Pet not found"
// @Security ApiKeyAuth
// @Router   /pets/{id} [put]
func UpdatePet() {}

// DeletePet 删除宠物（已弃用，请使用 PATCH /pets/{id}/status）。
//
// @Summary  Delete a pet
// @Tags     pets
// @Param    id   path     integer  true  int64 "Pet ID"
// @Success  204  "No Content"
// @Failure  404  {object} models.ErrorResponse "Pet not found"
// @Failure  403  {object} models.ErrorResponse "Forbidden"
// @Security ApiKeyAuth
// @Deprecated
// @Router   /pets/{id} [delete]
func DeletePet() {}

// UploadPetPhoto 上传宠物照片（multipart/form-data）。
//
// @Summary  Upload pet photo
// @Tags     pets
// @Accept   mpfd
// @Produce  json
// @Param    id       path     integer  true  int64  "Pet ID"
// @Param    file     formData file     true         "Photo file"
// @Param    caption  formData string   false        "Photo caption"
// @Success  200 {object} models.UploadResult "Uploaded"
// @Failure  400 {object} models.ErrorResponse "Invalid file"
// @Security ApiKeyAuth
// @Router   /pets/{id}/upload [post]
func UploadPetPhoto() {}

// ═══════════════════════════════════════════════════════════════════════════════
// Order 相关接口
// ═══════════════════════════════════════════════════════════════════════════════

// ListOrders 列出订单（带多条件过滤）。
//
// @Summary  List orders
// @Tags     orders
// @Produce  json
// @Param    status    query  string   false "Filter by order status"
// @Param    pet_id    query  integer  false int64 "Filter by pet ID"
// @Param    complete  query  boolean  false "Only completed orders"
// @Param    from      query  string   false date-time "Created after (ISO 8601)"
// @Param    to        query  string   false date-time "Created before (ISO 8601)"
// @Success  200 {object} common.PageData{list=[]models.Order} "Order list"
// @Failure  400 {object} models.ErrorResponse "Bad request"
// @Security ApiKeyAuth
// @Router   /store/orders [get]
func ListOrders() {}

// CreateOrder 创建订单。
//
// @Summary  Place a new order
// @Tags     orders
// @Accept   json
// @Produce  json
// @Param    body  body  models.CreateOrderRequest  true  "Order details"
// @Success  201 {object} models.Order "Order placed"
// @Failure  400 {object} models.ErrorResponse "Validation error"
// @Failure  404 {object} models.ErrorResponse "Pet not found"
// @Security BearerAuth
// @Router   /store/orders [post]
func CreateOrder() {}

// GetOrder 根据 ID 获取订单详情。
//
// @Summary  Get order by ID
// @Tags     orders
// @Produce  json
// @Param    id    path  integer  true  int64 "Order ID"
// @Success  200 {object} models.Order "Order found"
// @Failure  404 {object} models.ErrorResponse "Not found"
// @Security ApiKeyAuth
// @Router   /store/orders/{id} [get]
func GetOrder() {}

// CancelOrder 取消订单。
//
// @Summary  Cancel order
// @Tags     orders
// @Param    id    path  integer  true  int64 "Order ID"
// @Success  204   "Cancelled"
// @Failure  404   {object} models.ErrorResponse "Not found"
// @Failure  409   {object} models.ErrorResponse "Cannot cancel delivered order"
// @Security BearerAuth
// @Router   /store/orders/{id} [delete]
func CancelOrder() {}

// ═══════════════════════════════════════════════════════════════════════════════
// 非 @Router 函数（不应被 extractor 当作操作注解）
// ═══════════════════════════════════════════════════════════════════════════════

// initDB 内部初始化函数，无 @Router，应被忽略。
//
// @Summary Should be ignored
func initDB() {}

// helperFunc 无任何注解，应被忽略。
func helperFunc() {}
