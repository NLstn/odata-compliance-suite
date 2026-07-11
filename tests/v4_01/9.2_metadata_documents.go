package v4_01

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"mime"
	"strings"

	"github.com/nlstn/odata-compliance-suite/framework"
)

// MetadataDocuments validates the CSDL document requirements that are specific
// to an OData 4.01 service, including downgrade to a 4.0 XML representation and
// the JSON CSDL representation required at Advanced conformance.
func MetadataDocuments() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"9.2 OData 4.01 Metadata Documents",
		"Validates XML CSDL version negotiation and the required structure of JSON CSDL metadata.",
		"https://docs.oasis-open.org/odata/odata-csdl-xml/v4.01/os/odata-csdl-xml-v4.01-os.html#sec_CSDLXMLDocument",
	)

	suite.AddTest(
		"test_xml_csdl_4_01_structure",
		"OData 4.01 XML metadata has Edmx Version=4.01 and exactly one DataServices element",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/$metadata",
				framework.Header{Key: "Accept", Value: "application/xml"},
				framework.Header{Key: "OData-MaxVersion", Value: "4.01"},
			)
			if err != nil {
				return err
			}
			return validateXMLCSDL(resp, "4.01")
		},
	)

	suite.AddTest(
		"test_xml_csdl_downgrades_to_4_0",
		"OData-MaxVersion 4.0 returns XML metadata with Edmx Version=4.0",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/$metadata",
				framework.Header{Key: "Accept", Value: "application/xml"},
				framework.Header{Key: "OData-MaxVersion", Value: "4.0"},
			)
			if err != nil {
				return err
			}
			return validateXMLCSDL(resp, "4.0")
		},
	)

	suite.AddTest(
		"test_json_csdl_document_structure",
		"JSON metadata has application/json Content-Type, $Version, and a resolvable $EntityContainer",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/$metadata",
				framework.Header{Key: "Accept", Value: "application/json"},
				framework.Header{Key: "OData-MaxVersion", Value: "4.01"},
			)
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}
			mediaType, _, err := mime.ParseMediaType(resp.Headers.Get("Content-Type"))
			if err != nil || mediaType != "application/json" {
				return fmt.Errorf("JSON CSDL Content-Type = %q, want application/json", resp.Headers.Get("Content-Type"))
			}

			var document map[string]interface{}
			if err := json.Unmarshal(resp.Body, &document); err != nil {
				return fmt.Errorf("invalid JSON CSDL: %w", err)
			}
			version, ok := document["$Version"].(string)
			if !ok || (version != "4.0" && version != "4.01") {
				return fmt.Errorf("JSON CSDL $Version = %v, want 4.0 or 4.01", document["$Version"])
			}
			qualifiedContainer, ok := document["$EntityContainer"].(string)
			if !ok || !strings.Contains(qualifiedContainer, ".") {
				return fmt.Errorf("JSON CSDL $EntityContainer = %v, want a namespace-qualified name", document["$EntityContainer"])
			}
			namespace, containerName, _ := strings.Cut(qualifiedContainer, ".")
			// Split at the last dot because namespaces commonly contain dots.
			if dot := strings.LastIndex(qualifiedContainer, "."); dot > 0 {
				namespace, containerName = qualifiedContainer[:dot], qualifiedContainer[dot+1:]
			}
			schema, ok := document[namespace].(map[string]interface{})
			if !ok {
				return fmt.Errorf("$EntityContainer namespace %q is not present as a schema", namespace)
			}
			container, ok := schema[containerName].(map[string]interface{})
			if !ok {
				return fmt.Errorf("$EntityContainer target %q is not present", qualifiedContainer)
			}
			if kind, _ := container["$Kind"].(string); kind != "EntityContainer" {
				return fmt.Errorf("$EntityContainer target $Kind = %q, want EntityContainer", kind)
			}
			containerCount := 0
			for name, rawSchema := range document {
				if strings.HasPrefix(name, "$") {
					continue
				}
				schemaObject, ok := rawSchema.(map[string]interface{})
				if !ok {
					continue
				}
				for _, rawElement := range schemaObject {
					element, ok := rawElement.(map[string]interface{})
					if !ok {
						continue
					}
					if element["$Kind"] == "EntityContainer" {
						containerCount++
					}
				}
			}
			if containerCount != 1 {
				return fmt.Errorf("JSON metadata defines %d entity containers, want exactly one", containerCount)
			}
			return nil
		},
	)

	return suite
}

func validateXMLCSDL(resp *framework.HTTPResponse, expectedVersion string) error {
	if resp.StatusCode != 200 {
		return fmt.Errorf("metadata status = %d, want 200: %s", resp.StatusCode, string(resp.Body))
	}
	mediaType, _, err := mime.ParseMediaType(resp.Headers.Get("Content-Type"))
	if err != nil || mediaType != "application/xml" {
		return fmt.Errorf("XML CSDL Content-Type = %q, want application/xml", resp.Headers.Get("Content-Type"))
	}
	var doc struct {
		XMLName      xml.Name `xml:"Edmx"`
		Version      string   `xml:"Version,attr"`
		DataServices []struct {
			Schemas []struct{} `xml:"Schema"`
		} `xml:"DataServices"`
	}
	if err := xml.Unmarshal(resp.Body, &doc); err != nil {
		return fmt.Errorf("invalid XML CSDL: %w", err)
	}
	if doc.XMLName.Local != "Edmx" {
		return fmt.Errorf("XML CSDL root = %q, want Edmx", doc.XMLName.Local)
	}
	if doc.Version != expectedVersion {
		return fmt.Errorf("Edmx Version = %q, want %s", doc.Version, expectedVersion)
	}
	if len(doc.DataServices) != 1 {
		return fmt.Errorf("Edmx contains %d DataServices elements, want exactly one", len(doc.DataServices))
	}
	if len(doc.DataServices[0].Schemas) == 0 {
		return fmt.Errorf("DataServices must contain at least one Schema")
	}
	return nil
}
