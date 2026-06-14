package framework

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"strings"
)

// RunReport holds all results from a compliance run.
type RunReport struct {
	ToolVersion  string
	ServerURL    string
	TotalSuites  int
	PassedSuites int
	TotalTests   int
	PassedTests  int
	FailedTests  int
	SkippedTests int
	Suites       []SuiteRunResult
}

// SuiteRunResult holds results for a single test suite.
type SuiteRunResult struct {
	Name             string
	Version          string
	ConformanceLevel string
	Feature          string
	Results          *TestResults
}

// WriteJSON writes the report as indented JSON.
func (r *RunReport) WriteJSON(w io.Writer) error {
	type testJSON struct {
		Name   string `json:"name"`
		Status string `json:"status"`
		Error  string `json:"error,omitempty"`
	}
	type suiteJSON struct {
		Name             string     `json:"name"`
		Version          string     `json:"version"`
		ConformanceLevel string     `json:"conformanceLevel,omitempty"`
		Feature          string     `json:"feature,omitempty"`
		Total            int        `json:"total"`
		Passed           int        `json:"passed"`
		Failed           int        `json:"failed"`
		Skipped          int        `json:"skipped"`
		Tests            []testJSON `json:"tests"`
	}
	type summaryJSON struct {
		TotalSuites  int `json:"totalSuites"`
		PassedSuites int `json:"passedSuites"`
		TotalTests   int `json:"totalTests"`
		PassedTests  int `json:"passedTests"`
		FailedTests  int `json:"failedTests"`
		SkippedTests int `json:"skippedTests"`
	}
	type reportJSON struct {
		ToolVersion string      `json:"toolVersion"`
		ServerURL   string      `json:"serverURL"`
		Summary     summaryJSON `json:"summary"`
		Suites      []suiteJSON `json:"suites"`
	}

	suites := make([]suiteJSON, 0, len(r.Suites))
	for _, s := range r.Suites {
		tests := make([]testJSON, 0, len(s.Results.Details))
		for _, d := range s.Results.Details {
			tests = append(tests, testJSON{
				Name:   d.Name,
				Status: strings.ToLower(d.Status.String()),
				Error:  d.Error,
			})
		}
		suites = append(suites, suiteJSON{
			Name:             s.Name,
			Version:          s.Version,
			ConformanceLevel: s.ConformanceLevel,
			Feature:          s.Feature,
			Total:            s.Results.Total,
			Passed:           s.Results.Passed,
			Failed:           s.Results.Failed,
			Skipped:          s.Results.Skipped,
			Tests:            tests,
		})
	}

	report := reportJSON{
		ToolVersion: r.ToolVersion,
		ServerURL:   r.ServerURL,
		Summary: summaryJSON{
			TotalSuites:  r.TotalSuites,
			PassedSuites: r.PassedSuites,
			TotalTests:   r.TotalTests,
			PassedTests:  r.PassedTests,
			FailedTests:  r.FailedTests,
			SkippedTests: r.SkippedTests,
		},
		Suites: suites,
	}

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(report)
}

