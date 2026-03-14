// Package main is a runnable Pet Store API server that demonstrates the full
// swag3 workflow end-to-end:
//
//  1. Annotations in [testdata/petstore/petstore.go] drive spec generation.
//  2. swag3 CLI converts those annotations into [openapi.json].
//  3. [pkg/swaggin] serves the spec and interactive UIs at runtime via Gin.
//
// # Quick start
//
//	# Step 1 – generate the spec (run from project root)
//	go run ./cmd/swag3/ -dir testdata/petstore -output openapi.json
//
//	# Step 2 – start the server
//	go run ./examples/petstore/
//
// # Endpoints
//
//	GET  /openapi.json          raw OpenAPI 3 JSON spec
//	GET  /docs                  Swagger UI  (interactive Try-it-out)
//	GET  /redoc                 Redoc       (clean three-panel docs)
//	GET  /healthz               liveness probe
//
//	GET    /api/v1/pets                  list pets (paginated)
//	POST   /api/v1/pets                  create a pet
//	POST   /api/v1/pets/search           advanced keyword search
//	GET    /api/v1/pets/:id              get pet by ID
//	PUT    /api/v1/pets/:id              replace a pet
//	DELETE /api/v1/pets/:id              delete a pet
//	GET    /api/v1/pets/:id/events       list lifecycle events for a pet
//	POST   /api/v1/pets/:id/avatar       upload pet avatar (multipart/form-data)
//	GET    /api/v1/pets/:id/avatar       download pet avatar (binary)
//	POST   /api/v1/pets/import           batch-import pets (function-local struct)
//
//	POST   /api/v1/orders                create order (nested struct + time.Time)
//	GET    /api/v1/orders/:id            get order by ID
//
//	POST   /api/v1/categories            create category (recursive self-ref)
//	GET    /api/v1/categories/:id        get category tree
//
//	GET    /api/v1/stock                 list stock (bo.xx / sql.ddlx.xx annotation demo)
//	GET    /api/v1/stock/:petId          get stock for a pet
//	PUT    /api/v1/stock/:petId          adjust stock quantity
//
// # swag3 features exercised
//
//   - Nested composite pagination @Success 200 {object} BaseResponse{data=PagedList{list=[]Pet}}
//   - Function-local struct       SearchFilter / ImportItem defined inside handler bodies
//   - stdlib time.Time            Pet.CreatedAt / Order.PlacedAt → date-time
//   - map[string]string           PetEvent.Metadata → additionalProperties
//   - Nullable pointer            *time.Time / *Address → nullable: true
//   - Deep nesting                CreateOrderRequest → Address + []OrderItem
//   - Recursive reference         Category.Children []*Category → cycle via $ref
//   - Cross-file struct           Address/OrderItem in types_order.go, referenced by main.go annotations
//   - Dotted type references      bo.StockItem / sql.ddlx.StockPage in stock handlers (types_stock.go)
//
// # Configuration
//
// Set the PORT environment variable to override the default port 8088:
//
//	PORT=9000 go run ./examples/petstore/
package main

import (
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/shuttlefy/go-openapi3-swag/pkg/swaggin"
)

// ── Domain types ─────────────────────────────────────────────────────────────
//
// These types mirror the annotated declarations in testdata/petstore/petstore.go.
// swag3 reads that file to generate the schema; this file contains the actual
// runtime implementations used by the Gin handlers.

