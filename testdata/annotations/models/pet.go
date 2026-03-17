// Package models 提供宠物商店的核心数据模型。
package models

import "time"

// ── 枚举 ──────────────────────────────────────────────────────────────────────

// PetStatus 宠物状态（string enum）。
type PetStatus string

const (
	PetStatusAvailable PetStatus = "available"
	PetStatusPending   PetStatus = "pending"
	PetStatusSold      PetStatus = "sold"
)

// ── 叶节点 ─────────────────────────────────────────────────────────────────────

// Category 宠物分类。
type Category struct {
	ID   int64  `json:"id"`
	Name string `json:"name" validate:"required" minLength:"1" maxLength:"64"`
}

// Tag 宠物标签。
type Tag struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

// ── 核心模型 ───────────────────────────────────────────────────────────────────

// Pet 宠物。引用 Category（指针）、[]Tag（切片）、PetStatus（enum）和 time.Time（内置外部类型）。
type Pet struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"      validate:"required" minLength:"1" maxLength:"64"`
	PhotoURLs []string  `json:"photo_urls,omitempty"`
	Category  *Category `json:"category,omitempty"`
	Tags      []Tag     `json:"tags,omitempty"`
	Status    PetStatus `json:"status"    enums:"available,pending,sold"`
	CreatedAt time.Time `json:"created_at" readonly:"true"`
}

// ── 请求体 ─────────────────────────────────────────────────────────────────────

// CreatePetRequest 创建宠物请求体。
type CreatePetRequest struct {
	Name      string    `json:"name"      validate:"required" minLength:"1" maxLength:"64"`
	Category  *Category `json:"category,omitempty"`
	Tags      []Tag     `json:"tags,omitempty"`
	Status    PetStatus `json:"status"    enums:"available,pending,sold" default:"available"`
	PhotoURLs []string  `json:"photo_urls,omitempty"`
}

// UpdatePetRequest 更新宠物请求体（所有字段可选）。
type UpdatePetRequest struct {
	Name     string    `json:"name,omitempty"   minLength:"1" maxLength:"64"`
	Status   PetStatus `json:"status,omitempty" enums:"available,pending,sold"`
	Category *Category `json:"category,omitempty"`
}

// ── 响应体 ─────────────────────────────────────────────────────────────────────

// UploadResult 图片上传结果。
type UploadResult struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	URL     string `json:"url"`
}
