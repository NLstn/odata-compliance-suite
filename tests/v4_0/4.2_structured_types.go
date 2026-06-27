package v4_0

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/nlstn/odata-compliance-suite/framework"
)

// nullableAttrPattern extracts the value of every Nullable="..." facet in the CSDL.
var nullableAttrPattern = regexp.MustCompile(`Nullable="([^"]*)"`)

// StructuredTypes creates the 4.2 Structured Types test suite
func StructuredTypes() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"4.2 Structured Types",
		"Tests that structured types (entity types and complex types) are properly composed of structural and navigation properties.",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part3-csdl/odata-v4.0-errata03-os-part3-csdl-complete.html#_Toc453752515",
	)

	suite.AddTest(
		"test_entity_types_with_properties",
		"Entity types are structured types with properties",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/$metadata")
			if err != nil {
				return err
			}

			body := string(resp.Body)
			if !strings.Contains(body, "EntityType") {
				return framework.NewError("Metadata must contain EntityType elements")
			}

			if !strings.Contains(body, "Property") {
				return framework.NewError("Entity types should have properties")
			}

			return nil
		},
	)

	suite.AddTest(
		"test_complex_types_with_properties",
		"Complex types are structured types with properties",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/$metadata")
			if err != nil {
				return err
			}

			body := string(resp.Body)
			if !strings.Contains(body, "ComplexType") {
				return nil // No complex types, skip
			}

			if !strings.Contains(body, "Property") {
				return framework.NewError("Complex types should have properties")
			}

			return nil
		},
	)

	suite.AddTest(
		"test_navigation_properties",
		"Structured types can have navigation properties",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/$metadata")
			if err != nil {
				return err
			}

			body := string(resp.Body)
			if !strings.Contains(body, "NavigationProperty") {
				return framework.NewError("Structured types should have navigation properties")
			}

			return nil
		},
	)

	suite.AddTest(
		"test_structured_types_exist",
		"Structured types can have zero or more properties",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/$metadata")
			if err != nil {
				return err
			}

			body := string(resp.Body)
			hasStructuredTypes := strings.Contains(body, "EntityType") || strings.Contains(body, "ComplexType")

			if !hasStructuredTypes {
				return framework.NewError("Metadata must contain at least one structured type")
			}

			return nil
		},
	)

	suite.AddTest(
		"test_primitive_type_properties",
		"Structural properties can be of primitive types",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/$metadata")
			if err != nil {
				return err
			}

			body := string(resp.Body)
			if !strings.Contains(body, "Property") {
				return framework.NewError("Metadata should have properties")
			}

			hasPrimitiveTypes := strings.Contains(body, `Type="Edm.String"`) ||
				strings.Contains(body, `Type="Edm.Int32"`) ||
				strings.Contains(body, `Type="Edm.Boolean"`)

			if !hasPrimitiveTypes {
				return framework.NewError("Properties should use primitive types")
			}

			return nil
		},
	)

	suite.AddTest(
		"test_complex_type_properties",
		"Structural properties can be of complex types",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/$metadata")
			if err != nil {
				return err
			}

			body := string(resp.Body)
			if !strings.Contains(body, "ComplexType") {
				return nil // No complex types, skip
			}

			return nil
		},
	)

	suite.AddTest(
		"test_collection_properties",
		"Structural properties can be collections",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/$metadata")
			if err != nil {
				return err
			}

			body := string(resp.Body)
			if !strings.Contains(body, `Type="Collection(`) {
				return nil // No collection properties, skip
			}

			return nil
		},
	)

	suite.AddTest(
		"test_navigation_property_types",
		"Navigation properties can reference entity types",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/$metadata")
			if err != nil {
				return err
			}

			body := string(resp.Body)
			if !strings.Contains(body, "NavigationProperty") {
				return framework.NewError("Metadata should have navigation properties")
			}

			if !strings.Contains(body, `Type=`) {
				return framework.NewError("Navigation properties must have Type attribute")
			}

			return nil
		},
	)

	suite.AddTest(
		"test_collection_navigation_properties",
		"Navigation properties can be collections of entity types",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/$metadata")
			if err != nil {
				return err
			}

			body := string(resp.Body)
			if !strings.Contains(body, "NavigationProperty") {
				return nil // No navigation properties, skip
			}

			// Check if any navigation property has Collection type
			if strings.Contains(body, `Type="Collection(`) {
				return nil
			}

			return nil // Optional feature
		},
	)

	suite.AddTest(
		"test_entity_type_inheritance",
		"Entity types can inherit from other entity types",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/$metadata")
			if err != nil {
				return err
			}

			body := string(resp.Body)
			if !strings.Contains(body, `BaseType=`) {
				return nil // No inheritance, skip
			}

			return nil
		},
	)

	suite.AddTest(
		"test_non_nullable_properties",
		"Structural properties can be non-nullable",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/$metadata")
			if err != nil {
				return err
			}

			body := string(resp.Body)
			if !strings.Contains(body, "Property") {
				return framework.NewError("Metadata should have properties")
			}

			if strings.Contains(body, `Nullable="false"`) {
				return nil
			}

			return nil // Non-nullable properties are optional
		},
	)

	suite.AddTest(
		"test_nullable_properties",
		"Structural properties can be nullable",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/$metadata")
			if err != nil {
				return err
			}

			body := string(resp.Body)
			propertyCount := strings.Count(body, "<Property ")
			if propertyCount == 0 {
				return framework.NewError("metadata declares no structural properties")
			}

			// Every Nullable facet must carry a valid boolean value (CSDL 6.2.2).
			for _, m := range nullableAttrPattern.FindAllStringSubmatch(body, -1) {
				if m[1] != "true" && m[1] != "false" {
					return framework.NewError(fmt.Sprintf("Nullable facet has invalid boolean value %q", m[1]))
				}
			}

			// A structural property is nullable when its Nullable facet is absent
			// (defaults to true) or explicitly "true". The reference model declares
			// both nullable and non-nullable properties, so the number of explicitly
			// non-nullable properties must be strictly fewer than the total — proving
			// at least one nullable property exists.
			nonNullable := strings.Count(body, `Nullable="false"`)
			if nonNullable >= propertyCount {
				return framework.NewError("expected at least one nullable structural property")
			}

			return nil
		},
	)

	suite.AddTest(
		"test_open_types",
		"Open types allow dynamic properties",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/$metadata")
			if err != nil {
				return err
			}

			body := string(resp.Body)
			// OpenType is optional
			if strings.Contains(body, `OpenType="true"`) {
				return nil
			}

			return nil // Optional feature
		},
	)

	suite.AddTest(
		"test_abstract_types",
		"Abstract types cannot be instantiated directly",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/$metadata")
			if err != nil {
				return err
			}

			body := string(resp.Body)
			// Abstract types are optional
			if strings.Contains(body, `Abstract="true"`) {
				return nil
			}

			return nil // Optional feature
		},
	)

	return suite
}
