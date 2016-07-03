// Copyright 2015 Afshin Darian. All rights reserved.
// Use of this source code is governed by The MIT License
// that can be found in the LICENSE file.

// Package bear provides HTTP multiplexing with dynamic URL components and
// request contexts to form the nucleus of a middleware-based web service.
package bear

import "regexp"

const (
	asterisk  = "*"
	dynamic   = "\x00"
	empty     = ""
	lasterisk = "*/"
	slash     = "/"
	slashr    = '/'
	wildcard  = "\x00\x00"
)

var (
	dyn   = regexp.MustCompile(`\{(\w+)\}`)
	dbl   = regexp.MustCompile(`[\/]{2,}`)
	verbs = [8]string{
		"CONNECT",
		"DELETE",
		"GET",
		"HEAD",
		"OPTIONS",
		"POST",
		"PUT",
		"TRACE",
	}
)