// Pet represents an animal available in the store.
//
// swag3 mapping notes:
//   - CreatedAt time.Time  → OpenAPI string, format: date-time
//   - UpdatedAt *time.Time → same, plus nullable: true (pointer = optional update)
type Pet struct {
	ID   int64  `json:"id"                    example:"1"`
	Name string `json:"name"                  example:"Buddy"`
	// Status uses the PetStatus alias type so the generated schema references
	// the named enum in components/schemas instead of inlining a bare string.
	Status    PetStatus  `json:"status"                example:"available"`
	Tags      []PetTag   `json:"tags,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt *time.Time `json:"updated_at,omitempty"` // nil until first update
}

// CreatePetRequest is the JSON body for POST /pets and PUT /pets/:id.
// binding:"required" tags are enforced by gin's ShouldBindJSON.
type CreatePetRequest struct {
	Name   string    `json:"name"   binding:"required" example:"Buddy"`
	Status PetStatus `json:"status" binding:"required" example:"available"`
	Tags   []PetTag  `json:"tags,omitempty"`
}

// BaseResponse is the universal response envelope for every API endpoint.
//
// Successful responses carry code=0; application errors carry a non-zero code
// so that clients can differentiate HTTP-level errors from business errors
// without inspecting the HTTP status code alone.
//
// In the generated spec, the Data field is overridden per endpoint using the
// composite annotation syntax, e.g.:
//
//	@Success 200 {object} BaseResponse{data=Pet}
//	@Success 200 {object} BaseResponse{data=PagedList{list=[]Pet}}
type BaseResponse struct {
	Code    int         `json:"code"             example:"0"`
	Message string      `json:"message"          example:"ok"`
	Data    interface{} `json:"data,omitempty"` // absent on error / 204 responses
}

// PagedList is the pagination payload carried inside BaseResponse.Data for
// list and search endpoints.
//
// The List field holds the concrete item slice (e.g. []*Pet). In annotations
// the concrete type is always specified via the nested composite syntax:
//
//	@Success 200 {object} BaseResponse{data=PagedList{list=[]Pet}}
type PagedList struct {
	Total int64       `json:"total" example:"42"` // total records matching the query (before paging)
	List  interface{} `json:"list"`               // concrete item slice for this page
}

// PetEvent records a single lifecycle event that occurred to a pet.
//
// swag3 mapping notes:
//   - OccurredAt time.Time         → string, format: date-time
//   - Metadata   map[string]string → object with additionalProperties: {type: string}
type PetEvent struct {
	ID    int64 `json:"id"                  example:"1"`
	PetID int64 `json:"pet_id"              example:"1"`
	// EventType uses the EventType alias so the enum values are centralised.
	EventType  EventType         `json:"event_type"          example:"created"`
	Detail     string            `json:"detail,omitempty"    example:"pet was created via API"`
	OccurredAt time.Time         `json:"occurred_at"`
	Metadata   map[string]string `json:"metadata,omitempty"` // arbitrary key-value context
}

// UploadResult carries metadata about a successfully uploaded file,
// returned in the BaseResponse.Data field after a file upload.
type UploadResult struct {
	Filename    string `json:"filename"     example:"avatar.jpg"`            // original filename as provided by the client
	Size        int64  `json:"size"         example:"204800"`                // file size in bytes
	ContentType string `json:"content_type" example:"image/jpeg"`            // detected MIME type
	URL         string `json:"url"          example:"/api/v1/pets/1/avatar"` // relative URL to download the file
}

// ── Response helpers ──────────────────────────────────────────────────────────

// okResp wraps data in a successful BaseResponse (code=0, message="ok").
func okResp(data interface{}) BaseResponse {
	return BaseResponse{Code: 0, Message: "ok", Data: data}
}

// errResp wraps an application error in a BaseResponse (no Data field).
func errResp(code int, message string) BaseResponse {
	return BaseResponse{Code: code, Message: message}
}

// ── In-memory store ───────────────────────────────────────────────────────────
//
// store is a thread-safe in-memory repository used instead of a real database
// to keep the example self-contained.  It is not production-quality; it exists
// only to give the handlers realistic data to work with.

// avatarFile holds the raw bytes and metadata of an uploaded avatar image.
// All files are kept in memory to avoid any filesystem dependency in the example.
type avatarFile struct {
	Filename    string
	Content     []byte
	ContentType string
}

// store holds all pets, orders, categories, events, and avatar images.
type store struct {
	mu         sync.RWMutex
	pets       map[int64]*Pet
	events     map[int64][]*PetEvent // keyed by pet ID
	avatars    map[int64]*avatarFile // keyed by pet ID; at most one avatar per pet
	orders     map[int64]*Order
	categories map[int64]*Category
	next       int64 // next pet ID
	evNext     int64 // next event ID
	orderNext  int64 // next order ID
	catNext    int64 // next category ID
}

// newStore creates a store pre-populated with three sample pets and seed events.
func newStore() *store {
	now := time.Now()
	s := &store{
		pets:       make(map[int64]*Pet),
		events:     make(map[int64][]*PetEvent),
		avatars:    make(map[int64]*avatarFile),
		orders:     make(map[int64]*Order),
		categories: make(map[int64]*Category),
		next:       4,
		evNext:     3,
		orderNext:  1,
		catNext:    1,
	}

	s.pets[1] = &Pet{ID: 1, Name: "Buddy", Status: PetStatusAvailable, Tags: []PetTag{"dog"}, CreatedAt: now.Add(-72 * time.Hour)}
	s.pets[2] = &Pet{ID: 2, Name: "Whiskers", Status: PetStatusPending, Tags: []PetTag{"cat"}, CreatedAt: now.Add(-48 * time.Hour)}
	s.pets[3] = &Pet{ID: 3, Name: "Goldie", Status: PetStatusSold, Tags: []PetTag{"fish"}, CreatedAt: now.Add(-24 * time.Hour)}

	// Seed events for Buddy so GET /pets/1/events returns something useful.
	s.events[1] = []*PetEvent{
		{ID: 1, PetID: 1, EventType: EventTypeCreated, OccurredAt: now.Add(-72 * time.Hour), Metadata: map[string]string{"source": "web"}},
		{ID: 2, PetID: 1, EventType: EventTypeViewed, OccurredAt: now.Add(-1 * time.Hour)},
	}
	return s
}

// listPage returns a page of pets matching the given status filter.
// An empty status string returns all pets.
func (s *store) listPage(status PetStatus, page, perPage int64) (total int64, items []*Pet) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var all []*Pet
	for _, p := range s.pets {
		if status == "" || p.Status == status {
			all = append(all, p)
		}
	}

	total = int64(len(all))
	start := (page - 1) * perPage
	if start >= total {
		return total, []*Pet{}
	}
	end := start + perPage
	if end > total {
		end = total
	}
	return total, all[start:end]
}

// search returns pets that satisfy ALL of the given criteria:
//   - name contains every keyword (case-insensitive, AND logic)
//   - status is one of the requested statuses (OR logic; empty = any)
//   - tags contains every requested tag (AND logic; empty = any)
func (s *store) search(keywords, statuses, tags []string) []*Pet {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var out []*Pet
outer:
	for _, p := range s.pets {
		// Status filter (OR: pet must match at least one of the given statuses).
		if len(statuses) > 0 && !containsStr(statuses, string(p.Status)) {
			continue
		}
		// Keyword filter (AND: every keyword must appear in the name).
		for _, kw := range keywords {
			if !strings.Contains(strings.ToLower(p.Name), strings.ToLower(kw)) {
				continue outer
			}
		}
		// Tag filter (AND: pet must carry every requested tag).
		for _, tag := range tags {
			if !containsPetTag(p.Tags, PetTag(tag)) {
				continue outer
			}
		}
		out = append(out, p)
	}
	return out
}

// get returns the pet with the given ID, or (nil, false) if not found.
func (s *store) get(id int64) (*Pet, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	p, ok := s.pets[id]
	return p, ok
}

// create persists a new pet and appends a "created" event to its event log.
func (s *store) create(req CreatePetRequest) *Pet {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	p := &Pet{
		ID:        s.next,
		Name:      req.Name,
		Status:    req.Status,
		Tags:      req.Tags,
		CreatedAt: now,
	}
	s.pets[p.ID] = p
	s.events[p.ID] = append(s.events[p.ID], &PetEvent{
		ID:         s.evNext,
		PetID:      p.ID,
		EventType:  "created",
		OccurredAt: now,
		Metadata:   map[string]string{"source": "api"},
	})
	s.evNext++
	s.next++
	return p
}

// update replaces an existing pet's fields and appends an "updated" event.
// Returns (nil, false) when the ID does not exist.
func (s *store) update(id int64, req CreatePetRequest) (*Pet, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	p, ok := s.pets[id]
	if !ok {
		return nil, false
	}
	now := time.Now()
	p.Name, p.Status, p.Tags, p.UpdatedAt = req.Name, req.Status, req.Tags, &now
	s.events[p.ID] = append(s.events[p.ID], &PetEvent{
		ID: s.evNext, PetID: p.ID, EventType: "updated", OccurredAt: now,
	})
	s.evNext++
	return p, true
}

// delete removes a pet by ID. Returns false when the ID does not exist.
// Note: events are intentionally retained (audit trail).
func (s *store) delete(id int64) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.pets[id]; !ok {
		return false
	}
	delete(s.pets, id)
	return true
}

// listEvents returns all events for a pet, optionally filtered by eventType.
// The second return value is false when the pet ID does not exist.
func (s *store) listEvents(id int64, eventType string) ([]*PetEvent, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if _, ok := s.pets[id]; !ok {
		return nil, false
	}
	evs := s.events[id]
	if eventType == "" {
		return evs, true
	}
	et := EventType(eventType)
	var out []*PetEvent
	for _, e := range evs {
		if e.EventType == et {
			out = append(out, e)
		}
	}
	return out, true
}

// saveAvatar stores an uploaded avatar for the given pet ID, replacing any
// previous upload.  Returns false when the pet does not exist.
func (s *store) saveAvatar(id int64, af *avatarFile) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.pets[id]; !ok {
		return false
	}
	s.avatars[id] = af
	return true
}

// loadAvatar retrieves the stored avatar for the given pet ID.
// Returns (nil, false) when the pet has no avatar or does not exist.
func (s *store) loadAvatar(id int64) (*avatarFile, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	af, ok := s.avatars[id]
	return af, ok
}

// createOrder persists a new order and returns it.
func (s *store) createOrder(req CreateOrderRequest) *Order {
	s.mu.Lock()
	defer s.mu.Unlock()
	o := &Order{
		ID:              s.orderNext,
		CustomerName:    req.CustomerName,
		ShippingAddress: req.ShippingAddress,
		BillingAddress:  req.BillingAddress,
		Items:           req.Items,
		Status:          "pending",
		Note:            req.Note,
		PlacedAt:        req.PlacedAt,
	}
	if o.PlacedAt.IsZero() {
		o.PlacedAt = time.Now()
	}
	s.orders[o.ID] = o
	s.orderNext++
	return o
}

// getOrder retrieves an order by ID, or (nil, false) if not found.
func (s *store) getOrder(id int64) (*Order, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	o, ok := s.orders[id]
	return o, ok
}

// createCategory adds a category node to the tree.
// When ParentID is set, the new node is appended to that parent's Children slice.
func (s *store) createCategory(req CreateCategoryRequest) (*Category, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if req.ParentID != nil {
		if _, ok := s.categories[*req.ParentID]; !ok {
			return nil, false // parent does not exist
		}
	}
	c := &Category{ID: s.catNext, Name: req.Name, ParentID: req.ParentID}
	s.categories[c.ID] = c
	if req.ParentID != nil {
		parent := s.categories[*req.ParentID]
		parent.Children = append(parent.Children, c)
	}
	s.catNext++
	return c, true
}

// getCategory retrieves a category (with its Children tree) by ID.
func (s *store) getCategory(id int64) (*Category, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	c, ok := s.categories[id]
	return c, ok
}

// containsStr reports whether s is present in slice.
func containsStr(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}

// containsPetTag reports whether tag is present in a PetTag slice.
func containsPetTag(slice []PetTag, tag PetTag) bool {
	for _, v := range slice {
		if v == tag {
			return true
		}
	}
	return false
}

// ── HTTP handlers ─────────────────────────────────────────────────────────────
//
// handler wraps a *store and exposes one method per API operation.
// Each method signature matches what gin expects: func(*gin.Context).

// handler holds shared dependencies for all request handlers.
type handler struct{ db *store }

// listPets handles GET /api/v1/pets.
//
// Query params:
//   - status   optional status filter
//   - page     1-based page number (default 1)
//   - per_page page size, clamped to [1, 100] (default 20)
//
// @Summary List pets (paginated)
// @Tags pets
// @ID listPets
// @Param status   query string  false "filter by status (available, pending, sold)"
// @Param page     query integer false "page number (1-based, default 1)"
// @Param per_page query integer false "items per page (default 20, max 100)"
// @Success 200 {object} BaseResponse{data=PagedList{list=[]Pet}} "paginated pet list"
// @Failure 500 {object} BaseResponse "internal server error"
// @Router /pets [get]
func (h *handler) listPets(c *gin.Context) {
	status := c.Query("status")
	page, _ := strconv.ParseInt(c.DefaultQuery("page", "1"), 10, 64)
	perPage, _ := strconv.ParseInt(c.DefaultQuery("per_page", "20"), 10, 64)

	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}

	total, items := h.db.listPage(PetStatus(status), page, perPage)
	if items == nil {
		items = []*Pet{}
	}
	c.JSON(http.StatusOK, okResp(PagedList{Total: total, List: items}))
}

// getPet handles GET /api/v1/pets/:id.
//
// @Summary Get a pet by ID
// @Tags pets
// @ID getPet
// @Param id path integer true "pet ID"
// @Success 200 {object} BaseResponse{data=Pet} "the requested pet"
// @Failure 400 {object} BaseResponse           "invalid id"
// @Failure 404 {object} BaseResponse           "pet not found"
// @Router /pets/{id} [get]
func (h *handler) getPet(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, errResp(400, "invalid id"))
		return
	}
	pet, found := h.db.get(id)
	if !found {
		c.JSON(http.StatusNotFound, errResp(404, "pet not found"))
		return
	}
	c.JSON(http.StatusOK, okResp(pet))
}

// createPet handles POST /api/v1/pets.
//
// @Summary Create a new pet
// @Tags pets
// @ID createPet
// @Param body body CreatePetRequest true "pet to create"
// @Success 201 {object} BaseResponse{data=Pet} "created pet"
// @Failure 400 {object} BaseResponse           "invalid request body"
// @Security ApiKeyAuth
// @Router /pets [post]
func (h *handler) createPet(c *gin.Context) {
	var req CreatePetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errResp(400, err.Error()))
		return
	}
	c.JSON(http.StatusCreated, okResp(h.db.create(req)))
}

// updatePet handles PUT /api/v1/pets/:id.
//
// @Summary Update an existing pet
// @Tags pets
// @ID updatePet
// @Param id   path integer          true "pet ID"
// @Param body body CreatePetRequest true "updated pet data"
// @Success 200 {object} BaseResponse{data=Pet} "updated pet"
// @Failure 400 {object} BaseResponse           "invalid id or request body"
// @Failure 404 {object} BaseResponse           "pet not found"
// @Security ApiKeyAuth
// @Router /pets/{id} [put]
func (h *handler) updatePet(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, errResp(400, "invalid id"))
		return
	}
	var req CreatePetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errResp(400, err.Error()))
		return
	}
	pet, found := h.db.update(id, req)
	if !found {
		c.JSON(http.StatusNotFound, errResp(404, "pet not found"))
		return
	}
	c.JSON(http.StatusOK, okResp(pet))
}

// deletePet handles DELETE /api/v1/pets/:id.
// Returns 204 No Content on success; no response body is emitted.
//
// @Summary Delete a pet
// @Tags pets
// @ID deletePet
// @Param id path integer true "pet ID"
// @Success 204 "pet deleted (no body)"
// @Failure 400 {object} BaseResponse "invalid id"
// @Failure 404 {object} BaseResponse "pet not found"
// @Security ApiKeyAuth
// @Router /pets/{id} [delete]
func (h *handler) deletePet(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, errResp(400, "invalid id"))
		return
	}
	if !h.db.delete(id) {
		c.JSON(http.StatusNotFound, errResp(404, "pet not found"))
		return
	}
	c.Status(http.StatusNoContent)
}

// searchPets handles POST /api/v1/pets/search.
//
// This handler demonstrates the function-local struct pattern supported by
// swag3: SearchFilter is declared inside the function body rather than at
// package level. swag3 detects that SearchPets() in petstore.go carries a
// @Router annotation and therefore registers the local type as a named schema
// in components, allowing the $ref to resolve correctly in the generated spec.
//
// @Summary Search pets by keyword, status, and tags
// @Tags pets
// @ID searchPets
// @Param body body   SearchFilter true "search criteria"
// @Success 200 {object} BaseResponse{data=PagedList{list=[]Pet}} "matching pets (single page)"
// @Failure 400 {object} BaseResponse                 "invalid request body"
// @Security ApiKeyAuth
// @Router /pets/search [post]
func (h *handler) searchPets(c *gin.Context) {
	// SearchFilter mirrors the function-local struct in SearchPets() (petstore.go).
	// Defining it here keeps the request shape co-located with the handler that
	// uses it, which is a common pattern in large Gin codebases.
	type SearchFilter struct {
		Keywords []string `json:"keywords"`       // all keywords must appear in the name (AND)
		Statuses []string `json:"statuses"`       // pet must match one of these statuses (OR)
		Tags     []string `json:"tags,omitempty"` // pet must carry all of these tags (AND)
	}

	var req SearchFilter
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errResp(400, err.Error()))
		return
	}

	items := h.db.search(req.Keywords, req.Statuses, req.Tags)
	if items == nil {
		items = []*Pet{}
	}
	// All matches are returned in a single page; callers can page client-side.
	c.JSON(http.StatusOK, okResp(PagedList{Total: int64(len(items)), List: items}))
}

// getPetEvents handles GET /api/v1/pets/:id/events.
//
// This handler's response type (PetEvent) demonstrates two stdlib/map field
// mappings that swag3 handles automatically:
//   - time.Time  → OpenAPI string, format: date-time
//   - map[string]string → OpenAPI object, additionalProperties: {type: string}
//
// Query params:
//   - event_type  optional filter; returns only events of that type when set
//
// @Summary Get lifecycle events for a pet
// @Tags pets
// @ID getPetEvents
// @Param id         path  integer true  "pet ID"
// @Param event_type query string  false "filter by event type (created, updated, viewed, …)"
// @Success 200 {object} BaseResponse{data=[]PetEvent} "list of events"
// @Failure 400 {object} BaseResponse                  "invalid id"
// @Failure 404 {object} BaseResponse                  "pet not found"
// @Router /pets/{id}/events [get]
func (h *handler) getPetEvents(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, errResp(400, "invalid id"))
		return
	}

	evs, found := h.db.listEvents(id, c.Query("event_type"))
	if !found {
		c.JSON(http.StatusNotFound, errResp(404, "pet not found"))
		return
	}
	if evs == nil {
		evs = []*PetEvent{}
	}
	c.JSON(http.StatusOK, okResp(evs))
}

// uploadAvatar handles POST /api/v1/pets/:id/avatar.
//
// Accepts a multipart/form-data request with a single "file" field.
// The file is stored in memory (the in-memory store); a real service would
// persist it to object storage (S3, GCS, …) or a local filesystem.
//
// Size is capped at 8 MiB via gin's MaxMultipartMemory setting in main().
//
// @Summary Upload pet avatar
// @Tags pets
// @ID uploadPetAvatar
// @accept multipart/form-data
// @Param id   path     integer true "pet ID"
// @Param file formData file    true "avatar image (JPEG / PNG / GIF / WEBP)"
// @Success 200 {object} BaseResponse{data=UploadResult} "upload metadata"
// @Failure 400 {object} BaseResponse "invalid id or missing/oversized file"
// @Failure 404 {object} BaseResponse "pet not found"
// @Security ApiKeyAuth
// @Router /pets/{id}/avatar [post]
func (h *handler) uploadAvatar(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, errResp(400, "invalid id"))
		return
	}

	fileHeader, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, errResp(400, "file field is required: "+err.Error()))
		return
	}

	f, err := fileHeader.Open()
	if err != nil {
		c.JSON(http.StatusBadRequest, errResp(400, "cannot open uploaded file: "+err.Error()))
		return
	}
	defer f.Close()

	content, err := io.ReadAll(f)
	if err != nil {
		c.JSON(http.StatusBadRequest, errResp(400, "cannot read uploaded file: "+err.Error()))
		return
	}

	// Detect actual MIME type from file content (ignores the client-supplied header).
	contentType := http.DetectContentType(content)

	af := &avatarFile{
		Filename:    fileHeader.Filename,
		Content:     content,
		ContentType: contentType,
	}
	if !h.db.saveAvatar(id, af) {
		c.JSON(http.StatusNotFound, errResp(404, "pet not found"))
		return
	}

	c.JSON(http.StatusOK, okResp(UploadResult{
		Filename:    fileHeader.Filename,
		Size:        int64(len(content)),
		ContentType: contentType,
		URL:         "/api/v1/pets/" + strconv.FormatInt(id, 10) + "/avatar",
	}))
}

// downloadAvatar handles GET /api/v1/pets/:id/avatar.
//
// Streams the stored avatar bytes back to the client with the detected
// Content-Type.  Returns 404 when no avatar has been uploaded yet.
//
// @Summary Download pet avatar
// @Tags pets
// @ID downloadPetAvatar
// @produce application/octet-stream
// @Param id path integer true "pet ID"
// @Success 200 "avatar image bytes (Content-Type matches the uploaded format)"
// @Failure 400 {object} BaseResponse "invalid id"
// @Failure 404 {object} BaseResponse "pet or avatar not found"
// @Router /pets/{id}/avatar [get]
func (h *handler) downloadAvatar(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, errResp(400, "invalid id"))
		return
	}

	af, ok := h.db.loadAvatar(id)
	if !ok {
		c.JSON(http.StatusNotFound, errResp(404, "avatar not found"))
		return
	}

	// Send raw bytes; Content-Disposition triggers a download in browsers.
	c.Header("Content-Disposition", `attachment; filename="`+af.Filename+`"`)
	c.Data(http.StatusOK, af.ContentType, af.Content)
}

// ── Order handlers ────────────────────────────────────────────────────────────

// createOrder handles POST /api/v1/orders.
//
// Demonstrates a deeply nested request body:
//   - ShippingAddress Address     inline struct value (required)
//   - BillingAddress  *Address    nullable pointer to same struct
//   - Items           []OrderItem array of structs
//   - PlacedAt        time.Time   stdlib → OpenAPI string/date-time
//
// All nested types (Address, OrderItem) are defined in types_order.go,
// not in this file — demonstrating cross-file struct resolution by swag3.
//
// @Summary Create an order
// @Tags orders
// @ID createOrder
// @Param body body CreateOrderRequest true "order payload"
// @Success 201 {object} BaseResponse{data=Order} "created order"
// @Failure 400 {object} BaseResponse             "invalid request body"
// @Security ApiKeyAuth
// @Router /orders [post]
func (h *handler) createOrder(c *gin.Context) {
	var req CreateOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errResp(400, err.Error()))
		return
	}
	c.JSON(http.StatusCreated, okResp(h.db.createOrder(req)))
}

// getOrder handles GET /api/v1/orders/:id.
//
// @Summary Get an order by ID
// @Tags orders
// @ID getOrder
// @Param id path integer true "order ID"
// @Success 200 {object} BaseResponse{data=Order} "the order"
// @Failure 400 {object} BaseResponse             "invalid id"
// @Failure 404 {object} BaseResponse             "order not found"
// @Router /orders/{id} [get]
func (h *handler) getOrder(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, errResp(400, "invalid id"))
		return
	}
	order, found := h.db.getOrder(id)
	if !found {
		c.JSON(http.StatusNotFound, errResp(404, "order not found"))
		return
	}
	c.JSON(http.StatusOK, okResp(order))
}

// ── Category handlers ─────────────────────────────────────────────────────────

// createCategory handles POST /api/v1/categories.
//
// Category.Children is []*Category, forming a recursive (self-referential) schema.
// swag3 detects the cycle and breaks it with a $ref instead of expanding forever.
//
// @Summary Create a category
// @Tags categories
// @ID createCategory
// @Param body body CreateCategoryRequest true "category payload"
// @Success 201 {object} BaseResponse{data=Category} "created category"
// @Failure 400 {object} BaseResponse                "invalid request body"
// @Failure 404 {object} BaseResponse                "parent category not found"
// @Security ApiKeyAuth
// @Router /categories [post]
func (h *handler) createCategory(c *gin.Context) {
	var req CreateCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errResp(400, err.Error()))
		return
	}
	cat, ok := h.db.createCategory(req)
	if !ok {
		c.JSON(http.StatusNotFound, errResp(404, "parent category not found"))
		return
	}
	c.JSON(http.StatusCreated, okResp(cat))
}

// getCategory handles GET /api/v1/categories/:id.
//
// The returned Category carries its full Children subtree, illustrating how
// the recursive $ref schema is used in practice.
//
// @Summary Get a category (with children tree)
// @Tags categories
// @ID getCategory
// @Param id path integer true "category ID"
// @Success 200 {object} BaseResponse{data=Category} "category with children"
// @Failure 400 {object} BaseResponse                "invalid id"
// @Failure 404 {object} BaseResponse                "category not found"
// @Router /categories/{id} [get]
func (h *handler) getCategory(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, errResp(400, "invalid id"))
		return
	}
	cat, found := h.db.getCategory(id)
	if !found {
		c.JSON(http.StatusNotFound, errResp(404, "category not found"))
		return
	}
	c.JSON(http.StatusOK, okResp(cat))
}

// ── Import handler ────────────────────────────────────────────────────────────

// importPets handles POST /api/v1/pets/import.
//
// This handler defines both ImportItem and ImportRequest as function-local
// structs.  swag3 detects that importPets is an annotated operation (via the
// corresponding ImportPets() stub in petstore.go) and registers these local
// types as named components schemas so that the $ref can resolve correctly.
//
// @Summary Batch-import pets
// @Tags pets
// @ID importPets
// @Param body body ImportRequest true "import payload"
// @Success 200 {object} BaseResponse{data=[]Pet} "successfully imported pets"
// @Failure 400 {object} BaseResponse             "invalid request body"
// @Security ApiKeyAuth
// @Router /pets/import [post]
func (h *handler) importPets(c *gin.Context) {
	// ImportItem describes a single pet to import.
	// Defined locally to demonstrate function-local struct support.
	type ImportItem struct {
		Name   string    `json:"name"   binding:"required"`
		Status PetStatus `json:"status"`
		Tags   []PetTag  `json:"tags,omitempty"`
	}

	// ImportRequest wraps the source label and the items list.
	// Also defined locally — swag3 registers it because importPets is annotated.
	type ImportRequest struct {
		// Source identifies the upstream system ("csv", "json", "api", …).
		Source string       `json:"source" binding:"required"`
		DryRun bool         `json:"dry_run"` // when true, validate only; do not persist
		Items  []ImportItem `json:"items"  binding:"required"`
	}

	var req ImportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errResp(400, err.Error()))
		return
	}

	var imported []*Pet
	if !req.DryRun {
		for _, item := range req.Items {
			imported = append(imported, h.db.create(CreatePetRequest{
				Name:   item.Name,
				Status: item.Status,
				Tags:   item.Tags,
			}))
		}
	}
	if imported == nil {
		imported = []*Pet{}
	}
	c.JSON(http.StatusOK, okResp(imported))
}

// ── Server bootstrap ──────────────────────────────────────────────────────────

func main() {
	r := gin.Default()
	r.MaxMultipartMemory = 8 << 20 // 8 MiB max upload size

	// ── 1. Spec & UI routes (swaggin) ─────────────────────────────────────────
	//
	// swaggin.Register attaches three routes to the engine:
	//   GET /openapi.json  — raw spec served from disk (hot-reloaded on each request)
	//   GET /docs          — Swagger UI  (CDN: unpkg.com/swagger-ui-dist@5)
	//   GET /redoc         — Redoc       (CDN: cdn.jsdelivr.net/npm/redoc@latest)
	//
	// To use inline content instead of a file (e.g. after programmatic generation):
	//   swaggin.Register(r, swaggin.Options{SpecContent: generatedBytes, ...})
	swaggin.Register(r, swaggin.Options{
		SpecFile:  "openapi.json",
		Title:     "Pet Store API",
		JSONPath:  "/openapi.json",
		UIPath:    "/docs",  // Swagger UI
		RedocPath: "/redoc", // Redoc (alternative renderer)
		AllowCORS: true,
	})

	// ── 2. API routes ─────────────────────────────────────────────────────────
	//
	// All routes are grouped under /api/v1 to match the BasePath annotation
	// in testdata/petstore/petstore.go.
	//
	// Route order matters for Gin's radix-tree router: the static segment
	// /pets/search must be registered before the parametric /pets/:id so that
	// "search" is not mistakenly captured as an :id value.
	h := &handler{db: newStore()}
	api := r.Group("/api/v1")
	{
		api.GET("/pets", h.listPets)
		api.POST("/pets", h.createPet)
		api.POST("/pets/search", h.searchPets) // static — must precede :id routes
		api.POST("/pets/import", h.importPets) // function-local struct demo (static, before :id)
		api.GET("/pets/:id", h.getPet)
		api.PUT("/pets/:id", h.updatePet)
		api.DELETE("/pets/:id", h.deletePet)
		api.GET("/pets/:id/events", h.getPetEvents)
		api.POST("/pets/:id/avatar", h.uploadAvatar)  // multipart/form-data upload
		api.GET("/pets/:id/avatar", h.downloadAvatar) // binary download

		// Order endpoints — nested struct + time.Time + cross-file type reference
		api.POST("/orders", h.createOrder)
		api.GET("/orders/:id", h.getOrder)

		// Category endpoints — recursive self-referential schema
		api.POST("/categories", h.createCategory)
		api.GET("/categories/:id", h.getCategory)

		// Stock endpoints — bo.xx / sql.ddlx.xx dotted annotation demo
		api.GET("/stock", h.listStock)
		api.GET("/stock/:petId", h.getStock)
		api.PUT("/stock/:petId", h.adjustStock)
	}

	// ── 3. Health check ───────────────────────────────────────────────────────
	//
	// Kubernetes / Docker health probes can hit GET /healthz.
	// Returns 200 {"status":"ok","time":"<RFC3339>"}.
	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "time": time.Now().Format(time.RFC3339)})
	})

	// ── 4. Listen ─────────────────────────────────────────────────────────────
	port := os.Getenv("PORT")
	if port == "" {
		port = "8088"
	}
	r.Run(":" + port)
}