// WriteJUnit writes the report as JUnit XML (compatible with GitHub Actions test UI).
func (r *RunReport) WriteJUnit(w io.Writer) error {
	type junitFailure struct {
		XMLName xml.Name `xml:"failure"`
		Message string   `xml:"message,attr"`
		Text    string   `xml:",chardata"`
	}
	type junitSkipped struct {
		XMLName xml.Name `xml:"skipped"`
		Message string   `xml:"message,attr,omitempty"`
	}
	type junitTestCase struct {
		XMLName   xml.Name      `xml:"testcase"`
		Name      string        `xml:"name,attr"`
		Classname string        `xml:"classname,attr"`
		Failure   *junitFailure `xml:"failure,omitempty"`
		Skipped   *junitSkipped `xml:"skipped,omitempty"`
	}
	type junitTestSuite struct {
		XMLName   xml.Name        `xml:"testsuite"`
		Name      string          `xml:"name,attr"`
		Tests     int             `xml:"tests,attr"`
		Failures  int             `xml:"failures,attr"`
		Skipped   int             `xml:"skipped,attr"`
		Errors    int             `xml:"errors,attr"`
		TestCases []junitTestCase `xml:"testcase"`
	}
	type junitTestSuites struct {
		XMLName    xml.Name         `xml:"testsuites"`
		Name       string           `xml:"name,attr"`
		Tests      int              `xml:"tests,attr"`
		Failures   int              `xml:"failures,attr"`
		Skipped    int              `xml:"skipped,attr"`
		Errors     int              `xml:"errors,attr"`
		TestSuites []junitTestSuite `xml:"testsuite"`
	}

	suites := make([]junitTestSuite, 0, len(r.Suites))
	for _, s := range r.Suites {
		classname := "OData/" + s.Version
		cases := make([]junitTestCase, 0, len(s.Results.Details))
		for _, d := range s.Results.Details {
			tc := junitTestCase{
				Name:      d.Name,
				Classname: classname,
			}
			switch d.Status {
			case StatusFail:
				tc.Failure = &junitFailure{
					Message: d.Error,
					Text:    d.Error,
				}
			case StatusSkip:
				tc.Skipped = &junitSkipped{Message: d.Error}
			}
			cases = append(cases, tc)
		}
		suites = append(suites, junitTestSuite{
			Name:      s.Name,
			Tests:     s.Results.Total,
			Failures:  s.Results.Failed,
			Skipped:   s.Results.Skipped,
			Errors:    0,
			TestCases: cases,
		})
	}

	root := junitTestSuites{
		Name:       "OData Compliance Suite",
		Tests:      r.TotalTests,
		Failures:   r.FailedTests,
		Skipped:    r.SkippedTests,
		Errors:     0,
		TestSuites: suites,
	}

	if _, err := fmt.Fprint(w, xml.Header); err != nil {
		return err
	}
	enc := xml.NewEncoder(w)
	enc.Indent("", "  ")
	if err := enc.Encode(root); err != nil {
		return err
	}
	return enc.Flush()
}

// WriteSARIF writes the report as SARIF 2.1.0 JSON for GitHub code-scanning annotations.
func (r *RunReport) WriteSARIF(w io.Writer) error {
	type sarifMessage struct {
		Text string `json:"text"`
	}
	type sarifRule struct {
		ID               string       `json:"id"`
		ShortDescription sarifMessage `json:"shortDescription"`
	}
	type sarifDriver struct {
		Name    string      `json:"name"`
		Version string      `json:"version"`
		Rules   []sarifRule `json:"rules"`
	}
	type sarifTool struct {
		Driver sarifDriver `json:"driver"`
	}
	type sarifResult struct {
		RuleID  string       `json:"ruleId"`
		Level   string       `json:"level"`
		Message sarifMessage `json:"message"`
	}
	type sarifRun struct {
		Tool    sarifTool     `json:"tool"`
		Results []sarifResult `json:"results"`
	}
	type sarifRoot struct {
		Schema  string     `json:"$schema"`
		Version string     `json:"version"`
		Runs    []sarifRun `json:"runs"`
	}

	rulesMap := map[string]bool{}
	resultsMap := map[string]bool{}
	var rules []sarifRule
	var results []sarifResult

	for _, s := range r.Suites {
		for _, d := range s.Results.Details {
			if d.Status != StatusFail {
				continue
			}
			ruleID := s.Name + "/" + sanitizeSARIFID(d.Name)
			if !rulesMap[ruleID] {
				rulesMap[ruleID] = true
				rules = append(rules, sarifRule{
					ID:               ruleID,
					ShortDescription: sarifMessage{Text: d.Name},
				})
			}
			if !resultsMap[ruleID] {
				resultsMap[ruleID] = true
				results = append(results, sarifResult{
					RuleID:  ruleID,
					Level:   "error",
					Message: sarifMessage{Text: d.Error},
				})
			}
		}
	}

	if rules == nil {
		rules = []sarifRule{}
	}
	if results == nil {
		results = []sarifResult{}
	}

	root := sarifRoot{
		Schema:  "https://raw.githubusercontent.com/oasis-tcs/sarif-spec/master/Schemata/sarif-schema-2.1.0.json",
		Version: "2.1.0",
		Runs: []sarifRun{
			{
				Tool: sarifTool{
					Driver: sarifDriver{
						Name:    "odata-compliance-suite",
						Version: r.ToolVersion,
						Rules:   rules,
					},
				},
				Results: results,
			},
		},
	}

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(root)
}

func sanitizeSARIFID(s string) string {
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' || r == '.' {
			b.WriteRune(r)
		} else {
			b.WriteRune('_')
		}
	}
	return b.String()
}
