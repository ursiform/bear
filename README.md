# *b*ear: *e*mbeddable *a*pplication *r*outer
[![Build Status](https://drone.io/github.com/ursiform/bear/status.png)](https://drone.io/github.com/ursiform/bear/latest)

[![Coverage Status](https://coveralls.io/repos/ursiform/bear/badge.svg)](https://coveralls.io/r/ursiform/bear)

[`bear.Mux`](#type-mux) is an HTTP multiplexer. It uses a tree structure for fast routing, supports dynamic parameters, middleware,
and accepts both native [`http.HandlerFunc`](http://golang.org/pkg/net/http/#HandlerFunc) or [`bear.HandlerFunc`](#type-handlerfunc), which accepts an extra [`*Context`](#type-context) argument
that allows storing state (using the [`Get()`](#func-context-get) and [`Set()`](#func-context-set) methods) and calling the [`Next()`](#func-context-next) middleware.

## Install
```
go get github.com/ursiform/bear
```

## Quick start
```go
package main

import (
    "fmt"
    "github.com/ursiform/bear"
    "net/http"
)

func notfound(res http.ResponseWriter, req *http.Request, ctx *bear.Context) {
    res.Header().Set("Content-Type", "text/plain")
    res.WriteHeader(http.StatusNotFound)
    res.Write([]byte("Sorry, not found!\n"))
}
func one(res http.ResponseWriter, req *http.Request, ctx *bear.Context) {
    ctx.Set("one", "set in func one")
    ctx.Next(res, req)
}
func two(res http.ResponseWriter, req *http.Request, ctx *bear.Context) {
    ctx.Set("two", "set in func two")
    ctx.Next(res, req)
}
func three(res http.ResponseWriter, req *http.Request, ctx *bear.Context) {
    var (
        greet  string = fmt.Sprintf("Hello, %s!\n", ctx.Params["user"])
        first  string = ctx.Get("one").(string) // assert type: interface{} as string
        second string = ctx.Get("two").(string) // assert type: interface{} as string
        state  string = fmt.Sprintf("state one: %s\nstate two: %s\n", first, second)
    )
    res.Header().Set("Content-Type", "text/plain")
    res.Write([]byte(greet + state))
}
func main() {
    mux := bear.New()
    mux.On("GET", "/hello/{user}", one, two, three) // dynamic URL parameter {user}
    mux.On("*", "/*", notfound)                     // use wildcard for custom 404
    http.ListenAndServe(":1337", mux)
}
```
###To see it working:
```
$ curl http://localhost:1337/hello/world
Hello, world!
state one: set in func one
state two: set in func two
$ curl http://localhost:1337/hello/world/foo
Sorry, not found!
```

## Test
    go test github.com/ursiform/bear

## API

#### type Context

```go
type Context struct {
    // Params is a map of string keys with string values that is populated
    // by the dynamic URL parameters (if any).
    // Wildcard params are accessed by using an asterisk: Params["*"]
    Params map[string]string
}
```

#### func (*Context) Get

```go
func (ctx *Context) Get(key string) interface{}
```
`Get` allows retrieving a state value (`interface{}`)

#### func (*Context) Next

```go
func (ctx *Context) Next(res http.ResponseWriter, req *http.Request)
```
`Next` calls the next middleware (if any) that was registered as a handler for a
particular request pattern.

#### func (*Context) Pattern

```go
func (ctx *Context) Pattern() string
```
`Pattern` returns the URL pattern that a request matched.

#### func (*Context) Set

```go
func (ctx *Context) Set(key string, value interface{})
```
`Set` allows setting an arbitrary value (`interface{}`) to a string key to allow one
middleware to pass information to the next.

#### type HandlerFunc

```go
type HandlerFunc func(http.ResponseWriter, *http.Request, *Context)
```

`HandlerFunc` is similar to `http.HandlerFunc`, except it requires an extra
argument for the `*Context` of a request

#### type Mux

```go
type Mux struct {
}
```


#### func New

```go
func New() *Mux
```
`New` returns a reference to a bear `Mux` multiplexer

#### func (*Mux) On

```go
func (mux *Mux) On(verb string, pattern string, handlers ...interface{}) error
```
`On` adds HTTP verb handler(s) for a URL pattern. The handler argument(s) should
either be `http.HandlerFunc` or `bear.HandlerFunc` or conform to the signature
of one of those two. NOTE: if `http.HandlerFunc` (or a function conforming to
its signature) is used no other handlers can *follow* it, i.e. it is not
middleware.

It returns an error if it fails, but does not panic. Verb strings are
uppercase HTTP methods. There is a special verb `"*"` which can be used to
answer *all* HTTP methods. It is not uncommon for the verb `"*"` to return
errors, because a path may already have a listener associated with one HTTP verb
before the `"*"` verb is called. For example, this common and useful pattern
will return an error that can safely be ignored:

```go
handlerOne := func(http.ResponseWriter, *http.Request) {}
handlerTwo := func(http.ResponseWriter, *http.Request) {}
if err := mux.On("GET", "/foo/", handlerOne); err != nil {
    println(err.Error())
} // prints nothing to stderr
if err := mux.On("*", "/foo/", handlerTwo); err != nil {
    println(err.Error())
} // prints "bear: GET /foo/ exists, ignoring" to stderr
```

Pattern strings are composed of tokens that are separated by `"/"` characters.

There are three kinds of tokens:

1. static path strings: `"/foo/bar/baz/etc"`
2. dynamically populated parameters `"/foo/{bar}/baz"` (where `"bar"` will be populated in the `*Context.Params`)
3. wildcard tokens `"/foo/bar/*"` where `*` has to be the final token.

Parsed URL params are available in handlers via the `Params` map of the `*Context`.

*Notes:*

* A trailing slash `/` is always implied, even when not explicit.
* Wildcard (`*`) patterns are only matched if no other (more specific)
pattern matches. If multiple wildcard rules match, the most specific takes
precedence.
* Wildcard patterns do *not* match empty strings: a request to `/foo/bar` will
not match the pattern `"/foo/bar/*"`. The only exception to this is the root
wildcard pattern `"/*"` which will match the request path `/` if no root
handler exists.

#### func (*Mux) ServeHTTP

```go
func (mux *Mux) ServeHTTP(res http.ResponseWriter, req *http.Request)
```
`ServeHTTP` allows a `Mux` instance to conform to the `http.Handler` interface.

## License
[MIT License](LICENSE)
