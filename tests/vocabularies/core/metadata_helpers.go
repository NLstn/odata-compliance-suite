package core

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"strings"

	"github.com/nlstn/odata-compliance-suite/framework"
)

func metadataNamespace(metadataXML []byte) (string, error) {
	decoder := xml.NewDecoder(bytes.NewReader(metadataXML))
	for {
		token, err := decoder.Token()
		if err != nil {
			if err == io.EOF {
				break
			}
			return "", fmt.Errorf("failed to parse metadata XML: %w", err)
		}
		start, ok := token.(xml.StartElement)
		if !ok || start.Name.Local != "Schema" {
			continue
		}
		for _, attr := range start.Attr {
			if attr.Name.Local == "Namespace" {
				return attr.Value, nil
			}
		}
	}
	return "", fmt.Errorf("metadata namespace not found")
}

func hasAnnotation(metadataXML []byte, target, term string) (bool, error) {
	decoder := xml.NewDecoder(bytes.NewReader(metadataXML))
	targetEntityType, targetProperty, _ := strings.Cut(target, "/")
	var schemaNamespace, currentType, currentProperty, externalTarget string
	termMatches := func(candidate string) bool {
		return candidate == term ||
			strings.TrimPrefix(candidate, "Core.") == strings.TrimPrefix(term, "Org.OData.Core.V1.") ||
			strings.TrimPrefix(candidate, "Org.OData.Core.V1.") == strings.TrimPrefix(term, "Core.")
	}

	for {
		token, err := decoder.Token()
		if err != nil {
			if err == io.EOF {
				break
			}
			return false, fmt.Errorf("failed to parse metadata XML: %w", err)
		}
		switch node := token.(type) {
		case xml.StartElement:
			switch node.Name.Local {
			case "Schema":
				schemaNamespace = attribute(node, "Namespace")
			case "Annotations":
				externalTarget = attribute(node, "Target")
			case "EntityType", "ComplexType":
				name := attribute(node, "Name")
				currentType = name
				if schemaNamespace != "" {
					currentType = schemaNamespace + "." + name
				}
			case "Property":
				currentProperty = attribute(node, "Name")
			case "Annotation":
				if !termMatches(attribute(node, "Term")) {
					continue
				}
				if externalTarget == target {
					return true, nil
				}
				typeMatches := currentType == targetEntityType || strings.HasSuffix(targetEntityType, "."+currentType)
				if typeMatches && ((targetProperty == "" && currentProperty == "") || currentProperty == targetProperty) {
					return true, nil
				}
			}
		case xml.EndElement:
			switch node.Name.Local {
			case "Annotations":
				externalTarget = ""
			case "Property":
				currentProperty = ""
			case "EntityType", "ComplexType":
				currentType = ""
			case "Schema":
				schemaNamespace = ""
			}
		}
	}
	return false, nil
}

func attribute(start xml.StartElement, name string) string {
	for _, attr := range start.Attr {
		if attr.Name.Local == name {
			return attr.Value
		}
	}
	return ""
}

// coreAnnotationHit is a single <Annotation> match found inside an external
// <Annotations Target="..."> block, carrying whichever value form it used.
type coreAnnotationHit struct {
	Target     string
	Bool       *bool
	String     *string
	EnumMember *string
}

type coreAnnotationsDoc struct {
	DataServices struct {
		Schemas []struct {
			Annotations []struct {
				Target      string `xml:"Target,attr"`
				Annotations []struct {
					Term       string  `xml:"Term,attr"`
					Bool       *bool   `xml:"Bool,attr"`
					String     *string `xml:"String,attr"`
					EnumMember *string `xml:"EnumMember,attr"`
				} `xml:"Annotation"`
			} `xml:"Annotations"`
		} `xml:"Schema"`
	} `xml:"DataServices"`
}

// findAnnotationsByTerm scans every external <Annotations Target="..."> block
// in the metadata document for <Annotation Term="term"> and returns one hit
// per match, whatever attribute form (Bool/String/EnumMember) it used. It
// accepts both the "Core." alias and the fully-qualified "Org.OData.Core.V1."
// form of term. Inline annotations nested directly inside a Property or
// EntityType element are not scanned — this model only ever emits the
// external-Annotations-block form.
func findAnnotationsByTerm(metadataXML []byte, term string) ([]coreAnnotationHit, error) {
	var doc coreAnnotationsDoc
	if err := xml.Unmarshal(metadataXML, &doc); err != nil {
		return nil, fmt.Errorf("failed to parse metadata XML: %w", err)
	}

	shortForm := "Core." + strings.TrimPrefix(term, "Org.OData.Core.V1.")
	longForm := "Org.OData.Core.V1." + strings.TrimPrefix(term, "Core.")

	var hits []coreAnnotationHit
	for _, schema := range doc.DataServices.Schemas {
		for _, block := range schema.Annotations {
			for _, ann := range block.Annotations {
				if ann.Term != shortForm && ann.Term != longForm {
					continue
				}
				hits = append(hits, coreAnnotationHit{
					Target:     block.Target,
					Bool:       ann.Bool,
					String:     ann.String,
					EnumMember: ann.EnumMember,
				})
			}
		}
	}
	return hits, nil
}

