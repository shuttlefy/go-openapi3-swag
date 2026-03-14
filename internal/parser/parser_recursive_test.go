package parser

import "testing"

func TestRecursive_DirectSelfReference(t *testing.T) {
	result := parseComplex(t)

	s := findStruct(result.Structs, "TreeNode")
	if s == nil {
		t.Fatal("TreeNode not found")
	}

	children := findField(s.Fields, "Children")
	if children == nil {
		t.Fatal("Children field not found")
	}
	if children.TypeName != "[]*TreeNode" {
		t.Errorf("Children.TypeName = %q, want []*TreeNode", children.TypeName)
	}

	parent := findField(s.Fields, "Parent")
	if parent == nil {
		t.Fatal("Parent field not found")
	}
	if parent.TypeName != "*TreeNode" {
		t.Errorf("Parent.TypeName = %q, want *TreeNode", parent.TypeName)
	}
}

func TestRecursive_MutualRecursion(t *testing.T) {
	result := parseComplex(t)

	emp := findStruct(result.Structs, "Employee")
	if emp == nil {
		t.Fatal("Employee not found")
	}
	dept := findStruct(result.Structs, "Department")
	if dept == nil {
		t.Fatal("Department not found")
	}

	// Employee → *Department
	empDept := findField(emp.Fields, "Department")
	if empDept == nil {
		t.Fatal("Employee.Department not found")
	}
	if empDept.TypeName != "*Department" {
		t.Errorf("Employee.Department.TypeName = %q, want *Department", empDept.TypeName)
	}

	// Department → *Employee (manager)
	mgr := findField(dept.Fields, "Manager")
	if mgr == nil {
		t.Fatal("Department.Manager not found")
	}
	if mgr.TypeName != "*Employee" {
		t.Errorf("Department.Manager.TypeName = %q, want *Employee", mgr.TypeName)
	}

	// Department → []*Employee (employees)
	emps := findField(dept.Fields, "Employees")
	if emps == nil {
		t.Fatal("Department.Employees not found")
	}
	if emps.TypeName != "[]*Employee" {
		t.Errorf("Department.Employees.TypeName = %q, want []*Employee", emps.TypeName)
	}
}

func TestRecursive_DeepCycle(t *testing.T) {
	result := parseComplex(t)

	node := findStruct(result.Structs, "GraphNode")
	if node == nil {
		t.Fatal("GraphNode not found")
	}
	edge := findStruct(result.Structs, "GraphEdge")
	if edge == nil {
		t.Fatal("GraphEdge not found")
	}
	label := findStruct(result.Structs, "EdgeLabel")
	if label == nil {
		t.Fatal("EdgeLabel not found")
	}

	// GraphNode → []*GraphEdge
	edges := findField(node.Fields, "Edges")
	if edges == nil {
		t.Fatal("GraphNode.Edges not found")
	}
	if edges.TypeName != "[]*GraphEdge" {
		t.Errorf("Edges.TypeName = %q, want []*GraphEdge", edges.TypeName)
	}

	// GraphEdge → *GraphNode
	target := findField(edge.Fields, "Target")
	if target == nil {
		t.Fatal("GraphEdge.Target not found")
	}
	if target.TypeName != "*GraphNode" {
		t.Errorf("Target.TypeName = %q, want *GraphNode", target.TypeName)
	}

	// GraphEdge → []*EdgeLabel
	labels := findField(edge.Fields, "Labels")
	if labels == nil {
		t.Fatal("GraphEdge.Labels not found")
	}
	if labels.TypeName != "[]*EdgeLabel" {
		t.Errorf("Labels.TypeName = %q, want []*EdgeLabel", labels.TypeName)
	}

	// EdgeLabel → *GraphNode (completes the cycle: Node → Edge → Label → Node)
	labelNode := findField(label.Fields, "Node")
	if labelNode == nil {
		t.Fatal("EdgeLabel.Node not found")
	}
	if labelNode.TypeName != "*GraphNode" {
		t.Errorf("EdgeLabel.Node.TypeName = %q, want *GraphNode", labelNode.TypeName)
	}
}

func TestRecursive_SelfReferencingMap(t *testing.T) {
	result := parseComplex(t)

	s := findStruct(result.Structs, "Config")
	if s == nil {
		t.Fatal("Config not found")
	}

	children := findField(s.Fields, "Children")
	if children == nil {
		t.Fatal("Config.Children not found")
	}
	if children.TypeName != "map[string]*Config" {
		t.Errorf("Children.TypeName = %q, want map[string]*Config", children.TypeName)
	}
}

func TestRecursive_MultipleSelfRefPaths(t *testing.T) {
	result := parseComplex(t)

	s := findStruct(result.Structs, "Comment")
	if s == nil {
		t.Fatal("Comment not found")
	}

	cases := []struct {
		fieldName string
		wantType  string
	}{
		{"Replies", "[]*Comment"},
		{"ReplyTo", "*Comment"},
		{"BestReply", "*Comment"},
	}
	for _, tc := range cases {
		f := findField(s.Fields, tc.fieldName)
		if f == nil {
			t.Errorf("Comment.%s not found", tc.fieldName)
			continue
		}
		if f.TypeName != tc.wantType {
			t.Errorf("Comment.%s.TypeName = %q, want %q", tc.fieldName, f.TypeName, tc.wantType)
		}
	}
}

func TestRecursive_GenericSelfReference(t *testing.T) {
	result := parseComplex(t)

	s := findStruct(result.Structs, "LinkedList")
	if s == nil {
		t.Fatal("LinkedList not found")
	}

	next := findField(s.Fields, "Next")
	if next == nil {
		t.Fatal("LinkedList.Next not found")
	}
	if next.TypeName != "*LinkedList[T]" {
		t.Errorf("Next.TypeName = %q, want *LinkedList[T]", next.TypeName)
	}

	value := findField(s.Fields, "Value")
	if value == nil {
		t.Fatal("LinkedList.Value not found")
	}
	if value.TypeName != "T" {
		t.Errorf("Value.TypeName = %q, want T", value.TypeName)
	}
}
