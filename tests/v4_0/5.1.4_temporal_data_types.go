package v4_0

import (
	"github.com/nlstn/odata-compliance-suite/framework"
)

// TemporalDataTypes creates the 5.1.4 Temporal Data Types test suite
func TemporalDataTypes() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"5.1.4 Temporal Data Types",
		"Validates handling of OData temporal types including Edm.Date, Edm.TimeOfDay, and Edm.Duration in filters and metadata.",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part2-url-conventions/odata-v4.0-errata03-os-part2-url-conventions-complete.html#sec_PrimitiveTypes",
	)

	suite.AddTest(
		"test_datetime_offset_support",
		"year() function on Edm.DateTimeOffset returns matching entities",
		func(ctx *framework.TestContext) error {
			return assertProductFilter(ctx, "year(CreatedAt) eq 2024", func(p map[string]interface{}) bool {
				t, ok := productTime(p, "CreatedAt")
				return ok && t.Year() == 2024
			})
		},
	)

	suite.AddTest(
		"test_date_literal",
		"date() function on DateTimeOffset eq Date literal returns matching entities",
		func(ctx *framework.TestContext) error {
			return assertProductFilter(ctx, "date(CreatedAt) eq 2024-01-15", func(p map[string]interface{}) bool {
				t, ok := productTime(p, "CreatedAt")
				return ok && t.Format("2006-01-02") == "2024-01-15"
			})
		},
	)

	suite.AddTest(
		"test_time_literal",
		"time() function on DateTimeOffset eq TimeOfDay literal returns matching entities",
		func(ctx *framework.TestContext) error {
			return assertProductFilter(ctx, "time(CreatedAt) eq 14:30:00", func(p map[string]interface{}) bool {
				t, ok := productTime(p, "CreatedAt")
				return ok && t.Format("15:04:05") == "14:30:00"
			})
		},
	)

	suite.AddTest(
		"test_date_comparison",
		"date() function gt Date literal returns entities after that date",
		func(ctx *framework.TestContext) error {
			return assertProductFilter(ctx, "date(CreatedAt) gt 2024-01-01", func(p map[string]interface{}) bool {
				t, ok := productTime(p, "CreatedAt")
				return ok && t.Format("2006-01-02") > "2024-01-01"
			})
		},
	)

	suite.AddTest(
		"test_time_comparison",
		"time() function lt TimeOfDay literal returns entities before that time",
		func(ctx *framework.TestContext) error {
			return assertProductFilter(ctx, "time(CreatedAt) lt 12:00:00", func(p map[string]interface{}) bool {
				t, ok := productTime(p, "CreatedAt")
				return ok && t.Format("15:04:05") < "12:00:00"
			})
		},
	)

	suite.AddTest(
		"test_date_time_combination",
		"Combined date() and time() functions return correctly filtered entities",
		func(ctx *framework.TestContext) error {
			return assertProductFilter(ctx, "date(CreatedAt) eq 2024-01-15 and time(CreatedAt) gt 10:00:00", func(p map[string]interface{}) bool {
				t, ok := productTime(p, "CreatedAt")
				return ok && t.Format("2006-01-02") == "2024-01-15" && t.Format("15:04:05") > "10:00:00"
			})
		},
	)

	return suite
}
