package v4_0

import (
	"encoding/xml"
	"fmt"
	"strings"

	"github.com/nlstn/odata-compliance-suite/framework"
)

// IncludeElement creates the 3.4 Element edmx:Include test suite
func IncludeElement() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"3.4 Element edmx:Include",
		"Validates edmx:Include elements that include schemas from referenced documents according to the OData v4 specification.",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part3-csdl/odata-v4.0-errata03-os-part3-csdl-complete.html#_Toc453752504",
	)

	// Test: Validate edmx:Include element structure if present
	suite.AddTest(
		"test_include_element_structure",
		"edmx:Include elements have correct structure if present",
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

			// Check 2: edmx:Include is optional - test passes if not present
			if !strings.Contains(body, "<edmx:Include") {
				return nil
			}

			// Includes exist - validate their structure

			// Check 3: Includes MUST have Namespace attribute
			if !strings.Contains(body, "<edmx:Include") || !strings.Contains(body, "Namespace=") {
				return framework.NewError("edmx:Include elements must have Namespace attribute")
			}

			// Check 4: Namespace attributes must not be empty
			if strings.Contains(body, `Namespace=""`) {
				return framework.NewError("Namespace attribute must not be empty")
			}

			// Check 5: If Alias is present, it must not be empty
			if strings.Contains(body, `<edmx:Include`) && strings.Contains(body, `Alias=""`) {
				return framework.NewError("Alias attribute must not be empty if specified")
			}

			return nil
		},
	)

	// Test: Verify each edmx:Include has a non-empty Namespace attribute (spec §3.4)
	suite.AddTest(
		"test_include_has_namespace",
		"Each edmx:Include MUST have a non-empty Namespace attribute (spec §3.4)",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/$metadata", framework.Header{Key: "Accept", Value: "application/xml"})
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			if !strings.Contains(string(resp.Body), "<edmx:Include") {
				return ctx.Skip("no edmx:Include elements in metadata")
			}

			type Include struct {
				Namespace string `xml:"Namespace,attr"`
			}
			type Reference struct {
				Includes []Include `xml:"Include"`
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
				for ii, inc := range ref.Includes {
					if strings.TrimSpace(inc.Namespace) == "" {
						return framework.NewError(fmt.Sprintf(
							"edmx:Reference[%d]/edmx:Include[%d] is missing a non-empty Namespace attribute (spec §3.4)",
							ri, ii,
						))
					}
				}
			}
			return nil
		},
	)

	// Test: Verify Alias on edmx:Include does not start with reserved prefixes (spec §3.4)
	suite.AddTest(
		"test_include_alias_valid",
		"edmx:Include Alias MUST NOT start with \"Edm\" or \"odata\" (spec §3.4)",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/$metadata", framework.Header{Key: "Accept", Value: "application/xml"})
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			if !strings.Contains(string(resp.Body), "<edmx:Include") {
				return ctx.Skip("no edmx:Include elements in metadata")
			}

			type Include struct {
				Namespace string `xml:"Namespace,attr"`
				Alias     string `xml:"Alias,attr"`
			}
			type Reference struct {
				Includes []Include `xml:"Include"`
			}
			type EdmxDoc struct {
				XMLName    xml.Name    `xml:"Edmx"`
				References []Reference `xml:"Reference"`
			}
			var doc EdmxDoc
			if err := xml.Unmarshal(resp.Body, &doc); err != nil {
				return framework.NewError(fmt.Sprintf("Failed to parse metadata XML: %v", err))
			}

			reservedPrefixes := []string{"Edm", "odata"}
			for ri, ref := range doc.References {
				for ii, inc := range ref.Includes {
					if inc.Alias == "" {
						continue // Alias is optional
					}
					for _, prefix := range reservedPrefixes {
						if inc.Alias == prefix || strings.HasPrefix(inc.Alias, prefix+".") {
							return framework.NewError(fmt.Sprintf(
								"edmx:Reference[%d]/edmx:Include[%d] Alias %q must not start with reserved prefix %q (spec §3.4)",
								ri, ii, inc.Alias, prefix,
							))
						}
					}
				}
			}
			return nil
		},
	)

	return suite
}
