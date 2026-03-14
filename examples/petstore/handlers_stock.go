package main

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/shuttlefy/go-openapi3-swag/examples/petstore/bo"
)

// ── Stock handlers ─────────────────────────────────────────────────────────────
//
// This file demonstrates the strict package-qualified annotation syntax
// ("bo.Type") for handler types that live in a separate package.
//
// Dotted-type resolution rules (strict mode, as of this change):
//
//  1. Only a single dot is allowed: "bo.StockItem" is valid;
//     "sql.ddlx.StockPage" is NOT — "sql.ddlx" is not a valid Go identifier.
//
//  2. The package prefix must correspond to an actual scanned source directory.
//     swag3 registers every package name it encounters while walking the input
//     directories.  Because handlers_stock.go imports the "bo" sub-package and
//     the CLI is invoked with both the main dir and the bo dir, "bo" is known.
//
//  3. If the package is not found, or the name has multiple dots, the reference
//     is recorded as an unknown type and a diagnostic warning is emitted.
//     Silent last-segment stripping is intentionally NOT performed.
//
// See bo/types_stock.go for the corresponding Go type definitions.

// listStock handles GET /api/v1/stock.
//
// Demonstrates "bo.Type" for both the composite response and error response.
// The composite expression "bo.StockPage{items=[]bo.StockItem}" uses a single
// known package prefix on both the base type and the override value.
//
// @Summary List stock items
// @Description Returns a paginated inventory list filtered by quantity range or pet status.
// @Tags stock
// @ID listStock
// @Produce json
// @Param min_qty query integer false "minimum available quantity (inclusive)"
// @Param max_qty query integer false "maximum available quantity (inclusive)"
// @Param status  query string  false "pet status filter (available, pending, sold)"
// @Param page    query integer false "1-based page number (default 1)"
// @Param size    query integer false "page size (default 20, max 100)"
// @Success 200 {object} bo.StockPage{items=[]bo.StockItem} "paged stock list"
// @Failure 400 {object} bo.StockError "invalid query parameters"
// @Failure 500 {object} bo.StockError "internal server error"
// @Router /stock [get]
func (h *handler) listStock(c *gin.Context) {
	page, _ := strconv.ParseInt(c.DefaultQuery("page", "1"), 10, 32)
	size, _ := strconv.ParseInt(c.DefaultQuery("size", "20"), 10, 32)
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 20
	}

	h.db.mu.RLock()
	defer h.db.mu.RUnlock()

	statusFilter := c.Query("status")
	minQty, _ := strconv.ParseInt(c.DefaultQuery("min_qty", "0"), 10, 32)
	maxQty, _ := strconv.ParseInt(c.DefaultQuery("max_qty", "2147483647"), 10, 32)

	var items []bo.StockItem
	for _, p := range h.db.pets {
		if statusFilter != "" && string(p.Status) != statusFilter {
			continue
		}
		qty := int32(len(h.db.events[p.ID]))
		if qty < int32(minQty) || qty > int32(maxQty) {
			continue
		}
		items = append(items, bo.StockItem{
			PetID:     p.ID,
			PetName:   p.Name,
			Quantity:  qty,
			Reserved:  0,
			Available: qty,
		})
	}

	total := int64(len(items))
	start := (page - 1) * size
	if start >= total {
		items = []bo.StockItem{}
	} else {
		end := start + size
		if end > total {
			end = total
		}
		items = items[start:end]
	}

	c.JSON(http.StatusOK, okResp(bo.StockPage{
		Total: total,
		Page:  int32(page),
		Size:  int32(size),
		Items: items,
	}))
}

// getStock handles GET /api/v1/stock/:petId.
//
// Both response types use the "bo." prefix: bo.StockItem and bo.StockError.
//
// @Summary Get stock for a pet
// @Tags stock
// @ID getStockByPet
// @Param petId path integer true "pet ID"
// @Success 200 {object} bo.StockItem  "stock record for the pet"
// @Failure 400 {object} bo.StockError "invalid pet ID"
// @Failure 404 {object} bo.StockError "pet not found"
// @Router /stock/{petId} [get]
func (h *handler) getStock(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("petId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, errResp(400, "invalid pet ID"))
		return
	}

	h.db.mu.RLock()
	p, ok := h.db.pets[id]
	evCount := len(h.db.events[id])
	h.db.mu.RUnlock()

	if !ok {
		c.JSON(http.StatusNotFound, errResp(404, "pet not found"))
		return
	}

	qty := int32(evCount)
	c.JSON(http.StatusOK, okResp(bo.StockItem{
		PetID:     p.ID,
		PetName:   p.Name,
		Quantity:  qty,
		Reserved:  0,
		Available: qty,
	}))
}

// adjustStock handles PUT /api/v1/stock/:petId.
//
// Demonstrates "bo.StockAdjustRequest" as a body param — a package-qualified
// type name is used in @Param body exactly like for response types.
//
// @Summary Adjust stock quantity for a pet
// @Tags stock
// @ID adjustStock
// @Accept json
// @Produce json
// @Param petId path integer               true "pet ID"
// @Param body  body bo.StockAdjustRequest true "stock adjustment"
// @Success 200 {object} bo.StockItem      "updated stock record"
// @Failure 400 {object} bo.StockError     "invalid ID or request body"
// @Failure 404 {object} bo.StockError     "pet not found"
// @Security ApiKeyAuth
// @Router /stock/{petId} [put]
func (h *handler) adjustStock(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("petId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, errResp(400, "invalid pet ID"))
		return
	}

	var req bo.StockAdjustRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errResp(400, err.Error()))
		return
	}

	h.db.mu.RLock()
	p, ok := h.db.pets[id]
	evCount := len(h.db.events[id])
	h.db.mu.RUnlock()

	if !ok {
		c.JSON(http.StatusNotFound, errResp(404, "pet not found"))
		return
	}

	qty := int32(evCount) + req.Delta
	if qty < 0 {
		qty = 0
	}
	c.JSON(http.StatusOK, okResp(bo.StockItem{
		PetID:     p.ID,
		PetName:   p.Name,
		Quantity:  qty,
		Reserved:  0,
		Available: qty,
	}))
}
