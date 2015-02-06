// Copyright 2015 Afshin Darian. All rights reserved.
// Use of this source code is governed by The MIT License
// that can be found in the LICENSE file.

/*
Package bear (bear embeddable application router) is an HTTP multiplexer.
It uses a tree structure for fast routing, supports dynamic parameters,
middleware, and accepts both native http.HandlerFunc or bear.HandlerFunc
(which accepts an extra *Context argument that allows storing State and
calling the Next middleware)
*/
package bear

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"
)

const (
	asterisk    = "*"
	doubleslash = "//"
	dynamic     = "\x00"
	empty       = ""
	lasterisk   = "*/"
	slash       = "/"
	wildcard    = "\x00\x00"
)

var (
	dyn   *regexp.Regexp = regexp.MustCompile(`\{(\w+)\}`)
	verbs [8]string      = [8]string{"CONNECT", "DELETE", "GET", "HEAD", "OPTIONS", "POST", "PUT", "TRACE"}
)

type Context struct {
	// Params is a map of string keys with string values that is populated
	// by the dynamic URL parameters (if any).
	// Wildcard params are accessed by using an asterisk: Params["*"]
	Params  map[string]string
	state   map[string]interface{}
	handler int
	tree    *tree
}

// HandlerFunc is similar to http.HandlerFunc, except it requires
// an extra argument for the *Context of a request
type HandlerFunc func(http.ResponseWriter, *http.Request, *Context)

type Mux struct {
	connect *tree
	delete  *tree
	get     *tree
	head    *tree
	options *tree
	post    *tree
	put     *tree
	trace   *tree
	wild    bool
}

type tree struct {
	children map[string]*tree
	handlers []HandlerFunc
	name     string
	pattern  string
}

func handlerize(verb string, pattern string, fns []interface{}) (handlers []HandlerFunc, err error) {
	unreachable := false
	for _, fn := range fns {
		switch fn.(type) {
		case HandlerFunc:
			if unreachable {
				err = fmt.Errorf("bear: %s %s has unreachable middleware", verb, pattern)
				return
			}
			handlers = append(handlers, HandlerFunc(fn.(HandlerFunc)))
		case func(http.ResponseWriter, *http.Request, *Context):
			if unreachable {
				err = fmt.Errorf("bear: %s %s has unreachable middleware", verb, pattern)
				return
			}
			handlers = append(handlers, HandlerFunc(fn.(func(http.ResponseWriter, *http.Request, *Context))))
		case http.HandlerFunc:
			if unreachable {
				err = fmt.Errorf("bear: %s %s has unreachable middleware", verb, pattern)
				return
			}
			unreachable = true // after a non bear.HandlerFunc handler, all other handlers are unreachable
			handler := fn.(http.HandlerFunc)
			handlers = append(handlers, HandlerFunc(func(res http.ResponseWriter, req *http.Request, _ *Context) {
				handler(res, req)
			}))
		case func(http.ResponseWriter, *http.Request):
			if unreachable {
				err = fmt.Errorf("bear: %s %s has unreachable middleware", verb, pattern)
				return
			}
			unreachable = true // after a non bear.HandlerFunc handler, all other handlers are unreachable
			handler := fn.(func(http.ResponseWriter, *http.Request))
			handlers = append(handlers, HandlerFunc(func(res http.ResponseWriter, req *http.Request, _ *Context) {
				handler(res, req)
			}))
		default:
			err = fmt.Errorf("bear: handler needs to match http.HandlerFunc OR bear.HandlerFunc")
			return
		}
	}
	return
}
func sanitize(s string) string {
	if s == empty || s == slash {
		return slash
	}
	if !strings.HasPrefix(s, slash) { // start with slash
		s = slash + s
	}
	if slash != s[len(s)-1:] { // end with slash
		s = s + slash
	}
	return strings.Replace(s, doubleslash, slash, -1)
}
func set(verb string, tr *tree, pattern string, handlers []HandlerFunc) (wild bool, err error) {
	if pattern == slash {
		if nil != tr.handlers {
			return false, fmt.Errorf("bear: %s %s exists, ignoring", verb, pattern)
		} else {
			tr.pattern = pattern
			tr.handlers = handlers
			return false, nil
		}
	}
	if nil == tr.children {
		tr.children = make(map[string]*tree)
	}
	current := &tr.children
	components := strings.SplitAfter(sanitize(pattern), slash)
	components = components[1 : len(components)-1] // first token is "/" and last token is ""
	last := len(components) - 1
	for index, component := range components {
		var (
			match []string = dyn.FindStringSubmatch(component)
			key   string   = component
			name  string
		)
		if 0 < len(match) {
			key, name = dynamic, match[1]
		} else if key == lasterisk {
			key, name = wildcard, asterisk
			wild = true
		}
		if nil == (*current)[key] {
			(*current)[key] = &tree{children: make(map[string]*tree), name: name}
		}
		if index == last {
			if nil != (*current)[key].handlers {
				return false, fmt.Errorf("bear: %s %s exists, ignoring", verb, pattern)
			}
			(*current)[key].pattern = pattern
			(*current)[key].handlers = handlers
			return wild, nil
		} else if key == wildcard {
			return false, fmt.Errorf("bear: %s %s wildcard (%s) token must be last", verb, pattern, asterisk)
		}
		current = &(*current)[key].children
	}
	return wild, nil
}

