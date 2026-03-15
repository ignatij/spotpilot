package auth

// Session holds a persisted Spotify browser session.
type Session struct {
	Cookies []Cookie
}

// Cookie is a browser session cookie.
type Cookie struct {
	Name  string
	Value string
}
