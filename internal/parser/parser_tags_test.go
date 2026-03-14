package parser

import (
	"path/filepath"
	"testing"
)

func TestStructTags_JSONNameAndRequired(t *testing.T) {
	p := &GoParser{}
	abs, _ := filepath.Abs("../../testdata/go_complex")
	result, err := p.Parse([]string{abs})
	if err != nil {
		t.Fatal(err)
	}

	create := findStruct(result.Structs, "CreateUserRequest")
	if create == nil {
		t.Fatal("CreateUserRequest not found")
	}

	tests := []struct {
		fieldName string
		jsonName  string
		required  bool
		omitempty bool
		example   string
	}{
		{"Owner", "owner", true, false, "brayden"},          // binding:"required" example:"brayden"
		{"Remark", "remark", false, false, "some remark"},   // *string, example:"some remark"
		{"Name", "name", true, false, "John"},               // validate:"required" example:"John"
		{"Email", "email", true, false, "john@example.com"}, // binding:"required,email" example:"..."
		{"Age", "age", false, true, "25"},                   // omitempty, example:"25"
		{"Hidden", "-", false, false, ""},                   // json:"-", no example
		{"NoTag", "", false, false, ""},                     // no tag at all
		{"XMLOnly", "", false, false, ""},                   // xml tag only
	}

	if len(create.Fields) != len(tests) {
		t.Fatalf("expected %d fields, got %d", len(tests), len(create.Fields))
	}

	for i, tt := range tests {
		f := create.Fields[i]
		if f.Name != tt.fieldName {
			t.Errorf("field[%d].Name = %q, want %q", i, f.Name, tt.fieldName)
			continue
		}
		if f.JSONName != tt.jsonName {
			t.Errorf("%s.JSONName = %q, want %q", tt.fieldName, f.JSONName, tt.jsonName)
		}
		if f.Required != tt.required {
			t.Errorf("%s.Required = %v, want %v", tt.fieldName, f.Required, tt.required)
		}
		if f.Omitempty != tt.omitempty {
			t.Errorf("%s.Omitempty = %v, want %v", tt.fieldName, f.Omitempty, tt.omitempty)
		}
		if f.Example != tt.example {
			t.Errorf("%s.Example = %q, want %q", tt.fieldName, f.Example, tt.example)
		}
	}
}

func TestStructTags_ValidateRequired(t *testing.T) {
	p := &GoParser{}
	abs, _ := filepath.Abs("../../testdata/go_complex")
	result, err := p.Parse([]string{abs})
	if err != nil {
		t.Fatal(err)
	}

	update := findStruct(result.Structs, "UpdateUserRequest")
	if update == nil {
		t.Fatal("UpdateUserRequest not found")
	}

	if len(update.Fields) != 3 {
		t.Fatalf("expected 3 fields, got %d", len(update.Fields))
	}

	// Name: validate:"required,min=1" + json:",omitempty"
	if !update.Fields[0].Required || !update.Fields[0].Omitempty {
		t.Errorf("Name: Required=%v Omitempty=%v", update.Fields[0].Required, update.Fields[0].Omitempty)
	}

	// Status: binding:"required" enums:"active,inactive" example:"active"
	status := update.Fields[1]
	if !status.Required {
		t.Error("Status should be required")
	}
	if status.Example != "active" {
		t.Errorf("Status.Example = %q, want \"active\"", status.Example)
	}
	if len(status.Enums) != 2 || status.Enums[0] != "active" || status.Enums[1] != "inactive" {
		t.Errorf("Status.Enums = %v, want [active inactive]", status.Enums)
	}

	// Profile: validate:"required"
	if !update.Fields[2].Required {
		t.Error("Profile should be required")
	}
}

func TestStructTags_Enums(t *testing.T) {
	p := &GoParser{}
	abs, _ := filepath.Abs("../../testdata/go_complex")
	result, err := p.Parse([]string{abs})
	if err != nil {
		t.Fatal(err)
	}

	order := findStruct(result.Structs, "OrderRequest")
	if order == nil {
		t.Fatal("OrderRequest not found")
	}

	if len(order.Fields) != 3 {
		t.Fatalf("expected 3 fields, got %d", len(order.Fields))
	}

	// State: enums:"passed,rejected,terminated,cancelled,pending" example:"pending"
	state := order.Fields[0]
	wantEnums := []string{"passed", "rejected", "terminated", "cancelled", "pending"}
	if len(state.Enums) != len(wantEnums) {
		t.Fatalf("State.Enums len = %d, want %d", len(state.Enums), len(wantEnums))
	}
	for i, want := range wantEnums {
		if state.Enums[i] != want {
			t.Errorf("State.Enums[%d] = %q, want %q", i, state.Enums[i], want)
		}
	}
	if state.Example != "pending" {
		t.Errorf("State.Example = %q, want \"pending\"", state.Example)
	}

	// Type: enums:"1,2,3"
	typ := order.Fields[1]
	if len(typ.Enums) != 3 || typ.Enums[0] != "1" || typ.Enums[1] != "2" || typ.Enums[2] != "3" {
		t.Errorf("Type.Enums = %v, want [1 2 3]", typ.Enums)
	}

	// NoEnum: no enums tag
	if len(order.Fields[2].Enums) != 0 {
		t.Errorf("NoEnum.Enums should be empty, got %v", order.Fields[2].Enums)
	}
}

func floatPtr(v float64) *float64 { return &v }
func intPtr(v int64) *int64       { return &v }

