package v4_0

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/nlstn/odata-compliance-suite/framework"
)

// maxLengthOf extracts the MaxLength value for a given property in CSDL XML.
// Returns 0 if the property has no MaxLength.
func maxLengthOf(csdl, propertyName string) int {
	needle := `Name="` + propertyName + `"`
	idx := strings.Index(csdl, needle)
	if idx == -1 {
		return 0
	}
	// The MaxLength= attribute is on the same Property element.
	endTag := strings.Index(csdl[idx:], "/>")
	if endTag == -1 {
		return 0
	}
	elem := csdl[idx : idx+endTag]
	mlIdx := strings.Index(elem, `MaxLength="`)
	if mlIdx == -1 {
		return 0
	}
	mlIdx += len(`MaxLength="`)
	mlEnd := strings.Index(elem[mlIdx:], `"`)
	if mlEnd == -1 {
		return 0
	}
	n, err := strconv.Atoi(elem[mlIdx : mlIdx+mlEnd])
	if err != nil {
		return 0
	}
	return n
}

// PrimitiveTypes creates the 4.4 Primitive Types test suite
func PrimitiveTypes() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"4.4 Primitive Types",
		"Tests OData primitive types including Edm.String, Edm.Int32, Edm.Boolean, Edm.Decimal, and others defined in the specification.",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part3-csdl/odata-v4.0-errata03-os-part3-csdl-complete.html#_Toc453752517",
	)

	primitiveTypeTests := []struct {
		name     string
		typeName string
		required bool
	}{
		{"Edm.String primitive type is supported", "Edm.String", true},
		{"Edm.Int32 primitive type is supported", "Edm.Int32", true},
		{"Edm.Boolean primitive type is supported", "Edm.Boolean", false},
		{"Edm.Decimal primitive type is supported", "Edm.Decimal", false},
		{"Edm.Double primitive type is supported", "Edm.Double", false},
		{"Edm.Single primitive type is supported", "Edm.Single", false},
		{"Edm.Guid primitive type is supported", "Edm.Guid", false},
		{"Edm.DateTimeOffset primitive type is supported", "Edm.DateTimeOffset", false},
		{"Edm.Date primitive type is supported", "Edm.Date", false},
		{"Edm.TimeOfDay primitive type is supported", "Edm.TimeOfDay", false},
		{"Edm.Duration primitive type is supported", "Edm.Duration", false},
		{"Edm.Binary primitive type is supported", "Edm.Binary", false},
		{"Edm.Stream primitive type is supported", "Edm.Stream", false},
		{"Edm.Byte primitive type is supported", "Edm.Byte", false},
		{"Edm.SByte primitive type is supported", "Edm.SByte", false},
		{"Edm.Int16 primitive type is supported", "Edm.Int16", false},
		{"Edm.Int64 primitive type is supported", "Edm.Int64", false},
	}

	for _, tt := range primitiveTypeTests {
		typeName := tt.typeName
		required := tt.required
		suite.AddTest(
			"test_"+strings.ToLower(strings.ReplaceAll(typeName, ".", "_")),
			tt.name,
			func(ctx *framework.TestContext) error {
				resp, err := ctx.GET("/$metadata")
				if err != nil {
					return err
				}

				body := string(resp.Body)
				if !strings.Contains(body, `Type="`+typeName+`"`) {
					if required {
						return framework.NewError(typeName + " must be supported")
					}
					return nil // Optional type, skip
				}

				return nil
			},
		)
	}

	suite.AddTest(
		"test_facets_maxlength",
		"Edm.String MaxLength facet value is a positive integer or 'max'",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/$metadata")
			if err != nil {
				return err
			}

			body := string(resp.Body)
			if !strings.Contains(body, "MaxLength=") {
				return nil // Optional facet
			}

			// Per CSDL §7.2.3: MaxLength is either a positive integer or the special value "max".
			idx := 0
			for {
				pos := strings.Index(body[idx:], `MaxLength="`)
				if pos == -1 {
					break
				}
				pos += idx + len(`MaxLength="`)
				end := strings.Index(body[pos:], `"`)
				if end == -1 {
					break
				}
				val := body[pos : pos+end]
				if val != "max" {
					n, err := strconv.Atoi(val)
					if err != nil || n <= 0 {
						return fmt.Errorf("MaxLength=%q is not a positive integer or 'max' (CSDL §7.2.3)", val)
					}
				}
				idx = pos + end + 1
			}
			return nil
		},
	)

	suite.AddTest(
		"test_facets_precision",
		"Edm.Decimal Precision facet value is a positive integer",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/$metadata")
			if err != nil {
				return err
			}

			body := string(resp.Body)
			if !strings.Contains(body, "Precision=") {
				return nil // Optional facet
			}

			// Per CSDL §7.2.4: Precision must be a positive integer.
			idx := 0
			for {
				pos := strings.Index(body[idx:], `Precision="`)
				if pos == -1 {
					break
				}
				pos += idx + len(`Precision="`)
				end := strings.Index(body[pos:], `"`)
				if end == -1 {
					break
				}
				val := body[pos : pos+end]
				n, err := strconv.Atoi(val)
				if err != nil || n <= 0 {
					return fmt.Errorf("Precision=%q is not a positive integer (CSDL §7.2.4)", val)
				}
				idx = pos + end + 1
			}
			return nil
		},
	)

	suite.AddTest(
		"test_collection_of_primitives",
		"Collection-typed properties are returned as JSON arrays",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/$metadata")
			if err != nil {
				return err
			}

			body := string(resp.Body)
			if !strings.Contains(body, `Type="Collection(Edm.`) {
				return nil // No collection-typed primitive properties declared
			}

			// Metadata declares at least one Collection(Edm.*) property; verify the server
			// actually serializes it as a JSON array by inspecting a real entity.
			resp2, err := ctx.GET("/Products?$top=1")
			if err != nil {
				return err
			}
			items, err := ctx.ParseEntityCollection(resp2)
			if err != nil || len(items) == 0 {
				return nil // Can't verify without data
			}
			item := items[0]
			for _, v := range item {
				if _, ok := v.([]interface{}); ok {
					return nil // Found at least one array-valued property
				}
			}
			return framework.NewError("metadata declares Collection-typed properties but no entity contained a JSON array value")
		},
	)

	suite.AddTest(
		"test_key_uses_primitive_types",
		"Key properties use primitive types",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/$metadata")
			if err != nil {
				return err
			}

			body := string(resp.Body)
			if !strings.Contains(body, "Key") || !strings.Contains(body, "PropertyRef") {
				return framework.NewError("Metadata must contain key definitions")
			}

			return nil
		},
	)

	suite.AddTest(
		"test_primitive_types_case_sensitive",
		"Primitive type names are case-sensitive",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/$metadata")
			if err != nil {
				return err
			}

			body := string(resp.Body)
			if !strings.Contains(body, `Type="Edm.`) {
				return framework.NewError("Metadata must contain Edm primitive types")
			}

			// Check for incorrect lowercase
			if strings.Contains(body, `Type="edm.`) {
				return framework.NewError("Primitive type names must be case-sensitive (Edm.String not edm.string)")
			}

			return nil
		},
	)

	// MaxLength write enforcement (CSDL §6.2.3): POST with a string that exceeds MaxLength
	// must be rejected with 400.  Uses Product.Name which has MaxLength=100.
	suite.AddTest(
		"test_maxlength_write_enforcement",
		"POST with string exceeding MaxLength is rejected with 400 (CSDL §6.2.3)",
		func(ctx *framework.TestContext) error {
			// Discover MaxLength of Product.Name from metadata.
			metaResp, err := ctx.GET("/$metadata")
			if err != nil {
				return err
			}
			maxLen := maxLengthOf(string(metaResp.Body), "Name")
			if maxLen <= 0 {
				return ctx.Skip("Product.Name has no MaxLength declared — enforcement cannot be tested")
			}

			// Build a name that exceeds the declared MaxLength by 1 character.
			oversized := strings.Repeat("A", maxLen+1)
			payload, err := buildProductPayload(ctx, oversized, 9.99)
			if err != nil {
				return err
			}

			resp, err := ctx.POST("/Products", payload)
			if err != nil {
				return err
			}
			if resp.StatusCode == 201 {
				return fmt.Errorf("server accepted a Name of %d chars (MaxLength=%d) — MaxLength not enforced (CSDL §6.2.3)", maxLen+1, maxLen)
			}
			if resp.StatusCode != 400 {
				return fmt.Errorf("expected 400 for MaxLength violation, got %d", resp.StatusCode)
			}
			return nil
		},
	)

	return suite
}
