package v4_0

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"strings"

	"github.com/nlstn/odata-compliance-suite/framework"
)

// InstanceAnnotations creates the 11.6 Annotations test suite
func InstanceAnnotations() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"11.6 Annotations",
		"Tests handling of instance annotations, control information, and custom annotations in OData responses.",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part1-protocol/odata-v4.0-errata03-os-part1-protocol-complete.html#sec_InstanceAnnotations",
	)

	// Test 1: Standard @odata.context annotation
	suite.AddTest(
		"test_odata_context",
		"@odata.context annotation present",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products")
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			return ctx.AssertJSONField(resp, "@odata.context")
		},
	)

	// Test 2: @odata.count annotation
	suite.AddTest(
		"test_odata_count",
		"@odata.count annotation with $count",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$count=true")
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			var result map[string]interface{}
			if err := json.Unmarshal(resp.Body, &result); err != nil {
				return fmt.Errorf("invalid JSON response: %w", err)
			}

			countVal, ok := result["@odata.count"]
			if !ok {
				return fmt.Errorf("@odata.count required when $count=true is specified")
			}

			countNum, ok := countVal.(float64)
			if !ok {
				return fmt.Errorf("@odata.count must be a number, got %T", countVal)
			}

			if countNum < 0 {
				return fmt.Errorf("@odata.count must be non-negative, got %f", countNum)
			}

			if math.Trunc(countNum) != countNum {
				return fmt.Errorf("@odata.count must be an integer value, got %f", countNum)
			}

			return nil
		},
	)

	// Test 3: @odata.id annotation for entities
	suite.AddTest(
		"test_odata_id",
		"@odata.id annotation for entity",
		func(ctx *framework.TestContext) error {
			productPath, err := firstEntityPath(ctx, "Products")
			if err != nil {
				return err
			}
			resp, err := ctx.GET(productPath)
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			return ctx.AssertJSONField(resp, "@odata.id")
		},
	)

	// Test 4: Annotations in metadata=full
	suite.AddTest(
		"test_annotations_full_metadata",
		"Annotations in metadata=full",
		func(ctx *framework.TestContext) error {
			productPath, err := firstEntityPath(ctx, "Products")
			if err != nil {
				return err
			}
			resp, err := ctx.GET(productPath, framework.Header{Key: "Accept", Value: "application/json;odata.metadata=full"})
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			return ctx.AssertBodyContains(resp, "@odata")
		},
	)

	// Test 5: No annotations in metadata=none
	suite.AddTest(
		"test_no_annotations_none_metadata",
		"No annotations in metadata=none",
		func(ctx *framework.TestContext) error {
			productPath, err := firstEntityPath(ctx, "Products")
			if err != nil {
				return err
			}
			resp, err := ctx.GET(productPath, framework.Header{Key: "Accept", Value: "application/json;odata.metadata=none"})
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			var payload map[string]interface{}
			if err := json.Unmarshal(resp.Body, &payload); err != nil {
				return fmt.Errorf("invalid JSON response: %w", err)
			}
			for key := range payload {
				if strings.HasPrefix(key, "@odata.") {
					return framework.NewError(fmt.Sprintf("metadata=none should omit control information %q", key))
				}
			}

			return nil
		},
	)

	// Test 6: Annotations in collections
	suite.AddTest(
		"test_annotations_in_collections",
		"Annotations in collection responses",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products")
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			if err := ctx.AssertJSONField(resp, "value"); err != nil {
				return err
			}

			return ctx.AssertJSONField(resp, "@odata.context")
		},
	)


	// Test 7: Context control information is first for OData 4.0 JSON responses.
	suite.AddTest(
		"test_context_is_first_property",
		"@odata.context is the first JSON property",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products")
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			first, err := firstJSONObjectPropertyV40(resp.Body)
			if err != nil {
				return err
			}
			if first != "@odata.context" {
				return fmt.Errorf("first JSON property is %q, want @odata.context", first)
			}
			return nil
		},
	)

	// Test 8: id control information is not valid on a collection.
	suite.AddTest(
		"test_collection_has_no_odata_id",
		"Collection response does not contain @odata.id",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products")
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			var payload map[string]interface{}
			if err := json.Unmarshal(resp.Body, &payload); err != nil {
				return fmt.Errorf("invalid JSON response: %w", err)
			}
			if _, present := payload["@odata.id"]; present {
				return framework.NewError("collection response must not contain @odata.id")
			}
			return nil
		},
	)


	// Test 7: Context control information is first for OData 4.0 JSON responses.
	suite.AddTest(
		"test_context_is_first_property",
		"@odata.context is the first JSON property",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products")
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			first, err := firstJSONObjectPropertyV40(resp.Body)
			if err != nil {
				return err
			}
			if first != "@odata.context" {
				return fmt.Errorf("first JSON property is %q, want @odata.context", first)
			}
			return nil
		},
	)

	// Test 8: id control information is not valid on a collection.
	suite.AddTest(
		"test_collection_has_no_odata_id",
		"Collection response does not contain @odata.id",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products")
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			var payload map[string]interface{}
			if err := json.Unmarshal(resp.Body, &payload); err != nil {
				return fmt.Errorf("invalid JSON response: %w", err)
			}
			if _, present := payload["@odata.id"]; present {
				return framework.NewError("collection response must not contain @odata.id")
			}
			return nil
		},
	)

	return suite
}

func firstJSONObjectPropertyV40(body []byte) (string, error) {
	decoder := json.NewDecoder(bytes.NewReader(body))
	token, err := decoder.Token()
	if err != nil {
		return "", fmt.Errorf("read JSON object: %w", err)
	}
	if delimiter, ok := token.(json.Delim); !ok || delimiter != '{' {
		return "", fmt.Errorf("JSON response root is %v, want object", token)
	}
	token, err = decoder.Token()
	if err != nil {
		return "", fmt.Errorf("read first JSON property: %w", err)
	}
	name, ok := token.(string)
	if !ok {
		return "", fmt.Errorf("first JSON property token is %T, want string", token)
	}
	return name, nil
}
