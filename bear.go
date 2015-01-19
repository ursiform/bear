// Copyright 2015 Afshin Darian. All rights reserved.
// Use of this source code is governed by The MIT License
// that can be found in the LICENSE file.

/*
	Package bear (bear embeddedable application router) is an HTTP multiplexer.
	It uses a tree structure for fast routing, supports dynamic parameters,
	and accepts both native http.HandlerFunc or bear.HandlerFunc
	(which accepts an extra Context argument)
*/
package bear

import (
	"errors"
	"log"
	"net/http"
	"regexp"
	"strings"
)

const dynamic = "\x00"
const slash = "/"

// HandlerFunc is very similar to net/http HandlerFunc, except it requires
// an extra argument for the Context of a request
type HandlerFunc func(http.ResponseWriter, *http.Request, *Context)

// Mux holds references to separate trees for each HTTP verb
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

// Context is a struct that contains:
// Params: a map of string keys with string values that is populated
// by the dynamic URL parameters (if any)
// Pattern: the URL pattern string that was matched by a given request
type Context struct {
	Params  map[string]string
	Pattern string
}

type tree struct {
	children treemap
	dynamic  bool
	handler  HandlerFunc
	name     string
	pattern  string
}
type treemap map[string]*tree

func deploy(tr *tree, res http.ResponseWriter, req *http.Request) {
	if nil == tr {
		http.NotFound(res, req)
	} else {
		location, context := find(tr, req.URL.Path)
		if nil == location || nil == location.handler {
			http.NotFound(res, req)
		} else {
			location.handler(res, req, context)
		}
	}
}
func find(tr *tree, path string) (*tree, *Context) {
	var (
		components []string = split(path)
		current    *treemap = &tr.children
		context    *Context = &Context{Params: make(map[string]string)}
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
			context.Pattern = (*current)[key].pattern
			return (*current)[key], context
		}
		current = &(*current)[key].children
	}
	return nil, context
}
func handle(fn interface{}) (handler HandlerFunc, err error) {
	if _, ok := fn.(HandlerFunc); ok {
		handler = HandlerFunc(fn.(HandlerFunc))
	} else if _, ok := fn.(func(http.ResponseWriter, *http.Request, *Context)); ok {
		handler = HandlerFunc(fn.(func(http.ResponseWriter, *http.Request, *Context)))
	} else if _, ok := fn.(http.HandlerFunc); ok {
		listener := fn.(http.HandlerFunc)
		handler = HandlerFunc(func(res http.ResponseWriter, req *http.Request, ctx *Context) {
			listener(res, req)
		})
	} else if _, ok := fn.(func(http.ResponseWriter, *http.Request)); ok {
		listener := http.HandlerFunc(fn.(func(http.ResponseWriter, *http.Request)))
		handler = HandlerFunc(func(res http.ResponseWriter, req *http.Request, ctx *Context) {
			listener(res, req)
		})
	} else {
		err = errors.New("bear: handler needs to match http.HandlerFunc OR bear.HandlerFunc")
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
func set(verb string, tr *tree, pattern string, handler HandlerFunc) error {
	var (
		components []string = split(pattern)
		current    *treemap = &tr.children
		last       int      = len(components) - 1
	)
	if 0 == last {
		if nil != tr.handler {
			err := "bear: " + verb + " " + pattern + " exists, ignoring"
			log.Println(err)
			return errors.New(err)
		} else {
			tr.pattern = pattern
			tr.handler = handler
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
			if nil != (*current)[key].handler {
				err := "bear: " + verb + " " + pattern + " exists, ignoring"
				log.Println(err)
				return errors.New(err)
			}
			(*current)[key].pattern = pattern
			(*current)[key].handler = handler
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

// On adds an HTTP verb handler for a URL pattern.
// The third argument should either be an http.HandlerFunc or a bear.HandlerFunc
// or conform to the signature of one of those two.
// It returns an error if it fails, but does not panic.
func (mux *Mux) On(verb string, pattern string, handler interface{}) error {
	var tr *tree
	switch verb {
	default:
		return errors.New("bear: " + verb + " isn't a valid HTTP verb")
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
	if listener, err := handle(handler); err != nil {
		log.Println(err)
		return err
	} else {
		return set(verb, tr, pattern, listener)
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
