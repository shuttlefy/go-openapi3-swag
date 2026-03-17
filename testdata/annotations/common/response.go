package common

type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

type PageData struct {
	Total int         `json:"total"`
	Page  int         `json:"page"`
	List  interface{} `json:"list"`
}
