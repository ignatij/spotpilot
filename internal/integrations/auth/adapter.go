package auth

import (
	"context"

	"github.com/ignatij/spotpilot/internal/app"
)

// AppStoreAdapter adapts a CredentialStore to the app.SessionStore port.
type AppStoreAdapter struct {
	inner CredentialStore
}

// NewAppStoreAdapter wraps a CredentialStore to satisfy app.SessionStore.
func NewAppStoreAdapter(inner CredentialStore) *AppStoreAdapter {
	return &AppStoreAdapter{inner: inner}
}

func (a *AppStoreAdapter) Load(ctx context.Context) (*app.Session, error) {
	sess, err := a.inner.Load(ctx)
	if err != nil {
		return nil, err
	}
	if sess == nil {
		return nil, nil
	}
	out := &app.Session{}
	for _, c := range sess.Cookies {
		out.Cookies = append(out.Cookies, app.Cookie{Name: c.Name, Value: c.Value})
	}
	return out, nil
}

func (a *AppStoreAdapter) Save(ctx context.Context, s *app.Session) error {
	inner := &Session{}
	for _, c := range s.Cookies {
		inner.Cookies = append(inner.Cookies, Cookie{Name: c.Name, Value: c.Value})
	}
	return a.inner.Save(ctx, inner)
}

func (a *AppStoreAdapter) Clear(ctx context.Context) error {
	return a.inner.Clear(ctx)
}
