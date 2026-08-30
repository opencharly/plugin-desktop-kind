// Package desktopkind is the importable form of the charly DESKTOP-SURFACE kinds: `theme`,
// `session`, `displaymanager` and `desktopentry`.
//
// These four describe a graphical session the way `init:`/`service:` describes a supervised
// process: the KIND is a renderer VOCABULARY and the authored entity is DATA. One theme
// entity renders into a Hyprland config, a sway config and a GTK settings file without
// naming any of them.
//
// A KIND provider dispatches via the pb Invoke(OpLoad) envelope: decode the authored entity
// from op.Params into the core spec type and re-marshal as canonical JSON; the host lands it
// in uf.PluginKinds[<word>][<name>] — the FLAT opaque-body path. All four are
// Structural:false: none nests a deploy resource member.
//
// The values are SELF-CONTAINED (scalars, small records and inline templates — nothing rich
// or core-referencing like #Candy/#Vm), so they ride op.Params and are validated against this
// plugin's served self-contained #ThemeInput/#SessionInput/#DisplayManagerInput/
// #DesktopEntryInput schema (validateAuthoredPluginInput, the flat-kind load gate).
//
// Served OUT-OF-PROCESS by the cmd/serve shim: these words are consumed only at
// candy-compile time, when project plugins are already loaded, so nothing in the core parse
// path needs them and no charly change is required to adopt them.
//
// WHY THESE ARE KINDS AND NOT `write:` STEPS. Each could be hand-rolled as a write: with a
// heredoc. What a kind buys is a VALIDATOR and CROSS-REFERENCES that a write: cannot express:
//
//   - theme: a template naming a token the theme does not define is rejected at LOAD rather
//     than rendering an empty colour — a subtly wrong desktop instead of a failure.
//   - displaymanager: `session` must match a session entity's session_desktop.id. An
//     autologin pointing at a session file nothing installed is a black screen at boot with
//     an empty journal.
//   - desktopentry: `exec` and `url` are mutually exclusive, and startup_wm_class/window are
//     read BY the session renderer, so one entity feeds both the applications menu and the
//     compositor's window rules.
package desktopkind

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/opencharly/sdk"
	pb "github.com/opencharly/spec/proto"
	"github.com/opencharly/spec/spec"
)

//go:embed schema/*.cue
var schemaFS embed.FS

const calver = "2026.242.0930"

// NewProvider returns the desktop-kind provider for in-proc registration or out-of-proc serving.
func NewProvider() pb.ProviderServer { return &provider{} }

// NewMeta ships the four flat kind capabilities + their served self-contained schemas
// (fixedMeta.Describe compiles the embedded schemaFS's "schema" dir standalone).
func NewMeta() pb.PluginMetaServer {
	return sdk.NewMeta(calver,
		[]sdk.ProvidedCapability{
			{Class: "kind", Word: "theme", InputDef: "#ThemeInput"},
			{Class: "kind", Word: "session", InputDef: "#SessionInput"},
			{Class: "kind", Word: "displaymanager", InputDef: "#DisplayManagerInput"},
			{Class: "kind", Word: "desktopentry", InputDef: "#DesktopEntryInput"},
		},
		schemaFS)
}

type provider struct{ pb.UnimplementedProviderServer }

// Invoke handles OpLoad and OpValidate.
//
// OpLoad decodes the authored entity into its core spec type and returns it re-marshalled as
// canonical JSON — the host has already validated it against the served #*Input schema.
//
// OpValidate runs the checks CUE cannot express. That is the whole reason these are kinds
// rather than write: steps, so it is not optional decoration.
func (p provider) Invoke(_ context.Context, req *pb.InvokeRequest) (*pb.InvokeReply, error) {
	switch req.GetOp() {
	case sdk.OpLoad, sdk.OpValidate:
	default:
		return nil, fmt.Errorf("desktop kind: unsupported op %q (want %q or %q)", req.GetOp(), sdk.OpLoad, sdk.OpValidate)
	}
	if len(req.GetParamsJson()) == 0 {
		return nil, errors.New("desktop kind: load requires a CUE input payload")
	}

	word := req.GetReserved()
	var out any
	switch word {
	case "theme":
		out = &spec.Theme{}
	case "session":
		out = &spec.Session{}
	case "displaymanager":
		out = &spec.DisplayManager{}
	case "desktopentry":
		out = &spec.DesktopEntry{}
	default:
		return nil, fmt.Errorf("desktop kind: unsupported word %q", word)
	}
	if err := json.Unmarshal(req.GetParamsJson(), out); err != nil {
		return nil, fmt.Errorf("desktop kind %q: decode entity: %w", word, err)
	}
	if err := validateEntity(word, out); err != nil {
		return nil, fmt.Errorf("desktop kind %q: %w", word, err)
	}
	res, err := json.Marshal(out)
	if err != nil {
		return nil, fmt.Errorf("desktop kind %q: marshal entity: %w", word, err)
	}
	return &pb.InvokeReply{ResultJson: res}, nil
}

