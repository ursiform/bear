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
	asterisk  = "*"
	dynamic   = "\x00"
	lasterisk = "*/"
	slash     = "/"
	wildcard  = "\x00\x00"
	word      = `\{(\w+)\}`
)

var verbs []string = []string{"CONNECT", "DELETE", "GET", "HEAD", "OPTIONS", "POST", "PUT", "TRACE"}

type Context struct {
	// Params is a map of string keys with string values that is populated
	// by the dynamic URL parameters (if any).
	// Wildcard params are accessed by using an asterisk: Params["*"]
	Params map[string]string
	// State is a utility map of string keys and empty interface values
	// to allow one middleware to pass information to the next.
	State   map[string]interface{}
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
}

type tree struct {
	children map[string]*tree
	handlers []HandlerFunc
	name     string
	pattern  *string
}

func find(tr *tree, path string) *Context {
	var (
		components []string
		current    *map[string]*tree = &tr.children
		context    *Context          = &Context{
			State: make(map[string]interface{}),
			tree:  tr}
		last int
		wild *tree = nil
	)
	if nil != *current && nil != (*current)[wildcard] {
		wild = (*current)[wildcard]
	}
	if path == slash {
		if nil != tr.handlers {
			return context
		} else if nil != wild {
			context.tree = wild
			return context
		} else {
			return nil
		}
	}
	components = split(path)
	last = len(components) - 1
	for index, component := range components {
		key := component
		if nil == *current {
			if nil == wild {
				return nil
			} else {
				context.tree = wild
				return context
			}
		}
		if nil == (*current)[key] {
			if nil == (*current)[dynamic] && nil == (*current)[wildcard] {
				if nil == wild { // i.e. there is no wildcard up the tree
					return nil
				} else {
					context.tree = wild
					return context
				}
			} else {
				if nil != (*current)[wildcard] {
					wild = (*current)[wildcard]
					blob := strings.Join(components[index:], "")
					context.param(asterisk, blob[:len(blob)-1])
				}
				if nil != (*current)[dynamic] {
					key = dynamic
					// all components have trailing slashes because of split()
					// so the final character needs to be dropped
					context.param((*current)[key].name, component[:len(component)-1])
				} else {
					key = wildcard
				}
			}
		}
		if index == last {
			if (*current)[key].handlers == nil && wild != nil {
				context.tree = wild
			} else {
				context.tree = (*current)[key]
			}
			return context
		} else {
			current = &(*current)[key].children
			if nil != (*current)[wildcard] {
				wild = (*current)[wildcard]
				blob := strings.Join(components[index:], "")
				context.param(asterisk, blob[:len(blob)-1])
			}
		}
	}
	return nil
}
func handlerize(verb string, pattern string, fns []interface{}) (handlers []HandlerFunc, err error) {
	var unreachable = false
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
			handlers = append(handlers, HandlerFunc(func(res http.ResponseWriter, req *http.Request, _ *Context) {
				fn.(http.HandlerFunc)(res, req)
			}))
		case func(http.ResponseWriter, *http.Request):
			if unreachable {
				err = fmt.Errorf("bear: %s %s has unreachable middleware", verb, pattern)
				return
			}
			unreachable = true // after a non bear.HandlerFunc handler, all other handlers are unreachable
			handlers = append(handlers, HandlerFunc(func(res http.ResponseWriter, req *http.Request, _ *Context) {
				http.HandlerFunc(fn.(func(http.ResponseWriter, *http.Request)))(res, req)
			}))
		default:
			err = fmt.Errorf("bear: handler needs to match http.HandlerFunc OR bear.HandlerFunc")
			return
		}
	}
	return
}
func set(verb string, tr *tree, pattern string, handlers []HandlerFunc) error {
	var (
		components []string
		current    *map[string]*tree = &tr.children
		last       int
	)
	if pattern == slash {
		if nil != tr.handlers {
			return fmt.Errorf("bear: %s %s exists, ignoring", verb, pattern)
		} else {
			tr.pattern = &pattern
			tr.handlers = handlers
			return nil
		}
	}
	if nil == tr.children {
		tr.children = make(map[string]*tree)
	}
	dyn := regexp.MustCompile(word)
	components = split(pattern)
	last = len(components) - 1
	for index, component := range components {
		var (
			match []string = dyn.FindStringSubmatch(component)
			key   string   = component
			name  string
		)
		if 0 < len(match) {
			key = dynamic
			name = match[1]
		} else if key == lasterisk {
			key = wildcard
			name = asterisk
		}
		if nil == (*current)[key] {
			(*current)[key] = &tree{children: make(map[string]*tree), name: name}
		}
		if index == last {
			if nil != (*current)[key].handlers {
				return fmt.Errorf("bear: %s %s exists, ignoring", verb, pattern)
			}
			(*current)[key].pattern = &pattern
			(*current)[key].handlers = handlers
			return nil
		} else if key == wildcard {
			return fmt.Errorf("bear: %s %s wildcard (%s) tokens must be the final token", verb, pattern, asterisk)
		}
		current = &(*current)[key].children
	}
	return nil
}
func split(s string) []string {
	if s == "" || s == slash {
		return []string{slash}
	}
	var prefix, suffix string
	if !strings.HasPrefix(s, slash) {
		prefix = slash // prefix paths from root
	}
	if slash != s[len(s)-1:] {
		suffix = slash // end with slash
	}
	tokens := strings.SplitAfter(strings.Replace(prefix+s+suffix, slash+slash, slash, -1), slash)
	return tokens[1 : len(tokens)-1] // first token is always / and last token is always empty string
}

