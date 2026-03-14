package builder

import (
	spec "github.com/shuttlefy/go-openapi3-spec"
)

// Resolver tracks registered schema names and produces $ref references.
type Resolver struct {
	registered map[string]bool
}

func NewResolver() *Resolver {
	return &Resolver{registered: make(map[string]bool)}
}

func (r *Resolver) Register(name string) {
	r.registered[name] = true
}

func (r *Resolver) IsRegistered(name string) bool {
	return r.registered[name]
}

// RefSchema returns a $ref schema pointing to #/components/schemas/{name}.
func (r *Resolver) RefSchema(name string) *spec.Schema {
	return &spec.Schema{
		Reference: spec.Reference{Ref: "#/components/schemas/" + name},
	}
}
