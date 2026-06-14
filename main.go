package main

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/nlstn/odata-compliance-suite/framework"
	v4_0 "github.com/nlstn/odata-compliance-suite/tests/v4_0"
	v4_01 "github.com/nlstn/odata-compliance-suite/tests/v4_01"
	"github.com/nlstn/odata-compliance-suite/tests/vocabularies/capabilities"
	"github.com/nlstn/odata-compliance-suite/tests/vocabularies/core"
)

const validFormats = "text, junit, json, sarif"

// buildVersion is stamped at build time via -ldflags "-X main.buildVersion=...".
// It defaults to "dev" for `go run` / `go build` without ldflags.
var buildVersion = "dev"

var (
	serverURL  = flag.String("server", "http://localhost:9090", "URL of the OData service under test")
	version    = flag.String("version", "all", "OData version to test (4.0, 4.01, vocabularies, or all)")
	pattern    = flag.String("pattern", "", "Run only suites whose name matches this substring")
	debug      = flag.Bool("debug", false, "Enable debug mode with full HTTP request/response details")
	verbose    = flag.Bool("verbose", false, "Enable verbose mode to show all individual test results")
	timeout    = flag.Int("timeout", 30, "Seconds to wait for the server to become reachable before giving up")
	strict     = flag.Bool("strict", false, "Treat capability-skipped tests as failures (disables capability-aware skipping)")
	format     = flag.String("format", "text", "Output format: "+validFormats)
	outputFile = flag.String("output", "", "Write the structured report to this file (default: stdout for non-text formats)")
)

type TestSuiteInfo struct {
	Name             string
	Version          string
	Suite            func() *framework.TestSuite
	ConformanceLevel framework.ConformanceLevel
	Feature          string
}

type preparedSuite struct {
	info          TestSuiteInfo
	suite         *framework.TestSuite
	versionPrefix string
}

