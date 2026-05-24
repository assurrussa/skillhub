package defaults

import _ "embed"

// SourcesTSV is the built-in source preset registry used by standalone binaries.
//
//go:embed sources.tsv
var SourcesTSV string
