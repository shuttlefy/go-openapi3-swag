package parser

import "testing"

func TestComposite_NamedFuncType(t *testing.T) {
	result := parseComplex(t)

	hf := findTypeAlias(result.TypeAliases, "HandlerFunc")
	if hf == nil {
		t.Fatal("HandlerFunc not found")
	}
	if hf.Underlying != "func(http.ResponseWriter, *http.Request)" {
		t.Errorf("HandlerFunc.Underlying = %q, want func(http.ResponseWriter, *http.Request)", hf.Underlying)
	}

	mw := findTypeAlias(result.TypeAliases, "Middleware")
	if mw == nil {
		t.Fatal("Middleware not found")
	}
	if mw.Underlying != "func(HandlerFunc) HandlerFunc" {
		t.Errorf("Middleware.Underlying = %q, want func(HandlerFunc) HandlerFunc", mw.Underlying)
	}
}

func TestComposite_GenericStruct(t *testing.T) {
	result := parseComplex(t)

	pg := findStruct(result.Structs, "Pagination")
	if pg == nil {
		t.Fatal("Pagination not found")
	}
	if len(pg.Fields) != 4 {
		t.Fatalf("Pagination fields len = %d, want 4", len(pg.Fields))
	}
	items := findField(pg.Fields, "Items")
	if items == nil {
		t.Fatal("Items field not found")
	}
	if items.TypeName != "[]T" {
		t.Errorf("Items.TypeName = %q, want []T", items.TypeName)
	}

	rs := findStruct(result.Structs, "Result")
	if rs == nil {
		t.Fatal("Result not found")
	}
	data := findField(rs.Fields, "Data")
	if data == nil {
		t.Fatal("Data field not found")
	}
	if data.TypeName != "*T" {
		t.Errorf("Data.TypeName = %q, want *T", data.TypeName)
	}
	errField := findField(rs.Fields, "Error")
	if errField == nil {
		t.Fatal("Error field not found")
	}
	if errField.TypeName != "E" {
		t.Errorf("Error.TypeName = %q, want E", errField.TypeName)
	}
}

func TestComposite_GenericInstantiationEmbedded(t *testing.T) {
	result := parseComplex(t)

	ul := findStruct(result.Structs, "UserList")
	if ul == nil {
		t.Fatal("UserList not found")
	}
	if len(ul.Fields) < 2 {
		t.Fatalf("UserList fields len = %d, want >= 2", len(ul.Fields))
	}

	// Embedded Pagination[User] — no name, TypeName is generic instantiation
	embedded := ul.Fields[0]
	if embedded.Name != "" {
		t.Errorf("embedded field Name = %q, want empty", embedded.Name)
	}
	if embedded.TypeName != "Pagination[User]" {
		t.Errorf("embedded field TypeName = %q, want Pagination[User]", embedded.TypeName)
	}
}

func TestComposite_NestedMapSliceCombinations(t *testing.T) {
	result := parseComplex(t)

	nt := findStruct(result.Structs, "NestedTypes")
	if nt == nil {
		t.Fatal("NestedTypes not found")
	}

	cases := []struct {
		fieldName string
		wantType  string
	}{
		{"SliceOfMaps", "[]map[string]int"},
		{"MapOfSlices", "map[string][]int64"},
		{"MapOfMaps", "map[string]map[string]bool"},
		{"MapOfPtrSlice", "map[int][]*User"},
		{"DeepNested", "*[]map[string][]*User"},
	}

	for _, tc := range cases {
		f := findField(nt.Fields, tc.fieldName)
		if f == nil {
			t.Errorf("field %q not found", tc.fieldName)
			continue
		}
		if f.TypeName != tc.wantType {
			t.Errorf("%s.TypeName = %q, want %q", tc.fieldName, f.TypeName, tc.wantType)
		}
	}
}

func TestComposite_FuncFields(t *testing.T) {
	result := parseComplex(t)

	rt := findStruct(result.Structs, "RuntimeTypes")
	if rt == nil {
		t.Fatal("RuntimeTypes not found")
	}

	cases := []struct {
		fieldName string
		wantType  string
	}{
		{"Callback", "func(string) error"},
		{"Transform", "func(int, string) (bool, error)"},
		{"Events", "chan string"},
		{"SendOnly", "chan<- int"},
		{"RecvOnly", "<-chan bool"},
		{"InlineStruct", "struct{}"},
	}

	for _, tc := range cases {
		f := findField(rt.Fields, tc.fieldName)
		if f == nil {
			t.Errorf("field %q not found", tc.fieldName)
			continue
		}
		if f.TypeName != tc.wantType {
			t.Errorf("%s.TypeName = %q, want %q", tc.fieldName, f.TypeName, tc.wantType)
		}
	}
}

func TestComposite_TypeAliasEquals(t *testing.T) {
	result := parseComplex(t)

	sa := findTypeAlias(result.TypeAliases, "StringAlias")
	if sa == nil {
		t.Fatal("StringAlias not found")
	}
	if sa.Underlying != "string" {
		t.Errorf("StringAlias.Underlying = %q, want string", sa.Underlying)
	}

	as := findStruct(result.Structs, "AliasedStruct")
	if as == nil {
		t.Fatal("AliasedStruct not found")
	}
	label := findField(as.Fields, "Label")
	if label == nil {
		t.Fatal("Label field not found")
	}
	if label.TypeName != "StringAlias" {
		t.Errorf("Label.TypeName = %q, want StringAlias", label.TypeName)
	}
}

func TestComposite_FuncReturningGeneric(t *testing.T) {
	result := parseComplex(t)

	fn := findFunc(result.Functions, "ListUsers")
	if fn == nil {
		t.Fatal("ListUsers not found")
	}
	if len(fn.Params) != 2 {
		t.Fatalf("ListUsers.Params len = %d, want 2", len(fn.Params))
	}
	if fn.Params[0].Name != "page" || fn.Params[0].TypeName != "int" {
		t.Errorf("Params[0] = %+v, want {page int}", fn.Params[0])
	}

	if len(fn.Results) != 1 {
		t.Fatalf("ListUsers.Results len = %d, want 1", len(fn.Results))
	}
	if fn.Results[0].TypeName != "*Pagination[User]" {
		t.Errorf("Results[0].TypeName = %q, want *Pagination[User]", fn.Results[0].TypeName)
	}

	fn2 := findFunc(result.Functions, "ProcessResult")
	if fn2 == nil {
		t.Fatal("ProcessResult not found")
	}
	if len(fn2.Params) != 1 {
		t.Fatalf("ProcessResult.Params len = %d, want 1", len(fn2.Params))
	}
	if fn2.Params[0].TypeName != "Result[User, error]" {
		t.Errorf("Params[0].TypeName = %q, want Result[User, error]", fn2.Params[0].TypeName)
	}
}
