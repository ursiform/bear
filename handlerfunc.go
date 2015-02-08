// Copyright 2015 Afshin Darian. All rights reserved.
// Use of this source code is governed by The MIT License
// that can be found in the LICENSE file.

package bear

import (
	"fmt"
	"net/http"
)

// HandlerFunc is similar to http.HandlerFunc, except it requires
// an extra argument for the *Context of a request
type HandlerFunc func(http.ResponseWriter, *http.Request, *Context)

func handlerize(verb string, pattern string,
	fns []interface{}) (handlers []HandlerFunc, err error) {
	unreachable := false
	for _, fn := range fns {
		switch fn.(type) {
		case HandlerFunc:
			if unreachable {
				err = fmt.Errorf("bear: %s %s has unreachable middleware", verb,
					pattern)
				return
			}
			handlers = append(handlers, HandlerFunc(fn.(HandlerFunc)))
		case func(http.ResponseWriter, *http.Request, *Context):
			if unreachable {
				err = fmt.Errorf("bear: %s %s has unreachable middleware", verb,
					pattern)
				return
			}
			handlers = append(handlers, HandlerFunc(
				fn.(func(http.ResponseWriter, *http.Request, *Context))))
		case http.HandlerFunc:
			if unreachable {
				err = fmt.Errorf("bear: %s %s has unreachable middleware", verb,
					pattern)
				return
			}
			// after non HandlerFunc handlers, other handlers are unreachable
			unreachable = true
			handler := fn.(http.HandlerFunc)
			handlers = append(handlers, HandlerFunc(func(
				res http.ResponseWriter, req *http.Request, _ *Context) {
				handler(res, req)
			}))
		case func(http.ResponseWriter, *http.Request):
			if unreachable {
				err = fmt.Errorf("bear: %s %s has unreachable middleware", verb,
					pattern)
				return
			}
			// after non HandlerFunc handlers, other handlers are unreachable
			unreachable = true
			handler := fn.(func(http.ResponseWriter, *http.Request))
			handlers = append(handlers, HandlerFunc(func(
				res http.ResponseWriter, req *http.Request, _ *Context) {
				handler(res, req)
			}))
		default:
			err = fmt.Errorf(
				"bear: handler must match http.HandlerFunc OR bear.HandlerFunc")
			return
		}
	}
	return
}
