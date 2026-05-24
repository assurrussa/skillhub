package targets

import _ "embed"

// TargetsTSV is the built-in install target registry used by standalone binaries.
//
//go:embed targets.tsv
var TargetsTSV string
