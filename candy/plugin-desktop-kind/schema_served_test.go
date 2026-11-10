package desktopkind

// schema_served_test.go is the schema-LOAD gate, and the §10.1 S3 gate.
//
// `charly box validate` does NOT compile a served schema: a dangling reference inside a
// served def is green at plugin load and only surfaces when the host validates an authored
// entity (opencharly/spec#301). So this file composes the plugin's OWN Describe reply and
// compiles the body it serves, with the same CUE library the runtime links — the served
// body, not the file on disk, is the surface the host splices onto its base and validates
// every authored `kind: theme|session|displaymanager|desktopentry` entity against.
//
// It pins the S3 half of D-PAT-3 as well: every closed set of words is a vocabulary table,
// its word list is DERIVED from that table, and the enum the author meets is `or()` of the
// derived list — so a word cannot be added to the table and forgotten by the list, nor
// accepted by the enum while the table it claims to derive from lacks it.
//
// It also REPLACES a text parse. plugin_test.go used to assert the four advertised InputDefs
// by substring-matching the schema SOURCE (`strings.Contains(src, def+": {")`), which reads
// the file rather than the surface the host receives; a def written in any other shape
// (a derived enum has no `{`) would have been reported as missing while it resolved fine.
// TestAdvertisedInputDefsResolveInTheServedSchema reads the advertised words AND their defs
// out of the plugin's own Describe reply and resolves each def in the compiled body.

import (
	"context"
	"fmt"
	"sort"
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	pb "github.com/opencharly/spec/proto"
)

// vocabHarness LIFTS the hidden vocabulary tables and their derived word lists into one
// public definition. An `_`-prefixed label is hidden by CUE and cannot be looked up from
// outside the compilation unit, so the served body alone exposes only the enums; this
// fragment adds no semantics, it re-exports the tables the schema already carries so the
// test can read them instead of restating their words.
const vocabHarness = `
#TestVocabulary: {
	tables: {
		syntax:    _syntax
		manager:   _manager
		numlock:   _numlock
		placement: _placement
		category:  _category
	}
	words: {
		syntax:    _syntaxWord
		manager:   _managerWord
		numlock:   _numlockWord
		placement: _placementWord
		category:  _categoryWord
	}
}
`

// vocabProbe pairs a vocabulary table (named as in vocabHarness) with ONE authored entity
// that uses a word from it, so the enum is exercised through the same def the host
// validates against.
type vocabProbe struct {
	name string
	use  func(word string) string
}

// vocabProbes covers every vocabulary table this schema carries. A new table belongs in
// vocabHarness and here: the probe is what proves it is wired into the enum an author
// actually meets.
var vocabProbes = []vocabProbe{
	{"syntax", func(w string) string {
		return fmt.Sprintf(`{#SessionInput: {compositor: "sway", config_path_template: "/x", syntax: %q}}`, w)
	}},
	{"manager", func(w string) string {
		return fmt.Sprintf(`{#DisplayManagerInput: {manager: %q, session: "s"}}`, w)
	}},
	{"numlock", func(w string) string {
		return fmt.Sprintf(`{#DisplayManagerInput: {manager: "sddm", session: "s", numlock: %q}}`, w)
	}},
	{"placement", func(w string) string {
		return fmt.Sprintf(`{#DesktopEntryInput: {entry_name: "e", exec: "x", window: {placement: %q}}}`, w)
	}},
	{"category", func(w string) string {
		return fmt.Sprintf(`{#DesktopEntryInput: {entry_name: "e", exec: "x", categories: [%q]}}`, w)
	}},
}

// servedBody returns the CUE source the plugin serves over Describe — the body the host
// splices onto its base, which is what every test below judges.
func servedBody(t *testing.T) string {
	t.Helper()
	caps, err := NewMeta().Describe(context.Background(), &pb.Empty{})
	if err != nil {
		t.Fatalf("Describe: %v", err)
	}
	body := caps.GetSchemaCue()
	if body == "" {
		t.Fatal("Describe served an EMPTY CUE body")
	}
	return body
}

// servedSchema returns the plugin's served CUE body COMPILED, with the test's table
// re-export appended. NewMeta's Describe already compiles the concatenated body
// (sdk.BuildCapabilities), so a failure here means the schema the host receives does not
// stand on its own.
func servedSchema(t *testing.T) cue.Value {
	t.Helper()
	return compile(t, servedBody(t)+vocabHarness)
}

func compile(t *testing.T, src string) cue.Value {
	t.Helper()
	v := cuecontext.New().CompileString(src)
	if err := v.Err(); err != nil {
		t.Fatalf("served schema does not compile: %v", err)
	}
	return v
}

