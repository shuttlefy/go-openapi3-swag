package main

// ── Category domain types ─────────────────────────────────────────────────────
//
// Category is intentionally self-referential (Children []*Category) to
// demonstrate how swag3 handles recursive / cyclic type references.
//
// The schema builder detects the cycle when building the Category schema for
// the second time and emits a $ref instead of expanding infinitely:
//
//	Category:
//	  type: object
//	  properties:
//	    children:
//	      type: array
//	      items:
//	        $ref: '#/components/schemas/Category'   ← cycle resolved via $ref

// Category is a tree node in the product category hierarchy.
// A Category may contain zero or more child Categories, forming an arbitrary-
// depth tree.  The root nodes have no ParentID.
type Category struct {
	ID       int64       `json:"id"                  example:"1"`
	Name     string      `json:"name"                example:"Dogs"`
	ParentID *int64      `json:"parent_id,omitempty" example:"0"` // nil for root categories
	// Children contains the immediate sub-categories.
	// This field creates a recursive schema reference:
	// Category → []Category → Category → …
	// swag3 resolves the cycle by emitting $ref on the second encounter.
	Children []*Category `json:"children,omitempty"`
}

// CreateCategoryRequest is the body for POST /categories.
type CreateCategoryRequest struct {
	Name     string `json:"name"               binding:"required" example:"Dogs"`
	ParentID *int64 `json:"parent_id,omitempty" example:"1"` // attach to an existing parent
}
