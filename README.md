# *b*ear: *e*mbeddable *a*pplication *r*outer

[![Coverage status](https://coveralls.io/repos/ursiform/bear/badge.svg)](https://coveralls.io/r/ursiform/bear)

[![API documentation](https://godoc.org/github.com/ursiform/bear?status.svg)](https://godoc.org/github.com/ursiform/bear)

[`bear.Mux`](#type-mux) is an HTTP multiplexer. It uses a tree structure for fast routing, supports dynamic parameters, middleware,
and accepts both native [`http.HandlerFunc`](http://golang.org/pkg/net/http/#HandlerFunc) or [`bear.HandlerFunc`](https://godoc.org/github.com/ursiform/bear#HandlerFunc), which accepts an extra [`*Context`](https://godoc.org/github.com/ursiform/bear#Context) argument
that allows storing state (using the [`Get()`](https://godoc.org/github.com/ursiform/bear#Context.Get) and [`Set()`](https://godoc.org/github.com/ursiform/bear#Context.Set) methods) and calling the [`Next()`](https://godoc.org/github.com/ursiform/bear#Context.Next) middleware.

## Install
```
go get github.com/ursiform/bear
```

## Quick start
```go
package main

import (
    "fmt"
    "log"
    "github.com/ursiform/bear"
    "net/http"
)
func logRequest(res http.ResponseWriter, req *http.Request, ctx *bear.Context) {
    log.Printf("%s %s\n", req.Method, req.URL.Path)
    ctx.Next(res, req)
}
func notFound(res http.ResponseWriter, req *http.Request, ctx *bear.Context) {
    res.Header().Set("Content-Type", "text/plain")
    res.WriteHeader(http.StatusNotFound)
    res.Write([]byte("Sorry, not found!\n"))
}
func one(res http.ResponseWriter, req *http.Request, ctx *bear.Context) {
    ctx.Set("one", "set in func one").Next(res, req) // Set() allows chaining
}
func two(res http.ResponseWriter, req *http.Request, ctx *bear.Context) {
    ctx.Set("two", "set in func two").Next(res, req)
}
func three(res http.ResponseWriter, req *http.Request, ctx *bear.Context) {
    greet := fmt.Sprintf("Hello, %s!\n", ctx.Params["user"])
    first := ctx.Get("one").(string)  // assert type: interface{} as string
    second := ctx.Get("two").(string) // assert type: interface{} as string
    state := fmt.Sprintf("state one: %s\nstate two: %s\n", first, second)
    res.Header().Set("Content-Type", "text/plain")
    res.Write([]byte(greet + state))
}
func main() {
    mux := bear.New()
    mux.Always(logRequest)                          // log each incoming request
    mux.On("GET", "/hello/{user}", one, two, three) // dynamic URL param {user}
    mux.On("*", "/*", notFound)                     // wildcard method + path
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
    go test -cover github.com/ursiform/bear

## API

[![API documentation](https://godoc.org/github.com/ursiform/bear?status.svg)](https://godoc.org/github.com/ursiform/bear)

## License
[MIT License](LICENSE)
