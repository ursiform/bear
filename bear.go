// Copyright 2015 Afshin Darian. All rights reserved.
// Use of this source code is governed by The MIT License
// that can be found in the LICENSE file.

/*
Package bear (bear embeddable application router) is an HTTP multiplexer.
It uses a tree structure for fast routing, supports dynamic parameters,
middleware, and accepts both native http.HandlerFunc or bear.HandlerFunc
(which accepts an extra *Context argument that allows storing State and
calling the Next middleware)
*/
package bear

import "regexp"

const ( // global constants
	asterisk    = "*"
	doubleslash = "//"
	dynamic     = "\x00"
	empty       = ""
	lasterisk   = "*/"
	slash       = "/"
	wildcard    = "\x00\x00"
)

var ( // global variables
	dyn   *regexp.Regexp = regexp.MustCompile(`\{(\w+)\}`)
	verbs [8]string      = [8]string{
		"CONNECT", "DELETE", "GET", "HEAD", "OPTIONS", "POST", "PUT", "TRACE"}
)
