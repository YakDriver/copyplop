// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: MPL-2.0

package version

import (
	_ "embed"
	"strings"
)

//go:embed VERSION
var versionFile string

// Version returns the current version of copyplop
func Version() string {
	return strings.TrimSpace(versionFile)
}
