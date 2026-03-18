package models

// UserRole 用户角色（string enum）。
type UserRole string

const (
	UserRoleAdmin UserRole = "admin" // 管理员
	UserRoleUser  UserRole = "user"  // 普通用户
	UserRoleGuest UserRole = "guest" // 访客
)

// Address 地址。
type Address struct {
	Street  string `json:"street"`
	City    string `json:"city"    validate:"required"`
	Country string `json:"country" validate:"required"`
	Zip     string `json:"zip"     pattern:"^\\d{6}$"`
}

// User 用户。引用 UserRole（enum）和 Address（嵌套 struct）。
type User struct {
	ID      int64    `json:"id"`
	Name    string   `json:"name"  validate:"required" minLength:"2" maxLength:"64"`
	Email   string   `json:"email" validate:"required,email" format:"email"`
	Role    UserRole `json:"role"  enums:"admin,user,guest"`
	Address *Address `json:"address,omitempty"`
}

// UpdateUserRequest 更新用户资料请求体。
type UpdateUserRequest struct {
	Name    string   `json:"name,omitempty"  minLength:"2" maxLength:"64"`
	Email   string   `json:"email,omitempty" format:"email"`
	Role    UserRole `json:"role,omitempty"  enums:"admin,user,guest"`
	Address *Address `json:"address,omitempty"`
}

// LoginRequest 登录请求体。
type LoginRequest struct {
	Email    string `json:"email"    validate:"required,email"`
	Password string `json:"password" validate:"required" writeonly:"true" minLength:"8"`
}

// TokenResponse 登录成功响应。
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in" description:"Token 有效期（秒）"`
}
