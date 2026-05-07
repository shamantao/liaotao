# Config Schema Reference

## Sections

### `[app]`
| Key       | Type   | Default   | Description                             |
|-----------|--------|-----------|-----------------------------------------|
| name      | string | required  | Application name                        |
| version   | string | "0.1.0"   | Semantic version                        |
| mode      | string | "debug"   | `debug` keeps originals; `normal` trashes them |
| language  | string | "fr"      | UI language                             |

### `[config]`
| Key                  | Type    | Default | Description                              |
|----------------------|---------|---------|------------------------------------------|
| schema_version       | int     | 1       | Increment on breaking config changes     |
| enable_layered_merge | bool    | true    | Merge default→user→project→runtime       |
| strict_mode          | bool    | false   | Fail on unknown keys                     |

### `[path_manager]`
| Key                | Type     | Default        | Description                             |
|--------------------|----------|----------------|-----------------------------------------|
| allowed_roots      | [string] | required       | Whitelist of safe write locations       |
| temp_dir           | string   | .tmp/          | Temporary working directory             |
| logs_dir           | string   | logs/          | Log file directory                      |
| reports_dir        | string   | reports/       | Output reports directory                |
| collision_strategy | string   | "increment"    | `increment`, `suffix`, `short_hash`     |
| normalize_unicode  | bool     | false          | Normalize filenames to NFC              |
| trim_whitespace    | bool     | true           | Trim leading/trailing spaces in names   |

### `[logger]`
| Key                 | Type   | Default | Description                             |
|---------------------|--------|---------|-----------------------------------------|
| level               | string | "info"  | `trace`, `debug`, `info`, `warn`, `error` |
| console_pretty      | bool   | true    | Human-readable console output           |
| file_json           | bool   | true    | JSON-structured file output             |
| rotation_enabled    | bool   | true    | Rotate log files by size                |
| max_file_mb         | int    | 20      | Max size per file before rotation       |
| max_files           | int    | 5       | Max number of rotated log files         |
| include_context_ids | bool   | true    | Attach job_id / correlation_id to logs  |

### `[reporting]`
| Key            | Type | Default | Description                          |
|----------------|------|---------|--------------------------------------|
| enabled        | bool | true    | Enable report generation             |
| json_report    | bool | true    | Output JSON report                   |
| csv_report     | bool | true    | Output CSV report                    |
| include_failed | bool | true    | Include failed jobs in reports       |
