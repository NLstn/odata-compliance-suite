package aggregation

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"net/url"
	"sort"
	"strings"

	"github.com/nlstn/odata-compliance-suite/framework"
)

// RecursiveHierarchyAnnotation validates the vocabulary metadata that drives
// the hierarchy transformations exercised by the protocol suites.
func RecursiveHierarchyAnnotation() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"Aggregation.RecursiveHierarchy Annotation",
		"Validates the qualified hierarchy declaration and its relationship to hierarchy traversal.",
		"https://oasis-tcs.github.io/odata-vocabularies/vocabularies/Org.OData.Aggregation.V1.html#RecursiveHierarchy",
	)

	suite.AddTest(
		"xml_declares_tree_qualifier",
		"HierarchyNode has a RecursiveHierarchy annotation qualified as Tree",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/$metadata", framework.Header{Key: "Accept", Value: "application/xml"})
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}
			annotation, err := recursiveHierarchy(resp.Body)
			if err != nil {
				return err
			}
			if annotation.qualifier != "Tree" {
				return fmt.Errorf("RecursiveHierarchy qualifier = %q, want Tree", annotation.qualifier)
			}
			return nil
		},
	)

	suite.AddTest(
		"xml_declares_hierarchy_paths",
		"RecursiveHierarchy uses ID and Parent with their vocabulary-defined path expression types",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/$metadata", framework.Header{Key: "Accept", Value: "application/xml"})
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}
			annotation, err := recursiveHierarchy(resp.Body)
			if err != nil {
				return err
			}
			if annotation.nodeProperty != "ID" {
				return fmt.Errorf("NodeProperty PropertyPath = %q, want ID", annotation.nodeProperty)
			}
			if annotation.parentNavigationProperty != "Parent" {
				return fmt.Errorf("ParentNavigationProperty NavigationPropertyPath = %q, want Parent", annotation.parentNavigationProperty)
			}
			return nil
		},
	)

	suite.AddTest(
		"json_metadata_preserves_hierarchy",
		"JSON CSDL exposes the Tree hierarchy with ID and Parent paths",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/$metadata", framework.Header{Key: "Accept", Value: "application/json"})
			if err != nil {
				return err
			}
			if resp.StatusCode == 406 {
				return ctx.Skip("Service does not support JSON metadata format")
			}
			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}
			body := string(resp.Body)
			if !strings.Contains(body, "RecursiveHierarchy#Tree") ||
				!strings.Contains(body, "NodeProperty") ||
				!strings.Contains(body, "ParentNavigationProperty") {
				return fmt.Errorf("JSON CSDL does not preserve the Tree recursive hierarchy record")
			}
			return nil
		},
	)

	suite.AddTest(
		"declared_hierarchy_is_executable",
		"The advertised Tree hierarchy can resolve ancestors of node 4",
		func(ctx *framework.TestContext) error {
			expression := "ancestors($root/HierarchyNodes,Tree,ID,filter(ID eq 4))"
			resp, err := ctx.GET("/HierarchyNodes?$apply=" + url.QueryEscape(expression))
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}
			var payload struct {
				Value []struct {
					ID int `json:"ID"`
				} `json:"value"`
			}
			if err := ctx.GetJSON(resp, &payload); err != nil {
				return err
			}
			sort.Slice(payload.Value, func(i, j int) bool { return payload.Value[i].ID < payload.Value[j].ID })
			if len(payload.Value) != 2 || payload.Value[0].ID != 1 || payload.Value[1].ID != 2 {
				return fmt.Errorf("ancestors of node 4 = %+v, want IDs 1 and 2", payload.Value)
			}
			return nil
		},
	)

	return suite
}

type hierarchyAnnotation struct {
	qualifier                string
	nodeProperty             string
	parentNavigationProperty string
}

func recursiveHierarchy(metadataXML []byte) (hierarchyAnnotation, error) {
	decoder := xml.NewDecoder(bytes.NewReader(metadataXML))
	var result hierarchyAnnotation
	inside := false
	currentProperty := ""
	for {
		token, err := decoder.Token()
		if err != nil {
			if err == io.EOF {
				break
			}
			return result, fmt.Errorf("failed to parse metadata XML: %w", err)
		}
		switch node := token.(type) {
		case xml.StartElement:
			if node.Name.Local == "Annotation" {
				term := xmlAttribute(node, "Term")
				inside = term == "Aggregation.RecursiveHierarchy" || term == "Org.OData.Aggregation.V1.RecursiveHierarchy"
				if inside {
					result.qualifier = xmlAttribute(node, "Qualifier")
				}
			}
			if inside && node.Name.Local == "PropertyValue" {
				currentProperty = xmlAttribute(node, "Property")
				if currentProperty == "NodeProperty" {
					result.nodeProperty = xmlAttribute(node, "PropertyPath")
				}
				if currentProperty == "ParentNavigationProperty" {
					result.parentNavigationProperty = xmlAttribute(node, "NavigationPropertyPath")
				}
			}
			if inside && (node.Name.Local == "PropertyPath" || node.Name.Local == "NavigationPropertyPath") {
				var value string
				if err := decoder.DecodeElement(&value, &node); err != nil {
					return result, err
				}
				if currentProperty == "NodeProperty" && node.Name.Local == "PropertyPath" {
					result.nodeProperty = strings.TrimSpace(value)
				}
				if currentProperty == "ParentNavigationProperty" && node.Name.Local == "NavigationPropertyPath" {
					result.parentNavigationProperty = strings.TrimSpace(value)
				}
			}
		case xml.EndElement:
			if node.Name.Local == "PropertyValue" {
				currentProperty = ""
			}
			if node.Name.Local == "Annotation" && inside {
				return result, nil
			}
		}
	}
	return result, fmt.Errorf("RecursiveHierarchy annotation not found")
}

func xmlAttribute(start xml.StartElement, name string) string {
	for _, attr := range start.Attr {
		if attr.Name.Local == name {
			return attr.Value
		}
	}
	return ""
}
