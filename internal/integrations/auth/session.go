package auth

import "time"

// Session holds a persisted Spotify browser session.
type Session struct {
	Cookies     []Cookie
	AccessToken string
	TokenExpiry time.Time
}

// Cookie is a browser session cookie.
type Cookie struct {
	Name  string
	Value string
}
