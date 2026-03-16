package builder

import (
	"testing"

	spec "github.com/shuttlefy/go-openapi3-spec"
	"github.com/shuttlefy/go-openapi3-swag/internal/extractor"
	"github.com/shuttlefy/go-openapi3-swag/internal/parser"
)

func float64Ptr(v float64) *float64 { return &v }
func int64Ptr(v int64) *int64       { return &v }

func TestSchemaBuilder_PrimitiveTypes(t *testing.T) {
	resolver := NewResolver()
	sb := NewSchemaBuilder(resolver)

	tests := []struct {
		goType     string
		wantType   string
		wantFormat string
	}{
		{"string", "string", ""},
		{"int", "integer", "int32"},
		{"int64", "integer", "int64"},
		{"float32", "number", "float"},
		{"float64", "number", "double"},
		{"bool", "boolean", ""},
		{"byte", "string", "byte"},
		{"time.Time", "string", "date-time"},
	}

	for _, tt := range tests {
		s := sb.SchemaForType(tt.goType)
		if s.Type != tt.wantType {
			t.Errorf("SchemaForType(%q).Type = %q, want %q", tt.goType, s.Type, tt.wantType)
		}
		if s.Format != tt.wantFormat {
			t.Errorf("SchemaForType(%q).Format = %q, want %q", tt.goType, s.Format, tt.wantFormat)
		}
	}
}

func TestSchemaBuilder_AnyType(t *testing.T) {
	resolver := NewResolver()
	sb := NewSchemaBuilder(resolver)

	for _, typeName := range []string{"interface{}", "any"} {
		s := sb.SchemaForType(typeName)
		if s.Type != "" {
			t.Errorf("SchemaForType(%q).Type = %q, want empty (any type)", typeName, s.Type)
		}
		if s.Ref != "" {
			t.Errorf("SchemaForType(%q).Ref = %q, want empty", typeName, s.Ref)
		}
	}
}

func TestSchemaBuilder_PointerType(t *testing.T) {
	resolver := NewResolver()
	sb := NewSchemaBuilder(resolver)

	s := sb.SchemaForType("*string")
	if s.Type != "string" {
		t.Errorf("got type %q, want string", s.Type)
	}
	if !s.Nullable {
		t.Error("expected Nullable=true for *string")
	}
}

func TestSchemaBuilder_SliceType(t *testing.T) {
	resolver := NewResolver()
	sb := NewSchemaBuilder(resolver)

	s := sb.SchemaForType("[]string")
	if s.Type != "array" {
		t.Errorf("got type %q, want array", s.Type)
	}
	if s.Items == nil || s.Items.Type != "string" {
		t.Error("expected items.type=string")
	}
}

func TestSchemaBuilder_MapType(t *testing.T) {
	resolver := NewResolver()
	sb := NewSchemaBuilder(resolver)

	s := sb.SchemaForType("map[string]int")
	if s.Type != "object" {
		t.Errorf("got type %q, want object", s.Type)
	}
	addl, ok := s.AdditionalProperties.(*spec.Schema)
	if !ok {
		t.Fatal("expected AdditionalProperties to be *spec.Schema")
	}
	if addl.Type != "integer" {
		t.Errorf("additionalProperties.type = %q, want integer", addl.Type)
	}
}

func TestSchemaBuilder_StructRef(t *testing.T) {
	resolver := NewResolver()
	sb := NewSchemaBuilder(resolver)

	sb.RegisterStruct(&parser.RawStruct{
		Name: "User",
		Fields: []parser.RawField{
			{Name: "ID", TypeName: "int64", JSONName: "id"},
			{Name: "Name", TypeName: "string", JSONName: "name", Required: true},
		},
	})
	sb.BuildAll()

	userSchema := sb.Schemas().Get("User")
	if userSchema == nil {
		t.Fatal("User schema not registered")
	}
	if userSchema.Type != "object" {
		t.Errorf("User.Type = %q, want object", userSchema.Type)
	}

	idProp := userSchema.Properties.Get("id")
	if idProp == nil {
		t.Fatal("missing 'id' property")
	}
	if idProp.Type != "integer" || idProp.Format != "int64" {
		t.Errorf("id schema = %s/%s, want integer/int64", idProp.Type, idProp.Format)
	}

	if len(userSchema.Required) != 1 || userSchema.Required[0] != "name" {
		t.Errorf("Required = %v, want [name]", userSchema.Required)
	}

	ref := sb.SchemaForType("User")
	if ref.Ref != "#/components/schemas/User" {
		t.Errorf("ref = %q, want #/components/schemas/User", ref.Ref)
	}
}

