package v4_0

import (
	"time"

	"github.com/nlstn/odata-compliance-suite/framework"
)

// FilterDateFunctions creates the 11.3.2 Date/Time Functions test suite.
//
// Each test verifies the function's actual semantics: the filtered result set is
// compared against an oracle computed in Go from a full fetch (see
// assertProductFilter), not merely checked for HTTP 200.
func FilterDateFunctions() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"11.3.2 Date and Time Functions in $filter",
		"Tests date/time functions (year, month, day, hour, minute, second, fractionalseconds, date, time, now, mindatetime, maxdatetime, totaloffsetminutes) in filter expressions",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part2-url-conventions/odata-v4.0-errata03-os-part2-url-conventions-complete.html#sec_BuiltinFilterOperations",
	)

	suite.AddTest("test_year_function", "year() extracts the year from a DateTimeOffset",
		func(ctx *framework.TestContext) error {
			return assertProductFilter(ctx, "year(CreatedAt) eq 2024", func(p map[string]interface{}) bool {
				t, ok := productTime(p, "CreatedAt")
				return ok && t.Year() == 2024
			})
		})

	suite.AddTest("test_month_function", "month() extracts the month",
		func(ctx *framework.TestContext) error {
			return assertProductFilter(ctx, "month(CreatedAt) eq 1", func(p map[string]interface{}) bool {
				t, ok := productTime(p, "CreatedAt")
				return ok && t.Month() == time.January
			})
		})

	suite.AddTest("test_day_function", "day() extracts the day of month",
		func(ctx *framework.TestContext) error {
			return assertProductFilter(ctx, "day(CreatedAt) eq 15", func(p map[string]interface{}) bool {
				t, ok := productTime(p, "CreatedAt")
				return ok && t.Day() == 15
			})
		})

	suite.AddTest("test_hour_function", "hour() extracts the hour (UTC)",
		func(ctx *framework.TestContext) error {
			return assertProductFilter(ctx, "hour(CreatedAt) lt 12", func(p map[string]interface{}) bool {
				t, ok := productTime(p, "CreatedAt")
				return ok && t.UTC().Hour() < 12
			})
		})

	suite.AddTest("test_minute_function", "minute() extracts the minute",
		func(ctx *framework.TestContext) error {
			return assertProductFilter(ctx, "minute(CreatedAt) eq 30", func(p map[string]interface{}) bool {
				t, ok := productTime(p, "CreatedAt")
				return ok && t.Minute() == 30
			})
		})

	suite.AddTest("test_second_function", "second() extracts the second",
		func(ctx *framework.TestContext) error {
			return assertProductFilter(ctx, "second(CreatedAt) eq 0", func(p map[string]interface{}) bool {
				t, ok := productTime(p, "CreatedAt")
				return ok && t.Second() == 0
			})
		})

	suite.AddTest("test_fractionalseconds_function", "fractionalseconds() extracts the sub-second component",
		func(ctx *framework.TestContext) error {
			return assertProductFilter(ctx, "fractionalseconds(CreatedAt) eq 0", func(p map[string]interface{}) bool {
				t, ok := productTime(p, "CreatedAt")
				return ok && t.Nanosecond() == 0
			})
		})

	suite.AddTest("test_date_function", "date() extracts the date portion of a DateTimeOffset",
		func(ctx *framework.TestContext) error {
			return assertProductFilter(ctx, "date(CreatedAt) eq 2024-01-15", func(p map[string]interface{}) bool {
				t, ok := productTime(p, "CreatedAt")
				return ok && t.UTC().Format("2006-01-02") == "2024-01-15"
			})
		})

	suite.AddTest("test_time_function", "time() extracts the time-of-day portion",
		func(ctx *framework.TestContext) error {
			return assertProductFilter(ctx, "time(CreatedAt) lt 12:00:00", func(p map[string]interface{}) bool {
				t, ok := productTime(p, "CreatedAt")
				return ok && t.UTC().Hour() < 12
			})
		})

	suite.AddTest("test_totaloffsetminutes_function", "totaloffsetminutes() returns the timezone offset in minutes",
		func(ctx *framework.TestContext) error {
			return assertProductFilter(ctx, "totaloffsetminutes(CreatedAt) eq 0", func(p map[string]interface{}) bool {
				t, ok := productTime(p, "CreatedAt")
				if !ok {
					return false
				}
				_, offsetSeconds := t.Zone()
				return offsetSeconds == 0
			})
		})

	suite.AddTest("test_now_function", "now() returns a datetime later than all seed CreatedAt values",
		func(ctx *framework.TestContext) error {
			return assertProductFilter(ctx, "CreatedAt lt now()", func(p map[string]interface{}) bool {
				_, ok := productTime(p, "CreatedAt")
				return ok // all seed CreatedAt values are in the past
			})
		})

	suite.AddTest("test_mindatetime_function", "mindatetime() is earlier than every CreatedAt",
		func(ctx *framework.TestContext) error {
			return assertProductFilter(ctx, "CreatedAt gt mindatetime()", func(p map[string]interface{}) bool {
				_, ok := productTime(p, "CreatedAt")
				return ok
			})
		})

	suite.AddTest("test_maxdatetime_function", "maxdatetime() is later than every CreatedAt",
		func(ctx *framework.TestContext) error {
			return assertProductFilter(ctx, "CreatedAt lt maxdatetime()", func(p map[string]interface{}) bool {
				_, ok := productTime(p, "CreatedAt")
				return ok
			})
		})

	suite.AddTest("test_combined_date_functions", "year() and month() combine in a single predicate",
		func(ctx *framework.TestContext) error {
			return assertProductFilter(ctx, "year(CreatedAt) eq 2024 and month(CreatedAt) ge 6", func(p map[string]interface{}) bool {
				t, ok := productTime(p, "CreatedAt")
				return ok && t.Year() == 2024 && int(t.Month()) >= 6
			})
		})

	return suite
}