func main() {
	flag.Parse()

	// Validate --format early so we fail before touching the network.
	switch *format {
	case "text", "junit", "json", "sarif":
	default:
		fmt.Fprintf(os.Stderr, "Error: unknown format %q — valid values are: %s\n", *format, validFormats)
		os.Exit(1)
	}
	if *format == "text" && *outputFile != "" {
		fmt.Fprintln(os.Stderr, "Error: --output requires a non-text format (use --format junit, json, or sarif)")
		os.Exit(1)
	}

	// progressOut is where human-readable output goes.
	// For non-text formats we redirect it to stderr so that stdout (or --output)
	// carries only the structured report without interleaved human text.
	var progressOut io.Writer = os.Stdout
	if *format != "text" && *outputFile == "" {
		progressOut = os.Stderr
	}

	// useColor controls whether ANSI escape codes appear in progress output.
	useColor := isTerminal(progressOut) && os.Getenv("NO_COLOR") == ""

	fmt.Fprintln(progressOut)
	fmt.Fprintln(progressOut, "╔════════════════════════════════════════════════════════╗")
	fmt.Fprintln(progressOut, "║     OData v4 Compliance Test Suite                     ║")
	fmt.Fprintln(progressOut, "╚════════════════════════════════════════════════════════╝")
	fmt.Fprintln(progressOut)
	fmt.Fprintf(progressOut, "Suite Ver:  %s\n", buildVersion)
	fmt.Fprintf(progressOut, "Server URL: %s\n", *serverURL)
	fmt.Fprintf(progressOut, "Version:    %s\n", *version)
	if *format != "text" {
		fmt.Fprintf(progressOut, "Format:     %s\n", *format)
	}
	if *debug {
		fmt.Fprintln(progressOut, "Debug Mode: ENABLED")
	}
	if *verbose {
		fmt.Fprintln(progressOut, "Verbose Mode: ENABLED")
	}
	fmt.Fprintln(progressOut)

	// The suite runs entirely against an externally-provided OData service.
	// Wait for it to become reachable before running any tests.
	if !waitForServer(*serverURL, *timeout, progressOut, useColor) {
		fmt.Fprintf(os.Stderr, "Error: cannot reach OData service at %s after %ds\n", *serverURL, *timeout)
		fmt.Fprintln(os.Stderr, "Start your service and point -server at its root URL.")
		fmt.Fprintln(os.Stderr, "The service must expose the reference data model documented in CONTRACT.md.")
		os.Exit(1)
	}

	// Fetch and parse the service's $metadata to build a capability profile used for
	// intelligent test skipping. Failure is non-fatal: we warn and run all tests.
	var capProfile *framework.CapabilityProfile
	if !*strict {
		if profile, err := fetchCapabilityProfile(*serverURL); err != nil {
			fmt.Fprintf(progressOut, "⚠ WARNING: Could not parse capability profile from $metadata: %v\n", err)
			fmt.Fprintln(progressOut, "  Capability-aware skipping is disabled; all tests will run.")
			fmt.Fprintln(progressOut)
		} else {
			capProfile = profile
		}
	}

	// Gather test suites
	testSuites := []TestSuiteInfo{}

	// Register v4.0 tests
	if *version == "all" || *version == "4.0" {
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "1.1_introduction",
			Version: "4.0",
			Suite:   v4_0.Introduction,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "2.1_conformance",
			Version: "4.0",
			Suite:   v4_0.Conformance,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "3.1_edmx_element",
			Version: "4.0",
			Suite:   v4_0.EDMXElement,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "3.2_dataservices_element",
			Version: "4.0",
			Suite:   v4_0.DataServicesElement,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "3.3_reference_element",
			Version: "4.0",
			Suite:   v4_0.ReferenceElement,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "3.4_include_element",
			Version: "4.0",
			Suite:   v4_0.IncludeElement,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "3.5_includeannotations_element",
			Version: "4.0",
			Suite:   v4_0.IncludeAnnotationsElement,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "4.1_nominal_types",
			Version: "4.0",
			Suite:   v4_0.NominalTypes,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "4.2_structured_types",
			Version: "4.0",
			Suite:   v4_0.StructuredTypes,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "4.3_navigation_properties",
			Version: "4.0",
			Suite:   v4_0.NavigationProperties,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "4.4_primitive_types",
			Version: "4.0",
			Suite:   v4_0.PrimitiveTypes,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "4.5_builtin_abstract_types",
			Version: "4.0",
			Suite:   v4_0.BuiltInAbstractTypes,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "4.6_annotations",
			Version: "4.0",
			Suite:   v4_0.MetadataAnnotations,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "5.1.1_primitive_data_types",
			Version: "4.0",
			Suite:   v4_0.PrimitiveDataTypes,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "5.1.1.1_numeric_edge_cases",
			Version: "4.0",
			Suite:   v4_0.NumericEdgeCases,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "5.1.1.2_byte_types",
			Version: "4.0",
			Suite:   v4_0.ByteTypes,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "5.1.1.3_int16_type",
			Version: "4.0",
			Suite:   v4_0.Int16Type,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "5.1.1.4_single_type",
			Version: "4.0",
			Suite:   v4_0.SingleType,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "5.1.1.5_numeric_boundary_tests",
			Version: "4.0",
			Suite:   v4_0.NumericBoundaryTests,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "5.1.2_nullable_properties",
			Version: "4.0",
			Suite:   v4_0.NullableProperties,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "5.1.3_collection_properties",
			Version: "4.0",
			Suite:   v4_0.CollectionProperties,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "5.1.4_temporal_data_types",
			Version: "4.0",
			Suite:   v4_0.TemporalDataTypes,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "5.1.5_guid_type",
			Version: "4.0",
			Suite:   v4_0.GuidType,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "5.1.6_binary_type",
			Version: "4.0",
			Suite:   v4_0.BinaryType,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "5.1.7_date_timeofday_types",
			Version: "4.0",
			Suite:   v4_0.DateTimeOfDayTypes,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "5.1.8_duration_type",
			Version: "4.0",
			Suite:   v4_0.DurationType,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "5.2_complex_types",
			Version: "4.0",
			Suite:   v4_0.ComplexTypes,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "5.2.1_complex_filter",
			Version: "4.0",
			Suite:   v4_0.ComplexFilter,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "5.2.2_complex_orderby",
			Version: "4.0",
			Suite:   v4_0.ComplexOrderBy,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "5.2_custom_query_options",
			Version: "4.0",
			Suite:   v4_0.CustomQueryOptions,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "5.3_enum_types",
			Version: "4.0",
			Suite:   v4_0.EnumTypes,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "5.3_enum_metadata_members",
			Version: "4.0",
			Suite:   v4_0.EnumMetadataMembers,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "5.4_type_definitions",
			Version: "4.0",
			Suite:   v4_0.TypeDefinitions,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "6.1_extensibility",
			Version: "4.0",
			Suite:   v4_0.Extensibility,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "7.1.1_unicode_strings",
			Version: "4.0",
			Suite:   v4_0.UnicodeStrings,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "8.1.1_header_content_type",
			Version: "4.0",
			Suite:   v4_0.HeaderContentType,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "8.1.2_request_headers",
			Version: "4.0",
			Suite:   v4_0.RequestHeaders,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "8.1.3_response_headers",
			Version: "4.0",
			Suite:   v4_0.ResponseHeaders,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "8.1.5_response_status_codes",
			Version: "4.0",
			Suite:   v4_0.ResponseStatusCodes,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "8.1.6_invalid_query_parameters",
			Version: "4.0",
			Suite:   v4_0.InvalidQueryParameters,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "8.1.7_method_not_allowed",
			Version: "4.0",
			Suite:   v4_0.MethodNotAllowed,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "8.2.1_cache_control_header",
			Version: "4.0",
			Suite:   v4_0.CacheControlHeader,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "8.2.2_header_if_match",
			Version: "4.0",
			Suite:   v4_0.HeaderIfMatch,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "8.2.3_header_odata_entityid",
			Version: "4.0",
			Suite:   v4_0.HeaderODataEntityId,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "8.2.4_header_content_id",
			Version: "4.0",
			Suite:   v4_0.HeaderContentId,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "8.2.5_header_location",
			Version: "4.0",
			Suite:   v4_0.HeaderLocation,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "8.2.6_header_odata_version",
			Version: "4.0",
			Suite:   v4_0.HeaderODataVersion,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "8.2.7_header_accept",
			Version: "4.0",
			Suite:   v4_0.HeaderAccept,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "8.2.8_header_prefer",
			Version: "4.0",
			Suite:   v4_0.HeaderPrefer,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "8.2.8.1_preference_allow_entityreferences",
			Version: "4.0",
			Suite:   v4_0.PreferenceAllowEntityReferences,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "8.2.8.4_preference_include_annotations",
			Version: "4.0",
			Suite:   v4_0.PreferenceIncludeAnnotations,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "8.2.9_header_maxversion",
			Version: "4.0",
			Suite:   v4_0.HeaderMaxVersion,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "8.3_error_responses",
			Version: "4.0",
			Suite:   v4_0.ErrorResponses,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "8.4_error_response_consistency",
			Version: "4.0",
			Suite:   v4_0.ErrorResponseConsistency,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "9.1_service_document",
			Version: "4.0",
			Suite:   v4_0.ServiceDocument,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "9.2_metadata_document",
			Version: "4.0",
			Suite:   v4_0.MetadataDocument,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "9.3_annotations_metadata",
			Version: "4.0",
			Suite:   v4_0.AnnotationsMetadata,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "10.1_json_format",
			Version: "4.0",
			Suite:   v4_0.JSONFormat,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "10.2_odata_annotations",
			Version: "4.0",
			Suite:   v4_0.ODataAnnotations,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.1_resource_path",
			Version: "4.0",
			Suite:   v4_0.ResourcePath,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.2.1_addressing_entities",
			Version: "4.0",
			Suite:   v4_0.AddressingEntities,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.2.2_canonical_url",
			Version: "4.0",
			Suite:   v4_0.CanonicalURL,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.2.3_property_access",
			Version: "4.0",
			Suite:   v4_0.PropertyAccess,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.2.4_collection_operations",
			Version: "4.0",
			Suite:   v4_0.CollectionOperations,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.2.4.1_query_search",
			Version: "4.0",
			Suite:   v4_0.QuerySearch,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.2.4.2_count_segment",
			Version: "4.0",
			Suite:   v4_0.CountSegment,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.2.5.1_query_filter",
			Version: "4.0",
			Suite:   v4_0.QueryFilter,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.2.5.2_query_select_orderby",
			Version: "4.0",
			Suite:   v4_0.QuerySelectOrderby,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.2.5.3_query_top_skip",
			Version: "4.0",
			Suite:   v4_0.QueryTopSkip,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.2.5.4_query_apply",
			Version: "4.0",
			Suite:   v4_0.QueryApply,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.2.5.4.1_advanced_apply",
			Version: "4.0",
			Suite:   v4_0.AdvancedApply,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.2.5.4.2_apply_transformation_catalog",
			Version: "4.0",
			Suite:   v4_0.ApplyTransformationCatalog,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.2.5.4.3_apply_rollup",
			Version: "4.0",
			Suite:   v4_0.QueryApplyRollup,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.2.5.5_query_count",
			Version: "4.0",
			Suite:   v4_0.QueryCount,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.2.5.6_query_expand",
			Version: "4.0",
			Suite:   v4_0.QueryExpand,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.2.5.7_query_skiptoken",
			Version: "4.0",
			Suite:   v4_0.QuerySkiptoken,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.2.5.8_parameter_aliases",
			Version: "4.0",
			Suite:   v4_0.ParameterAliases,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.2.5.9_nested_expand_options",
			Version: "4.0",
			Suite:   v4_0.NestedExpandOptions,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.2.5.9_nested_expand_advanced",
			Version: "4.0",
			Suite:   v4_0.NestedExpandAdvanced,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.2.5.10_query_option_combinations",
			Version: "4.0",
			Suite:   v4_0.QueryOptionCombinations,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.2.5.11_query_select_with_navigation_filter",
			Version: "4.0",
			Suite:   v4_0.QuerySelectWithNavigationFilter,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.2.5.12_pagination_edge_cases",
			Version: "4.0",
			Suite:   v4_0.PaginationEdgeCases,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.2.6_query_format",
			Version: "4.0",
			Suite:   v4_0.QueryFormat,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.2.7_metadata_levels",
			Version: "4.0",
			Suite:   v4_0.MetadataLevels,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.2.8_delta_links",
			Version: "4.0",
			Suite:   v4_0.DeltaLinks,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.2.9_lambda_operators",
			Version: "4.0",
			Suite:   v4_0.LambdaOperators,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.2.10_addressing_operations",
			Version: "4.0",
			Suite:   v4_0.AddressingOperations,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.2.11_property_value",
			Version: "4.0",
			Suite:   v4_0.PropertyValue,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.2.12_stream_properties",
			Version: "4.0",
			Suite:   v4_0.StreamProperties,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.2.13_type_casting",
			Version: "4.0",
			Suite:   v4_0.TypeCasting,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.2.14_url_encoding",
			Version: "4.0",
			Suite:   v4_0.URLEncoding,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.2.15_entity_references",
			Version: "4.0",
			Suite:   v4_0.EntityReferences,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.2.16_singleton_operations",
			Version: "4.0",
			Suite:   v4_0.SingletonOperations,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.2.17_case_sensitivity",
			Version: "4.0",
			Suite:   v4_0.CaseSensitivity,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.3.1_filter_string_functions",
			Version: "4.0",
			Suite:   v4_0.FilterStringFunctions,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.3.2_filter_date_functions",
			Version: "4.0",
			Suite:   v4_0.FilterDateFunctions,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.3.3_filter_arithmetic_functions",
			Version: "4.0",
			Suite:   v4_0.FilterArithmeticFunctions,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.3.4_filter_type_functions",
			Version: "4.0",
			Suite:   v4_0.FilterTypeFunctions,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.3.5_filter_logical_operators",
			Version: "4.0",
			Suite:   v4_0.FilterLogicalOperators,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.3.6_filter_comparison_operators",
			Version: "4.0",
			Suite:   v4_0.FilterComparisonOperators,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.3.7_filter_geo_functions",
			Version: "4.0",
			Suite:   v4_0.FilterGeoFunctions,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.3.8_filter_expanded_properties",
			Version: "4.0",
			Suite:   v4_0.FilterExpandedProperties,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.3.9_string_function_edge_cases",
			Version: "4.0",
			Suite:   v4_0.StringFunctionEdgeCases,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.3.10_filter_single_entity_navigation",
			Version: "4.0",
			Suite:   v4_0.FilterOnSingleEntityNavigationProperties,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.3.11_orderby_navigation_property",
			Version: "4.0",
			Suite:   v4_0.OrderByNavigationProperty,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.4.1_requesting_entities",
			Version: "4.0",
			Suite:   v4_0.RequestingEntities,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.4.2_create_entity",
			Version: "4.0",
			Suite:   v4_0.CreateEntity,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.4.2.1_odata_bind",
			Version: "4.0",
			Suite:   v4_0.ODataBind,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.4.3_update_entity",
			Version: "4.0",
			Suite:   v4_0.UpdateEntity,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.4.4_delete_entity",
			Version: "4.0",
			Suite:   v4_0.DeleteEntity,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.4.5_upsert",
			Version: "4.0",
			Suite:   v4_0.Upsert,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.4.6_relationships",
			Version: "4.0",
			Suite:   v4_0.Relationships,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.4.6.1_navigation_property_operations",
			Version: "4.0",
			Suite:   v4_0.NavigationPropertyOperations,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.4.7_deep_insert",
			Version: "4.0",
			Suite:   v4_0.DeepInsert,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.4.8_modify_relationships",
			Version: "4.0",
			Suite:   v4_0.ModifyRelationships,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.4.9_batch_requests",
			Version: "4.0",
			Suite:   v4_0.BatchRequests,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.4.9.1_batch_error_handling",
			Version: "4.0",
			Suite:   v4_0.BatchErrorHandling,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.4.9.3_batch_content_id_referencing",
			Version: "4.0",
			Suite:   v4_0.BatchContentIDReferencing,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.4.10_asynchronous_requests",
			Version: "4.0",
			Suite:   v4_0.AsynchronousRequests,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.4.11_head_requests",
			Version: "4.0",
			Suite:   v4_0.HEADRequests,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.4.12_returning_results",
			Version: "4.0",
			Suite:   v4_0.ReturningResults,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.4.13_action_function_parameters",
			Version: "4.0",
			Suite:   v4_0.ActionFunctionParameters,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.4.14_null_value_handling",
			Version: "4.0",
			Suite:   v4_0.NullValueHandling,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.4.15_data_validation",
			Version: "4.0",
			Suite:   v4_0.DataValidation,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.5.1_conditional_requests",
			Version: "4.0",
			Suite:   v4_0.ConditionalRequests,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.6_annotations",
			Version: "4.0",
			Suite:   v4_0.InstanceAnnotations,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "12.1_operations",
			Version: "4.0",
			Suite:   v4_0.Operations,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "13.1_asynchronous_processing",
			Version: "4.0",
			Suite:   v4_0.AsynchronousProcessing,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "14.1_vocabulary_annotations",
			Version: "4.0",
			Suite:   v4_0.VocabularyAnnotations,
		})
	}

	// Register vocabulary tests (separate from protocol versions)
	if *version == "all" || *version == "vocabularies" || *version == "vocab" {
		// Core vocabulary tests
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "vocab_core_computed",
			Version: "vocabularies",
			Suite:   core.ComputedAnnotation,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "vocab_core_immutable",
			Version: "vocabularies",
			Suite:   core.ImmutableAnnotation,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "vocab_core_description",
			Version: "vocabularies",
			Suite:   core.DescriptionAnnotation,
		})

		// Capabilities vocabulary tests
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "vocab_capabilities_insert",
			Version: "vocabularies",
			Suite:   capabilities.InsertRestrictions,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "vocab_capabilities_update",
			Version: "vocabularies",
			Suite:   capabilities.UpdateRestrictions,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "vocab_capabilities_delete",
			Version: "vocabularies",
			Suite:   capabilities.DeleteRestrictions,
		})
	}

	// Register v4.01 tests
	if *version == "all" || *version == "4.01" {
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "8.2.6_header_odata_version",
			Version: "4.01",
			Suite:   v4_01.HeaderODataVersion,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "8.2.9_header_maxversion",
			Version: "4.01",
			Suite:   v4_01.HeaderMaxVersion,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.2.1_key_as_segments",
			Version: "4.01",
			Suite:   v4_01.KeyAsSegments,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.2.5.9_nested_expand_options",
			Version: "4.01",
			Suite:   v4_01.NestedExpandOptions,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "8.2.8.6_preference_omit_values",
			Version: "4.01",
			Suite:   v4_01.PreferenceOmitValues,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "8.3.1_header_async_result",
			Version: "4.01",
			Suite:   v4_01.HeaderAsyncResult,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.2.5.1_filter_in_operator",
			Version: "4.01",
			Suite:   v4_01.InOperator,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.5.1.1_filter_divby_operator",
			Version: "4.01",
			Suite:   v4_01.FilterDivByOperator,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.2.5.8_query_compute",
			Version: "4.01",
			Suite:   v4_01.QueryCompute,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.2.5.11_orderby_computed_properties",
			Version: "4.01",
			Suite:   v4_01.OrderByComputedProperties,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.2.5.13_query_index",
			Version: "4.01",
			Suite:   v4_01.QueryIndex,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.2.12_query_schemaversion",
			Version: "4.01",
			Suite:   v4_01.QuerySchemaVersion,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.2.17_case_insensitive_system_query_options",
			Version: "4.01",
			Suite:   v4_01.CaseInsensitiveSystemQueryOptions,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "12.2_function_action_overloading",
			Version: "4.01",
			Suite:   v4_01.FunctionActionOverloading,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "19_json_batch",
			Version: "4.01",
			Suite:   v4_01.JSONBatch,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.2.5.14_wildcard_select_expand",
			Version: "4.01",
			Suite:   v4_01.WildcardSelectExpand,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.5.3.3_filter_matches_pattern",
			Version: "4.01",
			Suite:   v4_01.MatchesPatternFilter,
		})
	}

	if len(testSuites) == 0 {
		fmt.Fprintf(progressOut, "No test suites found for version: %s\n", *version)
		os.Exit(1)
	}

	// capabilityRequirements maps a suite name to the capabilities it depends on.
	// If the service declares any of them unsupported (via Capabilities.V1 annotations in
	// $metadata), the suite is skipped rather than run-and-failed.
	// Entity-set requirements use "Products" — the primary fixture for all protocol suites.
	capabilityRequirements := map[string][]framework.RequiredCapability{
		// --- filter ---
		"11.2.5.1_query_filter":                         {framework.Require(framework.CapFilter, "Products")},
		"5.2.1_complex_filter":                          {framework.Require(framework.CapFilter, "Products")},
		"11.3.1_filter_string_functions":                {framework.Require(framework.CapFilter, "Products")},
		"11.3.2_filter_date_functions":                  {framework.Require(framework.CapFilter, "Products")},
		"11.3.3_filter_arithmetic_functions":            {framework.Require(framework.CapFilter, "Products")},
		"11.3.4_filter_type_functions":                  {framework.Require(framework.CapFilter, "Products")},
		"11.3.5_filter_logical_operators":               {framework.Require(framework.CapFilter, "Products")},
		"11.3.6_filter_comparison_operators":            {framework.Require(framework.CapFilter, "Products")},
		"11.3.7_filter_geo_functions":                   {framework.Require(framework.CapFilter, "Products")},
		"11.3.8_filter_expanded_properties":             {framework.Require(framework.CapFilter, "Products"), framework.Require(framework.CapExpand, "Products")},
		"11.3.9_string_function_edge_cases":             {framework.Require(framework.CapFilter, "Products")},
		"11.3.10_filter_single_entity_navigation":       {framework.Require(framework.CapFilter, "Products")},
		"11.2.9_lambda_operators":                       {framework.Require(framework.CapFilter, "Products")},
		"11.2.5.11_query_select_with_navigation_filter": {framework.Require(framework.CapFilter, "Products")},
		// v4.01 filter
		"11.2.5.1_filter_in_operator":     {framework.Require(framework.CapFilter, "Products")},
		"11.5.1.1_filter_divby_operator":  {framework.Require(framework.CapFilter, "Products")},
		"11.5.3.3_filter_matches_pattern": {framework.Require(framework.CapFilter, "Products")},
		// --- sort ---
		"11.2.5.2_query_select_orderby":         {framework.Require(framework.CapSort, "Products")},
		"5.2.2_complex_orderby":                 {framework.Require(framework.CapSort, "Products")},
		"11.3.11_orderby_navigation_property":   {framework.Require(framework.CapSort, "Products")},
		"11.2.5.11_orderby_computed_properties": {framework.Require(framework.CapSort, "Products")},
		// --- expand ---
		"11.2.5.6_query_expand":           {framework.Require(framework.CapExpand, "Products")},
		"11.2.5.9_nested_expand_options":  {framework.Require(framework.CapExpand, "Products")},
		"11.2.5.9_nested_expand_advanced": {framework.Require(framework.CapExpand, "Products")},
		// --- count ---
		"11.2.5.5_query_count":   {framework.Require(framework.CapCount, "Products")},
		"11.2.4.2_count_segment": {framework.Require(framework.CapCount, "Products")},
		// --- search ---
		"11.2.4.1_query_search": {framework.Require(framework.CapSearch, "Products")},
		// --- top / skip ---
		"11.2.5.3_query_top_skip":         {framework.Require(framework.CapTop, "Products"), framework.Require(framework.CapSkip, "Products")},
		"11.2.5.7_query_skiptoken":        {framework.Require(framework.CapSkip, "Products")},
		"11.2.5.12_pagination_edge_cases": {framework.Require(framework.CapTop, "Products"), framework.Require(framework.CapSkip, "Products")},
		// --- insert ---
		"11.4.2_create_entity":                    {framework.Require(framework.CapInsert, "Products")},
		"11.4.2.1_odata_bind":                     {framework.Require(framework.CapInsert, "Products")},
		"11.4.7_deep_insert":                      {framework.Require(framework.CapInsert, "Products")},
		"11.4.6.1_navigation_property_operations": {framework.Require(framework.CapInsert, "Products")},
		// --- update ---
		"11.4.3_update_entity":        {framework.Require(framework.CapUpdate, "Products")},
		"11.4.8_modify_relationships": {framework.Require(framework.CapUpdate, "Products")},
		// --- upsert (insert + update) ---
		"11.4.5_upsert": {framework.Require(framework.CapInsert, "Products"), framework.Require(framework.CapUpdate, "Products")},
		// --- delete ---
		"11.4.4_delete_entity": {framework.Require(framework.CapDelete, "Products")},
		// --- batch ---
		"11.4.9_batch_requests":                 {framework.Require(framework.CapBatch, "")},
		"11.4.9.1_batch_error_handling":         {framework.Require(framework.CapBatch, "")},
		"11.4.9.3_batch_content_id_referencing": {framework.Require(framework.CapBatch, "")},
		"19_json_batch":                         {framework.Require(framework.CapBatch, "")},
		// --- compute (v4.01; gated via SelectSupport.Computeable on entity set) ---
		"11.2.5.8_query_compute": {framework.Require(framework.CapCompute, "Products")},
		// --- returning results (Prefer: return=representation after mutations) ---
		"11.4.12_returning_results": {framework.Require(framework.CapInsert, "Products")},
		// --- null value handling (create + patch with nullable fields) ---
		"11.4.14_null_value_handling": {framework.Require(framework.CapInsert, "Products")},
		// --- data validation (POST with invalid payloads) ---
		"11.4.15_data_validation": {framework.Require(framework.CapInsert, "Products")},
		// --- conditional requests (ETags; all mutations are PATCH) ---
		"11.5.1_conditional_requests": {framework.Require(framework.CapUpdate, "Products")},
		// --- asynchronous processing (POST /Products as setup) ---
		"13.1_asynchronous_processing": {framework.Require(framework.CapInsert, "Products")},
	}

	// conformanceTags maps a suite name to its OData conformance level and feature area.
	type conformanceTag struct {
		Level   framework.ConformanceLevel
		Feature string
	}
	conformanceTags := map[string]conformanceTag{
		// --- Service Discovery ---
		"1.1_introduction":     {framework.LevelMinimal, "Service Discovery"},
		"2.1_conformance":      {framework.LevelMinimal, "Service Discovery"},
		"9.1_service_document": {framework.LevelMinimal, "Service Discovery"},
		// --- Metadata ---
		"3.1_edmx_element":               {framework.LevelMinimal, "Metadata"},
		"3.2_dataservices_element":       {framework.LevelMinimal, "Metadata"},
		"3.3_reference_element":          {framework.LevelMinimal, "Metadata"},
		"3.4_include_element":            {framework.LevelMinimal, "Metadata"},
		"3.5_includeannotations_element": {framework.LevelMinimal, "Metadata"},
		"9.2_metadata_document":          {framework.LevelMinimal, "Metadata"},
		"9.3_annotations_metadata":       {framework.LevelMinimal, "Metadata"},
		"11.2.12_query_schemaversion":    {framework.LevelAdvanced, "Metadata"},
		// --- Data Types ---
		"4.1_nominal_types":              {framework.LevelMinimal, "Data Types"},
		"4.2_structured_types":           {framework.LevelMinimal, "Data Types"},
		"4.3_navigation_properties":      {framework.LevelMinimal, "Data Types"},
		"4.4_primitive_types":            {framework.LevelMinimal, "Data Types"},
		"4.5_builtin_abstract_types":     {framework.LevelMinimal, "Data Types"},
		"4.6_annotations":                {framework.LevelMinimal, "Data Types"},
		"5.1.1_primitive_data_types":     {framework.LevelMinimal, "Data Types"},
		"5.1.1.1_numeric_edge_cases":     {framework.LevelMinimal, "Data Types"},
		"5.1.1.2_byte_types":             {framework.LevelMinimal, "Data Types"},
		"5.1.1.3_int16_type":             {framework.LevelMinimal, "Data Types"},
		"5.1.1.4_single_type":            {framework.LevelMinimal, "Data Types"},
		"5.1.1.5_numeric_boundary_tests": {framework.LevelMinimal, "Data Types"},
		"5.1.2_nullable_properties":      {framework.LevelMinimal, "Data Types"},
		"5.1.3_collection_properties":    {framework.LevelMinimal, "Data Types"},
		"5.1.4_temporal_data_types":      {framework.LevelMinimal, "Data Types"},
		"5.1.5_guid_type":                {framework.LevelMinimal, "Data Types"},
		"5.1.6_binary_type":              {framework.LevelMinimal, "Data Types"},
		"5.1.7_date_timeofday_types":     {framework.LevelMinimal, "Data Types"},
		"5.1.8_duration_type":            {framework.LevelMinimal, "Data Types"},
		"5.2_complex_types":              {framework.LevelMinimal, "Data Types"},
		"5.3_enum_types":                 {framework.LevelMinimal, "Data Types"},
		"5.3_enum_metadata_members":      {framework.LevelMinimal, "Data Types"},
		"5.4_type_definitions":           {framework.LevelMinimal, "Data Types"},
		// --- HTTP Protocol ---
		"6.1_extensibility":                             {framework.LevelMinimal, "HTTP Protocol"},
		"7.1.1_unicode_strings":                         {framework.LevelMinimal, "HTTP Protocol"},
		"8.1.1_header_content_type":                     {framework.LevelMinimal, "HTTP Protocol"},
		"8.1.2_request_headers":                         {framework.LevelMinimal, "HTTP Protocol"},
		"8.1.3_response_headers":                        {framework.LevelMinimal, "HTTP Protocol"},
		"8.1.5_response_status_codes":                   {framework.LevelMinimal, "HTTP Protocol"},
		"8.1.6_invalid_query_parameters":                {framework.LevelMinimal, "HTTP Protocol"},
		"8.1.7_method_not_allowed":                      {framework.LevelMinimal, "HTTP Protocol"},
		"8.2.1_cache_control_header":                    {framework.LevelMinimal, "HTTP Protocol"},
		"8.2.2_header_if_match":                         {framework.LevelMinimal, "HTTP Protocol"},
		"8.2.3_header_odata_entityid":                   {framework.LevelMinimal, "HTTP Protocol"},
		"8.2.4_header_content_id":                       {framework.LevelMinimal, "HTTP Protocol"},
		"8.2.5_header_location":                         {framework.LevelMinimal, "HTTP Protocol"},
		"8.2.6_header_odata_version":                    {framework.LevelMinimal, "HTTP Protocol"},
		"8.2.7_header_accept":                           {framework.LevelMinimal, "HTTP Protocol"},
		"8.2.8_header_prefer":                           {framework.LevelMinimal, "HTTP Protocol"},
		"8.2.8.1_preference_allow_entityreferences":     {framework.LevelIntermediate, "HTTP Protocol"},
		"8.2.8.4_preference_include_annotations":        {framework.LevelIntermediate, "HTTP Protocol"},
		"8.2.8.6_preference_omit_values":                {framework.LevelAdvanced, "HTTP Protocol"},
		"8.2.9_header_maxversion":                       {framework.LevelMinimal, "HTTP Protocol"},
		"8.3_error_responses":                           {framework.LevelMinimal, "HTTP Protocol"},
		"8.3.1_header_async_result":                     {framework.LevelAdvanced, "HTTP Protocol"},
		"8.4_error_response_consistency":                {framework.LevelMinimal, "HTTP Protocol"},
		"11.2.17_case_sensitivity":                      {framework.LevelMinimal, "HTTP Protocol"},
		"11.2.17_case_insensitive_system_query_options": {framework.LevelMinimal, "HTTP Protocol"},
		// --- JSON Format ---
		"10.1_json_format":       {framework.LevelMinimal, "JSON Format"},
		"10.2_odata_annotations": {framework.LevelMinimal, "JSON Format"},
		"11.2.6_query_format":    {framework.LevelMinimal, "JSON Format"},
		"11.2.7_metadata_levels": {framework.LevelIntermediate, "JSON Format"},
		// --- Entity Read ---
		"11.1_resource_path":            {framework.LevelMinimal, "Entity Read"},
		"11.2.1_addressing_entities":    {framework.LevelMinimal, "Entity Read"},
		"11.2.1_key_as_segments":        {framework.LevelMinimal, "Entity Read"},
		"11.2.2_canonical_url":          {framework.LevelMinimal, "Entity Read"},
		"11.2.3_property_access":        {framework.LevelMinimal, "Entity Read"},
		"11.2.4_collection_operations":  {framework.LevelMinimal, "Entity Read"},
		"11.2.10_addressing_operations": {framework.LevelIntermediate, "Entity Read"},
		"11.2.11_property_value":        {framework.LevelMinimal, "Entity Read"},
		"11.2.12_stream_properties":     {framework.LevelMinimal, "Entity Read"},
		"11.2.13_type_casting":          {framework.LevelMinimal, "Entity Read"},
		"11.2.14_url_encoding":          {framework.LevelMinimal, "Entity Read"},
		"11.2.15_entity_references":     {framework.LevelIntermediate, "Entity Read"},
		"11.2.16_singleton_operations":  {framework.LevelMinimal, "Entity Read"},
		"11.4.1_requesting_entities":    {framework.LevelMinimal, "Entity Read"},
		"11.4.11_head_requests":         {framework.LevelMinimal, "HTTP Protocol"},
		// --- Filtering ---
		"11.2.5.1_query_filter":                         {framework.LevelIntermediate, "Filtering"},
		"5.2.1_complex_filter":                          {framework.LevelIntermediate, "Filtering"},
		"11.3.1_filter_string_functions":                {framework.LevelIntermediate, "Filtering"},
		"11.3.2_filter_date_functions":                  {framework.LevelIntermediate, "Filtering"},
		"11.3.3_filter_arithmetic_functions":            {framework.LevelIntermediate, "Filtering"},
		"11.3.4_filter_type_functions":                  {framework.LevelIntermediate, "Filtering"},
		"11.3.5_filter_logical_operators":               {framework.LevelIntermediate, "Filtering"},
		"11.3.6_filter_comparison_operators":            {framework.LevelIntermediate, "Filtering"},
		"11.3.7_filter_geo_functions":                   {framework.LevelAdvanced, "Filtering"},
		"11.3.8_filter_expanded_properties":             {framework.LevelAdvanced, "Filtering"},
		"11.3.9_string_function_edge_cases":             {framework.LevelIntermediate, "Filtering"},
		"11.3.10_filter_single_entity_navigation":       {framework.LevelIntermediate, "Filtering"},
		"11.2.9_lambda_operators":                       {framework.LevelAdvanced, "Filtering"},
		"11.2.5.11_query_select_with_navigation_filter": {framework.LevelAdvanced, "Filtering"},
		"11.2.5.1_filter_in_operator":                   {framework.LevelIntermediate, "Filtering"},
		"11.5.1.1_filter_divby_operator":                {framework.LevelIntermediate, "Filtering"},
		"11.5.3.3_filter_matches_pattern":               {framework.LevelIntermediate, "Filtering"},
		// --- Sorting ---
		"11.2.5.2_query_select_orderby":         {framework.LevelIntermediate, "Sorting"},
		"5.2.2_complex_orderby":                 {framework.LevelIntermediate, "Sorting"},
		"11.3.11_orderby_navigation_property":   {framework.LevelAdvanced, "Sorting"},
		"11.2.5.11_orderby_computed_properties": {framework.LevelAdvanced, "Sorting"},
		// --- Paging ---
		"11.2.5.3_query_top_skip":         {framework.LevelIntermediate, "Paging"},
		"11.2.5.7_query_skiptoken":        {framework.LevelIntermediate, "Paging"},
		"11.2.5.12_pagination_edge_cases": {framework.LevelIntermediate, "Paging"},
		// --- Counting ---
		"11.2.5.5_query_count":   {framework.LevelIntermediate, "Counting"},
		"11.2.4.2_count_segment": {framework.LevelIntermediate, "Counting"},
		// --- Searching ---
		"11.2.4.1_query_search": {framework.LevelAdvanced, "Searching"},
		// --- Expanding ---
		"11.2.5.6_query_expand":            {framework.LevelAdvanced, "Expanding"},
		"11.2.5.9_nested_expand_options":   {framework.LevelAdvanced, "Expanding"},
		"11.2.5.9_nested_expand_advanced":  {framework.LevelAdvanced, "Expanding"},
		"11.2.5.14_wildcard_select_expand": {framework.LevelAdvanced, "Expanding"},
		// --- Data Modification ---
		"11.4.2_create_entity":                    {framework.LevelIntermediate, "Data Modification"},
		"11.4.2.1_odata_bind":                     {framework.LevelIntermediate, "Data Modification"},
		"11.4.3_update_entity":                    {framework.LevelIntermediate, "Data Modification"},
		"11.4.4_delete_entity":                    {framework.LevelIntermediate, "Data Modification"},
		"11.4.5_upsert":                           {framework.LevelAdvanced, "Data Modification"},
		"11.4.6_relationships":                    {framework.LevelIntermediate, "Data Modification"},
		"11.4.6.1_navigation_property_operations": {framework.LevelIntermediate, "Data Modification"},
		"11.4.7_deep_insert":                      {framework.LevelAdvanced, "Data Modification"},
		"11.4.8_modify_relationships":             {framework.LevelIntermediate, "Data Modification"},
		"11.4.12_returning_results":               {framework.LevelIntermediate, "Data Modification"},
		"11.4.14_null_value_handling":             {framework.LevelIntermediate, "Data Modification"},
		"11.4.15_data_validation":                 {framework.LevelIntermediate, "Data Modification"},
		// --- Batch ---
		"11.4.9_batch_requests":                 {framework.LevelAdvanced, "Batch"},
		"11.4.9.1_batch_error_handling":         {framework.LevelAdvanced, "Batch"},
		"11.4.9.3_batch_content_id_referencing": {framework.LevelAdvanced, "Batch"},
		"19_json_batch":                         {framework.LevelAdvanced, "Batch"},
		// --- Advanced Querying ---
		"5.2_custom_query_options":                {framework.LevelMinimal, "HTTP Protocol"},
		"11.2.5.4_query_apply":                    {framework.LevelAdvanced, "Advanced Querying"},
		"11.2.5.4.1_advanced_apply":               {framework.LevelAdvanced, "Advanced Querying"},
		"11.2.5.4.2_apply_transformation_catalog": {framework.LevelAdvanced, "Advanced Querying"},
		"11.2.5.4.3_apply_rollup":                 {framework.LevelAdvanced, "Advanced Querying"},
		"11.2.5.8_parameter_aliases":              {framework.LevelIntermediate, "Advanced Querying"},
		"11.2.5.8_query_compute":                  {framework.LevelAdvanced, "Advanced Querying"},
		"11.2.5.10_query_option_combinations":     {framework.LevelIntermediate, "Advanced Querying"},
		"11.2.5.13_query_index":                   {framework.LevelAdvanced, "Advanced Querying"},
		"11.2.8_delta_links":                      {framework.LevelAdvanced, "Advanced Querying"},
		// --- Operations (Functions & Actions) ---
		"12.1_operations":                    {framework.LevelAdvanced, "Operations"},
		"12.2_function_action_overloading":   {framework.LevelAdvanced, "Operations"},
		"11.4.13_action_function_parameters": {framework.LevelAdvanced, "Operations"},
		// --- Concurrency ---
		"11.5.1_conditional_requests": {framework.LevelAdvanced, "Concurrency"},
		// --- Async ---
		"11.4.10_asynchronous_requests": {framework.LevelAdvanced, "Async"},
		"13.1_asynchronous_processing":  {framework.LevelAdvanced, "Async"},
		// --- Annotations ---
		"11.6_annotations":            {framework.LevelAdvanced, "Annotations"},
		"14.1_vocabulary_annotations": {framework.LevelAdvanced, "Annotations"},
		// --- Vocabularies ---
		"vocab_core_computed":       {framework.LevelAdvanced, "Vocabularies"},
		"vocab_core_immutable":      {framework.LevelAdvanced, "Vocabularies"},
		"vocab_core_description":    {framework.LevelMinimal, "Vocabularies"},
		"vocab_capabilities_insert": {framework.LevelIntermediate, "Vocabularies"},
		"vocab_capabilities_update": {framework.LevelIntermediate, "Vocabularies"},
		"vocab_capabilities_delete": {framework.LevelIntermediate, "Vocabularies"},
	}

	// Prepare suites (apply pattern filter) so we can compute totals for concise progress output
	var suitesToRun []preparedSuite
	totalPlannedTests := 0

	for _, suiteInfo := range testSuites {
		if *pattern != "" && !strings.Contains(suiteInfo.Name, *pattern) {
			continue
		}

		suite := suiteInfo.Suite()
		suite.ServerURL = *serverURL
		suite.Debug = *debug
		suite.Verbose = *verbose
		suite.Quiet = !*verbose
		suite.Out = progressOut
		suite.Capabilities = capProfile
		suite.Strict = *strict
		if reqs, ok := capabilityRequirements[suiteInfo.Name]; ok {
			suite.RequiredCapabilities = reqs
		}
		if tag, ok := conformanceTags[suiteInfo.Name]; ok {
			suiteInfo.ConformanceLevel = tag.Level
			suiteInfo.Feature = tag.Feature
		}

		versionPrefix := "V4"
		if suiteInfo.Version == "4.01" {
			versionPrefix = "V4.01"
		}

		totalPlannedTests += len(suite.Tests)

		suitesToRun = append(suitesToRun, preparedSuite{
			info:          suiteInfo,
			suite:         suite,
			versionPrefix: versionPrefix,
		})
	}

	if len(suitesToRun) == 0 {
		fmt.Fprintln(progressOut, "No test suites matched the provided pattern.")
		os.Exit(1)
	}

	// Run tests
	fmt.Fprintln(progressOut, "═════════════════════════════════════════════════════════")
	fmt.Fprintln(progressOut)

	totalSuites := len(suitesToRun)
	passedSuites := 0
	totalTests := 0
	passedTests := 0
	failedTests := 0
	skippedTests := 0

	// Collect all failed tests for final summary
	type FailedTestInfo struct {
		SuiteName string
		TestName  string
		Error     string
	}
	var allFailedTests []FailedTestInfo

	if !*verbose {
		fmt.Fprintf(progressOut, "Running %d suites (%d total tests)\n", totalSuites, totalPlannedTests)
		fmt.Fprintln(progressOut)
	}

	for idx, prepared := range suitesToRun {
		suite := prepared.suite

		if *verbose {
			fmt.Fprintln(progressOut, ansi("0;34", fmt.Sprintf("Running: [%s] %s", prepared.versionPrefix, prepared.info.Name), useColor))
			fmt.Fprintln(progressOut, "─────────────────────────────────────────────────────────")
		}

		err := suite.Run()

		totalTests += suite.Results.Total
		passedTests += suite.Results.Passed
		failedTests += suite.Results.Failed
		skippedTests += suite.Results.Skipped

		// Collect failed tests from this suite
		for _, detail := range suite.Results.Details {
			if detail.Status == framework.StatusFail {
				allFailedTests = append(allFailedTests, FailedTestInfo{
					SuiteName: prepared.info.Name,
					TestName:  detail.Name,
					Error:     detail.Error,
				})
			}
		}

		if err == nil {
			passedSuites++
		}

		if *verbose {
			fmt.Fprintln(progressOut)
		} else {
			progressLine := fmt.Sprintf(
				"Progress: suites %d/%d | tests %d/%d | passed %d | failed %d | skipped %d",
				idx+1, totalSuites, totalTests, totalPlannedTests, passedTests, failedTests, skippedTests,
			)
			fmt.Fprintf(progressOut, "\r%-80s", progressLine)
		}
	}

	if !*verbose {
		fmt.Fprintln(progressOut)
		fmt.Fprintln(progressOut)
	}

	// Print overall summary
	fmt.Fprintln(progressOut, "═════════════════════════════════════════════════════════")
	fmt.Fprintln(progressOut)
	fmt.Fprintln(progressOut, "╔════════════════════════════════════════════════════════╗")
	fmt.Fprintln(progressOut, "║                  OVERALL SUMMARY                       ║")
	fmt.Fprintln(progressOut, "╚════════════════════════════════════════════════════════╝")
	fmt.Fprintln(progressOut)
	fmt.Fprintf(progressOut, "Test Scripts: %d/%d passed (%.0f%%)\n", passedSuites, totalSuites,
		float64(passedSuites)/float64(totalSuites)*100)
	fmt.Fprintln(progressOut, "Individual Tests:")
	fmt.Fprintf(progressOut, "  - Total: %d\n", totalTests)
	fmt.Fprintf(progressOut, "  - Passing: %d\n", passedTests)
	fmt.Fprintf(progressOut, "  - Failing: %d\n", failedTests)
	fmt.Fprintf(progressOut, "  - Skipped: %d\n", skippedTests)
	if totalTests > 0 {
		fmt.Fprintf(progressOut, "  - Pass Rate: %.0f%%\n", float64(passedTests)/float64(totalTests)*100)
	}
	fmt.Fprintln(progressOut)

	// Print list of failed tests if any
	if len(allFailedTests) > 0 {
		fmt.Fprintln(progressOut, "Failed Tests:")
		for _, failed := range allFailedTests {
			fmt.Fprintf(progressOut, "  ✗ [%s] %s\n", failed.SuiteName, failed.TestName)
			if failed.Error != "" {
				fmt.Fprintf(progressOut, "    Error: %s\n", failed.Error)
			}
		}
		fmt.Fprintln(progressOut)
	}

	// Conformance level reporting
	//
	// Group suites by (version, feature) and (version, level) to compute the
	// per-feature matrix and the highest conformance level met per OData version.
	type featureKey struct {
		version string
		feature string
	}
	type featureStats struct {
		level   framework.ConformanceLevel
		passed  int
		failed  int
		skipped int
	}
	featureMap := map[featureKey]*featureStats{}

	type levelStats struct {
		failed int
		total  int
	}
	// levelMap[version][level] → stats
	levelMap := map[string]map[framework.ConformanceLevel]*levelStats{}

	for _, ps := range suitesToRun {
		if ps.info.ConformanceLevel == framework.LevelUnspecified {
			continue
		}
		ver := ps.info.Version
		feat := ps.info.Feature
		lvl := ps.info.ConformanceLevel
		res := ps.suite.Results

		fk := featureKey{ver, feat}
		if featureMap[fk] == nil {
			featureMap[fk] = &featureStats{level: lvl}
		}
		fs := featureMap[fk]
		fs.passed += res.Passed
		fs.failed += res.Failed
		fs.skipped += res.Skipped
		// level reflects the highest (most restrictive) suite in the feature group
		if lvl > fs.level {
			fs.level = lvl
		}

		if levelMap[ver] == nil {
			levelMap[ver] = map[framework.ConformanceLevel]*levelStats{}
		}
		if levelMap[ver][lvl] == nil {
			levelMap[ver][lvl] = &levelStats{}
		}
		ls := levelMap[ver][lvl]
		ls.total++
		if res.Failed > 0 {
			ls.failed++
		}
	}

	// Determine highest conformance level met per OData version (cumulative).
	conformanceByVersion := map[string]framework.ConformanceLevel{}
	for ver, lvls := range levelMap {
		highest := framework.LevelUnspecified
		for _, lvl := range []framework.ConformanceLevel{
			framework.LevelMinimal,
			framework.LevelIntermediate,
			framework.LevelAdvanced,
		} {
			ls, ok := lvls[lvl]
			if !ok || ls.total == 0 {
				// No suites at this level; skip (don't block higher levels).
				continue
			}
			if ls.failed > 0 {
				break
			}
			highest = lvl
		}
		conformanceByVersion[ver] = highest
	}

	fmt.Fprintln(progressOut, "╔════════════════════════════════════════════════════════╗")
	fmt.Fprintln(progressOut, "║              CONFORMANCE LEVEL REPORT                  ║")
	fmt.Fprintln(progressOut, "╚════════════════════════════════════════════════════════╝")
	fmt.Fprintln(progressOut)

	for _, ver := range []string{"4.0", "4.01", "vocabularies"} {
		if _, ok := levelMap[ver]; !ok {
			continue
		}
		highest := conformanceByVersion[ver]
		versionLabel := "OData " + ver
		if ver == "vocabularies" {
			versionLabel = "Vocabularies"
		}
		for _, lvl := range []framework.ConformanceLevel{
			framework.LevelMinimal,
			framework.LevelIntermediate,
			framework.LevelAdvanced,
		} {
			ls := levelMap[ver][lvl]
			if ls == nil || ls.total == 0 {
				continue
			}
			var icon string
			if ls.failed > 0 {
				icon = ansi("0;31", "✗", useColor)
			} else {
				icon = ansi("0;32", "✓", useColor)
			}
			fmt.Fprintf(progressOut, "  %s [%s] %s: %s (%d/%d suites)\n",
				icon, versionLabel, lvl.String(),
				func() string {
					if ls.failed > 0 {
						return "Not Met"
					}
					return "Met"
				}(),
				ls.total-ls.failed, ls.total)
		}
		if highest != framework.LevelUnspecified {
			fmt.Fprintf(progressOut, "  → Highest level fully met: %s\n", ansi("0;32", versionLabel+" "+highest.String(), useColor))
		} else {
			fmt.Fprintf(progressOut, "  → Highest level fully met: %s\n", ansi("0;31", "None", useColor))
		}
		fmt.Fprintln(progressOut)
	}

	if *verbose && len(featureMap) > 0 {
		fmt.Fprintln(progressOut, "Per-Feature Matrix:")
		fmt.Fprintf(progressOut, "  %-32s %-14s  %6s  %6s  %6s  %s\n",
			"Feature", "Level", "Passed", "Failed", "Skipped", "Status")
		fmt.Fprintln(progressOut, "  "+strings.Repeat("─", 80))

		// Collect and sort feature keys for stable output.
		type featureRow struct {
			key  featureKey
			stat *featureStats
		}
		var rows []featureRow
		for k, v := range featureMap {
			rows = append(rows, featureRow{k, v})
		}
		// Sort by version, then level, then feature name.
		for i := 0; i < len(rows); i++ {
			for j := i + 1; j < len(rows); j++ {
				a, b := rows[i], rows[j]
				if a.key.version > b.key.version ||
					(a.key.version == b.key.version && a.stat.level > b.stat.level) ||
					(a.key.version == b.key.version && a.stat.level == b.stat.level && a.key.feature > b.key.feature) {
					rows[i], rows[j] = rows[j], rows[i]
				}
			}
		}

		for _, row := range rows {
			fs := row.stat
			var status string
			if fs.failed > 0 {
				status = ansi("0;31", "✗ FAIL", useColor)
			} else if fs.passed > 0 {
				status = ansi("0;32", "✓ PASS", useColor)
			} else {
				status = ansi("0;33", "⊘ SKIP", useColor)
			}
			fmt.Fprintf(progressOut, "  %-32s %-14s  %6d  %6d  %6d  %s [%s]\n",
				row.key.feature, fs.level.String(),
				fs.passed, fs.failed, fs.skipped,
				status, row.key.version)
		}
		fmt.Fprintln(progressOut)
	}

	// Clean exit with proper status code
	var exitCode int
	if passedSuites == totalSuites {
		fmt.Fprintln(progressOut, ansi("0;32", "✓ ALL TESTS PASSED", useColor))
		fmt.Fprintln(progressOut)
		exitCode = 0
	} else {
		fmt.Fprintln(progressOut, ansi("0;31", "✗ SOME TESTS FAILED", useColor))
		fmt.Fprintln(progressOut)
		exitCode = 1
	}

	// Build structured report and write it when a non-text format is requested.
	if *format != "text" {
		report := buildRunReport(buildVersion, *serverURL, passedSuites, totalSuites,
			passedTests, failedTests, skippedTests, totalTests, suitesToRun)

		reportDest, closeFile, err := openReportDest(*outputFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: cannot open output file %q: %v\n", *outputFile, err)
			os.Exit(1)
		}

		var writeErr error
		switch *format {
		case "junit":
			writeErr = report.WriteJUnit(reportDest)
		case "json":
			writeErr = report.WriteJSON(reportDest)
		case "sarif":
			writeErr = report.WriteSARIF(reportDest)
		}
		closeFile()
		if writeErr != nil {
			fmt.Fprintf(os.Stderr, "Error writing %s report: %v\n", *format, writeErr)
			os.Exit(1)
		}
	}

	os.Exit(exitCode)
}

