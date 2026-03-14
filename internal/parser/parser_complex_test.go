package parser

import (
	"testing"
)

func parseComplex(t *testing.T) *RawAST {
	t.Helper()
	p := &GoParser{}
	result, err := p.Parse([]string{"../../testdata/go_complex"})
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	return result
}

func TestComplex_EmbeddedStruct(t *testing.T) {
	result := parseComplex(t)

	product := findStruct(result.Structs, "Product")
	if product == nil {
		t.Fatal("Product not found")
	}

	if len(product.Fields) < 5 {
		t.Fatalf("Product fields len = %d, want >= 5", len(product.Fields))
	}

	// First field is embedded BaseModel (no name)
	embedded := product.Fields[0]
	if embedded.Name != "" {
		t.Errorf("embedded field Name = %q, want empty", embedded.Name)
	}
	if embedded.TypeName != "BaseModel" {
		t.Errorf("embedded field TypeName = %q, want BaseModel", embedded.TypeName)
	}
}

func TestComplex_MapTypes(t *testing.T) {
	result := parseComplex(t)

	product := findStruct(result.Structs, "Product")
	if product == nil {
		t.Fatal("Product not found")
	}

	metadata := findField(product.Fields, "Metadata")
	if metadata == nil {
		t.Fatal("Metadata field not found")
	}
	if metadata.TypeName != "map[string]any" {
		t.Errorf("Metadata.TypeName = %q, want map[string]any", metadata.TypeName)
	}

	order := findStruct(result.Structs, "Order")
	if order == nil {
		t.Fatal("Order not found")
	}
	extra := findField(order.Fields, "Extra")
	if extra == nil {
		t.Fatal("Extra field not found")
	}
	if extra.TypeName != "map[string][]string" {
		t.Errorf("Extra.TypeName = %q, want map[string][]string", extra.TypeName)
	}
}

func TestComplex_PointerAndSliceOfPointers(t *testing.T) {
	result := parseComplex(t)

	product := findStruct(result.Structs, "Product")
	if product == nil {
		t.Fatal("Product not found")
	}

	variants := findField(product.Fields, "Variants")
	if variants == nil {
		t.Fatal("Variants field not found")
	}
	if variants.TypeName != "[]*ProductVariant" {
		t.Errorf("Variants.TypeName = %q, want []*ProductVariant", variants.TypeName)
	}

	order := findStruct(result.Structs, "Order")
	if order == nil {
		t.Fatal("Order not found")
	}
	shipTo := findField(order.Fields, "ShipTo")
	if shipTo == nil {
		t.Fatal("ShipTo field not found")
	}
	if shipTo.TypeName != "*Address" {
		t.Errorf("ShipTo.TypeName = %q, want *Address", shipTo.TypeName)
	}
}

func TestComplex_TypeBlockDeclaration(t *testing.T) {
	result := parseComplex(t)

	for _, name := range []string{"Order", "OrderItem", "Address"} {
		if findStruct(result.Structs, name) == nil {
			t.Errorf("struct %q from type block not found", name)
		}
	}

	order := findStruct(result.Structs, "Order")
	if order == nil {
		t.Fatal("Order not found")
	}
	if len(order.Comments) == 0 {
		t.Error("Order should have doc comment from type block")
	}
	assertContains(t, order.Comments, "// Order represents a purchase order.")

	userID := findField(order.Fields, "UserID")
	if userID == nil {
		t.Fatal("UserID field not found")
	}
	if len(userID.Comments) == 0 {
		t.Error("UserID should have a doc comment")
	}
	assertContains(t, userID.Comments, "// UserID is the buyer.")
}

func TestComplex_EmptyStruct(t *testing.T) {
	result := parseComplex(t)

	empty := findStruct(result.Structs, "Empty")
	if empty == nil {
		t.Fatal("Empty struct not found")
	}
	if len(empty.Fields) != 0 {
		t.Errorf("Empty.Fields len = %d, want 0", len(empty.Fields))
	}
}

func TestComplex_ComplexFieldTypes(t *testing.T) {
	result := parseComplex(t)

	cf := findStruct(result.Structs, "ComplexFields")
	if cf == nil {
		t.Fatal("ComplexFields not found")
	}

	tags := findField(cf.Fields, "Tags")
	if tags == nil {
		t.Fatal("Tags field not found")
	}
	if tags.TypeName != "map[string]interface{}" {
		t.Errorf("Tags.TypeName = %q, want map[string]interface{}", tags.TypeName)
	}

	matrix := findField(cf.Fields, "Matrix")
	if matrix == nil {
		t.Fatal("Matrix field not found")
	}
	if matrix.TypeName != "[][]int" {
		t.Errorf("Matrix.TypeName = %q, want [][]int", matrix.TypeName)
	}

	nested := findField(cf.Fields, "Nested")
	if nested == nil {
		t.Fatal("Nested field not found")
	}
	if nested.TypeName != "*[]*Product" {
		t.Errorf("Nested.TypeName = %q, want *[]*Product", nested.TypeName)
	}
}

