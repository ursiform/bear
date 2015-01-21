package bear

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

var verbs []string = []string{"CONNECT", "DELETE", "GET", "HEAD", "OPTIONS", "POST", "PUT", "TRACE"}

type tester func(*testing.T)

// generates tests for param requests using bear.HandlerFunc
func paramBearTest(label string, method string, path string, pattern string, want map[string]string) tester {
	return func(t *testing.T) {
		var (
			mux *Mux = New()
			req *http.Request
			res *httptest.ResponseRecorder
		)
		req, _ = http.NewRequest(method, path, nil)
		res = httptest.NewRecorder()
		mux.On(method, pattern, HandlerFunc(func(res http.ResponseWriter, req *http.Request, ctx *Context) {
			if !reflect.DeepEqual(want, ctx.Params) {
				t.Errorf("%s %s (%s) %s got %v want %v", method, path, pattern, label, ctx.Params, want)
			}
		}))
		mux.ServeHTTP(res, req)
		if res.Code != http.StatusOK {
			t.Errorf("%s %s (%s) %s got %d want %d", method, path, pattern, label, res.Code, http.StatusOK)
		}
	}
}

// generates tests for param requests using anonymous bear.HandlerFunc compatible functions
func paramBearAnonTest(label string, method string, path string, pattern string, want map[string]string) tester {
	return func(t *testing.T) {
		var (
			mux *Mux = New()
			req *http.Request
			res *httptest.ResponseRecorder
		)
		req, _ = http.NewRequest(method, path, nil)
		res = httptest.NewRecorder()
		mux.On(method, pattern, func(res http.ResponseWriter, req *http.Request, ctx *Context) {
			if !reflect.DeepEqual(want, ctx.Params) {
				t.Errorf("%s %s (%s) %s got %v want %v", method, path, pattern, label, ctx.Params, want)
			}
		})
		mux.ServeHTTP(res, req)
		if res.Code != http.StatusOK {
			t.Errorf("%s %s (%s) %s got %d want %d", method, path, pattern, label, res.Code, http.StatusOK)
		}
	}
}