// buildRunReport assembles a RunReport from the completed suite run.
func buildRunReport(
	toolVersion, serverURL string,
	passedSuites, totalSuites, passedTests, failedTests, skippedTests, totalTests int,
	suitesToRun []preparedSuite,
) *framework.RunReport {
	suites := make([]framework.SuiteRunResult, 0, len(suitesToRun))
	for _, ps := range suitesToRun {
		lvl := ""
		if ps.info.ConformanceLevel != framework.LevelUnspecified {
			lvl = ps.info.ConformanceLevel.String()
		}
		suites = append(suites, framework.SuiteRunResult{
			Name:             ps.info.Name,
			Version:          ps.info.Version,
			ConformanceLevel: lvl,
			Feature:          ps.info.Feature,
			Results:          ps.suite.Results,
		})
	}
	return &framework.RunReport{
		ToolVersion:  toolVersion,
		ServerURL:    serverURL,
		TotalSuites:  totalSuites,
		PassedSuites: passedSuites,
		TotalTests:   totalTests,
		PassedTests:  passedTests,
		FailedTests:  failedTests,
		SkippedTests: skippedTests,
		Suites:       suites,
	}
}

// openReportDest returns a writer for the report output.
// If path is empty it returns os.Stdout and a no-op closer.
func openReportDest(path string) (io.Writer, func(), error) {
	if path == "" {
		return os.Stdout, func() {}, nil
	}
	f, err := os.Create(path)
	if err != nil {
		return nil, nil, err
	}
	return f, func() { _ = f.Close() }, nil
}

