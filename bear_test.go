package bear

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

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
func TestNotFoundCustom(t *testing.T) {
	var (
		method       string = "GET"
		mux          *Mux   = New()
		pathFound    string = "/foo/bar"
		pathLost     string = "/foo/bar/baz"
		patternFound string = "/foo/bar"
		patternLost  string = "/*"
		req          *http.Request
		res          *httptest.ResponseRecorder
		wantFound    int = http.StatusOK
		wantLost     int = http.StatusTeapot
	)
	handlerFound := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		res.WriteHeader(http.StatusOK)
		res.Write([]byte("found!"))
	})
	handlerLost := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		res.WriteHeader(http.StatusTeapot)
		res.Write([]byte("not found!"))
	})
	// test found to make sure wildcard doesn't overtake everything
	req, _ = http.NewRequest(method, pathFound, nil)
	res = httptest.NewRecorder()
	mux.On(method, patternFound, handlerFound)
	mux.ServeHTTP(res, req)
	if res.Code != wantFound {
		t.Errorf("%s %s (%s) got %d want %d", method, pathFound, patternFound, res.Code, wantFound)
	}
	// test lost to make sure wildcard can gets non-pattern-matching paths
	req, _ = http.NewRequest(method, pathLost, nil)
	res = httptest.NewRecorder()
	mux.On(method, patternLost, handlerLost)
	mux.ServeHTTP(res, req)
	if res.Code != wantLost {
		t.Errorf("%s %s (%s) got %d want %d", method, pathLost, patternLost, res.Code, wantLost)
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
func TestWildcardCompeting(t *testing.T) {
	var (
		method       string = "GET"
		mux          *Mux   = New()
		patternOne   string = "/*"
		pathOne      string = "/bar/baz"
		wantOne      string = "bar/baz"
		patternTwo   string = "/foo/*"
		pathTwo      string = "/foo/baz"
		wantTwo      string = "baz"
		patternThree string = "/foo/bar/*"
		pathThree    string = "/foo/bar/bar/baz"
		wantThree    string = "bar/baz"
		req          *http.Request
		res          *httptest.ResponseRecorder
	)
	handler := func(res http.ResponseWriter, req *http.Request, ctx *Context) {
		res.WriteHeader(http.StatusOK)
		res.Write([]byte(ctx.Params["*"]))
	}
	mux.On(method, patternOne, handler)
	mux.On(method, patternTwo, handler)
	mux.On(method, patternThree, handler)
	req, _ = http.NewRequest(method, pathOne, nil)
	res = httptest.NewRecorder()
	mux.ServeHTTP(res, req)
	if body := res.Body.String(); body != wantOne {
		t.Errorf("%s %s (%s) got %s want %s", method, pathOne, patternOne, body, wantOne)
	}
	req, _ = http.NewRequest(method, pathTwo, nil)
	res = httptest.NewRecorder()
	mux.ServeHTTP(res, req)
	if body := res.Body.String(); body != wantTwo {
		t.Errorf("%s %s (%s) got %s want %s", method, pathTwo, patternTwo, body, wantTwo)
	}
	req, _ = http.NewRequest(method, pathThree, nil)
	res = httptest.NewRecorder()
	mux.ServeHTTP(res, req)
	if body := res.Body.String(); body != wantThree {
		t.Errorf("%s %s (%s) got %s want %s", method, pathThree, patternThree, body, wantThree)
	}
}
func TestWildcardParams(t *testing.T) {
	var (
		method  string = "GET"
		mux     *Mux   = New()
		pattern string = "/foo/{bar}/*"
		path    string = "/foo/ABC/baz"
		want    string = "ABC"
		req     *http.Request
		res     *httptest.ResponseRecorder
	)
	handler := func(res http.ResponseWriter, req *http.Request, ctx *Context) {
		res.WriteHeader(http.StatusOK)
		res.Write([]byte(ctx.Params["bar"]))
	}
	mux.On(method, pattern, handler)
	req, _ = http.NewRequest(method, path, nil)
	res = httptest.NewRecorder()
	mux.ServeHTTP(res, req)
	if body := res.Body.String(); body != want {
		t.Errorf("%s %s (%s) got %s want %s", method, path, pattern, body, want)
	}
}
