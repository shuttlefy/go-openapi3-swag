package common

// PageData 通用分页响应包装。
type PageData struct {
	Total int         `json:"total" description:"总记录数" example:"100"`
	Page  int         `json:"page" description:"当前页码" example:"1"`
	Size  int         `json:"size" description:"每页条数" example:"20"`
	List  interface{} `json:"list" description:"数据列表"`
}

// Response 通用业务响应包装。
type Response struct {
	Code    int         `json:"code" description:"业务状态码，0 表示成功" example:"0"`
	Message string      `json:"message" description:"响应描述" example:"ok"`
	Data    interface{} `json:"data" description:"响应数据"`
}
