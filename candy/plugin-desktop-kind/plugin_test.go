package desktopkind

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/opencharly/sdk"
	pb "github.com/opencharly/spec/proto"
)

func invoke(t *testing.T, op, word, body string) (*pb.InvokeReply, error) {
	t.Helper()
	return NewProvider().Invoke(context.Background(), &pb.InvokeRequest{
		Op:         op,
		Reserved:   word,
		ParamsJson: []byte(body),
	})
}

// The four words must be served, and each must round-trip an authored body.
func TestInvoke_LoadsAllFourWords(t *testing.T) {
	cases := map[string]string{
		"theme": `{"name":"tokyo-night","token":{"accent":"#7aa2f7","foreground":"#c0caf5","background":"#1a1b26"}}`,
		"session": `{"compositor":"Hyprland","config_path_template":"${HOME}/.config/hypr/hyprland.lua",
			"session_desktop":{"id":"hyprland","name":"Hyprland","exec":"uwsm start hyprland.desktop"}}`,
		"displaymanager": `{"manager":"sddm","session":"hyprland","autologin":{"user":"user"}}`,
		"desktopentry":   `{"entry_name":"Disk Usage","exec":"dua i /","categories":["System"]}`,
	}
	for word, body := range cases {
		t.Run(word, func(t *testing.T) {
			reply, err := invoke(t, sdk.OpLoad, word, body)
			if err != nil {
				t.Fatalf("load %s: %v", word, err)
			}
			var got map[string]any
			if err := json.Unmarshal(reply.GetResultJson(), &got); err != nil {
				t.Fatalf("result is not JSON: %v", err)
			}
			if len(got) == 0 {
				t.Fatalf("load %s returned an empty entity", word)
			}
		})
	}
}

// THE reason `theme` is a kind: a template naming a token the theme does not define renders
// an EMPTY STRING — a terminal with no background, a bar with an invisible accent — which no
// build step reports. This must be a load error.
func TestInvoke_ThemeRejectsUndefinedToken(t *testing.T) {
	body := `{"name":"x","token":{"accent":"#ffffff","foreground":"#ffffff","background":"#000000"},
		"render":[{"app":"foot","path":"/tmp/c","content":"sel={{.Token.selection_background}}"}]}`
	_, err := invoke(t, sdk.OpLoad, "theme", body)
	if err == nil {
		t.Fatal("a render template naming an undefined token must be rejected")
	}
	for _, want := range []string{"selection_background", "empty colour", "/tmp/c"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error should name %q so the author can find it; got: %v", want, err)
		}
	}
}

// The same template is fine once the token IS defined — the check discriminates rather than
// rejecting every template that mentions a token.
func TestInvoke_ThemeAcceptsDefinedToken(t *testing.T) {
	body := `{"name":"x","token":{"accent":"#ffffff","foreground":"#ffffff","background":"#000000",
		"selection_background":"#33467c"},
		"render":[{"app":"foot","path":"/tmp/c","content":"sel={{.Token.selection_background}}"}]}`
	if _, err := invoke(t, sdk.OpLoad, "theme", body); err != nil {
		t.Fatalf("a template naming a DEFINED token must be accepted: %v", err)
	}
}

// A token present in the schema but left UNSET is not defined. This is the subtle half: the
// field exists on the struct either way, so a check that looked at the type rather than the
// value would pass here and let the empty colour through.
func TestInvoke_ThemeUnsetTokenIsNotDefined(t *testing.T) {
	body := `{"name":"x","token":{"accent":"#ffffff","foreground":"#ffffff","background":"#000000","color3":""},
		"render":[{"app":"btop","path":"/tmp/c","content":"c={{.Token.color3}}"}]}`
	if _, err := invoke(t, sdk.OpLoad, "theme", body); err == nil {
		t.Fatal("a token present but EMPTY must count as undefined — it renders an empty colour")
	}
}

// exec XOR url. CUE can express this only as a disjunction that doubles every other field
// and reports the whole shape; here the error can name the two fields at fault.
func TestInvoke_DesktopEntryExecXorURL(t *testing.T) {
	cases := []struct{ name, body, want string }{
		{"neither", `{"entry_name":"X"}`, "neither exec nor url"},
		{"both", `{"entry_name":"X","exec":"x","url":"https://e.invalid"}`, "BOTH exec and url"},
		{"browser_arg without url", `{"entry_name":"X","exec":"x","browser_arg":["--new-window"]}`, "browser_arg without url"},
		{"icon with both name and source", `{"entry_name":"X","exec":"x","icon":{"name":"a","source":"b"}}`, "both name and source"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := invoke(t, sdk.OpLoad, "desktopentry", tc.body)
			if err == nil {
				t.Fatalf("expected rejection for %s", tc.name)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("error should contain %q, got: %v", tc.want, err)
			}
		})
	}
	// An entry with NO icon at all is legal — only a half-declared one is not.
	if _, err := invoke(t, sdk.OpLoad, "desktopentry", `{"entry_name":"X","exec":"x"}`); err != nil {
		t.Fatalf("an entry with no icon must be accepted: %v", err)
	}
}

func TestInvoke_Rejects(t *testing.T) {
	if _, err := invoke(t, sdk.OpLoad, "wallpaper", `{}`); err == nil {
		t.Error("an unsupported word must be rejected")
	}
	if _, err := invoke(t, sdk.OpLoad, "theme", ``); err == nil {
		t.Error("an empty payload must be rejected")
	}
	if _, err := invoke(t, "op:deploy", "theme", `{"name":"x"}`); err == nil {
		t.Error("an unsupported op must be rejected")
	}
}

// The served capabilities are the contract the host reads to know these words exist. A
// mismatch between the advertised InputDef and the schema's actual def name makes the load
// gate silently validate against nothing.
func TestMeta_AdvertisesFourKindsWithMatchingDefs(t *testing.T) {
	schemaSrc, err := schemaFS.ReadFile("schema/desktop.cue")
	if err != nil {
		t.Fatalf("reading the embedded schema: %v", err)
	}
	for _, def := range []string{"#ThemeInput", "#SessionInput", "#DisplayManagerInput", "#DesktopEntryInput"} {
		if !strings.Contains(string(schemaSrc), def+": {") {
			t.Errorf("the served schema does not define %s, but NewMeta advertises it", def)
		}
	}
}
