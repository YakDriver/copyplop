// Copyright IBM Corp. 2014, 2025
// SPDX-License-Identifier: MPL-2.0

package copyright

type Issue struct {
	File    string
	Problem string
}

type FixResult struct {
	Fixed int
	Added int
}
