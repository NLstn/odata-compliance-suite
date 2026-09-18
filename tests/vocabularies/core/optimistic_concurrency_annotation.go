package core

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"strings"

	"github.com/nlstn/odata-compliance-suite/framework"
)

// OptimisticConcurrencyAnnotation validates the PropertyPath collection used
// to identify the Product concurrency token.
func OptimisticConcurrencyAnnotation() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"Core.OptimisticConcurrency Annotation",
		"Validates the Products concurrency-token declaration and its ETag behavior.",
		"https://oasis-tcs.github.io/odata-vocabularies/vocabularies/Org.OData.Core.V1.html#OptimisticConcurrency",
	)

	suite.AddTest(
		"xml_uses_property_path_collection",
		"Products declares Version as a PropertyPath, not a generic String expression",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/$metadata", framework.Header{Key: "Accept", Value: "application/xml"})
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}
			annotation, err := optimisticConcurrency(resp.Body)
			if err != nil {
				return err
			}
			if !strings.HasSuffix(annotation.target, "/Products") {
				return fmt.Errorf("OptimisticConcurrency target = %q, want Products entity set", annotation.target)
			}
			if len(annotation.paths) != 1 || annotation.paths[0] != "Version" {
				return fmt.Errorf("OptimisticConcurrency paths = %v, want [Version]", annotation.paths)
			}
			return nil
		},
	)

	suite.AddTest(
		"json_uses_collection_expression",
		"JSON CSDL encodes OptimisticConcurrency as a collection containing Version",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/$metadata", framework.Header{Key: "Accept", Value: "application/json"})
			if err != nil {
				return err
			}
			if resp.StatusCode == 406 {
				return ctx.Skip("Service does not support JSON metadata format")
			}
			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}
			body := string(resp.Body)
			if !strings.Contains(body, `"@Org.OData.Core.V1.OptimisticConcurrency"`) &&
				!strings.Contains(body, `"@Core.OptimisticConcurrency"`) {
				return fmt.Errorf("JSON CSDL is missing Core.OptimisticConcurrency")
			}
			if !strings.Contains(body, `"Version"`) {
				return fmt.Errorf("JSON CSDL OptimisticConcurrency does not reference Version")
			}
			return nil
		},
	)

	suite.AddTest(
		"advertised_concurrency_has_etag",
		"A Product entity response includes an ETag when optimistic concurrency is advertised",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$top=1")
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}
			if resp.Headers.Get("ETag") == "" && !strings.Contains(string(resp.Body), `"@odata.etag"`) {
				return fmt.Errorf("Product response has neither an ETag header nor @odata.etag")
			}
			return nil
		},
	)

	return suite
}

type optimisticConcurrencyAnnotation struct {
	target string
	paths  []string
}

func optimisticConcurrency(metadataXML []byte) (optimisticConcurrencyAnnotation, error) {
	decoder := xml.NewDecoder(bytes.NewReader(metadataXML))
	insideAnnotation := false
	currentTarget := ""
	var result optimisticConcurrencyAnnotation
	for {
		token, err := decoder.Token()
		if err != nil {
			if err == io.EOF {
				break
			}
			return result, fmt.Errorf("failed to parse metadata XML: %w", err)
		}
		switch node := token.(type) {
		case xml.StartElement:
			if node.Name.Local == "Annotations" {
				currentTarget = attribute(node, "Target")
			}
			if node.Name.Local == "Annotation" {
				term := attribute(node, "Term")
				insideAnnotation = term == "Core.OptimisticConcurrency" || term == "Org.OData.Core.V1.OptimisticConcurrency"
				if insideAnnotation {
					result.target = currentTarget
				}
			}
			if insideAnnotation && node.Name.Local == "PropertyPath" {
				var value string
				if err := decoder.DecodeElement(&value, &node); err != nil {
					return result, err
				}
				result.paths = append(result.paths, strings.TrimSpace(value))
			}
		case xml.EndElement:
			if node.Name.Local == "Annotations" {
				currentTarget = ""
			}
			if node.Name.Local == "Annotation" {
				insideAnnotation = false
			}
		}
	}
	if result.target == "" {
		return result, fmt.Errorf("Core.OptimisticConcurrency annotation not found")
	}
	return result, nil
}