func TestSchemaBuilder_StructFieldConstraints(t *testing.T) {
	resolver := NewResolver()
	sb := NewSchemaBuilder(resolver)

	sb.RegisterStruct(&parser.RawStruct{
		Name: "Product",
		Fields: []parser.RawField{
			{
				Name:        "Price",
				TypeName:    "float64",
				JSONName:    "price",
				Required:    true,
				Minimum:     float64Ptr(0),
				Maximum:     float64Ptr(9999.99),
				Description: "product price",
			},
			{
				Name:      "Tags",
				TypeName:  "[]string",
				JSONName:  "tags",
				MinItems:  int64Ptr(1),
				MaxItems:  int64Ptr(10),
				UniqueItems: true,
			},
			{
				Name:      "Code",
				TypeName:  "string",
				JSONName:  "code",
				Pattern:   "^[A-Z]{3}$",
				MinLength: int64Ptr(3),
				MaxLength: int64Ptr(3),
			},
			{
				Name:       "Status",
				TypeName:   "string",
				JSONName:   "status",
				Enums:      []string{"active", "inactive"},
				Default:    "active",
				ReadOnly:   true,
				Deprecated: true,
			},
		},
	})
	sb.BuildAll()

	schema := sb.Schemas().Get("Product")
	if schema == nil {
		t.Fatal("Product schema not found")
	}

	price := schema.Properties.Get("price")
	if price.Description != "product price" {
		t.Errorf("price.Description = %q", price.Description)
	}
	if price.Minimum == nil || *price.Minimum != 0 {
		t.Error("price.Minimum wrong")
	}
	if price.Maximum == nil || *price.Maximum != 9999.99 {
		t.Error("price.Maximum wrong")
	}

	tags := schema.Properties.Get("tags")
	if tags.Type != "array" {
		t.Errorf("tags.Type = %q, want array", tags.Type)
	}
	if !tags.UniqueItems {
		t.Error("tags.UniqueItems should be true")
	}
	if tags.MinItems == nil || *tags.MinItems != 1 {
		t.Error("tags.MinItems wrong")
	}

	code := schema.Properties.Get("code")
	if code.Pattern != "^[A-Z]{3}$" {
		t.Errorf("code.Pattern = %q", code.Pattern)
	}
	if code.MinLength == nil || *code.MinLength != 3 {
		t.Error("code.MinLength wrong")
	}

	status := schema.Properties.Get("status")
	if len(status.Enum) != 2 {
		t.Errorf("status.Enum len = %d, want 2", len(status.Enum))
	}
	if status.Default != "active" {
		t.Errorf("status.Default = %v", status.Default)
	}
	if !status.ReadOnly {
		t.Error("status.ReadOnly should be true")
	}
	if !status.Deprecated {
		t.Error("status.Deprecated should be true")
	}
}

func TestSchemaBuilder_NestedStructRef(t *testing.T) {
	resolver := NewResolver()
	sb := NewSchemaBuilder(resolver)

	sb.RegisterStruct(&parser.RawStruct{
		Name: "Address",
		Fields: []parser.RawField{
			{Name: "City", TypeName: "string", JSONName: "city"},
		},
	})
	sb.RegisterStruct(&parser.RawStruct{
		Name: "Person",
		Fields: []parser.RawField{
			{Name: "Name", TypeName: "string", JSONName: "name"},
			{Name: "Addr", TypeName: "Address", JSONName: "address"},
			{Name: "Addrs", TypeName: "[]Address", JSONName: "addresses"},
		},
	})
	sb.BuildAll()

	person := sb.Schemas().Get("Person")
	addr := person.Properties.Get("address")
	if addr.Ref != "#/components/schemas/Address" {
		t.Errorf("address.$ref = %q, want #/components/schemas/Address", addr.Ref)
	}

	addrs := person.Properties.Get("addresses")
	if addrs.Type != "array" {
		t.Errorf("addresses.Type = %q, want array", addrs.Type)
	}
	if addrs.Items == nil || addrs.Items.Ref != "#/components/schemas/Address" {
		t.Error("addresses.Items should be $ref to Address")
	}
}

func TestSchemaBuilder_JSONNameExcluded(t *testing.T) {
	resolver := NewResolver()
	sb := NewSchemaBuilder(resolver)

	sb.RegisterStruct(&parser.RawStruct{
		Name: "Foo",
		Fields: []parser.RawField{
			{Name: "Visible", TypeName: "string", JSONName: "visible"},
			{Name: "Hidden", TypeName: "string", JSONName: "-"},
		},
	})
	sb.BuildAll()

	schema := sb.Schemas().Get("Foo")
	keys := schema.Properties.Keys()
	if len(keys) != 1 || keys[0] != "visible" {
		t.Errorf("properties keys = %v, want [visible]", keys)
	}
}

func TestSchemaBuilder_CompositeType(t *testing.T) {
	resolver := NewResolver()
	sb := NewSchemaBuilder(resolver)

	sb.RegisterStruct(&parser.RawStruct{
		Name: "PageData",
		Fields: []parser.RawField{
			{Name: "Code", TypeName: "int", JSONName: "code"},
			{Name: "Data", TypeName: "interface{}", JSONName: "data"},
		},
	})
	sb.RegisterStruct(&parser.RawStruct{
		Name: "User",
		Fields: []parser.RawField{
			{Name: "ID", TypeName: "int64", JSONName: "id"},
		},
	})
	sb.BuildAll()

	te := extractor.TypeExpr{
		Name: "PageData",
		Overrides: []extractor.FieldOverride{
			{Field: "data", TypeExpr: "[]User"},
		},
	}
	schema := sb.SchemaForTypeExpr(te, false)

	if len(schema.AllOf) != 2 {
		t.Fatalf("expected allOf with 2 items, got %d", len(schema.AllOf))
	}
	if schema.AllOf[0].Ref != "#/components/schemas/PageData" {
		t.Errorf("allOf[0].$ref = %q", schema.AllOf[0].Ref)
	}
	override := schema.AllOf[1]
	dataProp := override.Properties.Get("data")
	if dataProp == nil {
		t.Fatal("missing override 'data' property")
	}
	if dataProp.Type != "array" {
		t.Errorf("data.Type = %q, want array", dataProp.Type)
	}
	if dataProp.Items == nil || dataProp.Items.Ref != "#/components/schemas/User" {
		t.Error("data.Items should be $ref to User")
	}
}

