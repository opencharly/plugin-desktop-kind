# plugin-desktop-kind

The desktop-surface kinds for OpenCharly — `theme`, `session`,
`displaymanager`, and `desktopentry` — served out-of-process
(`kind:theme`, `kind:session`, `kind:displaymanager`, `kind:desktopentry`).

Four first-class entities describing a graphical session the way `init:` /
`service:` describes a supervised process: the KIND is a renderer VOCABULARY and
the authored entity is DATA. One theme entity renders into a Hyprland config, a
sway config, and a GTK settings file without naming any of them.

## What it provides

| Capability | Surface |
|---|---|
| `kind:theme` | the `theme:` entity — colour tokens, fonts, cursor/icon themes, render targets |
| `kind:session` | the `session:` entity — a compositor + its config templates |
| `kind:displaymanager` | the `displaymanager:` entity — the login manager + autologin |
| `kind:desktopentry` | the `desktopentry:` entity — a freedesktop application entry |

All four are FLAT (`Structural:false` — no deploy members): their self-contained
values ride the step op params, are validated at load against this plugin's
served `#ThemeInput` / `#SessionInput` / `#DisplayManagerInput` /
`#DesktopEntryInput` schemas, and fold into `uf.PluginKinds[<word>][<name>]` as
opaque JSON.

## Why kinds and not `write:` steps

Each could be a `write:` with a heredoc. What a kind buys is a VALIDATOR and
cross-references a `write:` cannot express:

- **theme** — a render template naming a token the theme does not define is
  rejected at LOAD rather than rendering an empty colour (a subtly wrong desktop
  instead of a failure).
- **displaymanager** — `session` must match a session entity's
  `session_desktop.id`; an autologin pointing at a session file nothing installed
  is a black screen at boot.
- **desktopentry** — `exec` and `url` are mutually exclusive, and
  `startup_wm_class` / `window` are read BY the session renderer, so one entity
  feeds both the applications menu and the compositor's window rules.

The placement is out-of-process by design, not limitation: these words are
consumed only at candy-compile time, when project plugins are already loaded, so
nothing in the core parse path needs them.

## How to use it

Compose the plugin candy in a project's `candy:` list:

```yaml
- '@github.com/opencharly/plugin-desktop-kind/candy/plugin-desktop-kind:<tag>'
```

Then author a `theme:` entity:

```yaml
my-theme:
    theme:
        name: my-theme
        variant: dark
        token:
            accent: "#88c0d0"
            foreground: "#d8dee9"
            background: "#2e3440"
        render:
            - app: hyprland
              path: ~/.config/hypr/theme.conf
              content: "accent = {{.accent}}"
```

## Layout

- `candy/plugin-desktop-kind/` — the plugin module: `plugin.go` (the kind
  provider for all four words + `NewProvider()` / `NewMeta()`), `schema/desktop.cue`
  (the self-contained `#ThemeInput` / `#SessionInput` / `#DisplayManagerInput` /
  `#DesktopEntryInput`), `plugin_test.go` (drives the real `Invoke` for all four
  words + both validators), `schema_parity_test.go` (asserts the served
  `#*Input` defs still match spec's authored surface), `cmd/serve/main.go`.
- `charly.yml` — the root project manifest (`discover: candy`).
- `.github/workflows/tag-on-merge.yml` — CalVer tag + `CHANGELOG/` on merge.
- `README.md` — user overview only.

## Related

- Owning skill: `/charly-image:layer` — the candy authoring surface the desktop
  kinds extend. This candy carries no `skill:` entity of its own; the gap is
  tracked in
  [opencharly/opencharly#291](https://github.com/opencharly/opencharly/issues/291).
- `/charly-internals:plugin` — the plugin/provider model, including the `kind`
  provider class.
- [`opencharly/charly`](https://github.com/opencharly/charly) — the charly CLI.
