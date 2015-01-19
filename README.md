# *b*ear: *e*mbeddedable *a*pplication *r*outer
This is experimental software for now. The API might change.

    import "github.com/ursiform/bear"

Package bear (bear embeddedable application router) is an HTTP multiplexer. It
uses a tree structure for fast routing, supports dynamic parameters, and accepts
both native http.HandlerFunc or bear.HandlerFunc (which accepts an extra
Context)

## Usage

#### type Context

```go
type Context struct {
    Params  map[string]string
    Pattern string
}
```

Context is a struct that contains: Params: a map of string keys with string
values that is populated by the dynamic URL parameters (if any) Pattern: the URL
pattern string that was matched by a given request

#### type HandlerFunc

```go
type HandlerFunc func(http.ResponseWriter, *http.Request, *Context)
```

HandlerFunc is very similar to net/http HandlerFunc, except it requires an extra
argument for the Context of a request

#### type Mux

```go
type Mux struct {
}
```

Mux holds references to separate trees for each HTTP verb

#### func  New

```go
func New() *Mux
```
New returns a reference to a bear Mux multiplexer

#### func (*Mux) On

```go
func (mux *Mux) On(verb string, pattern string, handler interface{}) error
```
On adds an HTTP verb handler for a URL pattern. The third argument should either
be an http.HandlerFunc or a bear.HandlerFunc or conform to the signature of one
of those two. It returns an error if it fails, but does not panic.

#### func (*Mux) ServeHTTP

```go
func (mux *Mux) ServeHTTP(res http.ResponseWriter, req *http.Request)
```
ServeHTTP allows a Mux instance to conform to http.Handler interface.
