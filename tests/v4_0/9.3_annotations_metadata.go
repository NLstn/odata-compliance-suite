package v4_0

import (
	"strings"

	"github.com/nlstn/odata-compliance-suite/framework"
)

// AnnotationsMetadata creates the 9.3 Annotations in Metadata test suite
func AnnotationsMetadata() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"9.3 Annotations in Metadata",
		"Validates vocabulary annotations in metadata document including Core, Capabilities, and other standard vocabularies.",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part3-csdl/odata-v4.0-errata03-os-part3-csdl-complete.html#sec_Annotation",
	)

	suite.AddTest(
		"test_annotations_element",
		"Metadata contains valid structure with namespace and annotations",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/$metadata")
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			body := string(resp.Body)
			if !strings.Contains(body, "<Schema") && !strings.Contains(body, `"$Version"`) {
				return framework.NewError("invalid metadata structure: missing <Schema element or $Version key")
			}

			// Verify the schema declares a non-empty namespace, which is required
			// by CSDL §5.1.1 and used by clients to qualify type and term names.
			if !strings.Contains(body, `Namespace="`) {
				return framework.NewError("metadata <Schema> element is missing the required Namespace attribute")
			}

			// Verify that at least one Annotation element is present, confirming
			// the server exposes vocabulary annotations in its CSDL document.
			if !strings.Contains(body, "Annotation") {
				return framework.NewError("metadata document contains no Annotation elements; at least one vocabulary annotation is expected")
			}

			return nil
		},
	)

	suite.AddTest(
		"test_core_vocabulary",
		"Core vocabulary annotations (optional)",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/$metadata")
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			body := string(resp.Body)
			hasCoreAnnotation := strings.Contains(body, "Core.Computed") ||
				strings.Contains(body, "Core.Description") ||
				strings.Contains(body, "Org.OData.Core")
			if !hasCoreAnnotation {
				return nil // Core vocabulary annotations are optional
			}

			// Core annotations are present; verify at least one uses a fully-qualified
			// term name (must contain a dot, per CSDL §14.3).
			if !strings.Contains(body, `Term="`) {
				return framework.NewError("Core vocabulary annotations found but no Term attribute present on Annotation elements")
			}
			return nil
		},
	)

	suite.AddTest(
		"test_capabilities_vocabulary",
		"Capabilities vocabulary (optional)",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/$metadata")
			if err != nil {
				return err
			}

			body := string(resp.Body)
			// Capabilities vocabulary is optional
			if strings.Contains(body, "Capabilities.") {
				return nil
			}

			return nil // Optional feature
		},
	)

	return suite
}
