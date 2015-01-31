# *b*ear: *e*mbeddable *a*pplication *r*outer
This is experimental software for now. The API might change.

`bear.Mux` is an HTTP multiplexer. It uses a tree structure for fast routing, supports dynamic parameters, middleware,
and accepts both native `http.HandlerFunc` or `bear.HandlerFunc` (which accepts an extra `*Context` argument that allows
storing `State` and calling the `Next` middleware)

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
    ctx.State["one"] = "set in func one"
    ctx.Next(res, req)
}
func two(res http.ResponseWriter, req *http.Request, ctx *bear.Context) {
    ctx.State["two"] = "set in func two"
    ctx.Next(res, req)
}
func three(res http.ResponseWriter, req *http.Request, ctx *bear.Context) {
    var (
        greet  string = fmt.Sprintf("Hello, %s!\n", ctx.Params["user"])
        first  string = ctx.State["one"].(string) // assert type: interface{} as string
        second string = ctx.State["two"].(string) // assert type: interface{} as string
        state  string = fmt.Sprintf("state one: %s\nstate two: %s\n", first, second)
    )
    res.Header().Set("Content-Type", "text/plain")
    res.Write([]byte(greet + state))
}
func main() {
    mux := bear.New()
    mux.On("GET", "/hello/{user}", one, two, three) // custom URL param {user}
    mux.On("*", "/*", notfound)                     // wildcard to handle 404s
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
    // Pattern is the URL pattern string that was matched by a given request
    Pattern string
    // State is a utility map of string keys and empty interface values
    // to allow one middleware to pass information to the next.
    State map[string]interface{}
}
```


#### func (*Context) Next

```go
func (ctx *Context) Next(res http.ResponseWriter, req *http.Request)
```
Next calls the next middleware (if any) that was registered as a handler for a
particular request pattern.

#### type HandlerFunc

```go
type HandlerFunc func(http.ResponseWriter, *http.Request, *Context)
```

HandlerFunc is similar to `http.HandlerFunc`, except it requires an extra
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
New returns a reference to a bear `Mux` multiplexer

#### func (*Mux) On

```go
func (mux *Mux) On(verb string, pattern string, handlers ...interface{}) error
```
On adds HTTP verb handler(s) for a URL pattern. The handler argument(s) should
either be `http.HandlerFunc` or `bear.HandlerFunc` or conform to the signature
of one of those two. NOTE: if `http.HandlerFunc` (or a function conforming to
its signature) is used no other handlers can *follow* it, i.e. it is not
middleware.

It returns an error if it fails, but does not panic. Verb strings are uppercase
HTTP methods. There is a special verb `"*"` which can be used to answer *all*
HTTP methods.

Pattern strings are composed of tokens that are separated by `"/"`
characters.

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
ServeHTTP allows a `Mux` instance to conform to the `http.Handler` interface.

## License
[MIT License](LICENSE)