package validation

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"regexp"
	"strings"

	"github.com/nlstn/odata-compliance-suite/framework"
)

const productNamePattern = `^[A-Za-z0-9][A-Za-z0-9 -]{0,99}$`

// PatternAnnotation validates the Product Name constraint and verifies that
// the reference data conforms to the advertised regular expression.
func PatternAnnotation() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"Validation.Pattern Annotation",
		"Validates the Product Name regular-expression constraint in metadata and data.",
		"https://oasis-tcs.github.io/odata-vocabularies/vocabularies/Org.OData.Validation.V1.html#Pattern",
	)

	suite.AddTest(
		"xml_metadata_declares_pattern",
		"Product/Name declares the reference Validation.Pattern",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/$metadata", framework.Header{Key: "Accept", Value: "application/xml"})
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}
			value, target, err := patternFromMetadata(resp.Body)
			if err != nil {
				return err
			}
			if !strings.HasSuffix(target, ".Product/Name") {
				return fmt.Errorf("Validation.Pattern target = %q, want Product/Name", target)
			}
			if value != productNamePattern {
				return fmt.Errorf("Validation.Pattern = %q, want %q", value, productNamePattern)
			}
			return nil
		},
	)

	suite.AddTest(
		"json_metadata_declares_pattern",
		"JSON CSDL preserves the Product Name Validation.Pattern",
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
			if !strings.Contains(body, `"@Org.OData.Validation.V1.Pattern"`) &&
				!strings.Contains(body, `"@Validation.Pattern"`) {
				return fmt.Errorf("JSON CSDL is missing Validation.Pattern")
			}
			if !strings.Contains(body, productNamePattern) {
				return fmt.Errorf("JSON CSDL is missing the required Product Name pattern")
			}
			return nil
		},
	)

	suite.AddTest(
		"reference_data_matches_pattern",
		"Every Product Name conforms to the advertised Validation.Pattern",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$select=Name")
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}
			var payload struct {
				Value []struct {
					Name string `json:"Name"`
				} `json:"value"`
			}
			if err := ctx.GetJSON(resp, &payload); err != nil {
				return err
			}
			pattern := regexp.MustCompile(productNamePattern)
			for _, product := range payload.Value {
				if !pattern.MatchString(product.Name) {
					return fmt.Errorf("Product Name %q violates Validation.Pattern", product.Name)
				}
			}
			return nil
		},
	)

	return suite
}

func patternFromMetadata(metadataXML []byte) (string, string, error) {
	decoder := xml.NewDecoder(bytes.NewReader(metadataXML))
	target := ""
	for {
		token, err := decoder.Token()
		if err != nil {
			if err == io.EOF {
				break
			}
			return "", "", fmt.Errorf("failed to parse metadata XML: %w", err)
		}
		switch node := token.(type) {
		case xml.StartElement:
			if node.Name.Local == "Annotations" {
				target = validationXMLAttribute(node, "Target")
			}
			if node.Name.Local == "Annotation" {
				term := validationXMLAttribute(node, "Term")
				if term == "Validation.Pattern" || term == "Org.OData.Validation.V1.Pattern" {
					return validationXMLAttribute(node, "String"), target, nil
				}
			}
		case xml.EndElement:
			if node.Name.Local == "Annotations" {
				target = ""
			}
		}
	}
	return "", "", fmt.Errorf("Validation.Pattern annotation not found")
}

func validationXMLAttribute(start xml.StartElement, name string) string {
	for _, attr := range start.Attr {
		if attr.Name.Local == name {
			return attr.Value
		}
	}
	return ""
}
