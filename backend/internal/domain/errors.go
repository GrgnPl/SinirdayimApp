package domain

import "errors"

var ErrNotFound = errors.New("not found")

// ErrNoRoute means no truck route exists between the requested points.
var ErrNoRoute = errors.New("no route")
