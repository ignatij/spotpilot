package domain

import "errors"

// ErrNotFound is returned when no matching track, album, or artist is found.
var ErrNotFound = errors.New("not found")

// ErrNotLoggedIn is returned when no valid session exists.
var ErrNotLoggedIn = errors.New("not logged in")
