package v4_0

import (
	"encoding/xml"
	"testing"
)

func TestProductNameMaxLengthUsesProductProperty(t *testing.T) {
	const metadata = `<Edmx><DataServices>
		<Schema Namespace="Model" Alias="M">
			<EntityType Name="Category"><Property Name="Name" Type="Edm.String"/></EntityType>
			<EntityType Name="Product" BaseType="M.Named"/>
			<EntityType Name="Named"><Property Name="Name" Type="Edm.String" MaxLength="100"></Property></EntityType>
			<EntityContainer Name="Container"><EntitySet Name="Products" EntityType="M.Product"/></EntityContainer>
		</Schema>
	</DataServices></Edmx>`
	var doc csdlDocument
	if err := xml.Unmarshal([]byte(metadata), &doc); err != nil {
		t.Fatal(err)
	}
	limit, err := productNameMaxLength(&doc)
	if err != nil || limit != 100 {
		t.Fatalf("Product.Name MaxLength = %d, error = %v, want 100", limit, err)
	}
}
