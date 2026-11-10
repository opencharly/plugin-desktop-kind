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
	// S3: the words are rows of the `_syntax` table at the foot of this file, and the
	// enum is `or()` of that table's derived word list. The `*"lua"` default stays
	// written inline — S3 permits a literal default, and the default is a VALUE
	// choice the word table does not decide.
	syntax?: *"lua" | or(_syntaxWord)
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
	// S3: the words are rows of the `_manager` table at the foot of this file; the
	// `*"sddm"` default stays inline (a literal default is S3's one inline enum).
	manager: *"sddm" | or(_managerWord)
	// Cross-checked against a session's session_desktop.id by the compile-time consumer,
	// which can see every entity; this provider is handed one body and would have to guess
	// at the rest.
	session!: string & =~"^[a-z0-9]+(-[a-z0-9]+)*$"
	autologin?: #DkAutoLogin
	numlock?: *"none" | or(_numlockWord)
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
	// S3: the words are rows of the `_placement` table at the foot of this file; the
	// `*"default"` default stays inline.
	placement?: *"default" | or(_placementWord)
	workspace?: string & !=""
	size?:      string & =~"^[0-9]+x[0-9]+$"
}

// The freedesktop registered MAIN categories, verbatim. Closed because a typo here errors
// NOWHERE — the entry silently vanishes from every menu. S3: the words are rows of the
// `_category` table at the foot of this file.
#DkCategory: or(_categoryWord)

// ─── vocabulary tables (S3) ────────────────────────────────────────────────────────────
// Every closed set of words is a table keyed by word, and its word list is DERIVED from
// the table and used as `or()` of that list (§10.1 S3, D-PAT-3). The table and the enum it
// feeds therefore cannot disagree, and a new word is ONE row here.
//
// `@vocab` names the owner (§3.1). This plugin serves FOUR capability words rather than
// one, so the owner is the word whose input def cites the set — `session`,
// `displaymanager` or `desktopentry` — not the repo name, which no capability word spells.
//
// A two-value literal default stays written inline (`variant?: *"dark" | "light"`,
// `scope?: *"user" | "system"`, `model?: *"assembly" | "file_set"`,
// `type?: *"Application" | "XSession"`): S3 states that the two-value default is the only
// inline enum, and a two-word pair spelled where the author meets it reads better than a
// table a hundred lines away.
_syntax: {
	lua: {}
	ini: {}
	sway: {}
	xml: {}
	yaml: {}
} @vocab(session)
_syntaxWord: [for w, _ in _syntax {w}]

_manager: {
	sddm: {}
	greetd: {}
	gdm: {}
	ly: {}
} @vocab(displaymanager)
_managerWord: [for w, _ in _manager {w}]

_numlock: {
	none: {}
	on: {}
	off: {}
} @vocab(displaymanager)
_numlockWord: [for w, _ in _numlock {w}]

_placement: {
	default: {}
	float: {}
	tile: {}
	fullscreen: {}
	maximize: {}
} @vocab(desktopentry)
_placementWord: [for w, _ in _placement {w}]

// The MAIN categories the comment above `#DkCategory` names — the registry's additional
// and reserved categories are not part of that def and are not added here.
_category: {
	AudioVideo: {}
	Audio: {}
	Video: {}
	Development: {}
	Education: {}
	Game: {}
	Graphics: {}
	Network: {}
	Office: {}
	Science: {}
	Settings: {}
	System: {}
	Utility: {}
} @vocab(desktopentry)
_categoryWord: [for w, _ in _category {w}]
