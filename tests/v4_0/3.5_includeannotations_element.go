package v4_0

import (
	"encoding/xml"
	"fmt"
	"strings"

	"github.com/nlstn/odata-compliance-suite/framework"
)

// IncludeAnnotationsElement creates the 3.5 Element edmx:IncludeAnnotations test suite
func IncludeAnnotationsElement() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"3.5 Element edmx:IncludeAnnotations",
		"Validates edmx:IncludeAnnotations elements for including annotations from references according to the OData v4 specification.",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part3-csdl/odata-v4.0-errata03-os-part3-csdl-complete.html#_Toc453752505",
	)

	// Test: Validate edmx:IncludeAnnotations element structure if present
	suite.AddTest(
		"test_includeannotations_element_structure",
		"edmx:IncludeAnnotations elements have correct structure if present",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/$metadata")
			if err != nil {
				return err
			}

			// Check 1: Status code must be 200
			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			body := string(resp.Body)

			// Check 2: edmx:IncludeAnnotations is optional - test passes if not present
			if !strings.Contains(body, "<edmx:IncludeAnnotations") {
				return nil
			}

			// IncludeAnnotations exist - validate their structure

			// Check 3: MUST have TermNamespace attribute
			if !strings.Contains(body, "<edmx:IncludeAnnotations") || !strings.Contains(body, "TermNamespace=") {
				return framework.NewError("edmx:IncludeAnnotations elements must have TermNamespace attribute")
			}

			// Check 4: TermNamespace attribute must not be empty
			if strings.Contains(body, `TermNamespace=""`) {
				return framework.NewError("TermNamespace attribute must not be empty")
			}

			// Check 5: If Qualifier is present, it must not be empty
			if strings.Contains(body, `<edmx:IncludeAnnotations`) && strings.Contains(body, `Qualifier=""`) {
				return framework.NewError("Qualifier attribute must not be empty if specified")
			}

			// Check 6: If TargetNamespace is present, it must not be empty
			if strings.Contains(body, `<edmx:IncludeAnnotations`) && strings.Contains(body, `TargetNamespace=""`) {
				return framework.NewError("TargetNamespace attribute must not be empty if specified")
			}

			return nil
		},
	)

	// Test: Verify each edmx:IncludeAnnotations has a non-empty TermNamespace attribute (spec §3.5)
	suite.AddTest(
		"test_includeannotations_has_termnamespace",
		"Each edmx:IncludeAnnotations MUST have a non-empty TermNamespace attribute (spec §3.5)",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/$metadata", framework.Header{Key: "Accept", Value: "application/xml"})
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			if !strings.Contains(string(resp.Body), "<edmx:IncludeAnnotations") {
				return ctx.Skip("no edmx:IncludeAnnotations elements in metadata")
			}

			type IncludeAnnotations struct {
				TermNamespace string `xml:"TermNamespace,attr"`
			}
			type Reference struct {
				IncludeAnnotations []IncludeAnnotations `xml:"IncludeAnnotations"`
			}
			type EdmxDoc struct {
				XMLName    xml.Name    `xml:"Edmx"`
				References []Reference `xml:"Reference"`
			}
			var doc EdmxDoc
			if err := xml.Unmarshal(resp.Body, &doc); err != nil {
				return framework.NewError(fmt.Sprintf("Failed to parse metadata XML: %v", err))
			}

			for ri, ref := range doc.References {
				for ii, ia := range ref.IncludeAnnotations {
					if strings.TrimSpace(ia.TermNamespace) == "" {
						return framework.NewError(fmt.Sprintf(
							"edmx:Reference[%d]/edmx:IncludeAnnotations[%d] is missing a non-empty TermNamespace attribute (spec §3.5)",
							ri, ii,
						))
					}
				}
			}
			return nil
		},
	)

	// Test: Verify Qualifier on edmx:IncludeAnnotations is non-empty if present (spec §3.5)
	suite.AddTest(
		"test_includeannotations_qualifier_format",
		"edmx:IncludeAnnotations Qualifier attribute MUST be non-empty when present (spec §3.5)",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/$metadata", framework.Header{Key: "Accept", Value: "application/xml"})
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			if !strings.Contains(string(resp.Body), "<edmx:IncludeAnnotations") {
				return ctx.Skip("no edmx:IncludeAnnotations elements in metadata")
			}

			type IncludeAnnotations struct {
				TermNamespace string `xml:"TermNamespace,attr"`
				Qualifier     string `xml:"Qualifier,attr"`
			}
			type Reference struct {
				IncludeAnnotations []IncludeAnnotations `xml:"IncludeAnnotations"`
			}
			type EdmxDoc struct {
				XMLName    xml.Name    `xml:"Edmx"`
				References []Reference `xml:"Reference"`
			}
			var doc EdmxDoc
			if err := xml.Unmarshal(resp.Body, &doc); err != nil {
				return framework.NewError(fmt.Sprintf("Failed to parse metadata XML: %v", err))
			}

			// Only fail if the attribute is explicitly present but empty.
			// encoding/xml sets the field to "" both when absent and when empty="";
			// we use a string-check on the raw body to distinguish the two cases.
			body := string(resp.Body)
			if strings.Contains(body, `Qualifier=""`) {
				return framework.NewError(
					"edmx:IncludeAnnotations Qualifier attribute must not be empty when present (spec §3.5)",
				)
			}
			// For completeness, also validate via parsed struct (catches whitespace-only values).
			for ri, ref := range doc.References {
				for ii, ia := range ref.IncludeAnnotations {
					if ia.Qualifier != "" && strings.TrimSpace(ia.Qualifier) == "" {
						return framework.NewError(fmt.Sprintf(
							"edmx:Reference[%d]/edmx:IncludeAnnotations[%d] Qualifier attribute is whitespace-only (spec §3.5)",
							ri, ii,
						))
					}
				}
			}
			return nil
		},
	)

	return suite
}
