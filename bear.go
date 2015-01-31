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
	// Pattern is the URL pattern string that was matched by a given request
	Pattern string
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
	children treemap
	dynamic  bool
	handlers []HandlerFunc
	name     string
	pattern  string
}
type treemap map[string]*tree

func deploy(tr *tree, res http.ResponseWriter, req *http.Request) {
	if nil == tr {
		http.NotFound(res, req)
	} else {
		location, context := find(tr, req.URL.Path)
		if nil == location || nil == location.handlers {
			http.NotFound(res, req)
		} else {
			location.handlers[0](res, req, context)
		}
	}
}
func find(tr *tree, path string) (*tree, *Context) {
	var (
		components []string = split(path)
		current    *treemap = &tr.children
		context    *Context = &Context{Params: make(map[string]string), State: make(map[string]interface{}), tree: tr}
		last       int      = len(components) - 1
		wild       *tree    = new(tree)
	)
	if nil != *current && nil != (*current)[wildcard] {
		wild = (*current)[wildcard]
	}
	if 0 == last { // i.e. location is /
		if nil != tr.handlers {
			context.Pattern = tr.pattern
			return tr, context
		} else if nil != wild {
			context.Pattern = wild.pattern
			return wild, context
		} else {
			return nil, context
		}
	}
	components, last = components[1:], last-1 // ignore the initial "/" component
	for index, component := range components {
		key := component
		if nil == *current {
			if nil == wild {
				return nil, context
			} else {
				context.tree = wild
				context.Pattern = context.tree.pattern
				return context.tree, context
			}
		}
		if nil == (*current)[key] {
			if nil == (*current)[dynamic] && nil == (*current)[wildcard] {
				if nil == wild { // i.e. there is no wildcard up the tree
					return nil, context
				} else {
					context.tree = wild
					context.Pattern = context.tree.pattern
					return context.tree, context
				}
			} else {
				if nil != (*current)[wildcard] {
					wild = (*current)[wildcard]
					blob := strings.Join(components[index:], "")
					context.Params[wild.name] = blob[:len(blob)-1]
				}
				if nil != (*current)[dynamic] {
					key = dynamic
					// all components have trailing slashes because of sanitize
					// so the final character needs to be dropped
					context.Params[(*current)[key].name] = component[:len(component)-1]
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
			context.Pattern = context.tree.pattern
			return context.tree, context
		} else {
			current = &(*current)[key].children
			if nil != (*current)[wildcard] {
				wild = (*current)[wildcard]
				blob := strings.Join(components[index:], "")
				context.Params[wild.name] = blob[:len(blob)-1]
			}
		}
	}
	return nil, context
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
			unreachable = true // after the first non bear.HandlerFunc handler, all other handlers will be unreachable
			listener := fn.(http.HandlerFunc)
			handlers = append(handlers, HandlerFunc(func(res http.ResponseWriter, req *http.Request, ctx *Context) {
				listener(res, req)
			}))
		case func(http.ResponseWriter, *http.Request):
			if unreachable {
				err = fmt.Errorf("bear: %s %s has unreachable middleware", verb, pattern)
				return
			}
			unreachable = true // after the first non bear.HandlerFunc handler, all other handlers will be unreachable
			listener := http.HandlerFunc(fn.(func(http.ResponseWriter, *http.Request)))
			handlers = append(handlers, HandlerFunc(func(res http.ResponseWriter, req *http.Request, ctx *Context) {
				listener(res, req)
			}))
		default:
			err = fmt.Errorf("bear: handler needs to match http.HandlerFunc OR bear.HandlerFunc")
			return
		}
	}
	return
}
func initialize(tr **tree) *tree {
	if nil == *tr {
		*tr = new(tree)
	}
	return *tr
}
func sanitize(s string) string {
	// be nice about empty strings
	if s == "" {
		return slash
	}
	// always prefix paths from root
	if !strings.HasPrefix(s, slash) {
		s = slash + s
	}
	// always end with slash
	if !strings.HasSuffix(s, slash) {
		s = s + slash
	}
	// replace double slashes
	s = strings.Replace(s, slash+slash, slash, -1)
	return s
}
func set(verb string, tr *tree, pattern string, handlers []HandlerFunc) error {
	var (
		components []string = split(pattern)
		current    *treemap = &tr.children
		last       int      = len(components) - 1
	)
	if 0 == last {
		if nil != tr.handlers {
			return fmt.Errorf("bear: %s %s exists, ignoring", verb, pattern)
		} else {
			tr.pattern = pattern
			tr.handlers = handlers
			return nil
		}
	}
	if nil == tr.children {
		tr.children = make(treemap)
	}
	dyn := regexp.MustCompile(word)
	components, last = components[1:], last-1 // ignore the initial "/" component
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
			(*current)[key] = &tree{children: make(treemap), name: name}
		}
		if index == last {
			if nil != (*current)[key].handlers {
				return fmt.Errorf("bear: %s %s exists, ignoring", verb, pattern)
			}
			(*current)[key].pattern = pattern
			(*current)[key].handlers = handlers
			return nil
		} else if key == wildcard {
			return fmt.Errorf("bear: %s %s wildcard (%s) tokens must be the final token", verb, pattern, asterisk)
		}
		current = &(*current)[key].children
	}
	return nil
}
func split(s string) (components []string) {
	for _, component := range strings.SplitAfter(sanitize(s), "/") {
		if len(component) > 0 {
			components = append(components, component)
		}
	}
	return
}

// Next calls the next middleware (if any) that was registered as a handler for
// a particular request pattern.
func (ctx *Context) Next(res http.ResponseWriter, req *http.Request) {
	ctx.handler++
	if len(ctx.tree.handlers) > ctx.handler {
		ctx.tree.handlers[ctx.handler](res, req, ctx)
	}
}

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
		tr = initialize(&mux.connect)
	case "DELETE":
		tr = initialize(&mux.delete)
	case "GET":
		tr = initialize(&mux.get)
	case "HEAD":
		tr = initialize(&mux.head)
	case "OPTIONS":
		tr = initialize(&mux.options)
	case "POST":
		tr = initialize(&mux.post)
	case "PUT":
		tr = initialize(&mux.put)
	case "TRACE":
		tr = initialize(&mux.trace)
	}
	if fns, err := handlerize(verb, pattern, handlers); err != nil {
		return err
	} else {
		return set(verb, tr, pattern, fns)
	}
}

// ServeHTTP allows a Mux instance to conform to the http.Handler interface.
func (mux *Mux) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	switch req.Method {
	default:
		http.NotFound(res, req)
	case "CONNECT":
		deploy(mux.connect, res, req)
	case "DELETE":
		deploy(mux.delete, res, req)
	case "GET":
		deploy(mux.get, res, req)
	case "HEAD":
		deploy(mux.head, res, req)
	case "OPTIONS":
		deploy(mux.options, res, req)
	case "POST":
		deploy(mux.post, res, req)
	case "PUT":
		deploy(mux.put, res, req)
	case "TRACE":
		deploy(mux.trace, res, req)
	}
}

// New returns a pointer to a bear Mux multiplexer
func New() *Mux { return new(Mux) }
