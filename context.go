// Copyright 2015 Afshin Darian. All rights reserved.
// Use of this source code is governed by The MIT License
// that can be found in the LICENSE file.

package bear

import (
	"net/http"
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
// It returns a pointer to the current Context to allow chaining.
func (ctx *Context) Set(key string, value interface{}) *Context {
	if nil == ctx.state {
		ctx.state = make(map[string]interface{})
	}
	ctx.state[key] = value
	return ctx
}

// Pattern returns the URL pattern that a request matched.
func (ctx *Context) Pattern() string { return ctx.tree.pattern }