func TestSchemaBuilder_CompositeTypeAsArray(t *testing.T) {
	resolver := NewResolver()
	sb := NewSchemaBuilder(resolver)

	sb.RegisterStruct(&parser.RawStruct{
		Name: "Wrapper",
		Fields: []parser.RawField{
			{Name: "Item", TypeName: "interface{}", JSONName: "item"},
		},
	})
	sb.BuildAll()

	te := extractor.TypeExpr{
		Name: "Wrapper",
		Overrides: []extractor.FieldOverride{
			{Field: "item", TypeExpr: "string"},
		},
	}
	schema := sb.SchemaForTypeExpr(te, true)

	if schema.Type != "array" {
		t.Fatalf("expected type=array, got %q", schema.Type)
	}
	if schema.Items == nil || len(schema.Items.AllOf) != 2 {
		t.Fatal("expected items to be allOf composite")
	}
}

// TestSchemaBuilder_NestedComposite verifies that a doubly-nested composite
// annotation like BaseResponse{data=PagedList{list=[]User}} produces the
// correct two-level allOf schema instead of falling back to a plain object.
func TestSchemaBuilder_NestedComposite(t *testing.T) {
	resolver := NewResolver()
	sb := NewSchemaBuilder(resolver)

	// BaseResponse{code, message, data interface{}}
	sb.RegisterStruct(&parser.RawStruct{
		Name: "BaseResponse",
		Fields: []parser.RawField{
			{Name: "Code",    TypeName: "int",         JSONName: "code"},
			{Name: "Message", TypeName: "string",      JSONName: "message"},
			{Name: "Data",    TypeName: "interface{}", JSONName: "data"},
		},
	})
	// PagedList{total int64, list interface{}}
	sb.RegisterStruct(&parser.RawStruct{
		Name: "PagedList",
		Fields: []parser.RawField{
			{Name: "Total", TypeName: "int64",      JSONName: "total"},
			{Name: "List",  TypeName: "interface{}", JSONName: "list"},
		},
	})
	// User{id int64, name string}
	sb.RegisterStruct(&parser.RawStruct{
		Name: "User",
		Fields: []parser.RawField{
			{Name: "ID",   TypeName: "int64",  JSONName: "id"},
			{Name: "Name", TypeName: "string", JSONName: "name"},
		},
	})
	sb.BuildAll()

	// Parse annotation: BaseResponse{data=PagedList{list=[]User}}
	te := extractor.ParseTypeExpr("BaseResponse{data=PagedList{list=[]User}}")
	schema := sb.SchemaForTypeExpr(te, false)

	// Outer allOf: [$ref BaseResponse, {properties:{data: <inner>}}]
	if len(schema.AllOf) != 2 {
		t.Fatalf("outer allOf expected 2 items, got %d", len(schema.AllOf))
	}
	if schema.AllOf[0].Ref != "#/components/schemas/BaseResponse" {
		t.Errorf("outer allOf[0].$ref = %q, want BaseResponse", schema.AllOf[0].Ref)
	}
	dataProp := schema.AllOf[1].Properties.Get("data")
	if dataProp == nil {
		t.Fatal("missing 'data' property in outer override")
	}

	// data field must itself be an allOf: [$ref PagedList, {properties:{list: array}}]
	if len(dataProp.AllOf) != 2 {
		t.Fatalf("inner allOf (data) expected 2 items, got %d: %+v", len(dataProp.AllOf), dataProp)
	}
	if dataProp.AllOf[0].Ref != "#/components/schemas/PagedList" {
		t.Errorf("inner allOf[0].$ref = %q, want PagedList", dataProp.AllOf[0].Ref)
	}
	listProp := dataProp.AllOf[1].Properties.Get("list")
	if listProp == nil {
		t.Fatal("missing 'list' property in inner override")
	}
	if listProp.Type != "array" {
		t.Errorf("list.Type = %q, want array", listProp.Type)
	}
	if listProp.Items == nil || listProp.Items.Ref != "#/components/schemas/User" {
		t.Errorf("list.Items.$ref should be User, got %+v", listProp.Items)
	}
}

func TestSchemaBuilder_EmbeddedStruct(t *testing.T) {
	resolver := NewResolver()
	sb := NewSchemaBuilder(resolver)

	sb.RegisterStruct(&parser.RawStruct{
		Name: "Animal",
		Fields: []parser.RawField{
			{Name: "Name", TypeName: "string", JSONName: "name", Required: true},
		},
	})
	sb.RegisterStruct(&parser.RawStruct{
		Name: "Dog",
		Fields: []parser.RawField{
			// Embedded Animal (anonymous, no name)
			{Name: "", TypeName: "Animal"},
			// Own field
			{Name: "Breed", TypeName: "string", JSONName: "breed"},
		},
	})
	sb.BuildAll()

	dog := sb.Schemas().Get("Dog")
	if dog == nil {
		t.Fatal("Dog schema not found")
	}
	if len(dog.AllOf) < 2 {
		t.Fatalf("expected allOf with ≥2 items, got %d: %+v", len(dog.AllOf), dog.AllOf)
	}

	// First allOf entry should be $ref to Animal.
	if dog.AllOf[0].Ref != "#/components/schemas/Animal" {
		t.Errorf("allOf[0].$ref = %q, want #/components/schemas/Animal", dog.AllOf[0].Ref)
	}

	// Second allOf entry should have the own "breed" property.
	ownSchema := dog.AllOf[1]
	if ownSchema.Properties == nil {
		t.Fatal("own schema has no properties")
	}
	breedProp := ownSchema.Properties.Get("breed")
	if breedProp == nil || breedProp.Type != "string" {
		t.Errorf("breed property = %+v", breedProp)
	}
}