func TestStructTags_SchemaConstraints(t *testing.T) {
	p := &GoParser{}
	abs, _ := filepath.Abs("../../testdata/go_complex")
	result, err := p.Parse([]string{abs})
	if err != nil {
		t.Fatal(err)
	}

	product := findStruct(result.Structs, "ProductDetail")
	if product == nil {
		t.Fatal("ProductDetail not found")
	}

	if len(product.Fields) != 11 {
		t.Fatalf("expected 11 fields, got %d", len(product.Fields))
	}

	// Name: description, minLength, maxLength, example
	name := findField(product.Fields, "Name")
	assertStr(t, "Name.Description", name.Description, "Product name")
	assertIntPtr(t, "Name.MinLength", name.MinLength, intPtr(1))
	assertIntPtr(t, "Name.MaxLength", name.MaxLength, intPtr(200))
	assertStr(t, "Name.Example", name.Example, "Widget")

	// Price: minimum, maximum, default, format
	price := findField(product.Fields, "Price")
	assertFloatPtr(t, "Price.Minimum", price.Minimum, floatPtr(0))
	assertFloatPtr(t, "Price.Maximum", price.Maximum, floatPtr(99999.99))
	assertStr(t, "Price.Default", price.Default, "0")
	assertStr(t, "Price.Format", price.Format, "double")

	// SKU: pattern
	sku := findField(product.Fields, "SKU")
	assertStr(t, "SKU.Pattern", sku.Pattern, `^[A-Z]{2}-\d{6}$`)

	// Quantity: minimum, maximum, default (int)
	qty := findField(product.Fields, "Quantity")
	assertFloatPtr(t, "Quantity.Minimum", qty.Minimum, floatPtr(0))
	assertFloatPtr(t, "Quantity.Maximum", qty.Maximum, floatPtr(10000))
	assertStr(t, "Quantity.Default", qty.Default, "1")

	// Tags: minItems, maxItems, uniqueItems
	tags := findField(product.Fields, "Tags")
	assertIntPtr(t, "Tags.MinItems", tags.MinItems, intPtr(1))
	assertIntPtr(t, "Tags.MaxItems", tags.MaxItems, intPtr(10))
	if !tags.UniqueItems {
		t.Error("Tags.UniqueItems should be true")
	}

	// InternalID: readonly, format
	internalID := findField(product.Fields, "InternalID")
	if !internalID.ReadOnly {
		t.Error("InternalID.ReadOnly should be true")
	}
	assertStr(t, "InternalID.Format", internalID.Format, "uuid")

	// Password: writeonly, minLength
	pw := findField(product.Fields, "Password")
	if !pw.WriteOnly {
		t.Error("Password.WriteOnly should be true")
	}
	assertIntPtr(t, "Password.MinLength", pw.MinLength, intPtr(8))

	// OldField: deprecated
	old := findField(product.Fields, "OldField")
	if !old.Deprecated {
		t.Error("OldField.Deprecated should be true")
	}

	// CreatedAt: format, readonly
	createdAt := findField(product.Fields, "CreatedAt")
	assertStr(t, "CreatedAt.Format", createdAt.Format, "date-time")
	if !createdAt.ReadOnly {
		t.Error("CreatedAt.ReadOnly should be true")
	}

	// Email: format, description
	email := findField(product.Fields, "Email")
	assertStr(t, "Email.Format", email.Format, "email")
	assertStr(t, "Email.Description", email.Description, "Contact email")

	// Score: negative minimum
	score := findField(product.Fields, "Score")
	assertFloatPtr(t, "Score.Minimum", score.Minimum, floatPtr(-1.5))
	assertFloatPtr(t, "Score.Maximum", score.Maximum, floatPtr(1.5))
}

func TestStructTags_NoConstraints(t *testing.T) {
	p := &GoParser{}
	abs, _ := filepath.Abs("../../testdata/go_complex")
	result, err := p.Parse([]string{abs})
	if err != nil {
		t.Fatal(err)
	}

	order := findStruct(result.Structs, "OrderRequest")
	if order == nil {
		t.Fatal("OrderRequest not found")
	}

	noEnum := findField(order.Fields, "NoEnum")
	if noEnum.Format != "" || noEnum.Default != "" || noEnum.Description != "" {
		t.Errorf("NoEnum should have no constraints, got format=%q default=%q desc=%q",
			noEnum.Format, noEnum.Default, noEnum.Description)
	}
	if noEnum.Minimum != nil || noEnum.Maximum != nil {
		t.Error("NoEnum should have nil min/max")
	}
	if noEnum.ReadOnly || noEnum.WriteOnly || noEnum.Deprecated {
		t.Error("NoEnum should have no boolean flags")
	}
}

func assertStr(t *testing.T, label, got, want string) {
	t.Helper()
	if got != want {
		t.Errorf("%s = %q, want %q", label, got, want)
	}
}

func assertFloatPtr(t *testing.T, label string, got, want *float64) {
	t.Helper()
	if want == nil {
		if got != nil {
			t.Errorf("%s = %v, want nil", label, *got)
		}
		return
	}
	if got == nil {
		t.Errorf("%s = nil, want %v", label, *want)
		return
	}
	if *got != *want {
		t.Errorf("%s = %v, want %v", label, *got, *want)
	}
}

func assertIntPtr(t *testing.T, label string, got, want *int64) {
	t.Helper()
	if want == nil {
		if got != nil {
			t.Errorf("%s = %v, want nil", label, *got)
		}
		return
	}
	if got == nil {
		t.Errorf("%s = nil, want %v", label, *want)
		return
	}
	if *got != *want {
		t.Errorf("%s = %v, want %v", label, *got, *want)
	}
}