// Get allows retrieving a state value (interface{})
func (ctx *Context) Get(key string) interface{} {
	if nil == ctx.state {
		return nil
	} else {
		return ctx.state[key]
	}
}

// Next calls the next middleware (if any) that was registered as a handler for
// a particular request pattern.
func (ctx *Context) Next(res http.ResponseWriter, req *http.Request) {
	handlers := len(ctx.tree.handlers)
	ctx.handler++
	if handlers > ctx.handler {
		ctx.tree.handlers[ctx.handler](res, req, ctx)
	}
}

func (ctx *Context) param(key string, value string, capacity int) {
	if nil == ctx.Params {
		ctx.Params = make(map[string]string, capacity)
	}
	ctx.Params[key] = value[:len(value)-1]
}

// Set allows setting an arbitrary value (interface{}) to a string key
// to allow one middleware to pass information to the next.
func (ctx *Context) Set(key string, value interface{}) {
	if nil == ctx.state {
		ctx.state = make(map[string]interface{})
	}
	ctx.state[key] = value
}

// Pattern returns the URL pattern that a request matched.
func (ctx *Context) Pattern() string { return ctx.tree.pattern }

/*
On adds HTTP verb handler(s) for a URL pattern. The handler argument(s)
should either be http.HandlerFunc or bear.HandlerFunc or conform to the
signature of one of those two. NOTE: if http.HandlerFunc (or a function
conforming to its signature) is used no other handlers can *follow* it, i.e.
it is not middleware.

It returns an error if it fails, but does not panic. Verb strings are
uppercase HTTP methods. There is a special verb "*" which can be used to
answer *all* HTTP methods.

Pattern strings are composed of tokens that are separated by "/" characters.
There are three kinds of tokens:

1. static path strings: "/foo/bar/baz/etc"

2. dynamically populated parameters "/foo/{bar}/baz" (where "bar" will be
populated in the *Context.Params)

3. wildcard tokens "/foo/bar/*" where * has to be the final token.
Parsed URL params are available in handlers via the Params map of the
*Context argument.

Notes:

1. A trailing slash / is always implied, even when not explicit.

2. Wildcard (*) patterns are only matched if no other (more specific)
pattern matches. If multiple wildcard rules match, the most specific takes
precedence.

3. Wildcard patterns do *not* match empty strings: a request to /foo/bar will
not match the pattern "/foo/bar/*". The only exception to this is the root
wildcard pattern "/*" which will match the request path / if no root
handler exists.
*/
func (mux *Mux) On(verb string, pattern string, handlers ...interface{}) error {
	if verb == asterisk {
		for _, verb := range verbs {
			if err := mux.On(verb, pattern, handlers...); err != nil {
				return err
			}
		}
		return nil
	}
	var tr *tree = mux.tree(verb)
	if nil == tr {
		return fmt.Errorf("bear: %s isn't a valid HTTP verb", verb)
	}
	if fns, err := handlerize(verb, pattern, handlers); err != nil {
		return err
	} else {
		if wild, err := set(verb, tr, pattern, fns); err != nil {
			return err
		} else {
			mux.wild = mux.wild || wild
			return nil
		}
	}
}