func TestSchemaBuilder_EmbeddedStructOnly(t *testing.T) {
	resolver := NewResolver()
	sb := NewSchemaBuilder(resolver)

	sb.RegisterStruct(&parser.RawStruct{Name: "Base", Fields: []parser.RawField{
		{Name: "ID", TypeName: "int64", JSONName: "id"},
	}})
	sb.RegisterStruct(&parser.RawStruct{
		Name: "Child",
		Fields: []parser.RawField{
			// Only embedding, no own fields.
			{Name: "", TypeName: "Base"},
		},
	})
	sb.BuildAll()

	child := sb.Schemas().Get("Child")
	if child == nil {
		t.Fatal("Child schema not found")
	}
	// Should be allOf with just the Base ref.
	if len(child.AllOf) != 1 || child.AllOf[0].Ref != "#/components/schemas/Base" {
		t.Errorf("Child.AllOf = %+v", child.AllOf)
	}
}

func TestSchemaBuilder_CyclicRef(t *testing.T) {
	resolver := NewResolver()
	sb := NewSchemaBuilder(resolver)

	// A → B → A  (mutual cycle)
	sb.RegisterStruct(&parser.RawStruct{
		Name: "A",
		Fields: []parser.RawField{
			{Name: "Child", TypeName: "B", JSONName: "child"},
		},
	})
	sb.RegisterStruct(&parser.RawStruct{
		Name: "B",
		Fields: []parser.RawField{
			{Name: "Parent", TypeName: "A", JSONName: "parent"},
		},
	})
	sb.BuildAll()

	schemaA := sb.Schemas().Get("A")
	schemaB := sb.Schemas().Get("B")
	if schemaA == nil || schemaB == nil {
		t.Fatal("schemas A and B must both be built")
	}

	// A.child should be $ref to B.
	childProp := schemaA.Properties.Get("child")
	if childProp == nil || childProp.Ref != "#/components/schemas/B" {
		t.Errorf("A.child.$ref = %q", childProp.Ref)
	}

	// B.parent should be $ref to A (cycle broken via $ref).
	parentProp := schemaB.Properties.Get("parent")
	if parentProp == nil || parentProp.Ref != "#/components/schemas/A" {
		t.Errorf("B.parent.$ref = %q", parentProp.Ref)
	}
}

func TestSchemaBuilder_SelfRef(t *testing.T) {
	resolver := NewResolver()
	sb := NewSchemaBuilder(resolver)

	// Node self-references via Children field.
	sb.RegisterStruct(&parser.RawStruct{
		Name: "Node",
		Fields: []parser.RawField{
			{Name: "Value", TypeName: "string", JSONName: "value"},
			{Name: "Children", TypeName: "[]*Node", JSONName: "children"},
		},
	})
	sb.BuildAll()

	node := sb.Schemas().Get("Node")
	if node == nil {
		t.Fatal("Node schema not found")
	}

	children := node.Properties.Get("children")
	if children == nil {
		t.Fatal("missing children property")
	}
	if children.Type != "array" {
		t.Errorf("children.Type = %q, want array", children.Type)
	}
	if children.Items == nil || children.Items.Ref != "#/components/schemas/Node" {
		t.Errorf("children.Items.$ref = %q", children.Items.Ref)
	}
}

func TestSchemaBuilder_GenericTypeInstantiation(t *testing.T) {
	resolver := NewResolver()
	sb := NewSchemaBuilder(resolver)

	// GrayList is a registered generic base type.
	sb.RegisterStruct(&parser.RawStruct{
		Name:   "GrayList",
		Fields: []parser.RawField{{Name: "Items", TypeName: "[]string", JSONName: "items"}},
	})
	// Order has a field whose type is GrayList[string] — a generic instantiation.
	sb.RegisterStruct(&parser.RawStruct{
		Name:   "Order",
		Fields: []parser.RawField{{Name: "List", TypeName: "GrayList[string]", JSONName: "list"}},
	})
	sb.BuildAll()

	// No unknown-type warning should be emitted.
	if unknown := sb.UnknownTypeNames(); len(unknown) != 0 {
		t.Errorf("unexpected unknown types: %v", unknown)
	}

	order := sb.Schemas().Get("Order")
	listProp := order.Properties.Get("list")
	if listProp == nil {
		t.Fatal("missing list property")
	}
	if listProp.Ref != "#/components/schemas/GrayList" {
		t.Errorf("list.$ref = %q, want #/components/schemas/GrayList", listProp.Ref)
	}
}

func TestSchemaBuilder_UnknownType(t *testing.T) {
	resolver := NewResolver()
	sb := NewSchemaBuilder(resolver)

	sb.RegisterStruct(&parser.RawStruct{
		Name: "Order",
		Fields: []parser.RawField{
			{Name: "Item", TypeName: "NonExistentType", JSONName: "item"},
		},
	})
	sb.BuildAll()

	unknown := sb.UnknownTypeNames()
	if len(unknown) != 1 || unknown[0] != "NonExistentType" {
		t.Errorf("UnknownTypeNames = %v, want [NonExistentType]", unknown)
	}

	// The property should still be a plain object (graceful degradation).
	order := sb.Schemas().Get("Order")
	itemProp := order.Properties.Get("item")
	if itemProp == nil || itemProp.Type != "object" {
		t.Errorf("unknown type field = %+v", itemProp)
	}
}

