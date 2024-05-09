# reqbind

[![Go Coverage](https://github.com/codeallthethingz/reqbind/wiki/coverage.svg)](https://raw.githack.com/wiki/codeallthethingz/reqbind/coverage.html)
A small library to bind query string, path parameters, and request body to a struct with validation metadata.

## Installation

```shell
go get github.com/codeallthethingz/reqbind
```

## Usage

### Basic Usage

```go
import "github.com/codeallthethingz/reqbind"

// UnmarshalURLParams binds chi path parameters to a struct
u := &struct {
    ProjectID model.ID
    ColumnID  model.ID
}{}
if err := reqbind.UnmarshalURLParams(r, u); err != nil {
    http.Error(w, err.Error(), http.StatusBadRequest)
    return
}

// UnmarshalBody binds request body to a struct
b := &struct {
    Size int
}{}
if err := reqbind.UnmarshalBody(r, b); err != nil {
    http.Error(w, err.Error(), http.StatusBadRequest)
    return
}

// UnmarshalQuery binds query string to a struct
q := &struct {
    ProjectID model.ID
}{}
if err := reqbind.UnmarshalQuery(r, q); err != nil {
    http.Error(w, err.Error(), http.StatusBadRequest)
    return
}
```

### Custom Validation

```go
u := &struct {
    Email       string `required:"true" validate:"email" trimlower:"true"`
    Description string `max-length:"1000"`
    Phone       string `required:"true" validate:"phone"`
}{}
```

### Nested Objects

```go
b := &struct {
    Indexes []struct {
        ID       int64 `required:"true"`
        Position int   `required:"true"`
    } `required:"true"`
}{}
if err := reqbind.UnmarshalBody(r, b); err != nil {
    http.Error(w, err.Error(), http.StatusBadRequest)
    return
}

```
