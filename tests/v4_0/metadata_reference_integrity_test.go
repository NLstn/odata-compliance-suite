package v4_0

import "testing"

func TestResolveMetadataEntityType(t *testing.T) {
	metadata := &csdlMetadataDocument{
		References: []csdlMetadataReference{{Includes: []csdlMetadataInclude{{Namespace: "External.Model", Alias: "Ext"}}}},
		DataServices: csdlMetadataDataServices{Schemas: []csdlMetadataSchema{{
			Namespace: "Store.Model", Alias: "Store",
			EntityTypes:  []csdlMetadataEntityType{{Name: "Product"}},
			ComplexTypes: []csdlMetadataComplexType{{Name: "Address"}},
		}}},
	}
	types, aliases, referenced := metadataTypeIndex(metadata)
	for _, tc := range []struct {
		name string
		want bool
	}{
		{"Store.Model.Product", true},
		{"Store.Product", true},
		{"External.Model.RemoteEntity", true},
		{"Ext.RemoteEntity", true},
		{"Product", false},
		{"Store.Model.Address", false},
		{"Store.Address", false},
		{"Collection(Store.Model.Product)", false},
		{"Unknown.Model.Product", false},
	} {
		if got := resolveMetadataEntityType(tc.name, types, aliases, referenced); got != tc.want {
			t.Errorf("resolveMetadataEntityType(%q) = %v, want %v", tc.name, got, tc.want)
		}
	}
}

func TestComplexTypeMemberValidation(t *testing.T) {
	properties := []csdlMetadataProperty{{Name: "Street"}, {Name: "Street"}}
	if err := checkStructuralProperties("ComplexType", "Address", properties); err == nil {
		t.Fatal("duplicate complex type properties must fail")
	}
	properties = []csdlMetadataProperty{{Name: "Street"}}
	navigation := []metadataNavigationProperty{{Name: "Street"}}
	if err := checkNavigationProperties("ComplexType", "Address", properties, navigation); err == nil {
		t.Fatal("complex type navigation name colliding with a property must fail")
	}
}
