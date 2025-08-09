# valuesctl

Schema‑first CLI to generate config samples (with YAML comments), validate user configs, render a Go template, and patch an existing `values.yaml` **in place** using a JSON merge patch.

## What it does

- **Generate sample config** from a **JSON Schema** (authored in YAML or JSON)
  - Inserts each property’s `description` as a **comment above the key**
  - Honors `default → const → first enum → type‑based placeholder`
  - Recurses nested **objects/arrays** (deterministic key order)
- **Validate** a `config.yaml` against the schema (via `gojsonschema`)
- **Render** a Go `text/template` with the config map (helpers: `csv`, `jsonarr`)
- **Patch** an existing `values.yaml` using an RFC 7396 **JSON merge patch**
  - Atomic, in‑place write by default (with `--backup`)

## Install / Build

```sh
go mod tidy
make build

# or run without building
go run . --help
```

## CLI

### Generate a sample config (with comments)

```sh
./bin/valuesctl gen-sample \
  --schema ./config.schema.yaml \
  --out ./config.sample.yaml
```

### Validate + Patch an existing values.yaml

```sh
./bin/valuesctl patch \
  --file ./values.yaml \
  --config ./config.yaml \
  --template ./template.tmpl \
  --schema ./config.schema.yaml \
  --validate
```

Write to a different output file:

```sh
./bin/valuesctl patch \
  -f ./values.yaml \
  -c ./config.yaml \
  -t ./template.tmpl \
  -o ./values.new.yaml \
  --backup=false \
  -s ./config.schema.yaml \
  --validate
```

## Template data & helpers

The template runs against a `map[string]any` loaded from `config.yaml` with `missingkey=error` (referencing a missing key fails early). Helpers:

- `csv` – join a slice into `a,b,c`
- `jsonarr` – render a slice as a JSON array string like `["a","b"]`

Example:

```gotemplate
envVars:
  API_BASE_URL: "https://{ .host }/api"
  WS_BASE_URL: "wss://{ .host }"
  APP_NAME: "{ .appName }"
  ENABLE_MODULES: "{ csv .enabledModules }"
  ADMIN_EMAIL: "{ .adminEmail }"
```

## Schema details

- Author schemas in **YAML** or **JSON** (Draft‑07 style). The app autodetects format.
- Supported constructs: `type`, `properties`, `items`, `required`, `enum`, `const`, `default`, `allOf`, `oneOf`, `anyOf`, `additionalProperties`.
- **Validation** via `gojsonschema`.
- **Defaults application** (opt‑in): fills **missing** keys only; never overwrites user values.
  - Objects: deep fill per property (merge object defaults if configured).
  - Arrays: use schema default if array is missing; existing arrays unchanged.
  - `allOf`: apply defaults from each subschema in order; `oneOf/anyOf` not guessed.

## Sample generation (with comments)

The generator uses `yaml.v3` nodes so it can attach each property’s `description` as a comment **above** the key at every nesting level.

Precedence per field: `default → const → enum[0] → placeholder by type`.

- Objects: include all `properties` (sorted keys)
- Arrays: empty by default (toggle in code to emit one example item)
- `allOf`: shallow‑merge; comments cannot be preserved across composed schemas
- `oneOf / anyOf`: first branch is used

## Merge‑patch semantics

Patching uses **RFC 7396 JSON merge patch**:

1. Convert YAML ↔ JSON
2. Compute patch from `old` → `desired`
3. Apply patch
4. Convert back to YAML

This preserves fields you didn’t touch.
