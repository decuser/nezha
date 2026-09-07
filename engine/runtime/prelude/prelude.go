// Package prelude embeds the Nezha prelude source and documentation.
package prelude

import (
	_ "embed"
)

//go:embed prelude.nez
var Source string

//go:embed prelude.help
var HelpSource string

//go:embed prelude.doc
var DocSource string
