// Command serve is the OUT-OF-PROCESS entrypoint for the theme/session/displaymanager/
// desktopentry desktop-kind plugin: a thin shim serving the importable provider over
// go-plugin gRPC.
//
// Out-of-process is the RIGHT placement for these four, unlike the harness kinds. They are
// consumed only at candy-compile time, when project plugins are already loaded — nothing in
// the core parse path needs the words recognised — so adopting them requires no charly
// change at all.
package main

import (
	desktopkind "github.com/opencharly/plugin-desktop-kind/candy/plugin-desktop-kind"
	"github.com/opencharly/sdk"
)

func main() { sdk.Serve(desktopkind.NewProvider(), desktopkind.NewMeta()) }
