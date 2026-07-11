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
