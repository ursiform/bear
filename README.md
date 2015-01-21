# *b*ear: *e*mbeddable *a*pplication *r*outer
This is experimental software for now. The API might change.

`bear` is an HTTP multiplexer. It uses a tree structure for fast routing, supports dynamic parameters, middleware,
and accepts both native `http.HandlerFunc` or `bear.HandlerFunc` (which accepts an extra `*Context` argument that allows
storing state and calling the next middleware)

## Quick start
```go
package main

import (
    "fmt"
    "github.com/ursiform/bear"
    "net/http"
)

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
    port := ":1337"
    err := mux.On("GET", "/hello/{user}", one, two, three)
    if err != nil {
        fmt.Println(err)
    } else {
        fmt.Printf("running on %s\n", port)
        http.ListenAndServe(port, mux)
    }
}
```
###To see it working:
```
$ curl http://localhost:1337/hello/world
Hello, world!
state one: set in func one
state two: set in func two
```

## Test
    go test github.com/ursiform/bear

## API

#### type Context

```go
type Context struct {
    // Params is a map of string keys with string values that is populated
    // by the dynamic URL parameters (if any)
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

HandlerFunc is similar to net/http HandlerFunc, except it requires an extra
argument for the Context of a request

#### type Mux

```go
type Mux struct {
}
```

#### func  New

```go
func New() *Mux
```
New returns a reference to a bear Mux multiplexer

#### func (*Mux) On

```go
func (mux *Mux) On(verb string, pattern string, handlers ...interface{}) error
```
On adds HTTP verb handler(s) for a URL pattern. The handler argument(s) should
either be http.HandlerFunc or bear.HandlerFunc or conform to the signature of
one of those two. NOTE: if http.HandlerFunc (or a function conforming to its
signature) is used no other handlers can FOLLOW it, i.e. it is not middleware It
returns an error if it fails, but does not panic.

#### func (*Mux) ServeHTTP

```go
func (mux *Mux) ServeHTTP(res http.ResponseWriter, req *http.Request)
```
ServeHTTP allows a Mux instance to conform to http.Handler interface.

## License
[MIT License](LICENSE)
