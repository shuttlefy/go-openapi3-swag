package builder

import (
	spec "github.com/shuttlefy/go-openapi3-spec"
)

// Resolver tracks registered schema names and produces $ref references.
//
// Each type has two names:
//   - lookup name: the short Go identifier used as the internal map key
//     (e.g. "StockItem")
//   - schema name: the name emitted in components/schemas and $ref paths;
//     for sub-package types this is the fully-qualified form
//     (e.g. "bo.StockItem")
//
// When the type is in the primary (main) package the two names are identical.
type Resolver struct {
	registered  map[string]bool   // lookup name → registered
	schemaNames map[string]string // lookup name → schema component name
}

func NewResolver() *Resolver {
	return &Resolver{
		registered:  make(map[string]bool),
		schemaNames: make(map[string]string),
	}
}

// Register registers a type whose schema name equals its lookup name.
func (r *Resolver) Register(name string) {
	r.registered[name] = true
	if _, ok := r.schemaNames[name]; !ok {
		r.schemaNames[name] = name
	}
}

// RegisterWithSchemaName registers a type under lookupName but emits
// schemaName in $ref paths and components/schemas.
// Use this when the Go type lives in a sub-package and should be fully
// qualified in the output (e.g. lookupName="StockItem", schemaName="bo.StockItem").
func (r *Resolver) RegisterWithSchemaName(lookupName, schemaName string) {
	r.registered[lookupName] = true
	r.schemaNames[lookupName] = schemaName
}

// IsRegistered reports whether lookupName has been registered.
func (r *Resolver) IsRegistered(name string) bool {
	return r.registered[name]
}

// SchemaName returns the component schema name for the given lookup name.
func (r *Resolver) SchemaName(lookupName string) string {
	if full, ok := r.schemaNames[lookupName]; ok {
		return full
	}
	return lookupName
}

// RefSchema returns a $ref schema whose path uses the full schema name.
func (r *Resolver) RefSchema(lookupName string) *spec.Schema {
	return &spec.Schema{
		Reference: spec.Reference{Ref: "#/components/schemas/" + r.SchemaName(lookupName)},
	}
}
