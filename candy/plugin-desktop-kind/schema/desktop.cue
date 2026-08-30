package schema

// SELF-CONTAINED input schema for the four desktop-surface kinds.
//
// This mirrors spec's #Theme / #Session / #DisplayManager / #DesktopEntry, and the
// duplication is SANCTIONED, not an R3 violation: this file must compile STANDALONE (the
// host serves it over Describe and validates an authored body against it before dispatch),
// while spec's defs must generate Go. Neither can be the other. plugin-harness-kind's
// schema/skill.cue vs spec/schema/skill.cue is the shipped precedent.
//
// What keeps carrying two copies safe is a parity test, the same way plugin-distro's
// #DistroInput has one after it silently drifted behind spec.

// ─── theme ────────────────────────────────────────────────────────────────────────────

#ThemeInput: {
	name!: string & =~"^[a-z0-9]+(-[a-z0-9]+)*$"
	variant?: *"dark" | "light"
	token!: #DkThemeTokens
	font?:         string & !=""
	cursor_theme?: string & !=""
	icon_theme?:   string & !=""
	background?: [...string & !=""]
	render?: [...#DkRender]
}

// The token vocabulary is CLOSED. An undefined token renders as an EMPTY STRING — a
// terminal with no background, a bar with an invisible accent — which no build step reports.
// The provider's OpValidate additionally rejects a render template naming a token this
// theme does not define, which CUE cannot express because it requires reading the template.
#DkThemeTokens: {
	accent!:     #DkHexColor
	foreground!: #DkHexColor
	background!: #DkHexColor

	cursor?:               #DkHexColor
	selection_foreground?: #DkHexColor
	selection_background?: #DkHexColor

	color0?:  #DkHexColor
	color1?:  #DkHexColor
	color2?:  #DkHexColor
	color3?:  #DkHexColor
	color4?:  #DkHexColor
	color5?:  #DkHexColor
	color6?:  #DkHexColor
	color7?:  #DkHexColor
	color8?:  #DkHexColor
	color9?:  #DkHexColor
	color10?: #DkHexColor
	color11?: #DkHexColor
	color12?: #DkHexColor
	color13?: #DkHexColor
	color14?: #DkHexColor
	color15?: #DkHexColor
}

// #rrggbb only. Short form and alpha have no portable spelling across the formats a theme
// renders into, so accepting them would force the renderer to normalise.
#DkHexColor: string & =~"^#[0-9a-fA-F]{6}$"

#DkRender: {
	app!:     string & !=""
	path!:    string & !=""
	content!: string
	mode?:    string & =~"^0[0-7]{3,4}$"
	scope?: *"user" | "system"
	distro?: [...(string & !="")]
}

// ─── session ──────────────────────────────────────────────────────────────────────────

#SessionInput: {
	compositor!: string & !=""
	syntax?: *"lua" | "ini" | "sway" | "xml" | "yaml"
	config_path_template!: string & !=""
	model?: *"assembly" | "file_set"

	monitor_template?: string
	bind_template?:    string
	input_template?:   string
	exec_template?:    string
	env_template?:     string
	rule_template?:    string
	include_template?: string

	extra_file?: [...#DkRender]
	session_desktop?: #DkSessionDesktop
	theme_render?: [...string & !=""]
}

#DkSessionDesktop: {
	id!:   string & =~"^[a-z0-9]+(-[a-z0-9]+)*$"
	name!: string & !=""
	exec!: string & !=""
	type?: *"Application" | "XSession"
}

// ─── displaymanager ───────────────────────────────────────────────────────────────────

#DisplayManagerInput: {
	manager: *"sddm" | "greetd" | "gdm" | "ly"
	// Cross-checked against a session's session_desktop.id by the compile-time consumer,
	// which can see every entity; this provider is handed one body and would have to guess
	// at the rest.
	session!: string & =~"^[a-z0-9]+(-[a-z0-9]+)*$"
	autologin?: #DkAutoLogin
	numlock?: *"none" | "on" | "off"
	theme?: string & !=""
	config?: [...#DkRender]
	unit?: string & !=""
}

#DkAutoLogin: {
	user!:    string & !=""
	relogin?: bool
}

// ─── desktopentry ─────────────────────────────────────────────────────────────────────

#DesktopEntryInput: {
	entry_name!: string & !=""
	// exec and url are alternatives. Enforced by the provider's OpValidate rather than by a
	// CUE disjunction: a disjunction would double every other field and produce an error
	// naming the whole shape instead of the two fields at fault.
	exec?: string & !=""
	url?:  string & =~"^https?://"
	browser_arg?: [...string & !=""]
	icon?: #DkIcon
	categories?: [...#DkCategory]
	mime_type?: [...string & !=""]
	startup_wm_class?: string & !=""
	window?:           #DkWindow
	terminal?:         bool
	comment?:          string
}

#DkIcon: {
	name?:   string & !=""
	source?: string & !=""
}

#DkWindow: {
	placement?: *"default" | "float" | "tile" | "fullscreen" | "maximize"
	workspace?: string & !=""
	size?:      string & =~"^[0-9]+x[0-9]+$"
}

// The freedesktop registered MAIN categories, verbatim. Closed because a typo here errors
// NOWHERE — the entry silently vanishes from every menu.
#DkCategory: "AudioVideo" | "Audio" | "Video" | "Development" | "Education" |
	"Game" | "Graphics" | "Network" | "Office" | "Science" | "Settings" |
	"System" | "Utility"
