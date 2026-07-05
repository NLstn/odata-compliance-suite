package v4_0

import (
	"fmt"
	"strings"

	"github.com/nlstn/odata-compliance-suite/framework"
)

// DurationType creates the 5.1.8 Duration Type test suite
func DurationType() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"5.1.8 Duration Type",
		"Tests handling of Edm.Duration primitive type using ISO 8601 duration format (e.g., P1DT2H30M for 1 day, 2 hours, 30 minutes).",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part3-csdl/odata-v4.0-errata03-os-part3-csdl-complete.html#sec_PrimitiveTypes",
	)

	suite.AddTest(
		"test_duration_in_metadata",
		"Edm.Duration type appears in metadata as a genuine Property declaration",
		func(ctx *framework.TestContext) error {
			refs, err := propertiesDeclaredWithType(ctx, "Edm.Duration")
			if err != nil {
				return err
			}
			if len(refs) == 0 {
				return ctx.Skip("Edm.Duration is an optional primitive type not used by this model")
			}
			for _, ref := range refs {
				if ref.Property == "" {
					return framework.NewError("EntityType " + ref.EntityType + " declares an Edm.Duration property with no Name attribute")
				}
			}
			return nil
		},
	)

	suite.AddTest(
		"test_duration_literal_days",
		"Duration literal P1D (86400 s) matches entities with ShippingTime of exactly 1 day",
		func(ctx *framework.TestContext) error {
			return assertProductFilter(ctx, "ShippingTime eq duration'P1D'", func(p map[string]interface{}) bool {
				secs, ok := productDurationSeconds(p, "ShippingTime")
				return ok && secs == 86400
			})
		},
	)

	suite.AddTest(
		"test_duration_literal_hours",
		"Duration literal PT2H (7200 s) matches entities with ShippingTime of exactly 2 hours",
		func(ctx *framework.TestContext) error {
			return assertProductFilter(ctx, "ShippingTime eq duration'PT2H'", func(p map[string]interface{}) bool {
				secs, ok := productDurationSeconds(p, "ShippingTime")
				return ok && secs == 7200
			})
		},
	)

	suite.AddTest(
		"test_duration_literal_minutes",
		"Duration literal PT30M (1800 s) matches entities with ShippingTime of exactly 30 minutes",
		func(ctx *framework.TestContext) error {
			return assertProductFilter(ctx, "ShippingTime eq duration'PT30M'", func(p map[string]interface{}) bool {
				secs, ok := productDurationSeconds(p, "ShippingTime")
				return ok && secs == 1800
			})
		},
	)

	suite.AddTest(
		"test_duration_literal_seconds",
		"Duration literal PT45S (45 s) matches entities with ProcessingTime of exactly 45 seconds",
		func(ctx *framework.TestContext) error {
			return assertProductFilter(ctx, "ProcessingTime eq duration'PT45S'", func(p map[string]interface{}) bool {
				secs, ok := productDurationSeconds(p, "ProcessingTime")
				return ok && secs == 45
			})
		},
	)

	suite.AddTest(
		"test_duration_literal_combined",
		"Duration literal P1DT2H30M (95400 s) matches entities with ShippingTime of exactly 1d2h30m",
		func(ctx *framework.TestContext) error {
			return assertProductFilter(ctx, "ShippingTime eq duration'P1DT2H30M'", func(p map[string]interface{}) bool {
				secs, ok := productDurationSeconds(p, "ShippingTime")
				return ok && secs == 95400
			})
		},
	)

	suite.AddTest(
		"test_duration_literal_fractional_seconds",
		"Duration literal PT1.5S (1.5 s) matches entities with ProcessingTime of exactly 1.5 seconds",
		func(ctx *framework.TestContext) error {
			return assertProductFilter(ctx, "ProcessingTime eq duration'PT1.5S'", func(p map[string]interface{}) bool {
				secs, ok := productDurationSeconds(p, "ProcessingTime")
				return ok && secs == 1.5
			})
		},
	)

	suite.AddTest(
		"test_duration_negative",
		"Negative duration literal -P1D matches entities with Offset of exactly -1 day",
		func(ctx *framework.TestContext) error {
			return assertProductFilter(ctx, "Offset eq duration'-P1D'", func(p map[string]interface{}) bool {
				secs, ok := productDurationSeconds(p, "Offset")
				return ok && secs == -86400
			})
		},
	)

	suite.AddTest(
		"test_duration_comparison",
		"Duration gt PT1H returns entities with ShippingTime greater than 1 hour",
		func(ctx *framework.TestContext) error {
			return assertProductFilter(ctx, "ShippingTime gt duration'PT1H'", func(p map[string]interface{}) bool {
				secs, ok := productDurationSeconds(p, "ShippingTime")
				return ok && secs > 3600
			})
		},
	)

	suite.AddTest(
		"test_duration_equality",
		"Duration eq P2D matches entities with ShippingTime of exactly 2 days",
		func(ctx *framework.TestContext) error {
			return assertProductFilter(ctx, "ShippingTime eq duration'P2D'", func(p map[string]interface{}) bool {
				secs, ok := productDurationSeconds(p, "ShippingTime")
				return ok && secs == 172800
			})
		},
	)

	suite.AddTest(
		"test_duration_inequality",
		"Duration ne P0D returns entities where ShippingTime is not zero",
		func(ctx *framework.TestContext) error {
			return assertProductFilter(ctx, "ShippingTime ne duration'P0D'", func(p map[string]interface{}) bool {
				secs, ok := productDurationSeconds(p, "ShippingTime")
				return ok && secs != 0
			})
		},
	)

	suite.AddTest(
		"test_duration_zero",
		"Duration eq P0D returns entities with Offset of exactly zero",
		func(ctx *framework.TestContext) error {
			return assertProductFilter(ctx, "Offset eq duration'P0D'", func(p map[string]interface{}) bool {
				secs, ok := productDurationSeconds(p, "Offset")
				return ok && secs == 0
			})
		},
	)

	suite.AddTest(
		"test_duration_null_comparison",
		"Duration ne null returns only entities with a non-null ShippingTime",
		func(ctx *framework.TestContext) error {
			return assertProductFilter(ctx, "ShippingTime ne null", func(p map[string]interface{}) bool {
				_, ok := productDurationSeconds(p, "ShippingTime")
				return ok
			})
		},
	)

	suite.AddTest(
		"test_duration_cast",
		"cast() function supports Edm.Duration",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=cast(ShippingTime,'Edm.Duration') ne null")
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, 200)
		},
	)

	suite.AddTest(
		"test_duration_isof",
		"isof() function supports Edm.Duration",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=isof(ShippingTime,'Edm.Duration')")
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, 200)
		},
	)

	suite.AddTest(
		"test_duration_in_response",
		"Duration values are serialized as ISO 8601 strings (P... format)",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$top=10&$select=ID,Name,ShippingTime,ProcessingTime")
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
				for _, field := range []string{"ShippingTime", "ProcessingTime"} {
					val, ok := item[field]
					if !ok || val == nil {
						continue
					}
					s, isStr := val.(string)
					if !isStr {
						return fmt.Errorf("%s value %v is not a string; Edm.Duration must serialize as ISO 8601", field, val)
					}
					normalized := strings.TrimPrefix(s, "-")
					if len(normalized) == 0 || normalized[0] != 'P' {
						return fmt.Errorf("%s value %q does not start with 'P' as required by ISO 8601 duration format", field, s)
					}
				}
			}
			return nil
		},
	)

	suite.AddTest(
		"test_duration_invalid_format",
		"Invalid duration format returns 400 Bad Request",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=ShippingTime eq duration'invalid'")
			if err != nil {
				return err
			}
			// A syntactically invalid Edm.Duration literal must be rejected with
			// 400 Bad Request, not silently ignored (200) or surfaced as a
			// server error (5xx).
			return ctx.AssertStatusCode(resp, 400)
		},
	)

	return suite
}
