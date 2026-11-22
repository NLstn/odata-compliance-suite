package v4_01

import (
	"net/http"
	"testing"

	"github.com/nlstn/odata-compliance-suite/framework"
)

func TestOmitValuesResponse(t *testing.T) {
	for _, tc := range []struct {
		name, body, applied string
		annotated, wantPass bool
	}{
		{"preference ignored", `{"ID":"id","Name":"name","Description":null}`, "", false, true},
		{"omission confirmed", `{"ID":"id","Name":"name"}`, `another, omit-values="nulls"`, false, true},
		{"missing confirmation", `{"ID":"id","Name":"name"}`, "", false, false},
		{"wrong confirmation", `{"ID":"id","Name":"name"}`, "omit-values=defaults", false, false},
		{"annotated property omitted", `{"ID":"id","Name":"name"}`, "omit-values=nulls", true, false},
		{"non-null field omitted", `{"ID":"id"}`, "omit-values=nulls", false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			original := map[string]interface{}{"ID": "id", "Name": "name", "Description": nil}
			if tc.annotated {
				original["Description@Example.Term"] = true
			}
			headers := http.Header{}
			headers.Set("OData-Version", "4.01")
			if tc.applied != "" {
				headers.Set("Preference-Applied", tc.applied)
			}
			resp := &framework.HTTPResponse{StatusCode: http.StatusOK, Headers: headers, Body: []byte(tc.body)}
			err := assertOmitValuesResponse(&framework.TestContext{}, resp, original, "nulls")
			if (err == nil) != tc.wantPass {
				t.Fatalf("assertOmitValuesResponse returned %v, want pass = %t", err, tc.wantPass)
			}
		})
	}
}