// csdlEntityType and csdlEntitySet describe just enough of the CSDL EntityType/
// EntityContainer shape to build a minimally valid create payload below.
type csdlEntityTypeProperty struct {
	Name     string `xml:"Name,attr"`
	Type     string `xml:"Type,attr"`
	Nullable string `xml:"Nullable,attr"`
}

type csdlEntityType struct {
	Name string `xml:"Name,attr"`
	Key  struct {
		PropertyRefs []struct {
			Name string `xml:"Name,attr"`
		} `xml:"PropertyRef"`
	} `xml:"Key"`
	Properties []csdlEntityTypeProperty `xml:"Property"`
}

type csdlEntitySet struct {
	Name       string `xml:"Name,attr"`
	EntityType string `xml:"EntityType,attr"`
}

type csdlDoc struct {
	DataServices struct {
		Schemas []struct {
			Namespace       string           `xml:"Namespace,attr"`
			EntityTypes     []csdlEntityType `xml:"EntityType"`
			EntityContainer struct {
				EntitySets []csdlEntitySet `xml:"EntitySet"`
			} `xml:"EntityContainer"`
		} `xml:"Schema"`
	} `xml:"DataServices"`
}

// buildValidCreatePayload constructs a minimally valid POST body for
// entitySetName: one placeholder value per non-key property declared
// Nullable="false". This lets a restriction-rejection test send a request
// that would otherwise succeed, so a rejection can only be attributed to the
// restriction under test, not to an incomplete body the server would reject
// regardless (e.g. a missing required field).
func buildValidCreatePayload(metadataXML []byte, entitySetName string) (map[string]interface{}, error) {
	var doc csdlDoc
	if err := xml.Unmarshal(metadataXML, &doc); err != nil {
		return nil, fmt.Errorf("failed to parse metadata XML: %w", err)
	}

	var entityTypeName string
	for _, schema := range doc.DataServices.Schemas {
		for _, set := range schema.EntityContainer.EntitySets {
			if set.Name == entitySetName {
				entityTypeName = set.EntityType
			}
		}
	}
	if entityTypeName == "" {
		return nil, fmt.Errorf("entity set %q not found in metadata", entitySetName)
	}

	payload := map[string]interface{}{}
	for _, schema := range doc.DataServices.Schemas {
		for _, et := range schema.EntityTypes {
			qualifiedName := schema.Namespace + "." + et.Name
			if qualifiedName != entityTypeName && et.Name != entityTypeName {
				continue
			}
			keySet := make(map[string]bool, len(et.Key.PropertyRefs))
			for _, ref := range et.Key.PropertyRefs {
				keySet[ref.Name] = true
			}
			for _, prop := range et.Properties {
				if prop.Nullable == "false" && !keySet[prop.Name] {
					payload[prop.Name] = placeholderValueForType(prop.Type)
				}
			}
		}
	}
	return payload, nil
}

// placeholderValueForType returns an arbitrary but validly-typed value for
// edmType, for use in a synthetic create payload where the actual value is
// irrelevant to the behavior under test.
func placeholderValueForType(edmType string) interface{} {
	switch edmType {
	case "Edm.String":
		return "Core Permissions Test Value"
	case "Edm.Boolean":
		return true
	case "Edm.Guid":
		return "00000000-0000-0000-0000-000000000001"
	case "Edm.Date":
		return "2024-01-01"
	case "Edm.DateTimeOffset":
		return "2024-01-01T00:00:00Z"
	case "Edm.TimeOfDay":
		return "00:00:00"
	case "Edm.Duration":
		return "PT0S"
	case "Edm.Int16", "Edm.Int32", "Edm.Int64", "Edm.Byte", "Edm.SByte":
		return 1
	case "Edm.Double", "Edm.Single", "Edm.Decimal":
		return 1.0
	default:
		return "Core Permissions Test Value"
	}
}

func assertODataError(resp *framework.HTTPResponse) error {
	var payload map[string]interface{}
	if err := json.Unmarshal(resp.Body, &payload); err != nil {
		return fmt.Errorf("expected JSON error response, got parse error: %w", err)
	}

	errObjRaw, ok := payload["error"]
	if !ok {
		return fmt.Errorf("missing error object in response")
	}

	errObj, ok := errObjRaw.(map[string]interface{})
	if !ok {
		return fmt.Errorf("error object has unexpected type")
	}

	// OData JSON Format §9.3: error.code is a service-defined string — it is NOT
	// required to equal the HTTP status number. Only assert it is a non-empty string.
	code, ok := errObj["code"].(string)
	if !ok || strings.TrimSpace(code) == "" {
		return fmt.Errorf("error code is missing or empty, got %v", errObj["code"])
	}

	message, ok := errObj["message"].(string)
	if !ok || strings.TrimSpace(message) == "" {
		return fmt.Errorf("error message is missing or empty")
	}

	return nil
}
