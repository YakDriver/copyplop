<!-- Copyright IBM Corp. 2014, 2026 -->
<!-- SPDX-License-Identifier: MPL-2.0 -->

# copyplop

A fully configurable Go CLI tool for managing copyright headers in source code files.

## Features

- **Fully configurable**: Define any copyright format via templates
- **Self-updating**: Automatically updates headers when config changes (e.g., year updates)
- **Precise detection**: Only modifies actual header lines, preserves documentation mentions
- **Optional licensing**: Enable/disable SPDX license headers
- **Path filtering**: Include/exclude files using powerful glob patterns with `**` support
- **Multiple file types**: Support any file extension with custom comment styles (including block comments)
- **Smart detection**: Skip generated files, replace specific patterns
- **Third-party copyright handling**: Configure how to handle existing third-party copyrights
- **Git integration**: Only processes git-tracked files
- **Progress tracking**: Visual progress bar for large codebases
- **Template-based**: Use Go templates for flexible header formats

## Installation

```bash
go install github.com/YakDriver/copyplop@latest
```

## Configuration

Create `.copyplop.yaml` in your project root:

```yaml
copyright:
  holder: "Your Company"
  start_year: 2020
  current_year: 2026
  format: "Copyright {{.Holder}} {{.StartYear}}-{{.CurrentYear}}"
  
license:
  enabled: true
  identifier: "MIT"
files:
  extensions: [".go", ".js", ".py", ".sh"]
  comment_styles:
    ".go": "//"
    ".js": "//"
    ".py": "#"
    ".sh": "#"

detection:
  skip_generated: true
  generated_patterns: ["Code generated", "DO NOT EDIT"]
  replace_patterns: ["Copyright.*OldCompany"]

third_party:
  action: "above"  # "leave", "above", "below", "replace"
  patterns:
    - "Copyright.*Oracle"
    - "Copyright.*Microsoft"
```

## Third-Party Copyright Handling

Configure how to handle existing third-party copyrights with **precedence logic**:

- **`leave`**: Keep third-party copyrights as-is, add your copyright normally
- **`above`**: Add your copyright above third-party copyrights
- **`below`**: Add your copyright below third-party copyrights  
- **`replace`**: Replace third-party copyrights with your copyright

### Precedence Rules

**Replacement patterns take precedence over third-party patterns.** This allows you to use general third-party patterns without accidentally treating your own replacement targets as third-party.

```yaml
detection:
  # These get REPLACED (highest precedence)
  replace_patterns:
third_party:
  action: "above"
  # General pattern - but won't match HashiCorp due to precedence
  patterns:
    - "Copyright.*[a-zA-Z0-9].*"
```

**Result:** HashiCorp copyrights get replaced, all other copyrights are treated as third-party.

### Example Results

**Original file with Oracle copyright:**
```go
//Copyright (c) 2025, Oracle and/or its affiliates. All rights reserved.

package main
```

**With `action: "above"`:**
```go
// Copyright IBM Corp. 2014, 2026
//Copyright (c) 2025, Oracle and/or its affiliates. All rights reserved.

package main
```

**With `action: "below"`:**
```go
//Copyright (c) 2025, Oracle and/or its affiliates. All rights reserved.
// Copyright IBM Corp. 2014, 2026
package main
```

## Smart Extensions

Handle template files that could contain different content types using smart content detection:

```yaml
files:
  # Regular extensions with known comment styles
  extensions: [".go", ".py", ".md"]
  
  # Smart extensions detect content type automatically
  smart_extensions: [".gtpl", ".tmpl"]
  
  comment_styles:
    ".go": "//"
    ".py": "#"
    ".md": "<!--"
```

### How Smart Extensions Work

Smart extensions analyze file content to determine the actual file type:

- **`.gtpl`** (Go templates) - Could be Go code, Markdown, HCL, YAML, etc.
- **`.tmpl`** (Generic templates) - Detects the underlying content type

### Detection Patterns

| Content Type | Detection Patterns |
|--------------|-------------------|
| **Go** | `package `, `func `, `import (`, `type ... struct` |
| **Markdown** | `# `, `## `, ` ``` `, `[text](url)` |
| **HCL/Terraform** | `resource "`, `data "`, `variable "`, `output "` |
| **YAML** | `---`, `key: value` patterns |

**Binary File Safety:** Files with null bytes are automatically skipped to prevent processing binary content.

### Example Use Cases

```yaml
# For terraform-provider-aws templates
smart_extensions: [".gtpl", ".tmpl"]

