package v4_0

import "testing"

func TestInheritedKey(t *testing.T) {
	types := map[string]csdlMetadataEntityType{
		"Model.Root": {
			Name: "Root",
			Key:  &csdlMetadataKey{PropertyRefs: []csdlMetadataPropertyRef{{Name: "ID"}}},
		},
		"Model.Middle": {
			Name:     "Middle",
			BaseType: "M.Root",
		},
		"Model.Leaf": {
			Name:     "Leaf",
			BaseType: "Model.Middle",
		},
		"Model.Unkeyed": {Name: "Unkeyed"},
	}
	aliases := map[string]string{"M": "Model"}

	for _, name := range []string{"M.Root", "Model.Middle", "Model.Leaf"} {
		if !inheritedKey(name, types, aliases, map[string]bool{}) {
			t.Errorf("inheritedKey(%q) = false, want true", name)
		}
	}
	if inheritedKey("Model.Unkeyed", types, aliases, map[string]bool{}) {
		t.Error("unkeyed root unexpectedly reports an inherited key")
	}
	if inheritedKey("Referenced.External", types, aliases, map[string]bool{}) {
		t.Error("unresolved referenced type unexpectedly reports an inherited key")
	}
}

func TestInheritedKeyCycleTerminates(t *testing.T) {
	types := map[string]csdlMetadataEntityType{
		"Model.A": {Name: "A", BaseType: "Model.B"},
		"Model.B": {Name: "B", BaseType: "Model.A"},
	}
	if inheritedKey("Model.A", types, nil, map[string]bool{}) {
		t.Error("cyclic unkeyed inheritance unexpectedly reports a key")
	}
}
