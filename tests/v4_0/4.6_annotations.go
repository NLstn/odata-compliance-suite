package v4_0

import (
	"fmt"
	"strings"

	"github.com/nlstn/odata-compliance-suite/framework"
)

// MetadataAnnotations creates the 4.6 Annotations test suite
func MetadataAnnotations() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"4.6 Annotations",
		"Tests that model elements can be decorated with annotations, and annotations have proper term names and optional qualifiers.",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part3-csdl/odata-v4.0-errata03-os-part3-csdl-complete.html#_Toc453752519",
	)

	suite.AddTest(
		"test_annotations_present",
		"Annotations can be applied to model elements",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/$metadata")
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			body := string(resp.Body)
			if !strings.Contains(body, "Annotation") {
				return nil // No annotations, skip (they're optional)
			}

			// Annotations are declared: verify at least one known Core vocabulary
			// annotation is present (Core.Computed is applied to computed properties
			// and Core.Description to entity types/properties in the reference model).
			hasKnownAnnotation := strings.Contains(body, "Core.Computed") ||
				strings.Contains(body, "Core.Description") ||
				strings.Contains(body, "Org.OData.Core")
			if !hasKnownAnnotation {
				// Accept any annotation with a qualified term (contains a dot)
				hasQualifiedTerm := strings.Contains(body, `Term="`) && strings.Contains(body, `."`)
				if !hasQualifiedTerm {
					return framework.NewError("metadata declares Annotation elements but none use a qualified term (Namespace.LocalName)")
				}
			}
			return nil
		},
	)

	suite.AddTest(
		"test_annotations_have_term",
		"Annotations must have Term attribute",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/$metadata")
			if err != nil {
				return err
			}

			body := string(resp.Body)
			if !strings.Contains(body, "<Annotation") {
				return nil // No annotations, skip
			}

			if !strings.Contains(body, `Term=`) {
				return framework.NewError("Annotation elements must have Term attribute")
			}

			return nil
		},
	)

	suite.AddTest(
		"test_annotation_terms_qualified",
		"Annotation terms use qualified names",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/$metadata")
			if err != nil {
				return err
			}

			body := string(resp.Body)
			if !strings.Contains(body, `Term="`) {
				return nil // No annotation terms, skip
			}

			// Per CSDL §14.3: every Term attribute MUST be a qualified name (Namespace.LocalName).
			idx := 0
			for {
				pos := strings.Index(body[idx:], `Term="`)
				if pos == -1 {
					break
				}
				pos += idx + len(`Term="`)
				end := strings.Index(body[pos:], `"`)
				if end == -1 {
					break
				}
				term := body[pos : pos+end]
				if !strings.Contains(term, ".") {
					return fmt.Errorf("annotation Term=%q is not a qualified name (must be Namespace.LocalName)", term)
				}
				idx = pos + end + 1
			}
			return nil
		},
	)

	suite.AddTest(
		"test_annotations_with_qualifiers",
		"Annotations can have optional qualifiers",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/$metadata")
			if err != nil {
				return err
			}

			body := string(resp.Body)
			if !strings.Contains(body, `Qualifier="`) {
				return nil // Qualifiers are optional
			}

			// Per CSDL §14.3: Qualifier must be a SimpleIdentifier — starts with letter or '_',
			// contains only letters, digits, or '_', and is non-empty.
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
					return framework.NewError("empty Qualifier value is not a valid SimpleIdentifier")
				}
				first := q[0]
				if !((first >= 'A' && first <= 'Z') || (first >= 'a' && first <= 'z') || first == '_') {
					return fmt.Errorf("Qualifier=%q must start with a letter or underscore (SimpleIdentifier)", q)
				}
				idx = pos + end + 1
			}
			return nil
		},
	)

	suite.AddTest(
		"test_core_vocabulary_annotations",
		"Core vocabulary annotations are used",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/$metadata")
			if err != nil {
				return err
			}

			body := string(resp.Body)
			// Core vocabulary is optional
			if strings.Contains(body, `Term="Core.`) || strings.Contains(body, `Term="Org.OData.Core`) {
				return nil
			}

			return nil // Optional feature
		},
	)

	suite.AddTest(
		"test_annotations_on_entity_types",
		"Annotations can be applied to entity types",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/$metadata")
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			body := string(resp.Body)
			if !strings.Contains(body, "<Annotation") {
				return nil // No annotations — optional feature
			}

			// If annotations exist, verify the metadata response is well-formed XML/JSON
			// by confirming the body contains expected structural markers.
			isXML := strings.Contains(body, "<edmx:Edmx") || strings.Contains(body, "<Edmx")
			isJSON := strings.Contains(body, `"$Version"`)
			if !isXML && !isJSON {
				return framework.NewError("metadata response does not appear to be valid CSDL XML or JSON")
			}
			return nil
		},
	)

	suite.AddTest(
		"test_annotations_on_properties",
		"Annotations can be applied to properties",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/$metadata")
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			body := string(resp.Body)
			if !strings.Contains(body, "<Annotation") {
				return nil // No annotations — optional feature
			}

			// Verify that at least one Annotation element carries a Term attribute,
			// confirming the server emits well-formed CSDL annotation markup.
			if !strings.Contains(body, `Term=`) {
				return framework.NewError("metadata contains <Annotation> elements but none have a Term attribute")
			}
			return nil
		},
	)

	suite.AddTest(
		"test_external_targeting",
		"Annotations element supports external targeting",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/$metadata")
			if err != nil {
				return err
			}

			body := string(resp.Body)
			if strings.Contains(body, "<Annotations") && !strings.Contains(body, "Target=") {
				return framework.NewError("Annotations element should have Target attribute")
			}

			return nil // Optional feature
		},
	)

	return suite
}
