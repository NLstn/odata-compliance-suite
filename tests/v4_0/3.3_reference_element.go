package v4_0

import (
	"encoding/xml"
	"fmt"
	"strings"

	"github.com/nlstn/odata-compliance-suite/framework"
)

// ReferenceElement creates the 3.3 Element edmx:Reference test suite
func ReferenceElement() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"3.3 Element edmx:Reference",
		"Validates edmx:Reference elements that reference external CSDL documents according to the OData v4 specification.",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part3-csdl/odata-v4.0-errata03-os-part3-csdl-complete.html#_Toc453752503",
	)

	// Test: Validate edmx:Reference element structure if present
	suite.AddTest(
		"test_reference_element_structure",
		"edmx:Reference elements have correct structure if present",
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

			// Check 2: edmx:Reference is optional - test passes if not present
			if !strings.Contains(body, "<edmx:Reference") {
				return nil
			}

			// References exist - validate their structure

			// Check 3: References MUST have Uri attribute
			if !strings.Contains(body, "<edmx:Reference") || !strings.Contains(body, "Uri=") {
				return framework.NewError("edmx:Reference elements must have Uri attribute")
			}

			// Check 4: Uri attributes must not be empty
			if strings.Contains(body, `Uri=""`) {
				return framework.NewError("Uri attributes must not be empty")
			}

			// Check 5: References should contain Include or IncludeAnnotations (SHOULD requirement)
			// This is informational only, logged via context
			if !strings.Contains(body, "<edmx:Include") && !strings.Contains(body, "<edmx:IncludeAnnotations") {
				ctx.Log("Info: edmx:Reference should contain edmx:Include or edmx:IncludeAnnotations")
			}

			return nil
		},
	)

	// Test: Verify each edmx:Reference has a non-empty Uri attribute (spec §3.3)
	suite.AddTest(
		"test_reference_has_uri",
		"Each edmx:Reference MUST have a non-empty Uri attribute (spec §3.3)",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/$metadata", framework.Header{Key: "Accept", Value: "application/xml"})
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			if !strings.Contains(string(resp.Body), "<edmx:Reference") {
				return ctx.Skip("no edmx:Reference elements in metadata")
			}

			type Reference struct {
				Uri string `xml:"Uri,attr"`
			}
			type EdmxDoc struct {
				XMLName    xml.Name    `xml:"Edmx"`
				References []Reference `xml:"Reference"`
			}
			var doc EdmxDoc
			if err := xml.Unmarshal(resp.Body, &doc); err != nil {
				return framework.NewError(fmt.Sprintf("Failed to parse metadata XML: %v", err))
			}

			for i, ref := range doc.References {
				if strings.TrimSpace(ref.Uri) == "" {
					return framework.NewError(fmt.Sprintf(
						"edmx:Reference[%d] is missing a non-empty Uri attribute (spec §3.3)",
						i,
					))
				}
			}
			return nil
		},
	)

	// Test: Verify each edmx:Reference contains at least one Include or IncludeAnnotations child (spec §3.3)
	suite.AddTest(
		"test_reference_contains_include",
		"Each edmx:Reference MUST contain at least one edmx:Include or edmx:IncludeAnnotations child (spec §3.3)",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/$metadata", framework.Header{Key: "Accept", Value: "application/xml"})
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			if !strings.Contains(string(resp.Body), "<edmx:Reference") {
				return ctx.Skip("no edmx:Reference elements in metadata")
			}

			type Include struct {
				XMLName xml.Name `xml:"Include"`
			}
			type IncludeAnnotations struct {
				XMLName xml.Name `xml:"IncludeAnnotations"`
			}
			type Reference struct {
				Uri                string               `xml:"Uri,attr"`
				Includes           []Include            `xml:"Include"`
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

			for i, ref := range doc.References {
				if len(ref.Includes) == 0 && len(ref.IncludeAnnotations) == 0 {
					return framework.NewError(fmt.Sprintf(
						"edmx:Reference[%d] (Uri=%q) must contain at least one edmx:Include or edmx:IncludeAnnotations child (spec §3.3)",
						i, ref.Uri,
					))
				}
			}
			return nil
		},
	)

	return suite
}