// fieldWordList returns the sorted words of the enum an AUTHOR meets, addressed as a def
// plus one of its fields.
//
// Two things about the address and the read are measured, not stylistic:
//
//   - The field is looked up as an OPTIONAL constraint. Four of this schema's five enums sit
//     on an optional field (`syntax?`, `numlock?`, `placement?`; `#DkCategory` is a def with
//     no field at all), and a required-only lookup on `syntax?` reports "field not found",
//     which would turn this cross-check into a test of nothing.
//   - The words are collected by RECURSING through the expression. Four of the five carry a
//     literal default (`*"lua" | or(_syntaxWord)`), and `Expr()` hands back ONE nested
//     expression for that shape rather than the words — so a flat `alt.String()` fails with
//     "non-concrete value string", and `val.String()` would hand back only the default.
func fieldWordList(t *testing.T, v cue.Value, def, field string) []string {
	t.Helper()
	sels := []cue.Selector{cue.Def(def)}
	if field != "" {
		sels = append(sels, cue.Str(field).Optional())
	}
	seen := map[string]bool{}
	collectEnumWords(t, "#"+def+"."+field, v.LookupPath(cue.MakePath(sels...)), 0, seen)
	words := make([]string, 0, len(seen))
	for w := range seen {
		words = append(words, w)
	}
	sort.Strings(words)
	return words
}

func collectEnumWords(t *testing.T, label string, val cue.Value, depth int, seen map[string]bool) {
	t.Helper()
	if err := val.Err(); err != nil {
		t.Fatalf("%s does not resolve in the served schema: %v", label, err)
	}
	if depth > 16 {
		t.Fatalf("%s: the disjunction nests deeper than 16 levels", label)
	}
	if val.IncompleteKind() == cue.StringKind && val.IsConcrete() {
		w, err := val.String()
		if err != nil {
			t.Fatalf("%s: a concrete string does not stringify: %v", label, err)
		}
		seen[w] = true
		return
	}
	_, alts := val.Expr()
	if len(alts) == 0 {
		t.Fatalf("%s has no alternatives — it is not a word list", label)
	}
	for _, alt := range alts {
		collectEnumWords(t, label, alt, depth+1, seen)
	}
}

// liftedVocabulary is the harness's tables and derived lists, decoded once.
type liftedVocabulary struct {
	Tables map[string]map[string]any
	Words  map[string][]string
}

func liftVocabulary(t *testing.T, v cue.Value) liftedVocabulary {
	t.Helper()
	var out liftedVocabulary
	if err := v.LookupPath(cue.ParsePath("#TestVocabulary")).Decode(&out); err != nil {
		t.Fatalf("lift the vocabulary tables: %v", err)
	}
	return out
}

// TestAdvertisedInputDefsResolveInTheServedSchema is the def-presence gate, read from the
// REAL advertisement: NewMeta lists the four capability words and the def each one
// validates against, and a def that does not resolve in the served body makes the load gate
// silently validate against nothing.
func TestAdvertisedInputDefsResolveInTheServedSchema(t *testing.T) {
	caps, err := NewMeta().Describe(context.Background(), &pb.Empty{})
	if err != nil {
		t.Fatalf("Describe: %v", err)
	}
	v := servedSchema(t)
	want := map[string]string{
		"theme":          "#ThemeInput",
		"session":        "#SessionInput",
		"displaymanager": "#DisplayManagerInput",
		"desktopentry":   "#DesktopEntryInput",
	}
	got := map[string]string{}
	for _, pc := range caps.GetProvided() {
		got[pc.GetWord()] = pc.GetInputDef()
	}
	if len(got) != len(want) {
		t.Errorf("NewMeta advertises %d words %v, want %d", len(got), got, len(want))
	}
	for word, def := range want {
		if got[word] != def {
			t.Errorf("NewMeta advertises word %q against def %q, want %q", word, got[word], def)
		}
	}
	for word, def := range want {
		if err := v.LookupPath(cue.ParsePath(def)).Err(); err != nil {
			t.Errorf("advertised def %q for word %q does not resolve in the served schema: %v", def, word, err)
		}
	}
}

// TestS3VocabularyTablesDriveTheirEnums proves, for every vocabulary table, that (a) the
// derived word list IS the table's row set, and (b) each word is accepted where an author
// uses it while a non-word is refused. (b) is what a table that is merely DECLARED — never
// wired into the enum — fails.
func TestS3VocabularyTablesDriveTheirEnums(t *testing.T) {
	v := servedSchema(t)
	ctx := v.Context()
	lifted := liftVocabulary(t, v)
	if len(lifted.Tables) != len(vocabProbes) {
		t.Errorf("the schema carries %d vocabulary tables but %d are probed — a table nothing exercises is a table nothing keeps wired",
			len(lifted.Tables), len(vocabProbes))
	}
	for _, p := range vocabProbes {
		rows, ok := lifted.Tables[p.name]
		if !ok {
			t.Errorf("no vocabulary table %q in the served schema", p.name)
			continue
		}
		if len(rows) == 0 {
			t.Errorf("vocabulary table %q has no rows", p.name)
			continue
		}
		words, ok := lifted.Words[p.name]
		if !ok {
			t.Errorf("no derived word list for %q in the served schema", p.name)
			continue
		}
		keys := make([]string, 0, len(rows))
		for w := range rows {
			keys = append(keys, w)
		}
		sort.Strings(keys)
		sort.Strings(words)
		if fmt.Sprint(keys) != fmt.Sprint(words) {
			t.Errorf("%q's derived list %v is not its table's row set %v", p.name, words, keys)
		}
		for _, w := range keys {
			if err := v.Unify(ctx.CompileString(p.use(w))).Err(); err != nil {
				t.Errorf("%q word %q is refused where an author uses it: %v", p.name, w, err)
			}
		}
		if err := v.Unify(ctx.CompileString(p.use("zzz-not-a-word"))).Err(); err == nil {
			t.Errorf("%q accepts a non-word — the enum is not the table", p.name)
		}
	}
}

