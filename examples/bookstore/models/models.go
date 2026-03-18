package models

// BookCategory 书籍分类。
type BookCategory string

const (
	BookCategoryProgramming BookCategory = "programming" // 编程
	BookCategoryFiction     BookCategory = "fiction"     // 小说
	BookCategoryScience     BookCategory = "science"     // 科学
	BookCategoryHistory     BookCategory = "history"     // 历史
	BookCategoryOther       BookCategory = "other"       // 其他
)

// Book 书籍信息。
type Book struct {
	ID       int64        `json:"id" example:"1"`
	Title    string       `json:"title" example:"The Go Programming Language" description:"书名" minLength:"1" maxLength:"200"`
	Author   string       `json:"author" example:"Alan Donovan" description:"作者"`
	ISBN     string       `json:"isbn" example:"978-0134190440" description:"ISBN 编号" pattern:"^[0-9-]{10,17}$"`
	Price    float64      `json:"price" example:"59.9" description:"定价（元）" minimum:"0"`
	Category BookCategory `json:"category" example:"programming" description:"分类"`
	InStock  bool         `json:"in_stock" example:"true" description:"是否有货"`
}

// CreateBookRequest 创建书籍请求。
type CreateBookRequest struct {
	Title    string       `json:"title" binding:"required" description:"书名" minLength:"1" maxLength:"200" example:"Clean Code"`
	Author   string       `json:"author" binding:"required" description:"作者" example:"Robert C. Martin"`
	ISBN     string       `json:"isbn" description:"ISBN 编号" example:"978-0132350884"`
	Price    float64      `json:"price" description:"定价（元）" minimum:"0" default:"0" example:"49.9"`
	Category BookCategory `json:"category" description:"分类" default:"other"`
}

// UpdateBookRequest 更新书籍请求（所有字段可选）。
type UpdateBookRequest struct {
	Title    *string       `json:"title,omitempty" description:"书名" minLength:"1" maxLength:"200"`
	Author   *string       `json:"author,omitempty" description:"作者"`
	ISBN     *string       `json:"isbn,omitempty" description:"ISBN 编号"`
	Price    *float64      `json:"price,omitempty" description:"定价（元）" minimum:"0"`
	Category *BookCategory `json:"category,omitempty" description:"分类"`
	InStock  *bool         `json:"in_stock,omitempty" description:"是否有货"`
}

// ErrorResponse 统一错误响应。
type ErrorResponse struct {
	Code    int    `json:"code" example:"400" description:"业务错误码"`
	Message string `json:"message" example:"invalid request" description:"错误信息"`
}
