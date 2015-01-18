// Package bear (bear embeddedable application router) is an HTTP multiplexer.
// It uses a tree structure for fast routing, supports dynamic parameters,
// and accepts handlers that are very similar to the native http.HandlerFunc
// (except they accept an extra Params map[string]string argument)
package bear

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strings"
)

const dynamic = "\x00"
const slash = "/"

// HandlerFunc is very similar to net/http HandlerFunc, except it requires
// an extra Params map[string]string argument
type HandlerFunc func(http.ResponseWriter, *http.Request, Params)

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

// Params is a map of string keys with string values that is populated
// by the dynamic URL parameters, received by functions of type HandlerFunc
type Params map[string]string

type tree struct {
	children treemap
	dynamic  bool
	handler  HandlerFunc
	name     string
}
type treemap map[string]*tree

func deploy(t *tree, res http.ResponseWriter, req *http.Request) {
	if nil == t {
		http.NotFound(res, req)
	} else {
		location, params := find(t, req.URL.Path)
		if nil == location || nil == location.handler {
			http.NotFound(res, req)
		} else {
			location.handler(res, req, params)
		}
	}
}

func find(t *tree, path string) (*tree, Params) {
	var (
		components []string = split(path)
		current    *treemap = &t.children
		params     Params   = make(Params)
		last       int      = len(components) - 1
	)
	if 0 == last { // i.e. location is /
		return t, params
	}
	components, last = components[1:], last-1 // ignore the initial "/" component
	for index, component := range components {
		key := component
		if nil == *current {
			return nil, nil
		}
		if nil == (*current)[key] {
			if nil == (*current)[dynamic] {
				return nil, nil
			} else {
				key = dynamic
				// all components have trailing slashes because of sanitize
				// so the final character needs to be dropped
				params[(*current)[key].name] = component[:len(component)-1]
			}
		}
		if index == last {
			return (*current)[key], params
		}
		current = &(*current)[key].children
	}
	return nil, nil
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
func set(verb string, t *tree, pattern string, handler HandlerFunc) error {
	var (
		components []string = split(pattern)
		current    *treemap = &t.children
		last       int      = len(components) - 1
	)
	if 0 == last {
		if nil != t.handler {
			err := "bear: " + verb + " " + pattern + " exists, ignoring"
			log.Println(err)
			return errors.New(err)
		} else {
			t.handler = handler
			return nil
		}
	}
	if nil == t.children {
		t.children = make(treemap)
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
// It returns an error if it fails, but does not panic.
func (m *Mux) On(verb string, pattern string, handler HandlerFunc) error {
	var t *tree
	switch verb {
	default:
		return errors.New(fmt.Sprintf("bear: %s isn't a valid HTTP verb", verb))
	case "CONNECT":
		if nil == m.connect {
			m.connect = &tree{}
		}
		t = m.connect
	case "DELETE":
		if nil == m.delete {
			m.delete = &tree{}
		}
		t = m.delete
	case "GET":
		if nil == m.get {
			m.get = &tree{}
		}
		t = m.get
	case "HEAD":
		if nil == m.head {
			m.head = &tree{}
		}
		t = m.head
	case "OPTIONS":
		if nil == m.options {
			m.options = &tree{}
		}
		t = m.options
	case "POST":
		if nil == m.post {
			m.post = &tree{}
		}
		t = m.post
	case "PUT":
		if nil == m.put {
			m.put = &tree{}
		}
		t = m.put
	case "TRACE":
		if nil == m.trace {
			m.trace = &tree{}
		}
		t = m.trace
	}
	return set(verb, t, pattern, handler)
}

// ServeHTTP allows a Mux instance to conform to http.Handler interface.
func (m *Mux) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	switch req.Method {
	default:
		http.NotFound(res, req)
	case "CONNECT":
		deploy(m.get, res, req)
	case "DELETE":
		deploy(m.get, res, req)
	case "GET":
		deploy(m.get, res, req)
	case "HEAD":
		deploy(m.get, res, req)
	case "OPTIONS":
		deploy(m.get, res, req)
	case "POST":
		deploy(m.get, res, req)
	case "PUT":
		deploy(m.get, res, req)
	case "TRACE":
		deploy(m.get, res, req)
	}
}

// returns a reference to a bear Mux multiplexer
func New() *Mux {
	return new(Mux)
}
