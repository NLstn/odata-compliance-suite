package v4_02

import (
	"encoding/xml"
	"fmt"
	"net/http"
	"slices"

	"github.com/nlstn/odata-compliance-suite/framework"
)

func ProtocolVersioning() *framework.TestSuite {
	suite := framework.NewTestSuite("5.1 OData 4.02 protocol versioning", "Checks response negotiation and CSDL versions.", protocolSpec+"#sec_ProtocolVersioning")
	suite.AddTest("test_negotiate_402", "The response uses the highest supported version within the client cap", func(ctx *framework.TestContext) error {
		for _, tc := range []struct{ cap, want string }{
			{"4.0", "4.0"}, {"4.01", "4.01"}, {"4.019", "4.01"}, {"4.02", "4.02"}, {"4.1", "4.02"}, {"5.0", "4.02"},
		} {
			for _, path := range []string{"/", "/Products?$top=1"} {
				resp, err := ctx.GET(path, framework.Header{Key: "OData-MaxVersion", Value: tc.cap})
				if err != nil {
					return err
				}
				if err := assertResponse(ctx, resp, http.StatusOK, tc.want); err != nil {
					return fmt.Errorf("%s with cap %s: %w", path, tc.cap, err)
				}
			}
		}
		return nil
	})
	suite.AddTest("test_metadata_402", "XML and JSON CSDL identify the negotiated edmx version", func(ctx *framework.TestContext) error {
		for _, version := range []string{"4.01", "4.02"} {
			cap := framework.Header{Key: "OData-MaxVersion", Value: version}
			resp, err := ctx.GET("/$metadata", cap, framework.Header{Key: "Accept", Value: "application/xml"})
			if err != nil {
				return err
			}
			if err := assertResponse(ctx, resp, http.StatusOK, version); err != nil {
				return err
			}
			var edmx struct {
				XMLName xml.Name `xml:"http://docs.oasis-open.org/odata/ns/edmx Edmx"`
				Version string   `xml:"Version,attr"`
			}
			if err := xml.Unmarshal(resp.Body, &edmx); err != nil {
				return err
			}
			if edmx.Version != version {
				return fmt.Errorf("XML Edmx Version = %q, want %q", edmx.Version, version)
			}
			resp, err = ctx.GET("/$metadata", cap, framework.Header{Key: "Accept", Value: "application/json"})
			if err != nil {
				return err
			}
			if err := assertResponse(ctx, resp, http.StatusOK, version); err != nil {
				return err
			}
			var csdl struct {
				Version   string `json:"$Version"`
				Container string `json:"$EntityContainer"`
			}
			if err := ctx.GetJSON(resp, &csdl); err != nil {
				return err
			}
			if csdl.Version != version || csdl.Container == "" {
				return fmt.Errorf("expected JSON CSDL version %q and an entity container, got %+v", version, csdl)
			}
		}
		return nil
	})
	suite.AddTest("test_protocol_versions_advertised", "Core.ODataVersions advertises 4.02 and both supported earlier versions", func(ctx *framework.TestContext) error {
		resp, err := ctx.GET("/$metadata", framework.Header{Key: "Accept", Value: "application/xml"})
		if err != nil {
			return err
		}
		if err := ctx.AssertStatusCode(resp, http.StatusOK); err != nil {
			return err
		}
		type annotation struct {
			Term     string   `xml:"Term,attr"`
			Versions []string `xml:"Collection>String"`
		}
		var csdl struct {
			Schemas []struct {
				Namespace  string `xml:"Namespace,attr"`
				Containers []struct {
					Name        string       `xml:"Name,attr"`
					Annotations []annotation `xml:"Annotation"`
				} `xml:"EntityContainer"`
				Annotations []struct {
					Target string       `xml:"Target,attr"`
					Terms  []annotation `xml:"Annotation"`
				} `xml:"Annotations"`
			} `xml:"DataServices>Schema"`
		}
		if err := xml.Unmarshal(resp.Body, &csdl); err != nil {
			return err
		}
		announcesVersions := func(terms []annotation) bool {
			for _, term := range terms {
				if (term.Term == "Core.ODataVersions" || term.Term == "Org.OData.Core.V1.ODataVersions") &&
					slices.Contains(term.Versions, "4.0") && slices.Contains(term.Versions, "4.01") && slices.Contains(term.Versions, "4.02") {
					return true
				}
			}
			return false
		}
		for _, schema := range csdl.Schemas {
			for _, container := range schema.Containers {
				if announcesVersions(container.Annotations) {
					return nil
				}
			}
			for _, group := range schema.Annotations {
				for _, container := range schema.Containers {
					if group.Target == schema.Namespace+"."+container.Name && announcesVersions(group.Terms) {
						return nil
					}
				}
			}
		}
		return fmt.Errorf("metadata does not advertise all three versions in Core.ODataVersions")
	})

	return suite
}
