/*
Copyright 2023 Upbound Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package client

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"sync"
	"time"

	"github.com/crossplane/crossplane-runtime/v2/pkg/errors"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	"github.com/golang-jwt/jwt"
	"k8s.io/apimachinery/pkg/util/json"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/upbound/up-sdk-go"

	pcv1alpha1common "github.com/upbound/provider-upbound/apis/common/providerconfig/v1alpha1"
)

const (
	// UserAgent is the default user agent to use to make requests to the
	// Upbound API.
	UserAgent = "provider-upbound"
	// CookieName is the default cookie name used to identify a session token.
	CookieName = "SID"

	errNoIDInToken        = "no user id in personal access token"
	errInvalidAPIEndpoint = "unable to parse the API endpoint"
	errLoginFailed        = "unable to login"
	loginPath             = "/v1/login"
	errReadBody           = "unable to read response body"
	errParseCookieFmt     = "unable to parse session cookie: %s"
)

// sessionKey uniquely identifies an authentication identity.
// A cached session is only reused when all three fields match, making
// credential rotation and endpoint/org changes a natural cache miss.
type sessionKey struct {
	Endpoint string
	Org      string
	// CredHash is the hex-encoded SHA-256 of the raw credential bytes.
	// Using a hash avoids storing the raw credential and decouples the key
	// from any particular credential encoding (JWT, opaque token, etc.).
	CredHash string
}

type sessionCache struct {
	mu       sync.Mutex
	sessions map[sessionKey]Profile
}

var (
	DefaultAPIEndpoint, _ = url.Parse("https://api.upbound.io")
	cache                 = &sessionCache{sessions: make(map[sessionKey]Profile)}
)

// GetProviderConfigSpecFn returns the referenced ProviderConfig's spec from a
// legacy cluster-scoped MR or from a namespaced MR.
type GetProviderConfigSpecFn func(ctx context.Context, kube client.Client) (*pcv1alpha1common.ProviderConfigSpec, error)

func NewConfig(ctx context.Context, kube client.Client, getPCFn GetProviderConfigSpecFn) (*up.Config, Profile, error) {
	pcSpec, err := getPCFn(ctx, kube)
	if err != nil {
		return nil, Profile{}, errors.Wrap(err, "cannot get provider config")
	}

	data, err := resource.CommonCredentialExtractor(ctx, pcSpec.Credentials.Source, kube, pcSpec.Credentials.CommonCredentialSelectors)
	if err != nil {
		return nil, Profile{}, errors.Wrap(err, "cannot get credentials")
	}

	apiEndpoint, err := getAPIEndpoint(pcSpec)
	if err != nil {
		return nil, Profile{}, errors.Wrap(err, errInvalidAPIEndpoint)
	}

	profile, key, err := getOrCreateSession(ctx, data, pcSpec, apiEndpoint)
	if err != nil {
		return nil, Profile{}, err
	}

	cl := createUpClient(apiEndpoint, profile.Session, key)

	return up.NewConfig(func(conf *up.Config) {
		conf.Client = cl
	}), *profile, nil
}

// getOrCreateSession returns a valid cached session for the given identity, or
// logs in and caches a new one. It re-logins proactively when the session JWT
// is within 10 minutes of expiry, rather than returning an error and waiting
// for the next reconcile.
func getOrCreateSession(ctx context.Context, data []byte, pcSpec *pcv1alpha1common.ProviderConfigSpec, apiEndpoint *url.URL) (*Profile, sessionKey, error) {
	h := sha256.Sum256(data)
	key := sessionKey{
		Endpoint: apiEndpoint.String(),
		Org:      pcSpec.Organization,
		CredHash: hex.EncodeToString(h[:]),
	}

	cache.mu.Lock()
	defer cache.mu.Unlock()

	if p, ok := cache.sessions[key]; ok {
		// ParseUnverified is intentional: we only need to read the expiry claim
		// to decide whether to re-login. Signature verification would require the
		// server's public key; we don't have it and don't need it here since we
		// minted this session ourselves via the login endpoint.
		parser := jwt.Parser{}
		claims := &jwt.StandardClaims{}
		if _, _, err := parser.ParseUnverified(p.Session, claims); err == nil {
			// Session is still valid if it has no expiry or expires more than
			// 10 minutes from now.
			if claims.ExpiresAt == 0 || time.Now().Unix() <= claims.ExpiresAt-10*60 {
				return &p, key, nil
			}
		}
		// Session is invalid or approaching expiry — remove it and re-login
		// immediately rather than returning an error and failing the reconcile.
		delete(cache.sessions, key)
	}

	profile, err := login(ctx, data, pcSpec, apiEndpoint)
	if err != nil {
		return nil, key, err
	}

	cache.sessions[key] = *profile
	return profile, key, nil
}

// login authenticates against the Upbound API and returns a Profile containing
// the resulting session cookie.
func login(ctx context.Context, data []byte, pcSpec *pcv1alpha1common.ProviderConfigSpec, apiEndpoint *url.URL) (*Profile, error) {
	auth, err := constructAuth(string(data))
	if err != nil {
		return nil, errors.Wrap(err, errLoginFailed)
	}

	jsonStr, err := json.Marshal(auth)
	if err != nil {
		return nil, errors.Wrap(err, errLoginFailed)
	}

	loginURL := createLoginURL(apiEndpoint)
	req, err := createLoginRequest(ctx, loginURL, jsonStr)
	if err != nil {
		return nil, errors.Wrap(err, errLoginFailed)
	}

	req.Header.Set("Content-Type", "application/json")
	res, err := (&http.Client{}).Do(req)
	if err != nil {
		return nil, errors.Wrap(err, errLoginFailed)
	}
	defer func() { _ = res.Body.Close() }()

	session, err := extractSession(res, CookieName)
	if err != nil {
		return nil, errors.Wrap(err, errLoginFailed)
	}

	profile := Profile{
		Type:    TokenProfileType,
		ID:      auth.ID,
		Account: pcSpec.Organization,
	}
	if len(session) != 0 {
		profile.Session = session
	}
	return &profile, nil
}

// sessionClearingTransport is an http.RoundTripper that removes the cached
// session entry for a specific identity when the API returns 401 Unauthorized.
// This ensures the next Connect() call re-authenticates with current credentials
// rather than continuing to use an invalidated session.
type sessionClearingTransport struct {
	wrapped http.RoundTripper
	key     sessionKey
}

func (t *sessionClearingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := t.wrapped.RoundTrip(req)
	if err != nil {
		return resp, err
	}
	if resp.StatusCode == http.StatusUnauthorized {
		cache.mu.Lock()
		delete(cache.sessions, t.key)
		cache.mu.Unlock()
	}
	return resp, err
}

func createLoginURL(apiEndpoint *url.URL) *url.URL {
	return &url.URL{
		Scheme: apiEndpoint.Scheme,
		Host:   apiEndpoint.Host,
		Path:   loginPath,
	}
}

func createLoginRequest(ctx context.Context, loginURL *url.URL, jsonStr []byte) (*http.Request, error) {
	return http.NewRequestWithContext(ctx, http.MethodPost, loginURL.String(), bytes.NewReader(jsonStr))
}

func getAPIEndpoint(pcSpec *pcv1alpha1common.ProviderConfigSpec) (*url.URL, error) {
	if pcSpec.Endpoint == nil {
		return DefaultAPIEndpoint, nil
	}

	endpointURL, err := url.Parse(*pcSpec.Endpoint)
	if err != nil {
		return nil, err
	}

	// If the user provided only the host, assume HTTPS scheme by default
	if endpointURL.Scheme == "" {
		endpointURL.Scheme = "https"
	}

	return endpointURL, nil
}

func createUpClient(apiEndpoint *url.URL, session string, key sessionKey) up.Client {
	cj, _ := cookiejar.New(nil)
	cj.SetCookies(apiEndpoint, []*http.Cookie{
		{ //nolint:gosec
			Name:  CookieName,
			Value: session,
		},
	})

	cl := up.NewClient(func(u *up.HTTPClient) {
		u.BaseURL = apiEndpoint
		u.HTTP = &http.Client{
			Jar: cj,
			Transport: &sessionClearingTransport{
				wrapped: http.DefaultTransport,
				key:     key,
			},
		}
		u.UserAgent = UserAgent
	})

	return cl
}

// constructAuth constructs the body of an Upbound Cloud authentication request
// given the provided credentials.
func constructAuth(token string) (*auth, error) {
	id, err := parseID(token)
	if err != nil {
		return nil, err
	}
	return &auth{
		ID:       id,
		Password: token,
		Remember: true,
	}, nil
}

// parseID gets a user ID by either parsing a token.
func parseID(token string) (string, error) {
	p := jwt.Parser{}
	claims := &jwt.StandardClaims{}
	_, _, err := p.ParseUnverified(token, claims)
	if err != nil {
		return "", err
	}
	if claims.Id == "" {
		return "", errors.New(errNoIDInToken)
	}
	return claims.Id, nil
}

// extractSession extracts the specified cookie from an HTTP response. The
// caller is responsible for closing the response body.
func extractSession(res *http.Response, cookieName string) (string, error) {
	for _, cook := range res.Cookies() {
		if cook.Name == cookieName {
			return cook.Value, nil
		}
	}
	b, err := io.ReadAll(res.Body)
	if err != nil {
		return "", errors.Wrap(err, errReadBody)
	}
	return "", errors.Errorf(errParseCookieFmt, string(b))
}