func TestParamTypeSchema(t *testing.T) {
	tests := []struct {
		typeName string
		format   string
		wantType string
		wantFmt  string
	}{
		{"string", "", "string", ""},
		{"integer", "", "integer", "int64"},
		{"integer", "int32", "integer", "int32"},
		{"boolean", "", "boolean", ""},
		{"file", "", "string", "binary"},
		{"unknown", "", "string", ""},
	}
	for _, tt := range tests {
		s := ParamTypeSchema(tt.typeName, tt.format)
		if s.Type != tt.wantType {
			t.Errorf("ParamTypeSchema(%q, %q).Type = %q, want %q", tt.typeName, tt.format, s.Type, tt.wantType)
		}
		if s.Format != tt.wantFmt {
			t.Errorf("ParamTypeSchema(%q, %q).Format = %q, want %q", tt.typeName, tt.format, s.Format, tt.wantFmt)
		}
	}
}

// ── Type alias & const enum ───────────────────────────────────────────────────

func TestSchemaBuilder_TypeAlias_NoConsts(t *testing.T) {
	resolver := NewResolver()
	sb := NewSchemaBuilder(resolver)

	sb.RegisterTypeAlias(&parser.RawTypeAlias{
		Name:       "MyID",
		Underlying: "int64",
	})
	sb.BuildAll()

	schema := sb.schemas.Get("MyID")
	if schema == nil {
		t.Fatal("expected schema for MyID, got nil")
	}
	if schema.Type != "integer" {
		t.Errorf("MyID schema type = %q, want %q", schema.Type, "integer")
	}
	if schema.Format != "int64" {
		t.Errorf("MyID schema format = %q, want %q", schema.Format, "int64")
	}
	// Field referencing the alias should produce a $ref.
	fieldSchema := sb.SchemaForType("MyID")
	if fieldSchema.Ref == "" {
		t.Errorf("field schema for alias type should be a $ref, got type=%q", fieldSchema.Type)
	}
}

func TestSchemaBuilder_TypeAlias_WithStringConsts(t *testing.T) {
	resolver := NewResolver()
	sb := NewSchemaBuilder(resolver)

	sb.RegisterTypeAlias(&parser.RawTypeAlias{
		Name:       "Status",
		Underlying: "string",
		Comments:   []string{"// Status is the lifecycle state of a pet."},
	})
	sb.RegisterConst(parser.RawConst{
		Name: "StatusAvailable", TypeName: "Status", Value: `"available"`,
		Comments: []string{"// StatusAvailable means the pet is listed for adoption."},
	})
	sb.RegisterConst(parser.RawConst{
		Name: "StatusPending", TypeName: "Status", Value: `"pending"`,
		Comments: []string{"// StatusPending means an application is in progress."},
	})
	sb.RegisterConst(parser.RawConst{
		Name: "StatusSold", TypeName: "Status", Value: `"sold"`,
		Comments: []string{"// StatusSold means the pet has been adopted."},
	})
	sb.BuildAll()

	schema := sb.schemas.Get("Status")
	if schema == nil {
		t.Fatal("expected schema for Status, got nil")
	}
	if schema.Type != "string" {
		t.Errorf("Status schema type = %q, want %q", schema.Type, "string")
	}
	if len(schema.Enum) != 3 {
		t.Fatalf("Status schema enum len = %d, want 3", len(schema.Enum))
	}
	if schema.Enum[0] != "available" || schema.Enum[1] != "pending" || schema.Enum[2] != "sold" {
		t.Errorf("Status schema enum = %v, want [available pending sold]", schema.Enum)
	}
	if schema.Description == "" {
		t.Error("Status schema description should be set from alias comment")
	}

	// x-enum-varnames should carry the Go identifier names.
	varnames, ok := schema.Extensions.Get("x-enum-varnames").([]interface{})
	if !ok || len(varnames) != 3 {
		t.Fatalf("x-enum-varnames = %v, want 3-element slice", schema.Extensions.Get("x-enum-varnames"))
	}
	if varnames[0] != "StatusAvailable" {
		t.Errorf("x-enum-varnames[0] = %v, want StatusAvailable", varnames[0])
	}

	// x-enumDescriptions should carry the per-value doc comments.
	descs, ok := schema.Extensions.Get("x-enumDescriptions").([]interface{})
	if !ok || len(descs) != 3 {
		t.Fatalf("x-enumDescriptions = %v, want 3-element slice", schema.Extensions.Get("x-enumDescriptions"))
	}
	if descs[0] != "StatusAvailable means the pet is listed for adoption." {
		t.Errorf("x-enumDescriptions[0] = %q, want comment text", descs[0])
	}
}

func TestSchemaBuilder_TypeAlias_WithIntConsts(t *testing.T) {
	resolver := NewResolver()
	sb := NewSchemaBuilder(resolver)

	sb.RegisterTypeAlias(&parser.RawTypeAlias{
		Name:       "Priority",
		Underlying: "int",
	})
	sb.RegisterConst(parser.RawConst{Name: "PriorityLow", TypeName: "Priority", Value: "1"})
	sb.RegisterConst(parser.RawConst{Name: "PriorityHigh", TypeName: "Priority", Value: "2"})
	sb.BuildAll()

	schema := sb.schemas.Get("Priority")
	if schema == nil {
		t.Fatal("expected schema for Priority, got nil")
	}
	if len(schema.Enum) != 2 {
		t.Fatalf("Priority enum len = %d, want 2", len(schema.Enum))
	}
	if schema.Enum[0] != int64(1) || schema.Enum[1] != int64(2) {
		t.Errorf("Priority enum = %v, want [1 2]", schema.Enum)
	}
}

// ── Field & struct description from comments ─────────────────────────────────

