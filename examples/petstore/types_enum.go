package main

// ── Named-type enums ──────────────────────────────────────────────────────────
//
// This file demonstrates three swag3 features that work together:
//
//  1. 别名类型 (type alias)  — "type X <underlying>" produces a named schema in
//     components/schemas.  Fields whose Go type is X get a $ref instead of an
//     inlined primitive, which makes the spec easier to read and keeps enum
//     values in one place.
//
//  2. 常量枚举 (const enum)  — typed constants declared for the alias are
//     collected by swag3 and emitted as the "enum" array on the alias schema.
//
//  3. format tag            — the `format` struct tag overrides the default
//     OpenAPI format when the underlying type's default isn't specific enough
//     (e.g. email, uuid, uri).

// PetStatus is the lifecycle state of a pet in the store.
//
// The three canonical values map to the petstore industry convention:
// available → visible for adoption, pending → reserved, sold → no longer available.
type PetStatus string

const (
	// PetStatusAvailable means the pet is visible and can be adopted.
	PetStatusAvailable PetStatus = "available"
	// PetStatusPending means an adoption application is in progress.
	PetStatusPending PetStatus = "pending"
	// PetStatusSold means the pet has been adopted and is no longer listed.
	PetStatusSold PetStatus = "sold"
)

// EventType describes the kind of lifecycle event recorded for a pet.
//
// Values follow a simple verb-noun pattern so they sort and display clearly
// in audit trails and dashboards.
type EventType string

const (
	EventTypeCreated EventType = "created" // pet record was first inserted
	EventTypeUpdated EventType = "updated" // any field on the pet was changed
	EventTypeViewed  EventType = "viewed"  // pet detail page was fetched
	EventTypeDeleted EventType = "deleted" // pet record was removed
)

// OrderStatus represents the fulfilment state of an order.
//
// Transitions:  pending → paid → shipped → delivered
//               pending → cancelled  (any time before shipped)
type OrderStatus string

const (
	OrderStatusPending   OrderStatus = "pending"
	OrderStatusPaid      OrderStatus = "paid"
	OrderStatusShipped   OrderStatus = "shipped"
	OrderStatusDelivered OrderStatus = "delivered"
	OrderStatusCancelled OrderStatus = "cancelled"
)

// ContactEmail is an RFC-5321 email address.
// The `format:"email"` tag instructs swag3 to emit format: email in the schema.
type ContactEmail string

// PetTag is a free-form label attached to a pet for filtering and search.
// Using a named type instead of raw string keeps the enum extensible.
type PetTag string
