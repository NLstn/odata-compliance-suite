package core

import "testing"

func TestHasAnnotationScopesInlineAnnotationsToRequestedProperty(t *testing.T) {
	metadata := []byte(`<?xml version="1.0"?>
<edmx:Edmx xmlns:edmx="http://docs.oasis-open.org/odata/ns/edmx" Version="4.01">
  <edmx:DataServices>
    <Schema xmlns="http://docs.oasis-open.org/odata/ns/edm" Namespace="Example">
      <EntityType Name="Product">
        <Property Name="SerialNumber" Type="Edm.String" />
        <Property Name="CreatedAt" Type="Edm.DateTimeOffset">
          <Annotation Term="Core.Computed" Bool="true" />
        </Property>
      </EntityType>
    </Schema>
  </edmx:DataServices>
</edmx:Edmx>`)

	found, err := hasAnnotation(metadata, "Example.Product/SerialNumber", "Org.OData.Core.V1.Computed")
	if err != nil {
		t.Fatalf("hasAnnotation: %v", err)
	}
	if found {
		t.Fatal("annotation on CreatedAt incorrectly matched SerialNumber")
	}

	found, err = hasAnnotation(metadata, "Example.Product/CreatedAt", "Org.OData.Core.V1.Computed")
	if err != nil {
		t.Fatalf("hasAnnotation: %v", err)
	}
	if !found {
		t.Fatal("inline Core.Computed annotation on CreatedAt was not found")
	}
}