func TestComplex_MultipleParamsSameType(t *testing.T) {
	result := parseComplex(t)

	fn := findFunc(result.Functions, "ListProducts")
	if fn == nil {
		t.Fatal("ListProducts not found")
	}

	// func ListProducts(ctx context.Context, offset, limit int, q string)
	// Go AST: 4 params — ctx, offset, limit (grouped), q
	if len(fn.Params) != 4 {
		t.Fatalf("ListProducts.Params len = %d, want 4", len(fn.Params))
	}

	if fn.Params[0].Name != "ctx" || fn.Params[0].TypeName != "context.Context" {
		t.Errorf("Params[0] = %+v, want {ctx context.Context}", fn.Params[0])
	}
	if fn.Params[1].Name != "offset" || fn.Params[1].TypeName != "int" {
		t.Errorf("Params[1] = %+v, want {offset int}", fn.Params[1])
	}
	if fn.Params[2].Name != "limit" || fn.Params[2].TypeName != "int" {
		t.Errorf("Params[2] = %+v, want {limit int}", fn.Params[2])
	}
	if fn.Params[3].Name != "q" || fn.Params[3].TypeName != "string" {
		t.Errorf("Params[3] = %+v, want {q string}", fn.Params[3])
	}

	// Returns: ([]*Product, int64, error)
	if len(fn.Results) != 3 {
		t.Fatalf("ListProducts.Results len = %d, want 3", len(fn.Results))
	}
	if fn.Results[0].TypeName != "[]*Product" {
		t.Errorf("Results[0].TypeName = %q, want []*Product", fn.Results[0].TypeName)
	}
	if fn.Results[1].TypeName != "int64" {
		t.Errorf("Results[1].TypeName = %q, want int64", fn.Results[1].TypeName)
	}
}

func TestComplex_VariadicFunc(t *testing.T) {
	result := parseComplex(t)

	fn := findFunc(result.Functions, "VariadicFunc")
	if fn == nil {
		t.Fatal("VariadicFunc not found")
	}

	if len(fn.Params) != 2 {
		t.Fatalf("VariadicFunc.Params len = %d, want 2", len(fn.Params))
	}
	if fn.Params[1].TypeName != "...int64" {
		t.Errorf("Params[1].TypeName = %q, want ...int64", fn.Params[1].TypeName)
	}
}

func TestComplex_NamedReturns(t *testing.T) {
	result := parseComplex(t)

	fn := findFunc(result.Functions, "NamedReturns")
	if fn == nil {
		t.Fatal("NamedReturns not found")
	}

	if len(fn.Results) != 3 {
		t.Fatalf("NamedReturns.Results len = %d, want 3", len(fn.Results))
	}
	if fn.Results[0].Name != "product" || fn.Results[0].TypeName != "*Product" {
		t.Errorf("Results[0] = %+v, want {product *Product}", fn.Results[0])
	}
	if fn.Results[1].Name != "found" || fn.Results[1].TypeName != "bool" {
		t.Errorf("Results[1] = %+v, want {found bool}", fn.Results[1])
	}
	if fn.Results[2].Name != "err" || fn.Results[2].TypeName != "error" {
		t.Errorf("Results[2] = %+v, want {err error}", fn.Results[2])
	}
}

func TestComplex_SelectorParams(t *testing.T) {
	result := parseComplex(t)

	fn := findFunc(result.Functions, "SelectorParams")
	if fn == nil {
		t.Fatal("SelectorParams not found")
	}

	if len(fn.Params) != 2 {
		t.Fatalf("SelectorParams.Params len = %d, want 2", len(fn.Params))
	}
	if fn.Params[0].TypeName != "http.ResponseWriter" {
		t.Errorf("Params[0].TypeName = %q, want http.ResponseWriter", fn.Params[0].TypeName)
	}
	if fn.Params[1].TypeName != "*http.Request" {
		t.Errorf("Params[1].TypeName = %q, want *http.Request", fn.Params[1].TypeName)
	}
}

func TestComplex_NoDocFunction(t *testing.T) {
	result := parseComplex(t)

	fn := findFunc(result.Functions, "NoDocFunction")
	if fn == nil {
		t.Fatal("NoDocFunction not found")
	}

	if len(fn.Comments) != 0 {
		t.Errorf("NoDocFunction should have 0 comments, got %d", len(fn.Comments))
	}
	if len(fn.Params) != 0 {
		t.Errorf("NoDocFunction should have 0 params, got %d", len(fn.Params))
	}
	if len(fn.Results) != 0 {
		t.Errorf("NoDocFunction should have 0 results, got %d", len(fn.Results))
	}
}

func TestComplex_ValueReceiver(t *testing.T) {
	result := parseComplex(t)

	fn := findFunc(result.Functions, "CancelOrder")
	if fn == nil {
		t.Fatal("CancelOrder not found")
	}
	if fn.Receiver != "OrderService" {
		t.Errorf("CancelOrder.Receiver = %q, want OrderService (value receiver)", fn.Receiver)
	}
}

