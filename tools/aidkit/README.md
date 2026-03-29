# aidkit

Go toolkit for working with [AID](https://github.com/dan-strohschein/AID) (Agent Interface Document) files. Provides a parser library and CLI tools for parsing, validating, discovering, and generating AID documentation.

## Library

```go
import "github.com/dan-strohschein/aidkit/pkg/parser"

af, warnings, err := parser.ParseFile("path/to/file.aid")
// af.Header.Module, af.Entries, af.Workflows, af.Annotations
```

### Packages

| Package | Purpose |
|---------|---------|
| `pkg/parser` | Parse AID files into structured Go types |
| `pkg/validator` | Validate AID files against the spec |
| `pkg/discovery` | Discover .aid files in a project |
| `pkg/emitter` | Emit/generate AID file content |
| `pkg/l2` | L2 AID generation (enrichment, review, staleness) |

## CLI Tools

```bash
# Parse and dump an AID file
aid-parse path/to/file.aid

# Validate AID files
aid-validate path/to/.aidocs/

# Discover AID files in a project
aid-discover /path/to/project

# Generate manifest
aid-manifest-gen /path/to/.aidocs/

# Generate L2 enrichments
aid-gen-l2 path/to/file.aid
```

## Build

```bash
go build -o aid-parse ./cmd/aid-parse
go build -o aid-validate ./cmd/aid-validate
go build -o aid-discover ./cmd/aid-discover
go build -o aid-manifest-gen ./cmd/aid-manifest-gen
go build -o aid-gen-l2 ./cmd/aid-gen-l2
```

## Test

```bash
go test ./...
```

## License

MIT
