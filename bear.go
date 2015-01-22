// Copyright 2015 Afshin Darian. All rights reserved.
// Use of this source code is governed by The MIT License
// that can be found in the LICENSE file.

/*
	Package bear (bear embeddable application router) is an HTTP multiplexer.
	It uses a tree structure for fast routing, supports dynamic parameters,
	middleware, and accepts both native http.HandlerFunc or bear.HandlerFunc
	(which accepts an extra Context argument that allows storing State and
	calling the Next middleware)
*/
package bear

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"
)

const dynamic = "\x00"
const slash = "/"

type Context struct {
	// Params is a map of string keys with string values that is populated
	// by the dynamic URL parameters (if any)
	Params map[string]string
	// Pattern is the URL pattern string that was matched by a given request
	Pattern string
	// State is a utility map of string keys and empty interface values
	// to allow one middleware to pass information to the next.
	State   map[string]interface{}
	handler int
	tree    *tree
}

// HandlerFunc is similar to net/http HandlerFunc, except it requires
// an extra argument for the Context of a request
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
	)
	if 0 == last { // i.e. location is /
		context.Pattern = tr.pattern
		return tr, context
	}
	components, last = components[1:], last-1 // ignore the initial "/" component
	for index, component := range components {
		key := component
		if nil == *current {
			return nil, context
		}
		if nil == (*current)[key] {
			if nil == (*current)[dynamic] {
				return nil, context
			} else {
				key = dynamic
				// all components have trailing slashes because of sanitize
				// so the final character needs to be dropped
				context.Params[(*current)[key].name] = component[:len(component)-1]
			}
		}
		if index == last {
			context.tree = (*current)[key]
			context.Pattern = (*current)[key].pattern
			return context.tree, context
		}
		current = &(*current)[key].children
	}
	return nil, context
}
func handlerize(verb string, pattern string, fns []interface{}) (handlers []HandlerFunc, err error) {
	var unreachable = false
	for _, fn := range fns {
		if _, ok := fn.(HandlerFunc); ok {
			if unreachable {
				err = fmt.Errorf("bear: %s %s has unreachable middleware", verb, pattern)
				return
			}
			handlers = append(handlers, HandlerFunc(fn.(HandlerFunc)))
		} else if _, ok := fn.(func(http.ResponseWriter, *http.Request, *Context)); ok {
			if unreachable {
				err = fmt.Errorf("bear: %s %s has unreachable middleware", verb, pattern)
				return
			}
			handlers = append(handlers, HandlerFunc(fn.(func(http.ResponseWriter, *http.Request, *Context))))
		} else if _, ok := fn.(http.HandlerFunc); ok {
			if unreachable {
				err = fmt.Errorf("bear: %s %s has unreachable middleware", verb, pattern)
				return
			}
			// after the first non bear.HandlerFunc handler, all other handlers will be unreachable
			unreachable = true
			listener := fn.(http.HandlerFunc)
			handlers = append(handlers, HandlerFunc(func(res http.ResponseWriter, req *http.Request, ctx *Context) {
				listener(res, req)
			}))
		} else if _, ok := fn.(func(http.ResponseWriter, *http.Request)); ok {
			if unreachable {
				err = fmt.Errorf("bear: %s %s has unreachable middleware", verb, pattern)
				return
			}
			// after the first non bear.HandlerFunc handler, all other handlers will be unreachable
			unreachable = true
			listener := http.HandlerFunc(fn.(func(http.ResponseWriter, *http.Request)))
			handlers = append(handlers, HandlerFunc(func(res http.ResponseWriter, req *http.Request, ctx *Context) {
				listener(res, req)
			}))
		} else {
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
	dyn := regexp.MustCompile(`\{(\w+)\}`)
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

// Next calls the next middleware (if any) that was registered as a handler for a particular request pattern.
func (ctx *Context) Next(res http.ResponseWriter, req *http.Request) {
	ctx.handler++
	if len(ctx.tree.handlers) > ctx.handler {
		ctx.tree.handlers[ctx.handler](res, req, ctx)
	}
}

// On adds HTTP verb handler(s) for a URL pattern.
// The handler argument(s) should either be http.HandlerFunc or bear.HandlerFunc
// or conform to the signature of one of those two.
// NOTE: if http.HandlerFunc (or a function conforming to its signature) is used
// no other handlers can FOLLOW it, i.e. it is not middleware
// It returns an error if it fails, but does not panic.
func (mux *Mux) On(verb string, pattern string, handlers ...interface{}) error {
	var tr *tree
	switch verb {
	default:
		return fmt.Errorf("bear: %s isn't a valid HTTP verb", verb)
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

// ServeHTTP allows a Mux instance to conform to http.Handler interface.
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

// New returns a reference to a bear Mux multiplexer
func New() *Mux {
	return new(Mux)
}