// ServeHTTP allows a Mux instance to conform to the http.Handler interface.
func (mux *Mux) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	tr := mux.tree(req.Method)
	if nil == tr { // i.e. if req.Method is not found in HTTP verbs
		http.NotFound(res, req)
		return
	}
	if req.URL.Path == slash { // root is a special case because it is the top node in the tree
		if nil != tr.handlers { // root match
			tr.handlers[0](res, req, &Context{tree: tr})
			return
		} else if wild := tr.children[wildcard]; nil != wild { // root level wildcard pattern match
			wild.handlers[0](res, req, &Context{tree: wild})
			return
		}
		http.NotFound(res, req)
		return
	}
	var key string
	components := strings.SplitAfter(sanitize(req.URL.Path), slash)
	components = components[1 : len(components)-1] // first token is "/" and last token is ""
	context := new(Context)
	current := &tr.children
	capacity := len(components) // maximum number of URL params possible for this request
	last := capacity - 1
	if !mux.wild { // no wildcards: simpler, slightly faster
		for index, component := range components {
			key = component
			if nil == *current {
				http.NotFound(res, req)
				return
			} else if nil == (*current)[key] {
				if nil == (*current)[dynamic] {
					http.NotFound(res, req)
					return
				} else {
					key = dynamic
					context.param((*current)[key].name, component, capacity)
				}
			}
			if index == last {
				if nil == (*current)[key].handlers {
					http.NotFound(res, req)
				} else {
					context.tree = (*current)[key]
					context.tree.handlers[0](res, req, context)
				}
				return
			}
			current = &(*current)[key].children
		}
	} else {
		wild := tr.children[wildcard]
		for index, component := range components {
			key = component
			if nil == *current {
				if nil == wild {
					http.NotFound(res, req)
				} else { // wildcard pattern match
					context.tree = wild
					context.tree.handlers[0](res, req, context)
				}
				return
			}
			if nil == (*current)[key] {
				if nil == (*current)[dynamic] && nil == (*current)[wildcard] {
					if nil == wild { // i.e. there is no wildcard up the tree
						http.NotFound(res, req)
					} else { // wildcard pattern match
						context.tree = wild
						wild.handlers[0](res, req, context)
					}
					return
				} else {
					if nil != (*current)[wildcard] { // i.e. there is a more proximate wildcard
						wild = (*current)[wildcard]
						context.param(asterisk, strings.Join(components[index:], empty), capacity)
					}
					if nil != (*current)[dynamic] {
						key = dynamic
						context.param((*current)[key].name, component, capacity)
					} else { // wildcard pattern match
						context.tree = wild
						wild.handlers[0](res, req, context)
						return
					}
				}
			}
			if index == last {
				if nil == (*current)[key].handlers {
					http.NotFound(res, req)
				} else { // non-wildcard pattern match
					context.tree = (*current)[key]
					context.tree.handlers[0](res, req, context)
				}
				return
			}
			current = &(*current)[key].children
			if nil != (*current)[wildcard] { // i.e. there is a more proximate wildcard
				wild = (*current)[wildcard]
				context.param(asterisk, strings.Join(components[index:], empty), capacity)
			}
		}
	}
}

func (mux *Mux) tree(name string) *tree {
	switch name {
	case "CONNECT":
		return mux.connect
	case "DELETE":
		return mux.delete
	case "GET":
		return mux.get
	case "HEAD":
		return mux.head
	case "OPTIONS":
		return mux.options
	case "POST":
		return mux.post
	case "PUT":
		return mux.put
	case "TRACE":
		return mux.trace
	default:
		return nil
	}
}

// New returns a pointer to a bear Mux multiplexer
func New() *Mux {
	return &Mux{&tree{}, &tree{}, &tree{}, &tree{}, &tree{}, &tree{}, &tree{}, &tree{}, false}
}