// waitForServer polls the service root until it responds with HTTP 200 or the
// timeout (in seconds) elapses. It returns true once the server is reachable.
func waitForServer(serverURL string, timeoutSeconds int, out io.Writer, useColor bool) bool {
	fmt.Fprintf(out, "Waiting for OData service at %s ...\n", serverURL)
	for i := 0; i < timeoutSeconds; i++ {
		if checkServerConnectivity(serverURL) {
			fmt.Fprintln(out, ansi("0;32", "✓ Service is reachable!", useColor))
			fmt.Fprintln(out)
			return true
		}
		time.Sleep(1 * time.Second)
	}
	return false
}

// fetchCapabilityProfile fetches $metadata from the service and parses it into a
// CapabilityProfile. Returns an error if the document cannot be retrieved or parsed.
func fetchCapabilityProfile(serverURL string) (*framework.CapabilityProfile, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(serverURL + "/$metadata")
	if err != nil {
		return nil, fmt.Errorf("GET /$metadata: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET /$metadata returned status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading $metadata body: %w", err)
	}
	return framework.ParseCapabilityProfile(body)
}

// checkServerConnectivity returns true if the service root responds with 200.
func checkServerConnectivity(serverURL string) bool {
	resp, err := framework.NewTestSuite("", "", "").Client.Get(serverURL + "/")
	if err != nil {
		return false
	}
	//nolint:errcheck
	defer func() { _ = resp.Body.Close() }()
	return resp.StatusCode == 200
}

// isTerminal reports whether w is a file descriptor connected to a terminal.
func isTerminal(w io.Writer) bool {
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	fi, err := f.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

// ansi wraps text in an ANSI color escape sequence when enabled is true.
func ansi(code, text string, enabled bool) string {
	if !enabled {
		return text
	}
	return "\033[" + code + "m" + text + "\033[0m"
}
