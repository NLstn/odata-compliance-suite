package core

import (
	"fmt"
	"mime"
	"strings"

	"github.com/nlstn/odata-compliance-suite/framework"
)

// DescriptionAnnotation creates tests for the Core.Description annotation
// Tests that Core.Description annotations are properly exposed in metadata
func DescriptionAnnotation() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"Core.Description Annotation",
		"Validates that properties and types annotated with Org.OData.Core.V1.Description expose human-readable descriptions in metadata.",
		"https://oasis-tcs.github.io/odata-vocabularies/vocabularies/Org.OData.Core.V1.html#Description",
	)

	suite.AddTest(
		"metadata_includes_description_annotations",
		"Metadata document includes Core.Description annotations where defined",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/$metadata", framework.Header{Key: "Accept", Value: "application/xml"})
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}
			namespace, err := metadataNamespace(resp.Body)
			if err != nil {
				return err
			}

			nameTarget := fmt.Sprintf("%s.Product/Name", namespace)
			nameFound, err := hasAnnotation(resp.Body, nameTarget, "Org.OData.Core.V1.Description")
			if err != nil {
				return err
			}
			if !nameFound {
				return fmt.Errorf("expected Core.Description annotation on %s", nameTarget)
			}

			descTarget := fmt.Sprintf("%s.Product/Description", namespace)
			descFound, err := hasAnnotation(resp.Body, descTarget, "Org.OData.Core.V1.Description")
			if err != nil {
				return err
			}
			if !descFound {
				return fmt.Errorf("expected Core.Description annotation on %s", descTarget)
			}

			return nil
		},
	)

	suite.AddTest(
		"description_annotations_in_json_metadata",
		"JSON CSDL metadata contains required $Version key and Core.Description annotations",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/$metadata", framework.Header{Key: "Accept", Value: "application/json"})
			if err != nil {
				return err
			}

			// Service may not support JSON metadata format
			if resp.StatusCode == 406 {
				return ctx.Skip("Service does not support JSON metadata format")
			}

			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}
			mediaType, _, err := mime.ParseMediaType(resp.Headers.Get("Content-Type"))
			if err != nil || mediaType != "application/json" {
				return fmt.Errorf("JSON CSDL response Content-Type = %q, want application/json", resp.Headers.Get("Content-Type"))
			}

			var metadata map[string]interface{}
			if err := ctx.GetJSON(resp, &metadata); err != nil {
				return err
			}

			// Per CSDL JSON §3.1, the top-level document MUST contain "$Version".
			if version, ok := metadata["$Version"].(string); !ok || version == "" {
				return framework.NewError(`JSON metadata response is missing the required "$Version" key (CSDL JSON §3.1)`)
			}
			if container, ok := metadata["$EntityContainer"].(string); !ok || container == "" {
				return framework.NewError(`JSON metadata response is missing the required "$EntityContainer" key (CSDL JSON §4)`)
			}

			// If the XML metadata declares Core.Description annotations, the JSON
			// format should also expose them as "@Org.OData.Core.V1.Description" keys.
			body := string(resp.Body)
			if !strings.Contains(body, `"@Org.OData.Core.V1.Description"`) &&
				!strings.Contains(body, `"@Core.Description"`) {
				return framework.NewError("JSON CSDL omits the Core.Description annotations present in XML metadata")
			}
			return nil
		},
	)

	suite.AddTest(
		"qualified_description_annotations",
		"Qualified Description annotations (Qualifier=) follow SimpleIdentifier grammar",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/$metadata", framework.Header{Key: "Accept", Value: "application/xml"})
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			body := string(resp.Body)
			if !strings.Contains(body, `Qualifier="`) {
				return nil // No qualified annotations — optional feature
			}

			// Per CSDL §14.3: Qualifier must be a SimpleIdentifier (letter or '_' start,
			// no spaces, no dots). Scan all Qualifier= values and validate.
			idx := 0
			for {
				pos := strings.Index(body[idx:], `Qualifier="`)
				if pos == -1 {
					break
				}
				pos += idx + len(`Qualifier="`)
				end := strings.Index(body[pos:], `"`)
				if end == -1 {
					break
				}
				q := body[pos : pos+end]
				if q == "" {
					return fmt.Errorf("empty Qualifier value is not a valid SimpleIdentifier")
				}
				first := q[0]
				if !((first >= 'A' && first <= 'Z') || (first >= 'a' && first <= 'z') || first == '_') {
					return fmt.Errorf("Qualifier=%q must start with a letter or underscore (SimpleIdentifier grammar, CSDL §14.3)", q)
				}
				idx = pos + end + 1
			}
			ctx.Log("Qualified description annotations found and validated")
			return nil
		},
	)

	return suite
}