// TestServedEnumsAreTheDerivedListsCrossCheck reads each enum through the FIELD an author
// writes, not through the table: the table and the enum can each be right while the field
// still cites an extra literal. It compares the field's own alternative set against the
// table's derived list, so a word left in the field beyond the table fails HERE.
func TestServedEnumsAreTheDerivedListsCrossCheck(t *testing.T) {
	v := servedSchema(t)
	lifted := liftVocabulary(t, v)
	for _, tc := range []struct{ def, field, table string }{
		{"SessionInput", "syntax", "syntax"},
		{"DisplayManagerInput", "manager", "manager"},
		{"DisplayManagerInput", "numlock", "numlock"},
		{"DkWindow", "placement", "placement"},
		{"DkCategory", "", "category"},
	} {
		fieldWords := fieldWordList(t, v, tc.def, tc.field)
		tableWords := append([]string(nil), lifted.Words[tc.table]...)
		sort.Strings(tableWords)
		if fmt.Sprint(fieldWords) != fmt.Sprint(tableWords) {
			t.Errorf("#%s.%s accepts %v but the _%s table's word list is %v — the field cites its own literal instead of the derivation",
				tc.def, tc.field, fieldWords, tc.table, tableWords)
		}
	}
}

// TestDkCategoriesAreTheFreedesktopMains pins the ONE set whose contract is EXTERNAL. The
// schema states the claim in its own words — "the freedesktop registered MAIN categories,
// verbatim" — and an S3 table that shrinks stays self-consistent, so nothing else in this
// file can see a lost row: the table, the derived list and the enum would all agree on the
// smaller set. The registry is the authority, not this file, which is why restating it here
// is a contract pin rather than a second copy. The failure it prevents is the one the schema
// names: a typo or a dropped row errors NOWHERE and the entry silently vanishes from every menu.
func TestDkCategoriesAreTheFreedesktopMains(t *testing.T) {
	freedesktopMains := []string{
		"AudioVideo", "Audio", "Video", "Development", "Education", "Game", "Graphics",
		"Network", "Office", "Science", "Settings", "System", "Utility",
	}
	got := fieldWordList(t, servedSchema(t), "DkCategory", "")
	sort.Strings(freedesktopMains)
	if fmt.Sprint(got) != fmt.Sprint(freedesktopMains) {
		t.Errorf("#DkCategory accepts %v, but the freedesktop MAIN categories are %v — the entry that uses a missing one vanishes from every menu with no error anywhere",
			got, freedesktopMains)
	}
}

// TestVocabularyTablesKeepMoreThanOneRow is the coarse floor against a table that lost its
// tail: an S3 table that shrinks is self-consistent, so no cross-check in this file can see
// it, and every set here is at least three words. It also catches a READER that regressed to
// `String()`, which hands back the literal default and so reads exactly one word — the
// failure the omarchy precedent measured on a `*"cli" | or(_methodWord)` enum.
func TestVocabularyTablesKeepMoreThanOneRow(t *testing.T) {
	v := servedSchema(t)
	lifted := liftVocabulary(t, v)
	for _, p := range vocabProbes {
		if got := len(lifted.Words[p.name]); got < 2 {
			t.Errorf("the _%s table's derived list reads %d word(s) %v — a table that lost its tail, or a reader that returned the default",
				p.name, got, lifted.Words[p.name])
		}
	}
}

// TestEnumsFollowTheirTables is the only way a runtime test can see the S3 requirement itself —
// that the enum DERIVES from the table rather than restating it.
//
// Equality cannot see it: a field written `numlock?: *"none" | "on" | "off"` accepts exactly the
// table's three words, so every set- and probe-based test above passes over a table nothing
// cites. So this test MUTATES the table instead: a top-level `_x` struct is OPEN, so a second
// declaration of the same label in the appended harness ADDS a row, the derived list's
// comprehension recomputes over the merged table, and the field an author meets must then
// accept the injected word. An enum that cites the derivation follows; an enum written out by
// hand does not, and the row it lost is invisible to every other test in this file.
func TestEnumsFollowTheirTables(t *testing.T) {
	const injected = "probe-extra-word"
	for _, p := range vocabProbes {
		src := servedBody(t) + vocabHarness + fmt.Sprintf("\n_%s: {%q: {}}\n", p.name, injected)
		v := compile(t, src)
		ctx := v.Context()
		if err := v.Unify(ctx.CompileString(p.use(injected))).Err(); err != nil {
			t.Errorf("the %q table gained a row but the enum an author meets refuses it: %v — the field does not derive from _%sWord",
				p.name, err, p.name)
		}
	}
}
