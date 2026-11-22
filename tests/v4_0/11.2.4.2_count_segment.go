package v4_0

import (
	"fmt"
	"mime"
	"net/url"
	"strconv"
	"strings"

	"github.com/nlstn/odata-compliance-suite/framework"
)

// CountSegment creates the 11.2.4.2 $count segment test suite.
func CountSegment() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"11.2.4.2 $count Segment",
		"Tests the $count path segment for entity and navigation collections.",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part2-url-conventions/odata-v4.0-errata03-os-part2-url-conventions-complete.html#sec_Count",
	)

	// Test 1: $count on entity set returns text/plain integer matching the size
	suite.AddTest(
		"test_entityset_count_segment",
		"$count on entity set returns text/plain integer matching the full collection",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products/$count")
			if err != nil {
				return err
			}
			segmentCount, err := readSegmentCount(ctx, resp)
			if err != nil {
				return err
			}
			items, err := fetchAllProducts(ctx)
			if err != nil {
				return err
			}
			if segmentCount != len(items) {
				return fmt.Errorf("/Products/$count=%d, want %d entities across all pages", segmentCount, len(items))
			}
			return nil
		},
	)

	// Test 2: $count respects $filter and matches @odata.count
	suite.AddTestWithCapabilities(
		"test_count_segment_with_filter",
		"$count segment respects $filter",
		[]framework.RequiredCapability{framework.Require(framework.CapFilter, "Products")},
		func(ctx *framework.TestContext) error {
			filter := "Price gt 100"
			qp := url.Values{}
			qp.Set("$filter", filter)

			segmentResp, err := ctx.GET("/Products/$count?" + qp.Encode())
			if err != nil {
				return err
			}
			segmentCount, err := readSegmentCount(ctx, segmentResp)
			if err != nil {
				return err
			}
			products, err := fetchAllProducts(ctx)
			if err != nil {
				return err
			}
			expected := 0
			for _, product := range products {
				if price, ok := productFloat(product, "Price"); ok && price > 100 {
					expected++
				}
			}
			if segmentCount != expected {
				return fmt.Errorf("$count segment=%d, want %d products with Price gt 100", segmentCount, expected)
			}
			return nil
		},
	)

	// Test 3: $count on navigation collection
	suite.AddTestWithCapabilities(
		"test_navigation_count_segment",
		"$count on a populated navigation collection matches all related entities",
		[]framework.RequiredCapability{
			framework.Require(framework.CapFilter, "Products"),
			framework.Require(framework.CapTop, "Products"),
			framework.Require(framework.CapSelect, "Products"),
		},
		func(ctx *framework.TestContext) error {
			productPath, err := entityPathByFilter(ctx, "Products", "Name eq 'Laptop'")
			if err != nil {
				return err
			}
			resp, err := ctx.GET(productPath + "/Descriptions/$count")
			if err != nil {
				return err
			}
			segmentCount, err := readSegmentCount(ctx, resp)
			if err != nil {
				return err
			}
			items, err := collectEntityCollection(ctx, productPath+"/Descriptions")
			if err != nil {
				return err
			}
			if len(items) == 0 {
				return fmt.Errorf("seeded Laptop must have related Descriptions")
			}
			if segmentCount != len(items) {
				return fmt.Errorf("Descriptions/$count=%d, want %d related entities across all pages", segmentCount, len(items))
			}
			return nil
		},
	)

	return suite
}

func readSegmentCount(ctx *framework.TestContext, resp *framework.HTTPResponse) (int, error) {
	if err := ctx.AssertStatusCode(resp, 200); err != nil {
		return 0, err
	}
	contentType := resp.Headers.Get("Content-Type")
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil || mediaType != "text/plain" {
		return 0, fmt.Errorf("expected text/plain Content-Type, got %q", contentType)
	}
	return parseCountBody(resp.Body)
}

func parseCountBody(body []byte) (int, error) {
	trimmed := strings.TrimSpace(string(body))
	if trimmed == "" {
		return 0, fmt.Errorf("empty $count response body")
	}
	count, err := strconv.Atoi(trimmed)
	if err != nil {
		return 0, fmt.Errorf("invalid $count response %q: %w", trimmed, err)
	}
	if count < 0 {
		return 0, fmt.Errorf("$count must be non-negative, got %d", count)
	}
	return count, nil
}