func TestSchemaBuilder_FieldDescriptionFromComments(t *testing.T) {
	resolver := NewResolver()
	sb := NewSchemaBuilder(resolver)

	sb.RegisterStruct(&parser.RawStruct{
		Name: "Widget",
		Fields: []parser.RawField{
			{
				Name:     "Color",
				TypeName: "string",
				JSONName: "color",
				// no description tag — comment should be used instead
				Comments: []string{"// Color is the hex color code of the widget."},
			},
			{
				Name:        "Size",
				TypeName:    "int",
				JSONName:    "size",
				Description: "explicit description tag wins",
				Comments:    []string{"// this should be ignored"},
			},
		},
	})
	sb.BuildAll()

	schema := sb.schemas.Get("Widget")
	if schema == nil {
		t.Fatal("expected schema for Widget, got nil")
	}
	colorSchema := schema.Properties.Get("color")
	if colorSchema == nil {
		t.Fatal("expected property 'color'")
	}
	if colorSchema.Description != "Color is the hex color code of the widget." {
		t.Errorf("color description = %q, want comment text", colorSchema.Description)
	}
	sizeSchema := schema.Properties.Get("size")
	if sizeSchema == nil {
		t.Fatal("expected property 'size'")
	}
	if sizeSchema.Description != "explicit description tag wins" {
		t.Errorf("size description = %q, want tag text", sizeSchema.Description)
	}
}

func TestSchemaBuilder_StructDescriptionFromComments(t *testing.T) {
	resolver := NewResolver()
	sb := NewSchemaBuilder(resolver)

	sb.RegisterStruct(&parser.RawStruct{
		Name:     "Gadget",
		Comments: []string{"// Gadget is a useful device.", "// It has many features."},
		Fields:   []parser.RawField{{Name: "ID", TypeName: "int64", JSONName: "id"}},
	})
	sb.BuildAll()

	schema := sb.schemas.Get("Gadget")
	if schema == nil {
		t.Fatal("expected schema for Gadget, got nil")
	}
	wantDesc := "Gadget is a useful device. It has many features."
	if schema.Description != wantDesc {
		t.Errorf("Gadget description = %q, want %q", schema.Description, wantDesc)
	}
}

// ── commentsToDescription unit tests ─────────────────────────────────────────