func TestComplex_PointerReceiver(t *testing.T) {
	result := parseComplex(t)

	fn := findFunc(result.Functions, "CreateOrder")
	if fn == nil {
		t.Fatal("CreateOrder not found")
	}
	if fn.Receiver != "*OrderService" {
		t.Errorf("CreateOrder.Receiver = %q, want *OrderService", fn.Receiver)
	}

	if len(fn.Comments) == 0 {
		t.Fatal("CreateOrder should have comments")
	}
	assertContains(t, fn.Comments, "// @Router      /orders [post]")
}

func TestComplex_StructFieldTags(t *testing.T) {
	result := parseComplex(t)

	baseModel := findStruct(result.Structs, "BaseModel")
	if baseModel == nil {
		t.Fatal("BaseModel not found")
	}

	createdAt := findField(baseModel.Fields, "CreatedAt")
	if createdAt == nil {
		t.Fatal("CreatedAt not found")
	}
	if createdAt.TypeName != "time.Time" {
		t.Errorf("CreatedAt.TypeName = %q, want time.Time", createdAt.TypeName)
	}
	wantTag := "`" + `json:"created_at"` + "`"
	if createdAt.Tag != wantTag {
		t.Errorf("CreatedAt.Tag = %q, want %q", createdAt.Tag, wantTag)
	}
}

func TestComplex_EnumStringType(t *testing.T) {
	result := parseComplex(t)

	alias := findTypeAlias(result.TypeAliases, "OrderStatus")
	if alias == nil {
		t.Fatal("OrderStatus type alias not found")
	}
	if alias.Underlying != "string" {
		t.Errorf("OrderStatus.Underlying = %q, want string", alias.Underlying)
	}
	assertContains(t, alias.Comments, "// OrderStatus is an enum-like type.")

	consts := findConstsByType(result.Consts, "OrderStatus")
	if len(consts) != 5 {
		t.Fatalf("OrderStatus consts len = %d, want 5", len(consts))
	}

	wantValues := map[string]string{
		"OrderStatusPending":    `"pending"`,
		"OrderStatusProcessing": `"processing"`,
		"OrderStatusShipped":    `"shipped"`,
		"OrderStatusDelivered":  `"delivered"`,
		"OrderStatusCancelled":  `"cancelled"`,
	}
	for _, c := range consts {
		want, ok := wantValues[c.Name]
		if !ok {
			t.Errorf("unexpected const %q", c.Name)
			continue
		}
		if c.Value != want {
			t.Errorf("const %s value = %q, want %q", c.Name, c.Value, want)
		}
	}

	// Check comments on specific consts
	pending := findConst(result.Consts, "OrderStatusPending")
	if pending == nil {
		t.Fatal("OrderStatusPending not found")
	}
	assertContains(t, pending.Comments, "// OrderStatusPending means the order is awaiting payment.")

	cancelled := findConst(result.Consts, "OrderStatusCancelled")
	if cancelled == nil {
		t.Fatal("OrderStatusCancelled not found")
	}
	assertContains(t, cancelled.Comments, "// OrderStatusCancelled means the order was cancelled.")
}

func TestComplex_EnumIotaType(t *testing.T) {
	result := parseComplex(t)

	alias := findTypeAlias(result.TypeAliases, "Priority")
	if alias == nil {
		t.Fatal("Priority type alias not found")
	}
	if alias.Underlying != "int" {
		t.Errorf("Priority.Underlying = %q, want int", alias.Underlying)
	}

	consts := findConstsByType(result.Consts, "Priority")
	if len(consts) != 4 {
		t.Fatalf("Priority consts len = %d, want 4", len(consts))
	}

	wantNames := []string{"PriorityLow", "PriorityMedium", "PriorityHigh", "PriorityCritical"}
	for i, want := range wantNames {
		if consts[i].Name != want {
			t.Errorf("consts[%d].Name = %q, want %q", i, consts[i].Name, want)
		}
	}

	// First iota const has value "iota"
	if consts[0].Value != "iota" {
		t.Errorf("PriorityLow.Value = %q, want iota", consts[0].Value)
	}
	// Subsequent iota consts have no explicit value
	for i := 1; i < len(consts); i++ {
		if consts[i].Value != "" {
			t.Errorf("consts[%d].Value = %q, want empty (implicit iota)", i, consts[i].Value)
		}
	}
}

func findField(fields []RawField, name string) *RawField {
	for i := range fields {
		if fields[i].Name == name {
			return &fields[i]
		}
	}
	return nil
}

func findTypeAlias(aliases []RawTypeAlias, name string) *RawTypeAlias {
	for i := range aliases {
		if aliases[i].Name == name {
			return &aliases[i]
		}
	}
	return nil
}

func findConst(consts []RawConst, name string) *RawConst {
	for i := range consts {
		if consts[i].Name == name {
			return &consts[i]
		}
	}
	return nil
}

func findConstsByType(consts []RawConst, typeName string) []RawConst {
	var result []RawConst
	for _, c := range consts {
		if c.TypeName == typeName {
			result = append(result, c)
		}
	}
	return result
}