// Next calls the next middleware (if any) that was registered as a handler for
// a particular request pattern.
func (ctx *Context) Next(res http.ResponseWriter, req *http.Request) {
	ctx.handler++
	if len(ctx.tree.handlers) > ctx.handler {
		ctx.tree.handlers[ctx.handler](res, req, ctx)
	}
}

func (ctx *Context) param(key string, value string) {
	if nil == ctx.Params {
		ctx.Params = make(map[string]string)
	}
	ctx.Params[key] = value
}

// Pattern returns the URL pattern that a request matched.
func (ctx *Context) Pattern() string { return *(ctx.tree.pattern) }

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
	var tr *tree
	switch verb {
	default:
		return fmt.Errorf("bear: %s isn't a valid HTTP verb", verb)
	case "*":
		for _, verb := range verbs {
			if err := mux.On(verb, pattern, handlers...); err != nil {
				return err
			}
		}
		return nil
	case "CONNECT":
		tr = mux.connect
	case "DELETE":
		tr = mux.delete
	case "GET":
		tr = mux.get
	case "HEAD":
		tr = mux.head
	case "OPTIONS":
		tr = mux.options
	case "POST":
		tr = mux.post
	case "PUT":
		tr = mux.put
	case "TRACE":
		tr = mux.trace
	}
	if fns, err := handlerize(verb, pattern, handlers); err != nil {
		return err
	} else {
		return set(verb, tr, pattern, fns)
	}
}

// ServeHTTP allows a Mux instance to conform to the http.Handler interface.
func (mux *Mux) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	var tr *tree
	switch req.Method {
	default:
		http.NotFound(res, req)
		return
	case "CONNECT":
		tr = mux.connect
	case "DELETE":
		tr = mux.delete
	case "GET":
		tr = mux.get
	case "HEAD":
		tr = mux.head
	case "OPTIONS":
		tr = mux.options
	case "POST":
		tr = mux.post
	case "PUT":
		tr = mux.put
	case "TRACE":
		tr = mux.trace
	}
	if nil == tr {
		http.NotFound(res, req)
		return
	}
	context := find(tr, req.URL.Path)
	if nil == context || nil == context.tree.handlers {
		http.NotFound(res, req)
	} else {
		context.tree.handlers[0](res, req, context)
	}
}

// New returns a pointer to a bear Mux multiplexer
func New() *Mux {
	return &Mux{
		connect: &tree{},
		delete:  &tree{},
		get:     &tree{},
		head:    &tree{},
		options: &tree{},
		post:    &tree{},
		put:     &tree{},
		trace:   &tree{}}
}
