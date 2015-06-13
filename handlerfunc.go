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

func handlerizeLax(verb string, pattern string,
	functions []interface{}) (handlers []HandlerFunc, err error) {
	unreachable := false
	for _, function := range functions {
		switch function.(type) {
		case HandlerFunc:
			if unreachable {
				err = fmt.Errorf("bear: %s %s has unreachable middleware",
					verb, pattern)
				return
			}
			handler := function.(HandlerFunc)
			if handler == nil {
				err = fmt.Errorf("bear: %s %s has nil middleware",
					verb, pattern)
				return
			} else {
				handlers = append(handlers, HandlerFunc(handler))
			}
		case func(http.ResponseWriter, *http.Request, *Context):
			if unreachable {
				err = fmt.Errorf("bear: %s %s has unreachable middleware",
					verb, pattern)
				return
			}
			handler := function.(func(http.ResponseWriter, *http.Request, *Context))
			if handler == nil {
				err = fmt.Errorf("bear: %s %s has nil middleware",
					verb, pattern)
				return
			} else {
				handlers = append(handlers, HandlerFunc(handler))
			}
		case http.HandlerFunc:
			if unreachable {
				err = fmt.Errorf("bear: %s %s has unreachable middleware",
					verb, pattern)
				return
			}
			handler := function.(http.HandlerFunc)
			if handler == nil {
				err = fmt.Errorf("bear: %s %s has nil middleware",
					verb, pattern)
				return
			} else {
				// after non HandlerFunc handlers other handlers are unreachable
				unreachable = true
				handlers = append(handlers, HandlerFunc(func(
					res http.ResponseWriter, req *http.Request, _ *Context) {
					handler(res, req)
				}))
			}
		case func(http.ResponseWriter, *http.Request):
			if unreachable {
				err = fmt.Errorf("bear: %s %s has unreachable middleware",
					verb, pattern)
				return
			}
			handler := function.(func(http.ResponseWriter, *http.Request))
			if handler == nil {
				err = fmt.Errorf("bear: %s %s has nil middleware",
					verb, pattern)
				return
			} else {
				// after non HandlerFunc handlers other handlers are unreachable
				unreachable = true
				handlers = append(handlers, HandlerFunc(func(
					res http.ResponseWriter, req *http.Request, _ *Context) {
					handler(res, req)
				}))
			}
		default:
			err = fmt.Errorf(
				"bear: handler must match http.HandlerFunc OR bear.HandlerFunc")
			return
		}
	}
	return
}

func handlerizeStrict(functions []interface{}) (handlers []HandlerFunc, err error) {
	for _, function := range functions {
		switch function.(type) {
		case HandlerFunc:
			handler := function.(HandlerFunc)
			if handler == nil {
				return nil, fmt.Errorf("bear: nil middleware")
			} else {
				handlers = append(handlers, HandlerFunc(handler))
			}
		case func(http.ResponseWriter, *http.Request, *Context):
			handler := function.(func(http.ResponseWriter, *http.Request, *Context))
			if handler == nil {
				return nil, fmt.Errorf("bear: nil middleware")
			} else {
				handlers = append(handlers, HandlerFunc(handler))
			}
		default:
			return nil, fmt.Errorf(
				"bear: handler must be a bear.HandlerFunc or match its signature")
		}
	}
	return handlers, nil
}
