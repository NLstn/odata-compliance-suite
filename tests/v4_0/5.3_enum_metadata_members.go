package v4_0

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/nlstn/odata-compliance-suite/framework"
)

// EnumMetadataMembers creates the 5.3 Enumeration Types - Metadata Members test suite
func EnumMetadataMembers() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"5.3 Enumeration Types - Metadata Members",
		"Verifies that enum metadata reflects actual enum values and configured namespace.",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part3-csdl/odata-v4.0-errata03-os-part3-csdl-complete.html#sec_EnumerationType",
	)

	suite.AddTest(
		"test_enum_metadata_xml",
		"ProductStatus enum metadata declares the exact expected members, values, and IsFlags facet",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/$metadata")
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			body := string(resp.Body)
			if !strings.Contains(body, "EnumType") {
				return ctx.Skip("no EnumType declared in metadata")
			}

			// productStatusNames (defined in 11.3.5_filter_logical_operators.go)
			// is the same map the filter/comparison tests already rely on to
			// interpret Status values — reusing it here ties this metadata
			// check to a single source of truth instead of a second hardcoded
			// copy that could silently drift from it.
			enumPattern := regexp.MustCompile(`(?s)<EnumType Name="ProductStatus"([^>]*)>(.*?)</EnumType>`)
			m := enumPattern.FindStringSubmatch(body)
			if m == nil {
				return ctx.Skip("ProductStatus enum not declared in metadata")
			}
			attrs, membersBlock := m[1], m[2]

			if !strings.Contains(attrs, `IsFlags="true"`) {
				return fmt.Errorf("ProductStatus is used as a flags enum (comma-separated member values observed elsewhere in this suite) but metadata does not declare IsFlags=\"true\"")
			}

			memberPattern := regexp.MustCompile(`<Member Name="([^"]+)" Value="([^"]+)"`)
			declared := map[string]string{}
			for _, mm := range memberPattern.FindAllStringSubmatch(membersBlock, -1) {
				declared[mm[1]] = mm[2]
			}
			if len(declared) == 0 {
				return framework.NewError("ProductStatus EnumType declares no Member elements")
			}

			for name, value := range productStatusNames {
				declaredValue, ok := declared[name]
				if !ok {
					return fmt.Errorf("ProductStatus metadata is missing member %q (expected value %d)", name, value)
				}
				if declaredValue != fmt.Sprintf("%d", value) {
					return fmt.Errorf("ProductStatus member %q declared value %q, want %d", name, declaredValue, value)
				}
			}
			if len(declared) != len(productStatusNames) {
				return fmt.Errorf("ProductStatus metadata declares %d member(s), expected exactly %d (matching productStatusNames)", len(declared), len(productStatusNames))
			}
			return nil
		},
	)

	return suite
}
