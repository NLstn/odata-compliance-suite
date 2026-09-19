package v4_01

import (
	"encoding/xml"
	"fmt"
	"net/url"
	"regexp"
	"sort"

	"github.com/nlstn/odata-compliance-suite/framework"
)

// PrimitiveLiterals creates the OData 4.01 primitive-literal compatibility
// suite. OData 4.01 keeps the 4.0 prefixed forms valid while also requiring
// unprefixed duration and enumeration literals.
func PrimitiveLiterals() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"5.1.1.13 OData 4.01 Primitive Literals",
		"Validates OData 4.01 unprefixed duration and enumeration literals, string-to-primitive casts, and empty action bodies while preserving 4.0 syntax compatibility.",
		"https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part2-url-conventions.html#sec_PrimitiveLiterals",
	)

	queryIDs := func(ctx *framework.TestContext, filter string, maxVersion string) ([]string, error) {
		path := "/Products?$filter=" + url.QueryEscape(filter) + "&$select=ID"
		resp, err := ctx.GET(path, framework.Header{Key: "OData-MaxVersion", Value: maxVersion})
		if err != nil {
			return nil, err
		}
		if err := ctx.AssertStatusCode(resp, 200); err != nil {
			return nil, err
		}
		items, err := ctx.ParseEntityCollection(resp)
		if err != nil {
			return nil, err
		}
		ids := make([]string, 0, len(items))
		for _, item := range items {
			id, ok := item["ID"]
			if !ok || id == nil {
				return nil, fmt.Errorf("filtered product is missing ID")
			}
			ids = append(ids, fmt.Sprint(id))
		}
		sort.Strings(ids)
		return ids, nil
	}

	assertSameIDs := func(ctx *framework.TestContext, prefixed, unprefixed, maxVersion, label string) error {
		prefixedIDs, err := queryIDs(ctx, prefixed, maxVersion)
		if err != nil {
			return fmt.Errorf("%s prefixed literal: %w", label, err)
		}
		unprefixedIDs, err := queryIDs(ctx, unprefixed, maxVersion)
		if err != nil {
			return fmt.Errorf("%s unprefixed literal: %w", label, err)
		}
		if len(prefixedIDs) == 0 {
			return fmt.Errorf("%s prefixed literal returned no products", label)
		}
		if fmt.Sprint(prefixedIDs) != fmt.Sprint(unprefixedIDs) {
			return fmt.Errorf("%s prefixed and unprefixed forms returned different products: prefixed=%v unprefixed=%v", label, prefixedIDs, unprefixedIDs)
		}
		return nil
	}

	// OData 4.01 requires both literal forms regardless of whether the client
	// asks for a 4.0 or 4.01 response version.
	for _, maxVersion := range []string{"4.01", "4.0"} {
		maxVersion := maxVersion
		suite.AddTestWithCapabilities(
			"test_duration_literal_without_prefix_"+maxVersion,
			"Duration literals with and without the duration prefix are equivalent with OData-MaxVersion: "+maxVersion,
			[]framework.RequiredCapability{framework.Require(framework.CapFilter, "Products")},
			func(ctx *framework.TestContext) error {
				return assertSameIDs(ctx, "ShippingTime eq duration'P1D'", "ShippingTime eq 'P1D'", maxVersion, "duration")
			},
		)

		suite.AddTestWithCapabilities(
			"test_enum_literal_without_prefix_"+maxVersion,
			"Enumeration literals with and without the qualified type prefix are equivalent with OData-MaxVersion: "+maxVersion,
			[]framework.RequiredCapability{framework.Require(framework.CapFilter, "Products")},
			func(ctx *framework.TestContext) error {
				namespace, err := productNamespace(ctx)
				if err != nil {
					return err
				}
				qualified := fmt.Sprintf("Status eq %s.ProductStatus'InStock'", namespace)
				return assertSameIDs(ctx, qualified, "Status eq 'InStock'", maxVersion, "enumeration")
			},
		)
	}

	suite.AddTestWithCapabilities(
		"test_cast_string_to_primitive_4_01",
		"4.01 cast converts a string literal to a primitive type",
		[]framework.RequiredCapability{framework.Require(framework.CapFilter, "Products")},
		func(ctx *framework.TestContext) error {
			trueIDs, err := queryIDs(ctx, "cast('5',Edm.Int32) eq 5", "4.01")
			if err != nil {
				return fmt.Errorf("string-to-Int32 cast true expression: %w", err)
			}
			falseIDs, err := queryIDs(ctx, "cast('5',Edm.Int32) eq 6", "4.01")
			if err != nil {
				return fmt.Errorf("string-to-Int32 cast false expression: %w", err)
			}
			if len(trueIDs) == 0 {
				return fmt.Errorf("string-to-Int32 cast true expression returned no products")
			}
			if len(falseIDs) != 0 {
				return fmt.Errorf("string-to-Int32 cast false expression returned %d products", len(falseIDs))
			}
			return nil
		},
	)

	suite.AddTest(
		"test_action_without_body_4_01",
		"A bound action with no non-binding parameters accepts both an empty body and no body",
		func(ctx *framework.TestContext) error {
			path, err := firstEntityPath(ctx, "Products")
			if err != nil {
				return err
			}
			headers := []framework.Header{{Key: "OData-MaxVersion", Value: "4.01"}}
			noBody, err := ctx.POST(path+"/Activate", nil, headers...)
			if err != nil {
				return err
			}
			if noBody.StatusCode < 200 || noBody.StatusCode >= 300 {
				return fmt.Errorf("no-body Activate request returned %d: %s", noBody.StatusCode, string(noBody.Body))
			}
			emptyBody, err := ctx.POST(path+"/Activate", map[string]interface{}{}, headers...)
			if err != nil {
				return err
			}
			if emptyBody.StatusCode < 200 || emptyBody.StatusCode >= 300 {
				return fmt.Errorf("empty-object Activate request returned %d: %s", emptyBody.StatusCode, string(emptyBody.Body))
			}
			return nil
		},
	)

	return suite
}

func productNamespace(ctx *framework.TestContext) (string, error) {
	resp, err := ctx.GET("/$metadata")
	if err != nil {
		return "", err
	}
	if err := ctx.AssertStatusCode(resp, 200); err != nil {
		return "", err
	}

	// Keep the XML parse in the helper deliberately small: the namespace is
	// discovered from the Product EntityType rather than hard-coded to the
	// reference server's current namespace.
	var document struct {
		DataServices struct {
			Schemas []struct {
				Namespace string `xml:"Namespace,attr"`
				Entities  []struct {
					Name string `xml:"Name,attr"`
				} `xml:"EntityType"`
			} `xml:"Schema"`
		} `xml:"DataServices"`
	}
	if err := xml.Unmarshal(resp.Body, &document); err != nil {
		return "", fmt.Errorf("parse $metadata XML: %w", err)
	}
	for _, schema := range document.DataServices.Schemas {
		for _, entity := range schema.Entities {
			if entity.Name == "Product" && schema.Namespace != "" {
				return schema.Namespace, nil
			}
		}
	}

	// Some services use a non-standard metadata serializer; retain a helpful
	// fallback before skipping the enum-specific check.
	match := regexp.MustCompile(`EntityType="([^\"]+)\.Product"`).FindSubmatch(resp.Body)
	if match != nil {
		return string(match[1]), nil
	}
	return "", ctx.Skip("could not determine the Product entity type namespace")
}
