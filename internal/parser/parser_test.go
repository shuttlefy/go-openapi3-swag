package parser

import (
	"testing"
)

func TestGoParser_Parse_Simple(t *testing.T) {
	p := &GoParser{}
	result, err := p.Parse([]string{"../../testdata/go"})
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if result.Package != "simple" {
		t.Errorf("Package = %q, want %q", result.Package, "simple")
	}

	// --- Functions ---

	if got := len(result.Functions); got != 2 {
		t.Fatalf("len(Functions) = %d, want 2", got)
	}

	getUser := findFunc(result.Functions, "GetUser")
	if getUser == nil {
		t.Fatal("GetUser not found")
	}
	createUser := findFunc(result.Functions, "CreateUser")
	if createUser == nil {
		t.Fatal("CreateUser not found")
	}

	// GetUser: standalone function, no receiver
	if getUser.Receiver != "" {
		t.Errorf("GetUser.Receiver = %q, want empty", getUser.Receiver)
	}
	if len(getUser.Comments) == 0 {
		t.Fatal("GetUser.Comments is empty")
	}
	assertContains(t, getUser.Comments, "// @Summary     Get user by ID")
	assertContains(t, getUser.Comments, "// @Router      /users/{id} [get]")

	if len(getUser.Params) != 1 {
		t.Fatalf("GetUser.Params len = %d, want 1", len(getUser.Params))
	}
	if getUser.Params[0].Name != "id" || getUser.Params[0].TypeName != "int" {
		t.Errorf("GetUser.Params[0] = %+v, want {id int}", getUser.Params[0])
	}

	if len(getUser.Results) != 2 {
		t.Fatalf("GetUser.Results len = %d, want 2", len(getUser.Results))
	}
	if getUser.Results[0].TypeName != "*UserResponse" {
		t.Errorf("GetUser.Results[0].TypeName = %q, want *UserResponse", getUser.Results[0].TypeName)
	}
	if getUser.Results[1].TypeName != "error" {
		t.Errorf("GetUser.Results[1].TypeName = %q, want error", getUser.Results[1].TypeName)
	}

	// CreateUser: method with *UserController receiver
	if createUser.Receiver != "*UserController" {
		t.Errorf("CreateUser.Receiver = %q, want *UserController", createUser.Receiver)
	}
	if len(createUser.Params) != 1 {
		t.Fatalf("CreateUser.Params len = %d, want 1", len(createUser.Params))
	}
	if createUser.Params[0].TypeName != "UserResponse" {
		t.Errorf("CreateUser.Params[0].TypeName = %q, want UserResponse", createUser.Params[0].TypeName)
	}

	// --- Structs ---

	if got := len(result.Structs); got != 3 {
		t.Fatalf("len(Structs) = %d, want 3", got)
	}

	userResp := findStruct(result.Structs, "UserResponse")
	if userResp == nil {
		t.Fatal("UserResponse not found")
	}
	errResp := findStruct(result.Structs, "ErrorResponse")
	if errResp == nil {
		t.Fatal("ErrorResponse not found")
	}

	// UserResponse fields
	if len(userResp.Fields) != 4 {
		t.Fatalf("UserResponse fields len = %d, want 4", len(userResp.Fields))
	}

	assertField(t, userResp.Fields[0], "ID", "int", `json:"id"`)
	assertField(t, userResp.Fields[1], "Name", "string", `json:"name" binding:"required"`)
	assertField(t, userResp.Fields[2], "Email", "*string", `json:"email,omitempty"`)
	assertField(t, userResp.Fields[3], "Tags", "[]string", `json:"tags"`)

	// Parsed tag fields: JSONName, Required, Omitempty
	if userResp.Fields[0].JSONName != "id" {
		t.Errorf("ID.JSONName = %q, want \"id\"", userResp.Fields[0].JSONName)
	}
	if userResp.Fields[0].Required {
		t.Error("ID should not be required")
	}

	if userResp.Fields[1].JSONName != "name" {
		t.Errorf("Name.JSONName = %q, want \"name\"", userResp.Fields[1].JSONName)
	}
	if !userResp.Fields[1].Required {
		t.Error("Name should be required (binding:\"required\")")
	}

	if userResp.Fields[2].JSONName != "email" {
		t.Errorf("Email.JSONName = %q, want \"email\"", userResp.Fields[2].JSONName)
	}
	if !userResp.Fields[2].Omitempty {
		t.Error("Email should have Omitempty=true")
	}
	if userResp.Fields[2].Required {
		t.Error("Email (*string) should not be required")
	}

	if userResp.Fields[3].JSONName != "tags" {
		t.Errorf("Tags.JSONName = %q, want \"tags\"", userResp.Fields[3].JSONName)
	}

	// UserResponse comments
	assertContains(t, userResp.Comments, "// UserResponse represents a user in the system.")

	// ErrorResponse fields
	if len(errResp.Fields) != 2 {
		t.Fatalf("ErrorResponse fields len = %d, want 2", len(errResp.Fields))
	}
	assertField(t, errResp.Fields[0], "Code", "int", `json:"code"`)
	assertField(t, errResp.Fields[1], "Message", "string", `json:"message"`)
}

func findFunc(funcs []RawFunc, name string) *RawFunc {
	for i := range funcs {
		if funcs[i].Name == name {
			return &funcs[i]
		}
	}
	return nil
}

func findStruct(structs []RawStruct, name string) *RawStruct {
	for i := range structs {
		if structs[i].Name == name {
			return &structs[i]
		}
	}
	return nil
}

func assertContains(t *testing.T, comments []string, want string) {
	t.Helper()
	for _, c := range comments {
		if c == want {
			return
		}
	}
	t.Errorf("comments %v does not contain %q", comments, want)
}

func assertField(t *testing.T, f RawField, name, typeName, tag string) {
	t.Helper()
	wantTag := "`" + tag + "`"
	if f.Name != name {
		t.Errorf("field.Name = %q, want %q", f.Name, name)
	}
	if f.TypeName != typeName {
		t.Errorf("field %q TypeName = %q, want %q", name, f.TypeName, typeName)
	}
	if f.Tag != wantTag {
		t.Errorf("field %q Tag = %q, want %q", name, f.Tag, wantTag)
	}
}
