package desktopkind

import (
	"encoding/json"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/opencharly/sdk"
	"github.com/opencharly/spec/spec"
)

// This plugin serves a SELF-CONTAINED copy of spec's desktop defs, because the host compiles
// it standalone for Describe while spec's must generate Go. That duplication is sanctioned —
// and it can DRIFT.
//
// When it does, the failure is actively misleading: charly's core accepts a field and this
// plugin rejects it with `#ThemeInput.<field>: field not allowed`, which reads like the field
// does not exist anywhere. plugin-distro shipped exactly that bug — its #DistroInput was
// missing `installer` for weeks — and grew a guard for it. This is that guard, shipped WITH
// the schema rather than after the first drift.
//
// The comparison is against the AUTHORED wire keys (the json tags of spec's generated
// structs), not the Go field names: the wire surface is what an author writes and what this
// schema validates. Go names differ by design (`#DkRender` vs `#ThemeRender`).
func TestServedSchemaCoversEverySpecField(t *testing.T) {
	src, err := schemaFS.ReadFile("schema/desktop.cue")
	if err != nil {
		t.Fatalf("reading the served schema: %v", err)
	}
	schema := string(src)

	for _, tc := range []struct {
		def string
		typ any
	}{
		{"#ThemeInput", spec.Theme{}},
		{"#SessionInput", spec.Session{}},
		{"#DisplayManagerInput", spec.DisplayManager{}},
		{"#DesktopEntryInput", spec.DesktopEntry{}},
	} {
		t.Run(tc.def, func(t *testing.T) {
			want := wireKeys(tc.typ)
			got := cueTopLevelKeys(t, schema, tc.def)
			if strings.Join(got, ",") != strings.Join(want, ",") {
				t.Errorf("%s has drifted from spec's authored surface.\n got: %v\nwant: %v\n"+
					"If spec added a field, mirror it here; if spec removed one, remove it here.",
					tc.def, got, want)
			}
		})
	}
}

// wireKeys is the sorted set of json tag names on a spec struct — the AUTHORED keys.
func wireKeys(v any) []string {
	t := reflect.TypeOf(v)
	var out []string
	for i := 0; i < t.NumField(); i++ {
		tag := t.Field(i).Tag.Get("json")
		if tag == "" || tag == "-" {
			continue
		}
		name := strings.Split(tag, ",")[0]
		if name == "" {
			continue
		}
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

var cueField = regexp.MustCompile(`(?m)^\t([a-z_]+)!?\??:`)

func cueTopLevelKeys(t *testing.T, schema, def string) []string {
	t.Helper()
	i := strings.Index(schema, def+": {")
	if i < 0 {
		t.Fatalf("%s is not defined in the served schema", def)
	}
	rest := schema[i:]
	j := strings.Index(rest, "\n}")
	if j < 0 {
		t.Fatalf("%s is not closed", def)
	}
	var out []string
	for _, m := range cueField.FindAllStringSubmatch(rest[:j], -1) {
		out = append(out, m[1])
	}
	sort.Strings(out)
	return out
}

// A body carrying EVERY authored key must survive the round-trip with nothing dropped. The
// field-name guard above catches a missing def; this catches a def that is present but whose
// TYPE cannot hold what the author wrote.
func TestFullBodyRoundTripsWithoutLoss(t *testing.T) {
	body := `{"name":"full","variant":"dark",
		"token":{"accent":"#111111","foreground":"#222222","background":"#333333",
			"cursor":"#444444","selection_foreground":"#555555","selection_background":"#666666",
			"color0":"#000000","color15":"#ffffff"},
		"font":"F","cursor_theme":"C","icon_theme":"I",
		"background":["a.jpg","b.jpg"],
		"render":[{"app":"foot","path":"/tmp/x","content":"a={{.Token.accent}}","mode":"0644","scope":"user","distro":["omarchy"]}]}`

	reply, err := invoke(t, sdk.OpLoad, "theme", body)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	var in, out map[string]any
	if err := json.Unmarshal([]byte(body), &in); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(reply.GetResultJson(), &out); err != nil {
		t.Fatal(err)
	}
	for k := range in {
		if _, ok := out[k]; !ok {
			t.Errorf("key %q was DROPPED by the round-trip — the served schema and the spec type disagree on it", k)
		}
	}
}
