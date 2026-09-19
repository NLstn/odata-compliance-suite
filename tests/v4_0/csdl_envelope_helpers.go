package v4_0

import (
	"encoding/xml"
	"fmt"
	"strings"

	"github.com/nlstn/odata-compliance-suite/framework"
)

const csdlEdmxNamespace = "http://docs.oasis-open.org/odata/ns/edmx"

type csdlEnvelopeDocument struct {
	XMLName      xml.Name          `xml:"Edmx"`
	Version      string            `xml:"Version,attr"`
	DataServices *csdlDataServices `xml:"DataServices"`
	References   []csdlEnvelopeRef `xml:"Reference"`
}

type csdlDataServices struct {
	Schemas []csdlEnvelopeSchema `xml:"Schema"`
}

type csdlEnvelopeSchema struct {
	Namespace string `xml:"Namespace,attr"`
}

type csdlEnvelopeRef struct {
	URI                string                           `xml:"Uri,attr"`
	Includes           []csdlEnvelopeInclude            `xml:"Include"`
	IncludeAnnotations []csdlEnvelopeIncludeAnnotations `xml:"IncludeAnnotations"`
}

type csdlEnvelopeInclude struct {
	Namespace string `xml:"Namespace,attr"`
	Alias     string `xml:"Alias,attr"`
}

type csdlEnvelopeIncludeAnnotations struct {
	TermNamespace   string `xml:"TermNamespace,attr"`
	Qualifier       string `xml:"Qualifier,attr"`
	TargetNamespace string `xml:"TargetNamespace,attr"`
}

func fetchCSDLDocument(ctx *framework.TestContext) (*csdlEnvelopeDocument, error) {
	resp, err := ctx.GET("/$metadata")
	if err != nil {
		return nil, err
	}
	if err := ctx.AssertStatusCode(resp, 200); err != nil {
		return nil, err
	}

	var doc csdlEnvelopeDocument
	if err := xml.Unmarshal(resp.Body, &doc); err != nil {
		return nil, fmt.Errorf("metadata is not well-formed CSDL XML: %w", err)
	}
	if doc.XMLName.Local != "Edmx" || doc.XMLName.Space != csdlEdmxNamespace {
		return nil, fmt.Errorf("metadata root is {%s}%s, want {%s}Edmx", doc.XMLName.Space, doc.XMLName.Local, csdlEdmxNamespace)
	}
	if strings.TrimSpace(doc.Version) == "" {
		return nil, fmt.Errorf("edmx:Edmx Version attribute is empty")
	}
	if doc.DataServices == nil {
		return nil, fmt.Errorf("metadata is missing edmx:DataServices")
	}
	if len(doc.DataServices.Schemas) == 0 {
		return nil, fmt.Errorf("edmx:DataServices contains no Schema elements")
	}
	return &doc, nil
}

func validateCSDLReferenceAttributes(doc *csdlEnvelopeDocument) error {
	for i, ref := range doc.References {
		if strings.TrimSpace(ref.URI) == "" {
			return fmt.Errorf("edmx:Reference %d has an empty Uri attribute", i)
		}
		for j, include := range ref.Includes {
			if strings.TrimSpace(include.Namespace) == "" {
				return fmt.Errorf("edmx:Include %d in Reference %d has an empty Namespace", j, i)
			}
			if strings.ContainsAny(include.Namespace, " \t\r\n") || strings.ContainsAny(include.Alias, " \t\r\n") {
				return fmt.Errorf("edmx:Include %d in Reference %d contains whitespace in an attribute", j, i)
			}
		}
		for j, include := range ref.IncludeAnnotations {
			if strings.TrimSpace(include.TermNamespace) == "" {
				return fmt.Errorf("edmx:IncludeAnnotations %d in Reference %d has an empty TermNamespace", j, i)
			}
			for name, value := range map[string]string{
				"TermNamespace":   include.TermNamespace,
				"Qualifier":       include.Qualifier,
				"TargetNamespace": include.TargetNamespace,
			} {
				if strings.ContainsAny(value, " \t\r\n") {
					return fmt.Errorf("edmx:IncludeAnnotations %d in Reference %d contains whitespace in %s", j, i, name)
				}
			}
		}
	}
	return nil
}
