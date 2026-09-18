package v4_01

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/nlstn/odata-compliance-suite/framework"
)

// JSONControlInformation validates JSON control-information rules that are
// distinct from the existing 4.0 annotation smoke tests.
func JSONControlInformation() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"10.2 OData 4.01 JSON Control Information",
		"Validates ordering, collection identity, metadata suppression, and unknown annotation handling in OData 4.01 JSON payloads.",
		"https://docs.oasis-open.org/odata/odata-json-format/v4.01/cs02/odata-json-format-v4.01-cs02.html#sec_ControlInformation",
	)

	suite.AddTest(
		"test_context_is_first_property",
		"Context control information is the first property in a JSON response",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products",
				framework.Header{Key: "Accept", Value: "application/json"},
				framework.Header{Key: "OData-MaxVersion", Value: "4.01"},
			)
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			first, err := firstJSONObjectProperty(resp.Body)
			if err != nil {
				return err
			}
			if first != "@context" && first != "@odata.context" {
				return fmt.Errorf("first JSON property is %q, want @context or @odata.context", first)
			}
			return nil
		},
	)

	suite.AddTest(
		"test_collection_has_no_id_control_information",
		"Collection responses do not contain id control information",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products",
				framework.Header{Key: "Accept", Value: "application/json"},
				framework.Header{Key: "OData-MaxVersion", Value: "4.01"},
			)
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			var payload map[string]interface{}
			if err := json.Unmarshal(resp.Body, &payload); err != nil {
				return fmt.Errorf("invalid JSON collection response: %w", err)
			}
			for _, key := range []string{"@id", "@odata.id"} {
				if _, present := payload[key]; present {
					return fmt.Errorf("collection response contains forbidden %s control information", key)
				}
			}
			return nil
		},
	)

	suite.AddTest(
		"test_metadata_none_omits_control_information",
		"metadata=none omits JSON control information",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products",
				framework.Header{Key: "Accept", Value: "application/json;odata.metadata=none"},
				framework.Header{Key: "OData-MaxVersion", Value: "4.01"},
			)
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			var payload map[string]interface{}
			if err := json.Unmarshal(resp.Body, &payload); err != nil {
				return fmt.Errorf("invalid JSON metadata=none response: %w", err)
			}
			for key := range payload {
				if strings.HasPrefix(key, "@") {
					return fmt.Errorf("metadata=none response contains control information %q", key)
				}
			}
			return nil
		},
	)

	suite.AddTest(
		"test_unknown_annotations_are_ignored",
		"Unknown annotations in a 4.01 request do not cause an error",
		func(ctx *framework.TestContext) error {
			payload, err := productPayload401(ctx)
			if err != nil {
				return err
			}
			payload["@example.unknown"] = "must be ignored"

			resp, err := ctx.POST("/Products", payload,
				framework.Header{Key: "Content-Type", Value: "application/json"},
				framework.Header{Key: "OData-Version", Value: "4.01"},
				framework.Header{Key: "OData-MaxVersion", Value: "4.01"},
			)
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(resp, 201); err != nil {
				return fmt.Errorf("unknown annotation was not ignored: %w", err)
			}
			return nil
		},
	)

	return suite
}

func firstJSONObjectProperty(body []byte) (string, error) {
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

func productPayload401(ctx *framework.TestContext) (map[string]interface{}, error) {
	resp, err := ctx.GET("/Categories?$top=1&$select=ID")
	if err != nil {
		return nil, err
	}
	if err := ctx.AssertStatusCode(resp, 200); err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		return nil, fmt.Errorf("parse category response: %w", err)
	}
	items, ok := result["value"].([]interface{})
	if !ok || len(items) == 0 {
		return nil, fmt.Errorf("reference service returned no category rows")
	}
	first, ok := items[0].(map[string]interface{})
	if !ok || first["ID"] == nil {
		return nil, fmt.Errorf("reference service returned no category ID")
	}

	return map[string]interface{}{
		"Name":       "4.01 Unknown Annotation Probe",
		"Price":      12.34,
		"CategoryID": fmt.Sprintf("%v", first["ID"]),
		"Status":     1,
	}, nil
}
