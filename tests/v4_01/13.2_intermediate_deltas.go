package v4_01

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/nlstn/odata-compliance-suite/framework"
)

// IntermediateDeltas covers OData 4.01 URL and expression behavior that is
// not exercised by the individual operator/header suites.
func IntermediateDeltas() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"13.2 OData 4.01 Intermediate Deltas",
		"Validates 4.01 navigation null comparisons, nested $select, filtered collection counts, special-octet aliases, and optional 4.01 URL/query extensions.",
		"https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part1-protocol.html#sec_OData401IntermediateConformanceLevel",
	)

	collectionCount := func(ctx *framework.TestContext, path string) (int, error) {
		resp, err := ctx.GET(path,
			framework.Header{Key: "Accept", Value: "application/json"},
			framework.Header{Key: "OData-MaxVersion", Value: "4.01"},
		)
		if err != nil {
			return 0, err
		}
		if err := ctx.AssertStatusCode(resp, http.StatusOK); err != nil {
			return 0, err
		}
		var payload struct {
			Value []map[string]interface{} `json:"value"`
			Count interface{}              `json:"@odata.count"`
		}
		if err := json.Unmarshal(resp.Body, &payload); err != nil {
			return 0, fmt.Errorf("invalid JSON collection response: %w", err)
		}
		if payload.Count != nil {
			if count, ok := payload.Count.(float64); ok {
				return int(count), nil
			}
		}
		return len(payload.Value), nil
	}

	suite.AddTestWithCapabilities(
		"test_single_navigation_null_comparison",
		"eq and ne null comparisons on a single-valued navigation property partition the entity set",
		[]framework.RequiredCapability{framework.Require(framework.CapFilter, "Products")},
		func(ctx *framework.TestContext) error {
			total, err := collectionCount(ctx, "/Products?$select=ID&$count=true")
			if err != nil {
				return fmt.Errorf("baseline product count: %w", err)
			}
			nullCount, err := collectionCount(ctx, "/Products?$filter="+url.QueryEscape("Category eq null")+"&$select=ID&$count=true")
			if err != nil {
				return fmt.Errorf("Category eq null: %w", err)
			}
			nonNullCount, err := collectionCount(ctx, "/Products?$filter="+url.QueryEscape("Category ne null")+"&$select=ID&$count=true")
			if err != nil {
				return fmt.Errorf("Category ne null: %w", err)
			}
			if nullCount+nonNullCount != total {
				return fmt.Errorf("Category null comparisons do not partition Products: total=%d eq-null=%d ne-null=%d", total, nullCount, nonNullCount)
			}
			return nil
		},
	)

	suite.AddTestWithCapabilities(
		"test_filtered_collection_count_in_expression",
		"$count with a nested $filter can be used in a common filter expression",
		[]framework.RequiredCapability{framework.Require(framework.CapFilter, "Categories")},
		func(ctx *framework.TestContext) error {
			expression := "Products/$count($filter=Price gt 100) gt 0"
			path := "/Categories?$filter=" + url.QueryEscape(expression) + "&$select=ID"
			count, err := collectionCount(ctx, path)
			if err != nil {
				return fmt.Errorf("filtered collection count expression: %w", err)
			}
			if count == 0 {
				return fmt.Errorf("filtered collection count returned no categories; reference data contains categories with products priced over 100")
			}
			return nil
		},
	)

	suite.AddTest(
		"test_nested_select_on_complex_property",
		"A complex property in $select accepts a nested $select option",
		func(ctx *framework.TestContext) error {
			selectOption := url.QueryEscape("ID,ShippingAddress($select=City,Country)")
			filter := url.QueryEscape("Name eq 'Laptop'")
			resp, err := ctx.GET("/Products?$filter="+filter+"&$select="+selectOption,
				framework.Header{Key: "Accept", Value: "application/json"},
				framework.Header{Key: "OData-MaxVersion", Value: "4.01"},
			)
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(resp, http.StatusOK); err != nil {
				return err
			}
			items, err := ctx.ParseEntityCollection(resp)
			if err != nil {
				return err
			}
			if len(items) != 1 {
				return fmt.Errorf("nested complex $select returned %d Laptop entities, want 1", len(items))
			}
			address, ok := items[0]["ShippingAddress"].(map[string]interface{})
			if !ok {
				return fmt.Errorf("nested complex $select did not return ShippingAddress as an object: %T", items[0]["ShippingAddress"])
			}
			for _, property := range []string{"City", "Country"} {
				if _, ok := address[property]; !ok {
					return fmt.Errorf("nested complex $select omitted selected ShippingAddress.%s", property)
				}
			}
			for _, property := range []string{"Street", "State", "PostalCode"} {
				if _, ok := address[property]; ok {
					return fmt.Errorf("nested complex $select returned unselected ShippingAddress.%s", property)
				}
			}
			return nil
		},
	)

	suite.AddTest(
		"test_nested_select_query_options_optional",
		"A collection-valued navigation property in $select can carry nested query options when supported",
		func(ctx *framework.TestContext) error {
			selectOption := url.QueryEscape("ID,Descriptions($filter=LanguageKey eq 'EN';$select=LanguageKey;$top=1)")
			filter := url.QueryEscape("Name eq 'Laptop'")
			resp, err := ctx.GET("/Products?$filter="+filter+"&$select="+selectOption,
				framework.Header{Key: "Accept", Value: "application/json"},
				framework.Header{Key: "OData-MaxVersion", Value: "4.01"},
			)
			if err != nil {
				return err
			}
			if resp.StatusCode == http.StatusBadRequest || resp.StatusCode == http.StatusNotImplemented {
				return ctx.Skip("service does not support optional nested query options in $select")
			}
			if err := ctx.AssertStatusCode(resp, http.StatusOK); err != nil {
				return err
			}
			items, err := ctx.ParseEntityCollection(resp)
			if err != nil {
				return err
			}
			if len(items) != 1 {
				return fmt.Errorf("nested $select query options returned %d Laptop entities, want 1", len(items))
			}
			descriptions, ok := items[0]["Descriptions"].([]interface{})
			if !ok || len(descriptions) != 1 {
				return fmt.Errorf("nested $select query options returned %d descriptions, want 1", len(descriptions))
			}
			description, ok := descriptions[0].(map[string]interface{})
			if !ok {
				return fmt.Errorf("nested description is not an object")
			}
			if language, ok := description["LanguageKey"].(string); !ok || language != "EN" {
				return fmt.Errorf("nested description LanguageKey=%v, want EN", description["LanguageKey"])
			}
			if _, ok := description["Description"]; ok {
				return fmt.Errorf("nested $select returned unselected Description property")
			}
			return nil
		},
	)

	suite.AddTest(
		"test_structural_comparison_optional",
		"Equal and non-equal structural comparisons work when supported",
		func(ctx *framework.TestContext) error {
			compare := func(operator string) (*framework.HTTPResponse, error) {
				filter := url.QueryEscape("Name eq 'Laptop' and ShippingAddress " + operator + " ShippingAddress")
				return ctx.GET("/Products?$filter="+filter+"&$select=ID", framework.Header{Key: "OData-MaxVersion", Value: "4.01"})
			}
			equalResp, err := compare("eq")
			if err != nil {
				return err
			}
			if equalResp.StatusCode == http.StatusBadRequest || equalResp.StatusCode == http.StatusNotImplemented {
				return ctx.Skip("service does not support optional structural comparison")
			}
			if err := ctx.AssertStatusCode(equalResp, http.StatusOK); err != nil {
				return err
			}
			notEqualResp, err := compare("ne")
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(notEqualResp, http.StatusOK); err != nil {
				return err
			}
			equalItems, err := ctx.ParseEntityCollection(equalResp)
			if err != nil {
				return err
			}
			notEqualItems, err := ctx.ParseEntityCollection(notEqualResp)
			if err != nil {
				return err
			}
			if len(equalItems) != 1 || len(notEqualItems) != 0 {
				return fmt.Errorf("structural comparisons returned eq=%d, ne=%d; want eq=1, ne=0", len(equalItems), len(notEqualItems))
			}
			return nil
		},
	)

	suite.AddTest(
		"test_parameter_alias_special_octets",
		"Function parameter aliases preserve NUL, forward-slash, and backslash octets in string literals",
		func(ctx *framework.TestContext) error {
			productPath, err := firstEntityPath(ctx, "Products")
			if err != nil {
				return err
			}
			for _, octet := range []string{"\x00", "/", "\\"} {
				literal := url.QueryEscape("'alias-" + octet + "-value'")
				path := productPath + "/GetInfo(format=@format)?@format=" + literal
				resp, err := ctx.GET(path, framework.Header{Key: "OData-MaxVersion", Value: "4.01"})
				if err != nil {
					return err
				}
				if resp.StatusCode < 200 || resp.StatusCode >= 300 {
					return fmt.Errorf("GetInfo alias containing %q returned %d: %s", octet, resp.StatusCode, string(resp.Body))
				}
			}
			return nil
		},
	)

	suite.AddTest(
		"test_filter_path_segment",
		"The 4.01 /$filter path segment filters a collection before applying query options",
		func(ctx *framework.TestContext) error {
			path := "/Products/$filter(" + url.PathEscape("Price gt 100") + ")?$select=ID,Price"
			resp, err := ctx.GET(path, framework.Header{Key: "OData-MaxVersion", Value: "4.01"})
			if err != nil {
				return err
			}
			if resp.StatusCode == http.StatusBadRequest || resp.StatusCode == http.StatusNotImplemented || resp.StatusCode == http.StatusNotFound {
				return ctx.Skip("service does not support the optional 4.01 /$filter path segment")
			}
			if err := ctx.AssertStatusCode(resp, http.StatusOK); err != nil {
				return err
			}
			items, err := ctx.ParseEntityCollection(resp)
			if err != nil {
				return err
			}
			if len(items) == 0 {
				return fmt.Errorf("/$filter(Price gt 100) returned no products")
			}
			for i, item := range items {
				price, ok := item["Price"].(float64)
				if !ok || price <= 100 {
					return fmt.Errorf("filtered product %d has Price=%v, expected > 100", i, item["Price"])
				}
			}
			return nil
		},
	)

	suite.AddTest(
		"test_negative_substring_index",
		"Negative substring indexes address characters from the end when supported",
		func(ctx *framework.TestContext) error {
			filter := url.QueryEscape("substring(Name,-1) eq 'p'")
			resp, err := ctx.GET("/Products?$filter="+filter+"&$select=Name", framework.Header{Key: "OData-MaxVersion", Value: "4.01"})
			if err != nil {
				return err
			}
			if resp.StatusCode == http.StatusBadRequest || resp.StatusCode == http.StatusNotImplemented {
				return ctx.Skip("service does not support optional negative substring indexes")
			}
			if err := ctx.AssertStatusCode(resp, http.StatusOK); err != nil {
				return err
			}
			items, err := ctx.ParseEntityCollection(resp)
			if err != nil {
				return err
			}
			for i, item := range items {
				name, _ := item["Name"].(string)
				if !strings.HasSuffix(name, "p") {
					return fmt.Errorf("negative substring result %d has Name=%q, expected a final 'p'", i, name)
				}
			}
			return nil
		},
	)

	suite.AddTest(
		"test_annotation_value_in_expression",
		"A service advertising AnnotationValuesInQuerySupported can filter using a property annotation value",
		func(ctx *framework.TestContext) error {
			metadata, err := ctx.GET("/$metadata")
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(metadata, http.StatusOK); err != nil {
				return err
			}
			if !regexp.MustCompile(`(?i)AnnotationValuesInQuerySupported[^>]*Bool=["']true["']`).Match(metadata.Body) {
				return ctx.Skip("service does not advertise Capabilities.AnnotationValuesInQuerySupported")
			}
			expression := "Name/@Org.OData.Core.V1.Description eq 'Product display name'"
			resp, err := ctx.GET("/Products?$filter="+url.QueryEscape(expression)+"&$top=1", framework.Header{Key: "OData-MaxVersion", Value: "4.01"})
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(resp, http.StatusOK); err != nil {
				return err
			}
			items, err := ctx.ParseEntityCollection(resp)
			if err != nil {
				return err
			}
			if len(items) == 0 {
				return fmt.Errorf("annotation-value filter returned no products")
			}
			return nil
		},
	)

	suite.AddTest(
		"test_default_namespace_operation_aliases",
		"Operations in a declared default namespace work with and without namespace qualification",
		func(ctx *framework.TestContext) error {
			metadata, err := ctx.GET("/$metadata")
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(metadata, http.StatusOK); err != nil {
				return err
			}
			match := regexp.MustCompile(`(?i)Term=["'](?:Core\.)?DefaultNamespace["'][^>]*String=["']([^"']+)["']`).FindSubmatch(metadata.Body)
			if match == nil {
				return ctx.Skip("service does not declare Core.DefaultNamespace")
			}
			namespace := string(match[1])
			qualified := "/" + namespace + ".GetTopProducts()"
			unqualifiedResp, err := ctx.GET("/GetTopProducts()", framework.Header{Key: "OData-MaxVersion", Value: "4.01"})
			if err != nil {
				return err
			}
			qualifiedResp, err := ctx.GET(qualified, framework.Header{Key: "OData-MaxVersion", Value: "4.01"})
			if err != nil {
				return err
			}
			if unqualifiedResp.StatusCode < 200 || unqualifiedResp.StatusCode >= 300 {
				return fmt.Errorf("unqualified GetTopProducts returned %d", unqualifiedResp.StatusCode)
			}
