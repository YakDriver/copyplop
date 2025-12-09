<!-- Copyright IBM Corp. 2014, 2025 -->
<!-- SPDX-License-Identifier: MPL-2.0 -->

# copyplop

A fully configurable Go CLI tool for managing copyright headers in source code files.

## Features

- **Fully configurable**: Define any copyright format via templates
- **Optional licensing**: Enable/disable SPDX license headers
- **Path filtering**: Include/exclude files using powerful glob patterns with `**` support
- **Multiple file types**: Support any file extension with custom comment styles
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
  current_year: 2025
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
// Copyright IBM Corp. 2014, 2025
//Copyright (c) 2025, Oracle and/or its affiliates. All rights reserved.

package main
```

**With `action: "below"`:**
```go
//Copyright (c) 2025, Oracle and/or its affiliates. All rights reserved.
// Copyright IBM Corp. 2014, 2025
package main
```

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

## Usage

```bash
# Check for issues
copyplop check

# Fix copyright headers
copyplop fix

# Process specific path
copyplop check --path ./internal/service/ec2

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
Output: `// Copyright IBM Corp. 2014, 2025`

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
Output: `// Copyright 2025 Acme Corp`

See `examples/` directory for complete configurations.
