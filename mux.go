// Copyright 2015 Afshin Darian. All rights reserved.
// Use of this source code is governed by The MIT License
// that can be found in the LICENSE file.

package bear

import (
	"fmt"
	"net/http"
	"strings"
)

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

func set(verb string, tr *tree, pattern string,
	handlers []HandlerFunc) (wild bool, err error) {
	if pattern == slash {
		if nil != tr.handlers {
			return false, fmt.Errorf("bear: %s %s exists, ignoring", verb,
				pattern)
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
	// first token is "/" and last token is ""
	components = components[1 : len(components)-1]
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
			(*current)[key] = &tree{
				children: make(map[string]*tree), name: name}
		}
		if index == last {
			if nil != (*current)[key].handlers {
				return false, fmt.Errorf("bear: %s %s exists, ignoring", verb,
					pattern)
			}
			(*current)[key].pattern = pattern
			(*current)[key].handlers = handlers
			return wild, nil
		} else if key == wildcard {
			return false, fmt.Errorf(
				"bear: %s %s wildcard (%s) token must be last",
				verb, pattern, asterisk)
		}
		current = &(*current)[key].children
	}
	return wild, nil
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
	// root is a special case because it is the top node in the tree
	if req.URL.Path == slash {
		if nil != tr.handlers { // root match
			tr.handlers[0](res, req, &Context{tree: tr})
			return
		} else if wild := tr.children[wildcard]; nil != wild {
			// root level wildcard pattern match
			wild.handlers[0](res, req, &Context{tree: wild})
			return
		}
		http.NotFound(res, req)
		return
	}
	var key string
	components := strings.SplitAfter(sanitize(req.URL.Path), slash)
	// first token is "/" and last token is ""
	components = components[1 : len(components)-1]
	context := new(Context)
	current := &tr.children
	// maximum number of URL params possible for this request
	capacity := len(components)
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
					if nil != (*current)[wildcard] {
						// i.e. there is a more proximate wildcard
						wild = (*current)[wildcard]
						context.param(asterisk,
							strings.Join(components[index:], empty), capacity)
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
			if nil != (*current)[wildcard] {
				// i.e. there is a more proximate wildcard
				wild = (*current)[wildcard]
				context.param(asterisk,
					strings.Join(components[index:], empty), capacity)
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
	return &Mux{
		&tree{}, &tree{}, &tree{}, &tree{}, &tree{}, &tree{}, &tree{}, &tree{},
		false}
}