# Templates will be processed with appropriate comment style:
# - service.go.gtpl → detected as Go → uses "//" comments  
# - README.md.tmpl → detected as Markdown → uses "<!--" comments
# - main.tf.gtpl → detected as HCL → uses "#" comments
```

**Fallback:** Unknown content defaults to Go (configurable per project needs).

## Path Filtering

Control which files to process using include/exclude patterns with full doublestar glob support:

```yaml
files:
  # Only process specific paths
  include_paths:
    - "internal/service/[a-g]*"     # Services starting with a-g
    - "cmd/**"                      # All files under cmd/
    
  # Skip specific paths  
  exclude_paths:
    - ".github/**"                  # Skip all GitHub workflows
    - "internal/service/s3*"        # Skip S3-related services
    - "**/*_test.go"               # Skip all test files
```

### Pattern Logic
- **No filters**: Process all files
- **Include only**: Process only matching files
- **Exclude only**: Process all except matching files  
- **Both**: Process files that match includes AND don't match excludes

### Supported Patterns
- `*` - Single-level wildcard
- `**` - Recursive directory matching
- `[a-g]` - Character ranges
- `{foo,bar}` - Alternatives
- `internal/service/*` - Auto-expands to match subdirectories

### Examples
```bash
# Process only EC2 and ECS services
include_paths: ["internal/service/ec[2s]*"]

# Skip generated and test files
exclude_paths: ["**/*generated*", "**/*_test.go"]

# Process infrastructure code only
include_paths: ["internal/**", "cmd/**"]
exclude_paths: [".github/**", "examples/**"]
```

## Placement Exceptions

Copyplop supports configurable placement exceptions for cases where copyright headers cannot be the first line in a file.

### Configuration

```yaml
files:
  placement_exceptions:
    xml_declaration: true    # Allow <?xml version="1.0"?> before copyright
    markdown_heading: true   # Allow # Heading before copyright  
    frontmatter: ["md", "html.md"]  # YAML frontmatter extensions
```

### Exception Types

**Always Enabled:**
- **Shebang** (`#!/bin/bash`) - Always detected and preserved

**Configurable Exceptions:**
- **XML Declaration** - `<?xml version="1.0"?>` and similar
- **Markdown Heading** - `# Title` as first line
- **YAML Frontmatter** - Between `---` markers

### Processing Order

Exceptions are processed in this order:
1. Shebang (always)
2. XML Declaration (if enabled)
3. YAML Frontmatter (if configured)
4. Markdown Heading (if enabled)
5. Copyright header placement

### Examples

**XML File:**
```xml
<?xml version="1.0"?>
<!-- Copyright 2024 Your Corp -->

<root>content</root>
```

**Markdown File:**
```markdown
# My Document
<!-- Copyright 2024 Your Corp -->

Content here...
```

**Backward Compatibility:** The `below_frontmatter` configuration is deprecated. Use `placement_exceptions.frontmatter` instead.

## Self-Updating Headers

Copyplop automatically detects and updates its own copyright headers when configuration changes:

```yaml
# Change current_year from 2025 to 2026
copyright:
  current_year: 2026  # Updated
```

**Before:**
```go
// Copyright IBM Corp. 2014, 2025
// SPDX-License-Identifier: MPL-2.0
```

**After running `copyplop fix`:**
```go
// Copyright IBM Corp. 2014, 2026  
// SPDX-License-Identifier: MPL-2.0
```

### Precision Detection

Copyplop precisely identifies header lines vs. documentation mentions:

- ✅ **Updates**: Actual comment headers at the top of files
- ✅ **Preserves**: Documentation mentioning "Copyright" or "SPDX-License-Identifier"
- ✅ **Preserves**: Configuration values like `format: "SPDX-License-Identifier: {{.Identifier}}"`

### Block Comment Support

Works with all comment styles including block comments:

```javascript
/**
 * Copyright IBM Corp. 2014, 2025  // ← Gets updated
 * SPDX-License-Identifier: MPL-2.0
 */

function example() {
  // This mentions Copyright IBM Corp. in docs  // ← Preserved
}
```

## Usage

```bash
# Check for issues
copyplop check

# Fix copyright headers
copyplop fix

# Process specific path
copyplop check --path ./internal/service/ec2

# Show version
copyplop version
# or
copyplop --version

# Demo progress bar
copyplop demo
```

## Template Variables

Available in `copyright.format`:
- `{{.Holder}}` - Copyright holder
- `{{.StartYear}}` - Starting year
- `{{.CurrentYear}}` - Current year

Available in `license.format`:
- `{{.Identifier}}` - License identifier

## Examples

### IBM Style (with year range)
```yaml
copyright:
  format: "Copyright {{.Holder}} {{.StartYear}}, {{.CurrentYear}}"
```
Output: `// Copyright IBM Corp. 2014, 2026`

### HashiCorp Style (no year)
```yaml
copyright:
  format: "Copyright (c) {{.Holder}}"
```
### Simple Year Only
```yaml
copyright:
  format: "Copyright {{.CurrentYear}} {{.Holder}}"
```
Output: `// Copyright 2026 Acme Corp`

See `examples/` directory for complete configurations.
