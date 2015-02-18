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
	trees [8]*tree // pointers to a tree for each HTTP verb
	wild  [8]bool  // true if a tree has a wildcard (requires back-references)
}

func parsePath(s string) (components []string, last int) {
	start, offset := 0, 0
	if slashr == s[0] {
		start = 1
	}
	if slashr == s[len(s)-1] {
		offset = 1
	}
	components = strings.SplitAfter(s, slash)
	if start == 1 || offset == 1 {
		components = components[start : len(components)-offset]
	}
	last = len(components) - 1
	if offset == 0 {
		components[last] = components[last] + slash
	}
	return components, last
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
	tr, wildcards := mux.tree(verb)
	if nil == tr {
		return fmt.Errorf("bear: %s isn't a valid HTTP verb", verb)
	}
	if fns, err := handlerize(verb, pattern, handlers); err != nil {
		return err
	} else {
		tr.set(verb, pattern, fns, wildcards, &err)
		return err
	}
}

// ServeHTTP allows a Mux instance to conform to the http.Handler interface.
func (mux *Mux) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	tr, wildcards := mux.tree(req.Method)
	if nil == tr { // if req.Method is not found in HTTP verbs
		http.NotFound(res, req)
		return
	}
	// root is a special case because it is the top node in the tree
	if req.URL.Path == slash || req.URL.Path == empty {
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
	components, last := parsePath(req.URL.Path)
	capacity := last + 1 // maximum number of params possible for this request
	context := new(Context)
	current := &tr.children
	if !*wildcards { // no wildcards: simpler, slightly faster
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
			if nil == (*current)[key] {
				if nil == (*current)[dynamic] && nil == (*current)[wildcard] {
					if nil == wild { // there's no wildcard up the tree
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
				wild = (*current)[wildcard] // there's a more proximate wildcard
				context.param(asterisk,
					strings.Join(components[index:], empty), capacity)
			}
		}
	}
}

func (mux *Mux) tree(name string) (*tree, *bool) {
	switch name {
	case "CONNECT":
		return mux.trees[0], &mux.wild[0]
	case "DELETE":
		return mux.trees[1], &mux.wild[1]
	case "GET":
		return mux.trees[2], &mux.wild[2]
	case "HEAD":
		return mux.trees[3], &mux.wild[3]
	case "OPTIONS":
		return mux.trees[4], &mux.wild[4]
	case "POST":
		return mux.trees[5], &mux.wild[5]
	case "PUT":
		return mux.trees[6], &mux.wild[6]
	case "TRACE":
		return mux.trees[7], &mux.wild[7]
	default:
		return nil, nil
	}
}

// New returns a pointer to a bear Mux multiplexer
func New() *Mux {
	return &Mux{
		[8]*tree{
			&tree{}, &tree{}, &tree{}, &tree{},
			&tree{}, &tree{}, &tree{}, &tree{}},
		[8]bool{
			false, false, false, false,
			false, false, false, false}}
}
