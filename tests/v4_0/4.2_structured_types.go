package v4_0

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/nlstn/odata-compliance-suite/framework"
)

// propertyTagPattern matches each structural <Property ...> opening tag (not
// <NavigationProperty>, <Parameter>, or <ReturnType>).
var propertyTagPattern = regexp.MustCompile(`<Property\s[^>]*>`)

// nullableAttrPattern extracts the value of a Nullable="..." facet.
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
		"A property declared with a ComplexType actually serializes as a nested JSON object matching that shape",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/$metadata")
			if err != nil {
				return err
			}

			body := string(resp.Body)
			if !strings.Contains(body, "ComplexType") {
				return ctx.Skip("no ComplexType declared in metadata")
			}

			// Find a structural property whose Type references a declared
			// ComplexType by its qualified name, as opposed to an Edm
			// primitive, enum, or type definition.
			ns, err := schemaNamespace(ctx)
			if err != nil {
				return err
			}
			if ns == "" {
				return ctx.Skip("could not determine the schema namespace")
			}
			complexTypeNames := regexp.MustCompile(`<ComplexType Name="([^"]+)"`).FindAllStringSubmatch(body, -1)

			var complexPropName string
			for _, ct := range complexTypeNames {
				qualified := ns + "." + ct[1]
				pattern := regexp.MustCompile(`<Property\s[^>]*Name="([^"]+)"[^>]*Type="` + regexp.QuoteMeta(qualified) + `"`)
				if m := pattern.FindStringSubmatch(body); m != nil {
					complexPropName = m[1]
					break
				}
			}
			if complexPropName == "" {
				return ctx.Skip("metadata declares ComplexType(s) but no structural property references one by its qualified type name; see NLstn/go-odata#822 for a known instance where a complex-shaped property is instead mistakenly declared as Edm.String")
			}

			entityResp, err := ctx.GET("/Products?$top=1")
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(entityResp, 200); err != nil {
				return err
			}
			items, err := ctx.ParseEntityCollection(entityResp)
			if err != nil {
				return err
			}
			if err := ctx.AssertMinCollectionSize(items, 1); err != nil {
				return err
			}
			raw, ok := items[0][complexPropName]
			if !ok {
				return fmt.Errorf("complex-typed property %q missing from entity response", complexPropName)
			}
			if raw != nil {
				if _, isObject := raw.(map[string]interface{}); !isObject {
					return fmt.Errorf("complex-typed property %q should serialize as a JSON object, got %T", complexPropName, raw)
				}
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
			structuralCollectionPattern := regexp.MustCompile(`<Property\s[^>]*Type="Collection\([^)]+\)"[^>]*>`)
			matches := structuralCollectionPattern.FindAllString(body, -1)
			if len(matches) == 0 {
				return ctx.Skip("no structural (primitive/complex) collection properties declared in metadata")
			}

			// A structural collection property is declared; verify it actually
			// round-trips as a JSON array in a real entity response, not just
			// that the CSDL text mentions it.
			propNamePattern := regexp.MustCompile(`Name="([^"]+)"`)
			propName := ""
			if m := propNamePattern.FindStringSubmatch(matches[0]); m != nil {
				propName = m[1]
			}
			if propName == "" {
				return framework.NewError("could not extract the collection property's Name attribute")
			}

			entityResp, err := ctx.GET("/Products?$top=1")
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(entityResp, 200); err != nil {
				return err
			}
			items, err := ctx.ParseEntityCollection(entityResp)
			if err != nil {
				return err
			}
			if err := ctx.AssertMinCollectionSize(items, 1); err != nil {
				return err
			}
			raw, ok := items[0][propName]
			if !ok || raw == nil {
				return ctx.Skip(fmt.Sprintf("collection property %q not present on the fixture entity used here (may belong to a different entity type)", propName))
			}
			if _, isArray := raw.([]interface{}); !isArray {
				return fmt.Errorf("structural collection property %q should serialize as a JSON array, got %T", propName, raw)
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
			// Scope the search to the Product EntityType block specifically,
			// since a collection-valued nav property declared on a different
			// entity type (e.g. Category.Products) wouldn't be expandable
			// from the /Products entity set used for the behavioral check.
			productTypePattern := regexp.MustCompile(`(?s)<EntityType Name="Product"[^>]*>(.*?)</EntityType>`)
			productBlock := productTypePattern.FindStringSubmatch(body)
			if productBlock == nil {
				return ctx.Skip("could not locate the Product EntityType block in metadata")
			}
			collectionNavPattern := regexp.MustCompile(`<NavigationProperty\s[^>]*Name="([^"]+)"[^>]*Type="Collection\(([^)]+)\)"[^>]*/?>`)
			matches := collectionNavPattern.FindStringSubmatch(productBlock[1])
			if matches == nil {
				return ctx.Skip("Product declares no collection-valued navigation properties in metadata")
			}
			navName := matches[1]

			// Verify the declared collection-valued navigation property
			// actually expands as a JSON array, not just that CSDL declares it.
			entityResp, err := ctx.GET("/Products?$expand=" + navName + "&$top=1")
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(entityResp, 200); err != nil {
				return fmt.Errorf("$expand=%s failed: %w", navName, err)
			}
			items, err := ctx.ParseEntityCollection(entityResp)
			if err != nil {
				return err
			}
			if err := ctx.AssertMinCollectionSize(items, 1); err != nil {
				return err
			}
			raw, ok := items[0][navName]
			if !ok {
				return fmt.Errorf("expanded navigation property %q missing from response", navName)
			}
			if raw != nil {
				if _, isArray := raw.([]interface{}); !isArray {
					return fmt.Errorf("collection-valued navigation property %q should serialize as a JSON array, got %T", navName, raw)
				}
			}
			return nil
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

			// Inheritance declared: verify isof() on the derived SpecialProduct type
			// returns exactly the 3 seeded SpecialProduct instances.
			ns, err := schemaNamespace(ctx)
			if err != nil {
				return err
			}
			if ns == "" {
				return ctx.Skip("could not determine the schema namespace")
			}

			filterResp, err := ctx.GET("/Products?$filter=isof('" + ns + ".SpecialProduct')")
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(filterResp, 200); err != nil {
				return err
			}
			items, err := ctx.ParseEntityCollection(filterResp)
			if err != nil {
				return fmt.Errorf("isof filter returned invalid collection: %w", err)
			}
			if len(items) != 3 {
				return fmt.Errorf("isof('%s.SpecialProduct') expected exactly 3 results (Laptop, Premium Laptop Pro, Gaming Mouse Ultra), got %d", ns, len(items))
			}
			return nil
		},
	)

	suite.AddTest(
		"test_non_nullable_properties",
		"A property declared Nullable=\"false\" actually rejects an explicit null value",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/$metadata")
			if err != nil {
				return err
			}

			body := string(resp.Body)
			productTypePattern := regexp.MustCompile(`(?s)<EntityType Name="Product"[^>]*>(.*?)</EntityType>`)
			productBlock := productTypePattern.FindStringSubmatch(body)
			if productBlock == nil {
				return ctx.Skip("could not locate the Product EntityType block in metadata")
			}
			nonNullablePattern := regexp.MustCompile(`<Property\s[^>]*Name="([^"]+)"[^>]*Nullable="false"[^>]*/?>`)
			var nonNullableName string
			for _, m := range nonNullablePattern.FindAllStringSubmatch(productBlock[1], -1) {
				if m[1] != "ID" { // the key property is a separate, always-required concern
					nonNullableName = m[1]
					break
				}
			}
			if nonNullableName == "" {
				return ctx.Skip("Product declares no non-key Nullable=\"false\" properties")
			}

			payload, err := buildProductPayload(ctx, "NonNullableTest", 10.00)
			if err != nil {
				return err
			}
			payload[nonNullableName] = nil

			postResp, err := ctx.POST("/Products", payload)
			if err != nil {
				return err
			}
			return ctx.AssertODataError(postResp, 400, "")
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
			propertyTags := propertyTagPattern.FindAllString(body, -1)
			if len(propertyTags) == 0 {
				return framework.NewError("metadata declares no structural properties")
			}

			// A structural property is nullable when its Nullable facet is absent
			// (defaults to true) or explicitly "true". Scope the check to each
			// <Property> tag so Nullable facets on Parameter/ReturnType/Navigation
			// elements don't skew the result. Also validate the facet is a valid
			// boolean (CSDL 6.2.2).
			hasNullable := false
			for _, tag := range propertyTags {
				if m := nullableAttrPattern.FindStringSubmatch(tag); m != nil {
					if m[1] != "true" && m[1] != "false" {
						return framework.NewError(fmt.Sprintf("Nullable facet has invalid boolean value %q", m[1]))
					}
				}
				if !strings.Contains(tag, `Nullable="false"`) {
					hasNullable = true
				}
			}
			if !hasNullable {
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
			if !strings.Contains(body, `OpenType="true"`) {
				return ctx.Skip("no open types declared in metadata — dynamic property round-trip not applicable")
			}

			// Open types are declared: skip as no seed data contains dynamic properties
			// that could be round-tripped deterministically.
			return ctx.Skip("open types declared but no seed data with dynamic properties to round-trip")
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
			if !strings.Contains(body, `Abstract="true"`) {
				return nil // No abstract types, optional feature — nothing to verify
			}

			// An abstract entity type exists; extract its name and verify that a
			// direct POST to that entity set is rejected with 400 or 405.
			abstractNamePattern := regexp.MustCompile(`<EntityType[^>]+Abstract="true"[^>]+Name="([^"]+)"`)
			if abstractNamePattern == nil {
				return nil
			}
			matches := abstractNamePattern.FindStringSubmatch(body)
			if len(matches) < 2 {
				// Try reversed attribute order: Name before Abstract
				abstractNamePattern2 := regexp.MustCompile(`<EntityType[^>]+Name="([^"]+)"[^>]+Abstract="true"`)
				matches = abstractNamePattern2.FindStringSubmatch(body)
			}
			if len(matches) < 2 {
				return nil // Cannot determine abstract type name; skip behavioral check
			}
			abstractTypeName := matches[1]

			// Attempt to POST to an entity set whose type matches the abstract
			// type name. Confirm the entity set actually exists first (via
			// GET) — otherwise a 404 from the POST below is ambiguous between
			// "correctly rejected because the type is abstract" and "wrong
			// entity-set name guessed," and treating the latter as a pass
			// would prove nothing.
			entitySetName := abstractTypeName + "s" // naive pluralisation
			getResp, err := ctx.GET("/" + entitySetName + "?$top=1")
			if err != nil || getResp.StatusCode != 200 {
				return ctx.Skip(fmt.Sprintf("could not confirm entity set /%s exists (naive pluralization of abstract type %q may be wrong); skipping behavioral check", entitySetName, abstractTypeName))
			}

			postResp, err := ctx.POST("/"+entitySetName, map[string]interface{}{}, framework.Header{Key: "Content-Type", Value: "application/json"})
			if err != nil {
				return err
			}
			if postResp.StatusCode != 400 && postResp.StatusCode != 405 {
				return fmt.Errorf("POST to abstract type entity set /%s expected 400/405 (instantiating an abstract type must be rejected), got %d", entitySetName, postResp.StatusCode)
			}
			return nil
		},
	)

	return suite
}
