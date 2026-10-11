package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v6"
	"k8s.io/apimachinery/pkg/api/resource"
	"sigs.k8s.io/yaml"
)

// The patterns must agree with the parsers the app and the API server use: a
// false reject would block a valid install.
func TestQuantityPattern(t *testing.T) {
	re := regexp.MustCompile(quantityPattern)
	for _, s := range []string{"64Mi", "500m", "1.5", "5Gi", "1e3", "1E3", "1e-3", "0", "+1", "-1", ".5", "1k", "1K", "1Ki", "1ki", "1M", "1Mi", "big", "", "1 Gi", "1.Gi"} {
		_, err := resource.ParseQuantity(s)
		if got := re.MatchString(s); got != (err == nil) {
			t.Errorf("%q: pattern match = %v, ParseQuantity ok = %v", s, got, err == nil)
		}
	}
}

// parseTokenExpiryOK mirrors ParseTokenExpiry in internal/config, which lives
// in the app's module and can't be imported from here.
func parseTokenExpiryOK(s string) bool {
	if strings.EqualFold(s, "never") {
		return true
	}
	s = strings.ToLower(strings.TrimSpace(s))
	if !strings.HasSuffix(s, "d") {
		return false
	}
	days, err := strconv.Atoi(strings.TrimSuffix(s, "d"))
	return err == nil && days > 0
}

func TestTokenExpiryPattern(t *testing.T) {
	re := regexp.MustCompile(tokenExpiryPattern)
	for _, s := range []string{
		"never", "Never", "NEVER", "10d", "10D", "1d", "30d", "365d", "007d", "+5d", " 10d ", "10d\n",
		"0d", "00d", "-5d", "10", "d", "5 d", "10 days", "1w", "forever", "never ", " never", "1.5d", "1_0d", "d10",
	} {
		if got, want := re.MatchString(s), parseTokenExpiryOK(s); got != want {
			t.Errorf("%q: pattern match = %v, ParseTokenExpiry ok = %v", s, got, want)
		}
	}
	// The chart leaves TOKEN_EXPIRY unset when empty, so the app default applies.
	if !re.MatchString("") {
		t.Error(`"" must match: it means "use the app default"`)
	}
}

func TestMongoURIPattern(t *testing.T) {
	re := regexp.MustCompile(mongoURIPattern)
	for in, want := range map[string]bool{
		"": true, "mongodb://mongo:27017/clipboard": true, "mongodb+srv://u:p@db.example.com/clipboard": true,
		"mongo://db": false, "http://db": false, "mongodb://": false, "mongodb+srv://": false, "db:27017": false,
	} {
		if got := re.MatchString(in); got != want {
			t.Errorf("%q: got %v, want %v", in, got, want)
		}
	}
}

// compile builds the schema the way main does.
func compile(t *testing.T) *jsonschema.Schema {
	t.Helper()
	raw, err := build()
	if err != nil {
		t.Fatal(err)
	}
	doc, err := jsonschema.UnmarshalJSON(bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	c := jsonschema.NewCompiler()
	if err := c.AddResource("values.schema.json", doc); err != nil {
		t.Fatal(err)
	}
	schema, err := c.Compile("values.schema.json")
	if err != nil {
		t.Fatal(err)
	}
	return schema
}

func validateYAML(schema *jsonschema.Schema, doc string) error {
	j, err := yaml.YAMLToJSON([]byte(doc))
	if err != nil {
		return err
	}
	var v any
	if err := json.Unmarshal(j, &v); err != nil {
		return err
	}
	return schema.Validate(v)
}

// The schema must reject what the templates or the app would choke on, and
// accept what they tolerate.
func TestValidation(t *testing.T) {
	schema := compile(t)
	for name, tc := range map[string]struct {
		doc string
		ok  bool
	}{
		"unknown top-level key":            {"bogus: 1", false},
		"typo in nested key":               {"httpRoute: {parentRef: {nmae: gw}}", false},
		"bad service type":                 {"service: {type: Nope}", false},
		"port as string":                   {"service: {port: abc}", false},
		"port out of range":                {"service: {port: 70000}", false},
		"bad pull policy":                  {"image: {pullPolicy: Sometimes}", false},
		"empty image tag":                  {"image: {tag: ''}", false},
		"bad volume size":                  {"persistence: {size: big}", false},
		"volume size":                      {"persistence: {size: 50Gi}", true},
		"bad access mode":                  {"persistence: {accessModes: [ReadWriteTwice]}", false},
		"access mode":                      {"persistence: {accessModes: [ReadWriteMany]}", true},
		"token expiry without unit":        {"config: {tokenExpiry: '10'}", false},
		"token expiry as number":           {"config: {tokenExpiry: 10}", false},
		"token expiry days":                {"config: {tokenExpiry: 10d}", true},
		"token expiry never":               {"config: {tokenExpiry: never}", true},
		"bad mongo scheme":                 {"config: {mongodbUri: 'mongo://db'}", false},
		"mongo srv":                        {"config: {mongodbUri: 'mongodb+srv://u:p@db.example.com/clipboard'}", true},
		"numeric password from --set":      {"config: {passwords: 1234}", true},
		"boolean password":                 {"config: {passwords: true}", false},
		"route enabled without gateway":    {"httpRoute: {enabled: true}", false},
		"route enabled with empty gateway": {"httpRoute: {enabled: true, parentRef: {name: ''}}", false},
		"route enabled with gateway":       {"httpRoute: {enabled: true, parentRef: {name: gw}}", true},
		"route disabled without gateway":   {"httpRoute: {enabled: false, parentRef: {name: ''}}", true},
		"bad name override":                {"nameOverride: Bad_Name", false},
		"name override":                    {"nameOverride: clip", true},
		"null label map":                   {"commonLabels: null", true},
		"numeric label value":              {"commonLabels: {team: 1}", false},
		"numeric cpu":                      {"resources: {limits: {cpu: 1}}", true},
		"bad cpu":                          {"resources: {limits: {cpu: lots}}", false},
		"extra env without name":           {"extraEnv: [{value: x}]", false},
		"extra env":                        {"extraEnv: [{name: A, value: x}]", true},
		"bad toleration operator":          {"tolerations: [{key: a, operator: Maybe}]", false},
	} {
		t.Run(name, func(t *testing.T) {
			if err := validateYAML(schema, tc.doc); (err == nil) != tc.ok {
				t.Errorf("valid = %v, want %v: %v", err == nil, tc.ok, err)
			}
		})
	}
}

// Every values file in the repo must validate: the schema is closed, so a
// key missing from values.go would otherwise break installs.
func TestRepoValuesValidate(t *testing.T) {
	schema := compile(t)
	files := []string{"../../charts/clipboard/values.yaml"}
	fixtures, err := filepath.Glob("../../tests/values/*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	files = append(files, fixtures...)

	for _, f := range files {
		t.Run(filepath.Base(f), func(t *testing.T) {
			y, err := os.ReadFile(f)
			if err != nil {
				t.Fatal(err)
			}
			if err := validateYAML(schema, string(y)); err != nil {
				t.Error(err)
			}
		})
	}
}
