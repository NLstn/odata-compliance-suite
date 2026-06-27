package v4_0

import (
	"strings"

	"github.com/nlstn/odata-compliance-suite/framework"
)

// UnicodeStrings creates the 7.1.1 Unicode and Internationalization test suite
func UnicodeStrings() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"7.1.1 Unicode and Internationalization",
		"Tests handling of Unicode characters including multi-byte characters, emoji, international text, and proper URL encoding.",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part2-url-conventions/odata-v4.0-errata03-os-part2-url-conventions-complete.html#sec_LiteralDataValues",
	)

	suite.AddTest(
		"test_latin_extended",
		"Basic multi-byte Unicode characters (Latin Extended) return matching entities",
		func(ctx *framework.TestContext) error {
			return assertProductFilter(ctx, "contains(Name,'café')", func(p map[string]interface{}) bool {
				return strings.Contains(productString(p, "Name"), "café")
			})
		},
	)

	suite.AddTest(
		"test_cyrillic",
		"Cyrillic characters in filter return matching entities",
		func(ctx *framework.TestContext) error {
			return assertProductFilter(ctx, "contains(Name,'Привет')", func(p map[string]interface{}) bool {
				return strings.Contains(productString(p, "Name"), "Привет")
			})
		},
	)

	suite.AddTest(
		"test_chinese",
		"Chinese characters in filter return matching entities",
		func(ctx *framework.TestContext) error {
			return assertProductFilter(ctx, "contains(Name,'中文')", func(p map[string]interface{}) bool {
				return strings.Contains(productString(p, "Name"), "中文")
			})
		},
	)

	suite.AddTest(
		"test_arabic",
		"Arabic characters in filter return matching entities",
		func(ctx *framework.TestContext) error {
			return assertProductFilter(ctx, "contains(Name,'مرحبا')", func(p map[string]interface{}) bool {
				return strings.Contains(productString(p, "Name"), "مرحبا")
			})
		},
	)

	suite.AddTest(
		"test_emoji",
		"Emoji characters in filter return matching entities",
		func(ctx *framework.TestContext) error {
			return assertProductFilter(ctx, "contains(Name,'🚀')", func(p map[string]interface{}) bool {
				return strings.Contains(productString(p, "Name"), "🚀")
			})
		},
	)

	suite.AddTest(
		"test_mixed_scripts",
		"Mixed script characters in filter return matching entities",
		func(ctx *framework.TestContext) error {
			return assertProductFilter(ctx, "contains(Name,'Hello世界')", func(p map[string]interface{}) bool {
				return strings.Contains(productString(p, "Name"), "Hello世界")
			})
		},
	)

	return suite
}
