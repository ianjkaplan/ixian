# Ixian

OpenAPI 3.x to Go CLI code generator. Reads an OpenAPI spec and generates a fully functional Go CLI application using Cobra.

> Ixian refers to the planet machine makers in the Dune universe

## Quick Start

```bash
# Build
make build

# Generate a CLI from the example petstore spec
make generate

# Run all tests
make test
```

## Usage

```bash
./ixian --spec path/to/openapi.yaml --output ./output
```

Flags:

| Flag       | Short | Default    | Description                             |
| ---------- | ----- | ---------- | --------------------------------------- |
| `--spec`   | `-s`  | (required) | Path to OpenAPI 3.x spec (YAML or JSON) |
| `--output` | `-o`  | `./output` | Output directory for generated code     |

## Pipeline

The generator runs a six-stage pipeline:

1. **Parser** — Reads the spec file into a raw AST (no $ref resolution)
2. **Binder** — Resolves all `$ref` pointers, builds symbol table and dependency graph
3. **Checker** — Validates the bound AST, produces errors and warnings
4. **Planner** — Transforms the AST into a Go+CLI intermediate representation (type mapping, command grouping, flag strategy)
5. **Emitter** — Generates Go source files from the IR using `text/template`
6. **Formatter** — Runs `gofmt` on all generated files and writes to disk

## Authentication

Ixian reads `securitySchemes` from the OpenAPI spec and generates corresponding CLI flags and client logic.

| OpenAPI Scheme               | Generated Flag | Behavior                                          |
| ---------------------------- | -------------- | ------------------------------------------------- |
| `type: http, scheme: bearer` | `--auth-token` | Sends `Authorization: Bearer <token>` header      |
| `type: http, scheme: basic`  | `--auth-basic` | Sends `Authorization: Basic <credentials>` header |
| `type: apiKey, in: header`   | `--<key-name>` | Sends the API key as the specified header         |
| `type: apiKey, in: query`    | `--<key-name>` | Appends the API key as a query parameter          |

Example:

```bash
# Bearer token auth
./mycli --auth-token eyJhbG... pets list-pets

# API key auth
./mycli --x-api-key sk-abc123 pets list-pets
```

## Custom Headers

The generated CLI supports a repeatable `--header` (`-H`) flag for passing arbitrary headers:

```bash
./mycli --header "X-Request-ID:abc123" --header "Accept-Language:en" pets list-pets
```

Headers are passed through to every HTTP request. Custom headers are applied after auth headers, so they can override auth if needed.

## Generated Output

```
output/
├── main.go              # Entry point
├── cmd/
│   ├── root.go          # Root command, global flags
│   ├── pets.go          # pets subcommand group
│   └── owners.go        # owners subcommand group
├── client/
│   └── client.go        # HTTP client
└── types/
    └── types.go         # Generated structs and enums
```

## Development

```bash
make check     # fmt + vet + test
make test-race # tests with race detector
make fmt-fix   # auto-fix formatting
make clean     # remove build artifacts
```

## Project Structure

```
cmd/ixian/           # CLI entry point
internal/
  ast/               # Raw OpenAPI AST types
  parser/            # YAML/JSON → AST
  binder/            # $ref resolution, symbol table
  checker/           # Validation and diagnostics
  planner/           # AST → codegen IR
  ir/                # Intermediate representation types
  emitter/           # IR → Go source files
  formatter/         # gofmt + file writer
testdata/            # Golden file specs for testing
```

## Architecture

```
                    ┌──────────────┐
                    │  CLI Entry   │
                    │  (Cobra)     │
                    └──────┬───────┘
                           │
                           ▼
                    ┌──────────────┐
                    │   Parser     │
                    │              │
                    │ YAML/JSON →  │
                    │ Raw AST      │
                    └──────┬───────┘
                           │
                           ▼
                    ┌──────────────┐
                    │   Binder     │
                    │              │
                    │ Resolve $ref │
                    │ Symbol Table │
                    │ Dep Graph    │
                    └──────┬───────┘
                           │
                           ▼
                    ┌──────────────┐
                    │   Checker    │
                    │              │
                    │ Validate     │
                    │ Diagnostics  │
                    └──────┬───────┘
                           │
                           ▼
                    ┌──────────────┐
                    │   Planner    │
                    │              │
                    │ Type mapping │
                    │ Cmd grouping │
                    │ Flag strategy│
                    │ → Codegen IR │
                    └──────┬───────┘
                           │
                           ▼
                    ┌──────────────┐
                    │   Emitter    │
                    │              │
                    │ IR → .go     │
                    │ (templates)  │
                    └──────┬───────┘
                           │
                           ▼
                    ┌──────────────┐
                    │  Formatter   │
                    │              │
                    │ gofmt + write│
                    └──────────────┘
```

Each stage has a single input type and single output type. No stage reaches back to a previous stage's output. The planner owns all decisions (type mapping, command structure, flag names) — the emitter is a dumb printer.
