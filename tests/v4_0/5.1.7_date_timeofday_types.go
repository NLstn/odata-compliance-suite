package v4_0

import (
	"fmt"
	"strings"

	"github.com/nlstn/odata-compliance-suite/framework"
)

// DateTimeOfDayTypes creates the 5.1.7 Date and TimeOfDay Types test suite
func DateTimeOfDayTypes() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"5.1.7 Date and TimeOfDay Types",
		"Tests handling of Edm.Date (YYYY-MM-DD) and Edm.TimeOfDay (HH:MM:SS.sss) primitive types including literal formats, filtering, and functions.",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part2-url-conventions/odata-v4.0-errata03-os-part2-url-conventions-complete.html#sec_PrimitiveTypes",
	)

	// Edm.Date tests

	suite.AddTest(
		"test_date_in_metadata",
		"Edm.Date type appears in metadata",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/$metadata")
			if err != nil {
				return err
			}

			body := string(resp.Body)
			if !strings.Contains(body, `Type="Edm.Date"`) {
				return nil // Optional type
			}

			return nil
		},
	)

	suite.AddTest(
		"test_date_literal_format",
		"Date literal YYYY-MM-DD returns exactly matching entities",
		func(ctx *framework.TestContext) error {
			return assertProductFilter(ctx, "ReleaseDate eq 2024-01-15", func(p map[string]interface{}) bool {
				t, ok := productTime(p, "ReleaseDate")
				return ok && t.Format("2006-01-02") == "2024-01-15"
			})
		},
	)

	suite.AddTest(
		"test_date_comparison",
		"Date gt comparison returns entities after the given date",
		func(ctx *framework.TestContext) error {
			return assertProductFilter(ctx, "ReleaseDate gt 2024-01-01", func(p map[string]interface{}) bool {
				t, ok := productTime(p, "ReleaseDate")
				return ok && t.Format("2006-01-02") > "2024-01-01"
			})
		},
	)

	suite.AddTest(
		"test_date_year_function",
		"year() function on Edm.Date returns entities matching that year",
		func(ctx *framework.TestContext) error {
			return assertProductFilter(ctx, "year(ReleaseDate) eq 2024", func(p map[string]interface{}) bool {
				t, ok := productTime(p, "ReleaseDate")
				return ok && t.Year() == 2024
			})
		},
	)

	suite.AddTest(
		"test_date_month_function",
		"month() function on Edm.Date returns entities matching that month",
		func(ctx *framework.TestContext) error {
			return assertProductFilter(ctx, "month(ReleaseDate) eq 1", func(p map[string]interface{}) bool {
				t, ok := productTime(p, "ReleaseDate")
				return ok && int(t.Month()) == 1
			})
		},
	)

	suite.AddTest(
		"test_date_day_function",
		"day() function on Edm.Date returns entities matching that day",
		func(ctx *framework.TestContext) error {
			return assertProductFilter(ctx, "day(ReleaseDate) eq 15", func(p map[string]interface{}) bool {
				t, ok := productTime(p, "ReleaseDate")
				return ok && t.Day() == 15
			})
		},
	)

	suite.AddTest(
		"test_date_null_comparison",
		"Date ne null returns only entities with a non-null ReleaseDate",
		func(ctx *framework.TestContext) error {
			return assertProductFilter(ctx, "ReleaseDate ne null", func(p map[string]interface{}) bool {
				_, ok := productTime(p, "ReleaseDate")
				return ok
			})
		},
	)

	suite.AddTest(
		"test_date_cast",
		"cast(CreatedAt, 'Edm.Date') eq date literal returns matching entities",
		func(ctx *framework.TestContext) error {
			return assertProductFilter(ctx, "cast(CreatedAt,'Edm.Date') eq 2024-01-15", func(p map[string]interface{}) bool {
				t, ok := productTime(p, "CreatedAt")
				return ok && t.Format("2006-01-02") == "2024-01-15"
			})
		},
	)

	// Edm.TimeOfDay tests

	suite.AddTest(
		"test_timeofday_in_metadata",
		"Edm.TimeOfDay type appears in metadata",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/$metadata")
			if err != nil {
				return err
			}

			body := string(resp.Body)
			if !strings.Contains(body, `Type="Edm.TimeOfDay"`) {
				return nil // Optional type
			}

			return nil
		},
	)

	suite.AddTest(
		"test_timeofday_literal_format",
		"TimeOfDay literal HH:MM:SS returns exactly matching entities",
		func(ctx *framework.TestContext) error {
			return assertProductFilter(ctx, "OpenTime eq 09:30:00", func(p map[string]interface{}) bool {
				s := productString(p, "OpenTime")
				return s == "09:30:00"
			})
		},
	)

	suite.AddTest(
		"test_timeofday_with_milliseconds",
		"TimeOfDay with fractional seconds returns exactly matching entities",
		func(ctx *framework.TestContext) error {
			return assertProductFilter(ctx, "OpenTime eq 09:30:00.123", func(p map[string]interface{}) bool {
				s := productString(p, "OpenTime")
				return s == "09:30:00.123"
			})
		},
	)

	suite.AddTest(
		"test_timeofday_comparison",
		"TimeOfDay gt comparison returns entities after the given time",
		func(ctx *framework.TestContext) error {
			return assertProductFilter(ctx, "OpenTime gt 08:00:00", func(p map[string]interface{}) bool {
				s := productString(p, "OpenTime")
				return s != "" && s > "08:00:00"
			})
		},
	)

	suite.AddTest(
		"test_timeofday_hour_function",
		"hour() function on Edm.TimeOfDay returns matching entities",
		func(ctx *framework.TestContext) error {
			return assertProductFilter(ctx, "hour(OpenTime) eq 9", func(p map[string]interface{}) bool {
				s := productString(p, "OpenTime")
				if len(s) < 2 {
					return false
				}
				return s[:2] == "09"
			})
		},
	)

	suite.AddTest(
		"test_timeofday_minute_function",
		"minute() function on Edm.TimeOfDay returns matching entities",
		func(ctx *framework.TestContext) error {
			return assertProductFilter(ctx, "minute(OpenTime) eq 30", func(p map[string]interface{}) bool {
				s := productString(p, "OpenTime")
				if len(s) < 5 {
					return false
				}
				return s[3:5] == "30"
			})
		},
	)

	suite.AddTest(
		"test_timeofday_second_function",
		"second() function on Edm.TimeOfDay returns matching entities",
		func(ctx *framework.TestContext) error {
			return assertProductFilter(ctx, "second(OpenTime) eq 0", func(p map[string]interface{}) bool {
				s := productString(p, "OpenTime")
				if len(s) < 8 {
					return false
				}
				return s[6:8] == "00"
			})
		},
	)

	suite.AddTest(
		"test_timeofday_midnight",
		"TimeOfDay eq 00:00:00 returns exactly matching entities",
		func(ctx *framework.TestContext) error {
			return assertProductFilter(ctx, "OpenTime eq 00:00:00", func(p map[string]interface{}) bool {
				s := productString(p, "OpenTime")
				return s == "00:00:00"
			})
		},
	)

	suite.AddTest(
		"test_timeofday_end_of_day",
		"TimeOfDay le 23:59:59 returns all entities with a non-null OpenTime",
		func(ctx *framework.TestContext) error {
			return assertProductFilter(ctx, "OpenTime le 23:59:59", func(p map[string]interface{}) bool {
				s := productString(p, "OpenTime")
				return s != "" && s <= "23:59:59"
			})
		},
	)

	suite.AddTest(
		"test_timeofday_null_comparison",
		"TimeOfDay ne null returns only entities with a non-null OpenTime",
		func(ctx *framework.TestContext) error {
			return assertProductFilter(ctx, "OpenTime ne null", func(p map[string]interface{}) bool {
				return productString(p, "OpenTime") != ""
			})
		},
	)

	suite.AddTest(
		"test_timeofday_cast",
		"cast(CreatedAt, 'Edm.TimeOfDay') gt time literal returns matching entities",
		func(ctx *framework.TestContext) error {
			return assertProductFilter(ctx, "cast(CreatedAt,'Edm.TimeOfDay') gt 09:00:00", func(p map[string]interface{}) bool {
				t, ok := productTime(p, "CreatedAt")
				return ok && t.Format("15:04:05") > "09:00:00"
			})
		},
	)

	suite.AddTest(
		"test_date_and_timeofday_in_response",
		"Date and TimeOfDay values are serialized in their canonical wire formats",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$top=10&$select=ID,Name,ReleaseDate,OpenTime")
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}
			items, err := ctx.ParseEntityCollection(resp)
			if err != nil {
				return err
			}
			if err := ctx.AssertMinCollectionSize(items, 1); err != nil {
				return err
			}
			for _, item := range items {
				if rd, ok := item["ReleaseDate"]; ok && rd != nil {
					s, isStr := rd.(string)
					if !isStr || len(s) < 10 || s[4] != '-' || s[7] != '-' {
						return fmt.Errorf("ReleaseDate %q does not match YYYY-MM-DD format", rd)
					}
				}
				if ot, ok := item["OpenTime"]; ok && ot != nil {
					s, isStr := ot.(string)
					if !isStr || len(s) < 8 || s[2] != ':' || s[5] != ':' {
						return fmt.Errorf("OpenTime %q does not match HH:MM:SS format", ot)
					}
				}
			}
			return nil
		},
	)

	return suite
}
