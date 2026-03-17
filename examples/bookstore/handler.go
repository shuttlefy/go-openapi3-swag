package main

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/shuttlefy/go-openapi3-swag/examples/bookstore/common"
	"github.com/shuttlefy/go-openapi3-swag/examples/bookstore/models"
)

// listBooks 返回分页书籍列表。
//
// @Summary  列出书籍
// @Tags     books
// @Accept   json
// @Produce  json
// @Param    category  query   string   false "按分类过滤" enums(programming,fiction,science,history,other)
// @Param    in_stock  query   boolean  false "仅显示有货"
// @Param    page      query   integer  false int32 "页码（从 1 开始）"
// @Param    size      query   integer  false int32 "每页条数（默认 20）"
// @Success  200 {object} common.PageData{list=[]models.Book} "书籍分页列表"
// @Failure  400 {object} models.ErrorResponse "请求参数错误"
// @Failure  500 {object} models.ErrorResponse "服务器内部错误"
// @Security ApiKeyAuth
// @Router   /books [get]
func listBooks(c *gin.Context) {
	// 示例：直接返回空列表
	c.JSON(http.StatusOK, common.PageData{Total: 0, Page: 1, Size: 20, List: []models.Book{}})
}

// createBook 创建新书籍。
//
// @Summary     创建书籍
// @Description 创建一条新书记录，书名（title）和作者（author）为必填。
// @ID          createBook
// @Tags        books
// @Accept      json
// @Produce     json
// @Param       body  body  models.CreateBookRequest  true  "书籍信息"
// @Success     201 {object} models.Book "创建成功"
// @Failure     400 {object} models.ErrorResponse "请求体校验失败"
// @Failure     401 {object} models.ErrorResponse "未授权"
// @Security    BearerAuth
// @Router      /books [post]
func createBook(c *gin.Context) {
	var req models.CreateBookRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Code: 400, Message: err.Error()})
		return
	}
	book := models.Book{ID: 1, Title: req.Title, Author: req.Author, Price: req.Price}
	c.JSON(http.StatusCreated, book)
}

// getBook 根据 ID 获取书籍详情。
//
// @Summary  获取书籍
// @Tags     books
// @Produce  json
// @Param    id   path  integer  true  int64 "书籍 ID"
// @Success  200 {object} models.Book "书籍详情"
// @Failure  404 {object} models.ErrorResponse "书籍不存在"
// @Failure  400 {object} models.ErrorResponse "无效的 ID 格式"
// @Router   /books/{id} [get]
func getBook(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Code: 400, Message: "invalid id"})
		return
	}
	c.JSON(http.StatusOK, models.Book{ID: id, Title: "Example Book", Author: "Author", InStock: true})
}

// updateBook 更新书籍信息。
//
// @Summary  更新书籍
// @Tags     books
// @Accept   json
// @Produce  json
// @Param    id    path  integer                   true  int64 "书籍 ID"
// @Param    body  body  models.UpdateBookRequest  true  "待更新字段"
// @Success  200 {object} models.Book "更新后的书籍"
// @Failure  400 {object} models.ErrorResponse "请求参数错误"
// @Failure  404 {object} models.ErrorResponse "书籍不存在"
// @Security ApiKeyAuth
// @Router   /books/{id} [put]
func updateBook(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Code: 400, Message: "invalid id"})
		return
	}
	var req models.UpdateBookRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Code: 400, Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, models.Book{ID: id})
}

// deleteBook 删除书籍。
//
// @Summary  删除书籍
// @Tags     books
// @Param    id   path  integer  true  int64 "书籍 ID"
// @Success  204  "已删除"
// @Failure  404 {object} models.ErrorResponse "书籍不存在"
// @Failure  403 {object} models.ErrorResponse "无删除权限"
// @Security ApiKeyAuth
// @Router   /books/{id} [delete]
func deleteBook(c *gin.Context) {
	_, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Code: 400, Message: "invalid id"})
		return
	}
	c.Status(http.StatusNoContent)
}
