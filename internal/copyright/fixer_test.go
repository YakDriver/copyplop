package copyright

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/YakDriver/copyplop/internal/config"
)

func TestFixer_fixFile(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &config.Config{
		Copyright: config.Copyright{
			Holder:      "IBM Corp.",
			StartYear:   2014,
			CurrentYear: 2025,
			Format:      "Copyright {{.Holder}} {{.StartYear}}, {{.CurrentYear}}",
		},
		License: config.License{
			Enabled:    true,
			Identifier: "MPL-2.0",
			Format:     "SPDX-License-Identifier: {{.Identifier}}",
		},
		Files: config.Files{
			CommentStyles: map[string]string{".go": "//", ".sh": "#"},
		},
		Detection: config.Detection{
			SkipGenerated:     true,
			GeneratedPatterns: []string{"Code generated"},
			ReplacePatterns:   []string{"Copyright.*HashiCorp"},
		},
		ThirdParty: config.ThirdParty{
			Action:   "above",
			Patterns: []string{"Copyright.*Oracle"},
		},
	}

	fixer := NewFixer(cfg)

	tests := []struct {
		name           string
		filename       string
		input          string
		expectedOutput string
		shouldFix      bool
	}{
		{
			name:     "replace HashiCorp header",
			filename: "replace.go",
			input: `// Copyright HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package main`,
			expectedOutput: `// Copyright IBM Corp. 2014, 2025
// SPDX-License-Identifier: MPL-2.0

package main`,
			shouldFix: true,
		},
		{
			name:     "add missing header",
			filename: "missing.go",
			input: `package main

func main() {}`,
			expectedOutput: `// Copyright IBM Corp. 2014, 2025
// SPDX-License-Identifier: MPL-2.0

package main

func main() {}`,
			shouldFix: true,
		},
		{
			name:     "third-party copyright - above action",
			filename: "oracle.go",
			input: `//Copyright (c) 2025, Oracle and/or its affiliates. All rights reserved.

package main`,
			expectedOutput: `// Copyright IBM Corp. 2014, 2025
// SPDX-License-Identifier: MPL-2.0
//Copyright (c) 2025, Oracle and/or its affiliates. All rights reserved.

package main`,
			shouldFix: true,
		},
		{
			name:     "already correct header",
			filename: "correct.go",
			input: `// Copyright IBM Corp. 2014, 2025
// SPDX-License-Identifier: MPL-2.0

package main`,
			expectedOutput: `// Copyright IBM Corp. 2014, 2025
// SPDX-License-Identifier: MPL-2.0

package main`,
			shouldFix: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filePath := filepath.Join(tmpDir, tt.filename)
			err := os.WriteFile(filePath, []byte(tt.input), 0644)
			if err != nil {
				t.Fatalf("Failed to write test file: %v", err)
			}

			fixed := fixer.fixFile(filePath)
			if fixed != tt.shouldFix {
				t.Errorf("fixFile() fixed = %v, want %v", fixed, tt.shouldFix)
			}

			content, err := os.ReadFile(filePath)
			if err != nil {
				t.Fatalf("Failed to read file after fix: %v", err)
			}

			result := string(content)
			if result != tt.expectedOutput {
				t.Errorf("File content mismatch:\nGot:\n%s\nWant:\n%s", result, tt.expectedOutput)
			}
		})
	}
}

func TestThirdPartyActions(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name           string
		action         string
		input          string
		expectedOutput string
	}{
		{
			name:   "below action",
			action: "below",
			input: `//Copyright (c) 2025, Oracle and/or its affiliates. All rights reserved.

package main`,
			expectedOutput: `//Copyright (c) 2025, Oracle and/or its affiliates. All rights reserved.
// Copyright IBM Corp. 2014, 2025
// SPDX-License-Identifier: MPL-2.0

package main`,
		},
		{
			name:   "replace action",
			action: "replace",
			input: `//Copyright (c) 2025, Oracle and/or its affiliates. All rights reserved.

package main`,
			expectedOutput: `// Copyright IBM Corp. 2014, 2025
// SPDX-License-Identifier: MPL-2.0

package main`,
		},
		{
			name:   "leave action",
			action: "leave",
			input: `//Copyright (c) 2025, Oracle and/or its affiliates. All rights reserved.

package main`,
			expectedOutput: `// Copyright IBM Corp. 2014, 2025
// SPDX-License-Identifier: MPL-2.0

//Copyright (c) 2025, Oracle and/or its affiliates. All rights reserved.

package main`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				Copyright: config.Copyright{
					Holder:      "IBM Corp.",
					StartYear:   2014,
					CurrentYear: 2025,
					Format:      "Copyright {{.Holder}} {{.StartYear}}, {{.CurrentYear}}",
				},
				License: config.License{
					Enabled:    true,
					Identifier: "MPL-2.0",
					Format:     "SPDX-License-Identifier: {{.Identifier}}",
				},
				Files: config.Files{
					CommentStyles: map[string]string{".go": "//"},
				},
				ThirdParty: config.ThirdParty{
					Action:   tt.action,
					Patterns: []string{"Copyright.*Oracle"},
				},
			}

			fixer := NewFixer(cfg)
			filePath := filepath.Join(tmpDir, "test.go")

			err := os.WriteFile(filePath, []byte(tt.input), 0644)
			if err != nil {
				t.Fatalf("Failed to write test file: %v", err)
			}

			fixer.fixFile(filePath)

			content, err := os.ReadFile(filePath)
			if err != nil {
				t.Fatalf("Failed to read file after fix: %v", err)
			}

			result := string(content)
			if result != tt.expectedOutput {
				t.Errorf("Action %s failed:\nGot:\n%s\nWant:\n%s", tt.action, result, tt.expectedOutput)
			}
		})
	}
}
