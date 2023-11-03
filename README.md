# Zappac Lang

Zappac is a calculator. Zappac Lang is the processing utilities for the language used by Zappac's
various forms.

## Testing

Install [gow](https://github.com/mitranim/gow)

```shell
go install golang.org/x/tools/cmd/stringer@latest
go install github.com/mitranim/gow@latest
```

Run tests and wait for any changes before running them again

```shell
gow test ./...
```

If you change any of the `iota` types, you will need to trigger stringer codegen:

```shell
go generate ./...
```