func TestCommentsToDescription(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected string
	}{
		{"empty", nil, ""},
		{"single line comment", []string{"// Hello world"}, "Hello world"},
		{"no space after //", []string{"//hello"}, "hello"},
		{"multi line", []string{"// Line one.", "// Line two."}, "Line one. Line two."},
		{"block comment", []string{"/* block comment */"}, "block comment"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := commentsToDescription(tt.input)
			if got != tt.expected {
				t.Errorf("commentsToDescription(%v) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

// ── xx.xx (package-qualified) type reference resolution ───────────────────────

// TestSchemaBuilder_SubPackage_FullName verifies that a type from a sub-package
// is emitted with its fully-qualified name in both $ref and components/schemas.
//
// Given: primary package = "main", struct User in package "models"
// Expected:
//   - $ref produced by "models.User" annotation → #/components/schemas/models.User
//   - components/schemas key                    → "models.User"
func TestSchemaBuilder_SubPackage_FullName(t *testing.T) {
	resolver := NewResolver()
	sb := NewSchemaBuilder(resolver)

	sb.SetPrimaryPackage("main")
	sb.RegisterPackage("models")
	sb.RegisterStruct(&parser.RawStruct{
		Name:        "User",
		PackageName: "models",
		Fields:      []parser.RawField{{Name: "ID", TypeName: "int64", JSONName: "id"}},
	})
	sb.BuildAll()

	// Annotation "models.User" must resolve to $ref with full name.
	s := sb.SchemaForType("models.User")
	if s.Ref != "#/components/schemas/models.User" {
		t.Errorf("models.User $ref = %q, want #/components/schemas/models.User", s.Ref)
	}

	// The components/schemas key must also be the full qualified name.
	schemas := sb.Schemas()
	if schemas.Get("models.User") == nil {
		t.Error("components/schemas should have key 'models.User', not 'User'")
	}
	if schemas.Get("User") != nil {
		t.Error("components/schemas must NOT have bare key 'User' for a sub-package type")
	}
}

// TestSchemaBuilder_PrimaryPackage_ShortName verifies that a type from the
// primary package keeps its short name (no prefix) in the output.
func TestSchemaBuilder_PrimaryPackage_ShortName(t *testing.T) {
	resolver := NewResolver()
	sb := NewSchemaBuilder(resolver)

	sb.SetPrimaryPackage("main")
	sb.RegisterStruct(&parser.RawStruct{
		Name:        "Pet",
		PackageName: "main",
		Fields:      []parser.RawField{{Name: "ID", TypeName: "int64", JSONName: "id"}},
	})
	sb.BuildAll()

	s := sb.SchemaForType("Pet")
	if s.Ref != "#/components/schemas/Pet" {
		t.Errorf("Pet $ref = %q, want #/components/schemas/Pet", s.Ref)
	}
	if sb.Schemas().Get("Pet") == nil {
		t.Error("components/schemas must have key 'Pet'")
	}
}

// TestSchemaBuilder_SubPackage_CompositeFullName verifies that a composite
// annotation like "bo.StockPage{items=[]bo.StockItem}" resolves both the base
// type and the override to their fully-qualified schema names.
func TestSchemaBuilder_SubPackage_CompositeFullName(t *testing.T) {
	resolver := NewResolver()
	sb := NewSchemaBuilder(resolver)

	sb.SetPrimaryPackage("main")
	sb.RegisterPackage("bo")
	sb.RegisterStruct(&parser.RawStruct{
		Name:        "StockPage",
		PackageName: "bo",
		Fields: []parser.RawField{
			{Name: "Total", TypeName: "int64", JSONName: "total"},
			{Name: "Items", TypeName: "interface{}", JSONName: "items"},
		},
	})
	sb.RegisterStruct(&parser.RawStruct{
		Name:        "StockItem",
		PackageName: "bo",
		Fields:      []parser.RawField{{Name: "PetID", TypeName: "int64", JSONName: "pet_id"}},
	})
	sb.BuildAll()

	te := extractor.ParseTypeExpr("bo.StockPage{items=[]bo.StockItem}")
	schema := sb.SchemaForTypeExpr(te, false)

	// allOf[0] must point to bo.StockPage
	if len(schema.AllOf) < 2 {
		t.Fatalf("allOf len = %d, want >= 2", len(schema.AllOf))
	}
	if schema.AllOf[0].Ref != "#/components/schemas/bo.StockPage" {
		t.Errorf("allOf[0].$ref = %q, want #/components/schemas/bo.StockPage", schema.AllOf[0].Ref)
	}

	// Override items: array of bo.StockItem
	overrideProps := schema.AllOf[1].Properties
	if overrideProps == nil {
		t.Fatal("allOf[1] must have properties")
	}
	itemsProp := overrideProps.Get("items")
	if itemsProp == nil {
		t.Fatal("items property missing")
	}
	if itemsProp.Type != "array" {
		t.Fatalf("items.type = %q, want array", itemsProp.Type)
	}
	if itemsProp.Items == nil || itemsProp.Items.Ref != "#/components/schemas/bo.StockItem" {
		t.Errorf("items.items.$ref = %v, want #/components/schemas/bo.StockItem", itemsProp.Items)
	}
}

// TestSchemaBuilder_DottedType_KnownPackage verifies that "pkg.Type" resolves
// to $ref:pkg.Type (full qualified name) when pkg was registered via
// RegisterPackage and the type belongs to that package.
func TestSchemaBuilder_DottedType_KnownPackage(t *testing.T) {
	resolver := NewResolver()
	sb := NewSchemaBuilder(resolver)

	sb.SetPrimaryPackage("main")
	sb.RegisterPackage("models")
	sb.RegisterStruct(&parser.RawStruct{
		Name:        "User",
		PackageName: "models",
		Fields:      []parser.RawField{{Name: "ID", TypeName: "int64", JSONName: "id"}},
	})
	sb.BuildAll()

	s := sb.SchemaForType("models.User")
	if s.Ref == "" {
		t.Errorf("models.User should produce a $ref, got type=%q", s.Type)
	}
	// Full name must be preserved in the $ref.
	if s.Ref != "#/components/schemas/models.User" {
		t.Errorf("models.User $ref = %q, want #/components/schemas/models.User", s.Ref)
	}
}

// TestSchemaBuilder_DottedType_UnknownPackage verifies that "pkg.Type" is
// treated as an unknown type when pkg has NOT been registered — the builder
// must NOT silently fall back to the short name.
func TestSchemaBuilder_DottedType_UnknownPackage(t *testing.T) {
	resolver := NewResolver()
	sb := NewSchemaBuilder(resolver)

	// Register "User" as a type but do NOT register any package.
	sb.RegisterStruct(&parser.RawStruct{
		Name:   "User",
		Fields: []parser.RawField{{Name: "ID", TypeName: "int64", JSONName: "id"}},
	})
	sb.BuildAll()

	s := sb.SchemaForType("models.User")
	// Package "models" is unknown → must NOT produce a $ref.
	if s.Ref != "" {
		t.Errorf("models.User with unknown package should not produce a $ref, got %q", s.Ref)
	}
	unknowns := sb.UnknownTypeNames()
	found := false
	for _, u := range unknowns {
		if u == "models.User" {
			found = true
		}
	}
	if !found {
		t.Errorf("models.User should appear in unknown types: %v", unknowns)
	}
}

// TestSchemaBuilder_DottedType_MultiDot verifies that names with more than one
// dot ("sql.ddlx.Row") are always rejected — "sql.ddlx" cannot be a Go
// package identifier.
func TestSchemaBuilder_DottedType_MultiDot(t *testing.T) {
	resolver := NewResolver()
	sb := NewSchemaBuilder(resolver)

	// Register both "sql" and "ddlx" as packages, and "Row" as a type.
	sb.RegisterPackage("sql")
	sb.RegisterPackage("ddlx")
	sb.RegisterStruct(&parser.RawStruct{
		Name:   "Row",
		Fields: []parser.RawField{{Name: "ID", TypeName: "int64", JSONName: "id"}},
	})
	sb.BuildAll()

	s := sb.SchemaForType("sql.ddlx.Row")
	// Multi-dot name must never silently resolve — even if "Row" is registered.
	if s.Ref != "" {
		t.Errorf("sql.ddlx.Row should not produce a $ref, got %q", s.Ref)
	}
	unknowns := sb.UnknownTypeNames()
	found := false
	for _, u := range unknowns {
		if u == "sql.ddlx.Row" {
			found = true
		}
	}
	if !found {
		t.Errorf("sql.ddlx.Row should appear in unknown types: %v", unknowns)
	}
}

// TestSchemaBuilder_DottedType_Array verifies []models.User → array of $ref User
// when "models" is a registered package.
func TestSchemaBuilder_DottedType_Array(t *testing.T) {
	resolver := NewResolver()
	sb := NewSchemaBuilder(resolver)

	sb.RegisterPackage("models")
	sb.RegisterStruct(&parser.RawStruct{
		Name:   "User",
		Fields: []parser.RawField{{Name: "Name", TypeName: "string", JSONName: "name"}},
	})
	sb.BuildAll()

	s := sb.SchemaForType("[]models.User")
	if s.Type != "array" {
		t.Fatalf("[]models.User schema type = %q, want array", s.Type)
	}
	if s.Items == nil || s.Items.Ref != "#/components/schemas/User" {
		t.Errorf("items.$ref = %v, want #/components/schemas/User", s.Items)
	}
}

// TestSchemaBuilder_DottedType_Primitive verifies that stdlib primitives
// (time.Time) are NOT affected by the short-name fallback — they still produce
// their primitive schemas directly.
func TestSchemaBuilder_DottedType_Primitive(t *testing.T) {
	resolver := NewResolver()
	sb := NewSchemaBuilder(resolver)
	sb.BuildAll()

	s := sb.SchemaForType("time.Time")
	if s.Type != "string" || s.Format != "date-time" {
		t.Errorf("time.Time = {type:%q, format:%q}, want {string, date-time}", s.Type, s.Format)
	}
	// Must not produce a $ref even if "Time" happened to be registered.
	if s.Ref != "" {
		t.Errorf("time.Time should not be a $ref, got %q", s.Ref)
	}
}

// TestSchemaBuilder_DottedType_UnknownFallsThrough verifies that a dotted name
// whose short name is also unregistered falls through to the unknown-type path.
func TestSchemaBuilder_DottedType_UnknownFallsThrough(t *testing.T) {
	resolver := NewResolver()
	sb := NewSchemaBuilder(resolver)
	sb.BuildAll()

	s := sb.SchemaForType("uuid.UUID")
	// uuid.UUID is not registered, so we expect the fallback unknown schema.
	if s.Ref != "" {
		t.Errorf("uuid.UUID should not be a $ref when not registered, got %q", s.Ref)
	}
	if s.Type != "object" {
		t.Errorf("uuid.UUID should fallback to type:object, got %q", s.Type)
	}
	unknowns := sb.UnknownTypeNames()
	found := false
	for _, u := range unknowns {
		if u == "uuid.UUID" {
			found = true
		}
	}
	if !found {
		t.Errorf("uuid.UUID should appear in unknown types: %v", unknowns)
	}
}

// TestSchemaBuilder_DottedType_CompositeResponse verifies that
// "models.BaseResponse{data=models.PagedList{list=[]models.User}}" resolves
// correctly when all three types are registered under their short names.
func TestSchemaBuilder_DottedType_CompositeResponse(t *testing.T) {
	resolver := NewResolver()
	sb := NewSchemaBuilder(resolver)

	sb.RegisterPackage("models")
	sb.RegisterStruct(&parser.RawStruct{
		Name: "BaseResponse",
		Fields: []parser.RawField{
			{Name: "Code", TypeName: "int", JSONName: "code"},
			{Name: "Data", TypeName: "interface{}", JSONName: "data"},
		},
	})
	sb.RegisterStruct(&parser.RawStruct{
		Name: "PagedList",
		Fields: []parser.RawField{
			{Name: "Total", TypeName: "int64", JSONName: "total"},
			{Name: "List", TypeName: "interface{}", JSONName: "list"},
		},
	})
	sb.RegisterStruct(&parser.RawStruct{
		Name:   "User",
		Fields: []parser.RawField{{Name: "ID", TypeName: "int64", JSONName: "id"}},
	})
	sb.BuildAll()

	// Build schema for a fully-qualified composite type expression.
	te := extractor.ParseTypeExpr("models.BaseResponse{data=models.PagedList{list=[]models.User}}")
	schema := sb.SchemaForTypeExpr(te, false)

	// Top level: allOf[  $ref:BaseResponse,  {properties:{data: allOf[...]}}  ]
	if len(schema.AllOf) != 2 {
		t.Fatalf("top-level allOf len = %d, want 2", len(schema.AllOf))
	}
	baseRef := schema.AllOf[0]
	if baseRef.Ref != "#/components/schemas/BaseResponse" {
		t.Errorf("allOf[0].$ref = %q, want #/components/schemas/BaseResponse", baseRef.Ref)
	}

	// The override inlines the data property.
	overrideObj := schema.AllOf[1]
	if overrideObj.Properties == nil {
		t.Fatal("allOf[1] should have properties")
	}
	dataProp := overrideObj.Properties.Get("data")
	if dataProp == nil {
		t.Fatal("data property should exist in override")
	}
	// data is itself allOf[$ref:PagedList, {properties:{list:…}}]
	if len(dataProp.AllOf) != 2 {
		t.Fatalf("data.allOf len = %d, want 2", len(dataProp.AllOf))
	}
	pagedRef := dataProp.AllOf[0]
	if pagedRef.Ref != "#/components/schemas/PagedList" {
		t.Errorf("data.allOf[0].$ref = %q, want #/components/schemas/PagedList", pagedRef.Ref)
	}
}

// ── unquoteConstValue unit tests ──────────────────────────────────────────────

func TestUnquoteConstValue(t *testing.T) {
	tests := []struct {
		input interface{}
		want  interface{}
	}{
		{`"hello"`, "hello"},
		{"42", int64(42)},
		{"3.14", 3.14},
		{"plain", "plain"},
	}
	for _, tt := range tests {
		got := unquoteConstValue(tt.input.(string))
		if got != tt.want {
			t.Errorf("unquoteConstValue(%q) = %v (%T), want %v (%T)", tt.input, got, got, tt.want, tt.want)
		}
	}
}
