package bo

// ── Stock domain types ────────────────────────────────────────────────────────
//
// This package (bo — business objects) demonstrates how swag3 handles
// package-qualified type references in annotations.
//
// When a handler in package main annotates a response as "bo.StockItem",
// swag3 resolves the reference by:
//  1. Splitting on the first (and only) dot: pkg="bo", type="StockItem".
//  2. Checking that "bo" was registered as a scanned package name.
//  3. Looking up "StockItem" in the schema registry → emit $ref.
//
// Rules enforced by the builder:
//   - Exactly one dot: "bo.StockItem" ✓  "sql.ddlx.StockItem" ✗ (multi-dot)
//   - The package prefix must be a known scanned package.
//   - No silent fallback: unknown prefix → diagnostic error.
//
// See also: handlers_stock.go in the parent package for the annotated handlers.

// StockItem holds the inventory record for a single pet.
// Referenced in annotations as: bo.StockItem
type StockItem struct {
	PetID     int64  `json:"pet_id"              example:"1"`
	PetName   string `json:"pet_name"            example:"Buddy"`
	Quantity  int32  `json:"quantity"            example:"10"  minimum:"0"`
	Reserved  int32  `json:"reserved"            example:"2"   minimum:"0"`
	Available int32  `json:"available"           example:"8"   minimum:"0"  readonly:"true"`
}

// StockAdjustRequest is the request body for adjusting a pet's stock quantity.
// Referenced in annotations as: bo.StockAdjustRequest
type StockAdjustRequest struct {
	Delta  int32  `json:"delta"             binding:"required" example:"5"`
	Reason string `json:"reason,omitempty"                    example:"restock"`
}

// StockFilter holds query-string parameters for the stock list endpoint.
// Referenced in annotations as: bo.StockFilter
type StockFilter struct {
	MinQty int32  `json:"min_qty,omitempty" example:"0"   minimum:"0"`
	MaxQty int32  `json:"max_qty,omitempty" example:"100" minimum:"0"`
	Status string `json:"status,omitempty"  example:"available"`
}

// StockPage is the pagination envelope returned by the stock list endpoint.
// The Items field is overridden per endpoint via the composite annotation:
//
//	@Success 200 {object} bo.StockPage{items=[]bo.StockItem}
//
// Referenced in annotations as: bo.StockPage
type StockPage struct {
	Total int64       `json:"total" example:"42"`
	Page  int32       `json:"page"  example:"1"`
	Size  int32       `json:"size"  example:"20"`
	Items interface{} `json:"items"`
}

// StockError is the error detail payload returned by the stock endpoints.
// Referenced in annotations as: bo.StockError
type StockError struct {
	Code    int    `json:"code"    example:"4001"`
	Message string `json:"message" example:"pet not found"`
	Detail  string `json:"detail,omitempty"`
}