// validateEntity runs the per-word invariants that a closed CUE struct cannot state.
func validateEntity(word string, v any) error {
	switch e := v.(type) {
	case *spec.Theme:
		return validateTheme(e)
	case *spec.DesktopEntry:
		return validateDesktopEntry(e)
	case *spec.Session, *spec.DisplayManager:
		// Both carry cross-ENTITY invariants — a displaymanager's `session` must match a
		// session's session_desktop.id — which need the whole project, not one body. They
		// are enforced by the compile-time consumer that can see every entity, not here:
		// this provider is handed exactly one body and would have to guess at the rest.
		return nil
	}
	return nil
}

// themeTokenRef finds {{.Token.<name>}} references in a template.
var themeTokenRef = regexp.MustCompile(`\{\{\s*\.Token\.([A-Za-z0-9_]+)\s*\}\}`)

// validateTheme rejects a render template that names a token this theme does not define.
//
// This is the check that makes `theme` worth being a kind. An undefined token renders as an
// EMPTY STRING, so the failure is a terminal with no background colour or a bar with an
// invisible accent — a subtly wrong desktop that no build step reports. Catching it at load
// turns a silent visual defect into a named error.
func validateTheme(t *spec.Theme) error {
	defined := definedTokens(t.Token)
	var bad []string
	for _, r := range t.Render {
		for _, m := range themeTokenRef.FindAllStringSubmatch(r.Content, -1) {
			if !defined[m[1]] {
				bad = append(bad, fmt.Sprintf("%s (%s) references .Token.%s", r.Path, r.App, m[1]))
			}
		}
	}
	if len(bad) > 0 {
		return fmt.Errorf("render template references a token this theme does not define — it would render an empty colour:\n  %s",
			strings.Join(bad, "\n  "))
	}
	return nil
}

// definedTokens reports which token names carry a value. A token left unset is NOT defined:
// the point of the check is that a template must not reach one.
func definedTokens(tk spec.ThemeTokens) map[string]bool {
	d := map[string]bool{}
	// val is spec.HexColor, a named string type — the generated shape for the closed
	// #HexColor def. Taking it as that type rather than plain string keeps the call sites
	// free of conversions and makes a future non-colour token a compile error here.
	set := func(name string, val spec.HexColor) {
		if val != "" {
			d[name] = true
		}
	}
	set("accent", tk.Accent)
	set("foreground", tk.Foreground)
	set("background", tk.Background)
	set("cursor", tk.Cursor)
	set("selection_foreground", tk.SelectionForeground)
	set("selection_background", tk.SelectionBackground)
	for i, v := range []spec.HexColor{
		tk.Color0, tk.Color1, tk.Color2, tk.Color3, tk.Color4, tk.Color5, tk.Color6, tk.Color7,
		tk.Color8, tk.Color9, tk.Color10, tk.Color11, tk.Color12, tk.Color13, tk.Color14, tk.Color15,
	} {
		set(fmt.Sprintf("color%d", i), v)
	}
	return d
}

// validateDesktopEntry enforces exec XOR url.
//
// CUE can express "at most one of these" only by splitting the def into a disjunction, which
// would double every other field and produce an error naming the whole shape rather than the
// two fields at fault. Stated here so the message can say which one to remove.
func validateDesktopEntry(e *spec.DesktopEntry) error {
	switch {
	case e.Exec == "" && e.URL == "":
		return errors.New("declares neither exec nor url — a desktop entry with no command does nothing when launched")
	case e.Exec != "" && e.URL != "":
		return errors.New("declares BOTH exec and url — they are alternatives: url renders the browser --app=<url> form, exec runs a command. Remove one")
	}
	if e.URL == "" && len(e.BrowserArgs) > 0 {
		return errors.New("declares browser_arg without url — browser_arg only applies to the --app=<url> form")
	}
	// spec.DesktopIcon is a VALUE, not a pointer, so "no icon declared" is the zero struct
	// rather than nil — an absent icon is legal, and only a HALF-declared one is not.
	if e.Icon.Name != "" && e.Icon.Source != "" {
		return errors.New("icon declares both name and source — name is a stock icon, source is a candy-relative file. Remove one")
	}
	return nil
}