// generates tests for param AND no-param requests (i.e. no *Context) using http.HandlerFunc
func simpleHttpTest(label string, method string, path string, pattern string, want int) tester {
	return func(t *testing.T) {
		var (
			mux *Mux = New()
			req *http.Request
			res *httptest.ResponseRecorder
		)
		req, _ = http.NewRequest(method, path, nil)
		res = httptest.NewRecorder()
		mux.On(method, pattern, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
		mux.ServeHTTP(res, req)
		if res.Code != want {
			t.Errorf("%s %s (%s) %s got %d want %d", method, path, pattern, label, res.Code, want)
		}
	}
}

// generates tests for param AND no-param requests (i.e. no *Context) using anonymous http.HandlerFunc compatible func
func simpleHttpAnonTest(label string, method string, path string, pattern string, want int) tester {
	return func(t *testing.T) {
		var (
			mux *Mux = New()
			req *http.Request
			res *httptest.ResponseRecorder
		)
		req, _ = http.NewRequest(method, path, nil)
		res = httptest.NewRecorder()
		mux.On(method, pattern, func(http.ResponseWriter, *http.Request) {})
		mux.ServeHTTP(res, req)
		if res.Code != want {
			t.Errorf("%s %s (%s) %s got %d want %d", method, path, pattern, label, res.Code, want)
		}
	}
}

// generates tests for simple no-param requests using bear.HandlerFunc
func simpleBearTest(label string, method string, path string, pattern string, want int) tester {
	return func(t *testing.T) {
		var (
			mux *Mux = New()
			req *http.Request
			res *httptest.ResponseRecorder
		)
		req, _ = http.NewRequest(method, path, nil)
		res = httptest.NewRecorder()
		mux.On(method, pattern, HandlerFunc(func(http.ResponseWriter, *http.Request, *Context) {}))
		mux.ServeHTTP(res, req)
		if res.Code != want {
			t.Errorf("%s %s (%s) %s got %d want %d", method, path, pattern, label, res.Code, want)
		}
	}
}

// generates tests for simple no-param requests using anonymous bear.HandlerFunc compatible functions
func simpleBearAnonTest(label string, method string, path string, pattern string, want int) tester {
	return func(t *testing.T) {
		var (
			mux *Mux = New()
			req *http.Request
			res *httptest.ResponseRecorder
		)
		req, _ = http.NewRequest(method, path, nil)
		res = httptest.NewRecorder()
		mux.On(method, pattern, func(http.ResponseWriter, *http.Request, *Context) {})
		mux.ServeHTTP(res, req)
		if res.Code != want {
			t.Errorf("%s %s (%s) %s got %d want %d", method, path, pattern, label, res.Code, want)
		}
	}
}

func TestDuplicateFailure(t *testing.T) {
	var (
		handler HandlerFunc = HandlerFunc(func(http.ResponseWriter, *http.Request, *Context) {})
		mux     *Mux        = New()
		pattern string      = "/foo/{bar}"
	)
	for _, verb := range verbs {
		if err := mux.On(verb, pattern, handler); err != nil {
			t.Error(err)
		} else if err := mux.On(verb, pattern, handler); err == nil {
			t.Errorf("%s %s addition should have failed because it is a duplicate", verb, pattern)
		}
	}
}
func TestMiddleware(t *testing.T) {
	var (
		middlewares int                    = 3
		mux         *Mux                   = New()
		params      map[string]string      = map[string]string{"bar": "BAR", "qux": "QUX"}
		path        string                 = "/foo/BAR/baz/QUX"
		pattern     string                 = "/foo/{bar}/baz/{qux}"
		state       map[string]interface{} = map[string]interface{}{"one": 1, "two": 2}
	)
	run := func(method string) {
		var (
			req     *http.Request
			res     *httptest.ResponseRecorder
			visited int = 0
		)
		one := func(res http.ResponseWriter, req *http.Request, ctx *Context) {
			visited++
			ctx.State["one"] = 1
			ctx.Next(res, req)
		}
		two := func(res http.ResponseWriter, req *http.Request, ctx *Context) {
			visited++
			ctx.State["two"] = 2
			ctx.Next(res, req)
		}
		last := func(res http.ResponseWriter, req *http.Request, ctx *Context) {
			visited++
			if !reflect.DeepEqual(params, ctx.Params) {
				t.Errorf("%s %s (%s) got %v want %v", method, path, pattern, ctx.Params, params)
			}
			if !reflect.DeepEqual(state, ctx.State) {
				t.Errorf("%s %s (%s) got %v want %v", method, path, pattern, ctx.State, state)
			}
		}
		req, _ = http.NewRequest(method, path, nil)
		res = httptest.NewRecorder()
		mux.On(method, pattern, one, two, last)
		mux.ServeHTTP(res, req)
		if visited != middlewares {
			t.Errorf("%s %s (%s) expected %d middlewares, visited %d", method, path, pattern, middlewares, visited)
		}
	}
	for _, verb := range verbs {
		run(verb)
	}
}
func TestMiddlewareRejection(t *testing.T) {
	var (
		mux     *Mux   = New()
		path    string = "/foo/BAR/baz/QUX"
		pattern string = "/foo/{bar}/baz/{qux}"
	)
	run := func(method string) {
		one := func(res http.ResponseWriter, req *http.Request, ctx *Context) { ctx.Next(res, req) }
		two := func(res http.ResponseWriter, req *http.Request) {}
		last := func(res http.ResponseWriter, req *http.Request, ctx *Context) {}
		err := mux.On(method, pattern, one, two, last)
		if err == nil {
			t.Errorf("%s %s (%s) middleware with wrong signature was accepted", method, path, pattern)
		}
	}
	for _, verb := range verbs {
		run(verb)
	}
}
func TestOKNoParams(t *testing.T) {
	var (
		path    string = "/foo/bar"
		pattern string = "/foo/bar"
		want    int    = http.StatusOK
	)
	for _, verb := range verbs {
		simpleHttpTest("http.HandlerFunc", verb, path, pattern, want)(t)
		simpleHttpAnonTest("anonymous http.HandlerFunc", verb, path, pattern, want)(t)
		simpleBearTest("bear.HandlerFunc", verb, path, pattern, want)(t)
		simpleBearAnonTest("anonymous http.HandlerFunc", verb, path, pattern, want)(t)
	}
}
func TestOKParams(t *testing.T) {
	var (
		path    string            = "/foo/BAR/baz/QUX"
		pattern string            = "/foo/{bar}/baz/{qux}"
		want    map[string]string = map[string]string{"bar": "BAR", "qux": "QUX"}
	)
	for _, verb := range verbs {
		simpleHttpTest("http.HandlerFunc", verb, path, pattern, http.StatusOK)(t)
		simpleHttpAnonTest("anonymous http.HandlerFunc", verb, path, pattern, http.StatusOK)(t)
		paramBearTest("bear.HandlerFunc", verb, path, pattern, want)(t)
		paramBearAnonTest("anonymous http.HandlerFunc", verb, path, pattern, want)(t)
	}
}
func TestOKRoot(t *testing.T) {
	var (
		path    string = "/"
		pattern string = "/"
		want    int    = http.StatusOK
	)
	for _, verb := range verbs {
		simpleHttpTest("http.HandlerFunc", verb, path, pattern, want)(t)
		simpleHttpAnonTest("anonymous http.HandlerFunc", verb, path, pattern, want)(t)
		simpleBearTest("bear.HandlerFunc", verb, path, pattern, want)(t)
		simpleBearAnonTest("anonymous http.HandlerFunc", verb, path, pattern, want)(t)
	}
}
func TestNotFoundNoParams(t *testing.T) {
	var (
		path    string = "/foo/bar"
		pattern string = "/foo"
		want    int    = http.StatusNotFound
	)
	for _, verb := range verbs {
		simpleHttpTest("http.HandlerFunc", verb, path, pattern, want)(t)
		simpleHttpAnonTest("anonymous http.HandlerFunc", verb, path, pattern, want)(t)
		simpleBearTest("bear.HandlerFunc", verb, path, pattern, want)(t)
		simpleBearAnonTest("anonymous http.HandlerFunc", verb, path, pattern, want)(t)
	}
}
func TestNotFoundParams(t *testing.T) {
	var (
		path    string = "/foo/BAR/baz"
		pattern string = "/foo/{bar}/baz/{qux}"
		want    int    = http.StatusNotFound
	)
	for _, verb := range verbs {
		simpleHttpTest("http.HandlerFunc", verb, path, pattern, want)(t)
		simpleHttpAnonTest("anonymous http.HandlerFunc", verb, path, pattern, want)(t)
		simpleBearTest("bear.HandlerFunc", verb, path, pattern, want)(t)
		simpleBearAnonTest("anonymous http.HandlerFunc", verb, path, pattern, want)(t)
	}
}
