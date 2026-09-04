/*
Copyright 2026 Upbound Inc.

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
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/golang-jwt/jwt"
	"github.com/google/go-cmp/cmp"

	pcv1alpha1common "github.com/upbound/provider-upbound/apis/common/providerconfig/v1alpha1"
)

// makeCredJWT returns a PAT-style JWT (the bytes stored in the k8s credential
// secret). id is the "jti" claim so parseID can extract a user ID from it.
func makeCredJWT(t *testing.T, id string) []byte {
	t.Helper()
	tok, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.StandardClaims{
		Id: id,
	}).SignedString([]byte("cred-secret"))
	if err != nil {
		t.Fatalf("makeCredJWT: %v", err)
	}
	return []byte(tok)
}

// makeSessionJWT returns a session JWT (SID cookie value). Pass expiresAt=0 to
// produce a token with no expiry claim; ParseUnverified will leave ExpiresAt=0.
func makeSessionJWT(t *testing.T, expiresAt int64) string {
	t.Helper()
	tok, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.StandardClaims{
		ExpiresAt: expiresAt,
	}).SignedString([]byte("session-secret"))
	if err != nil {
		t.Fatalf("makeSessionJWT: %v", err)
	}
	return tok
}

func TestGetOrCreateSession(t *testing.T) {
	const (
		org       = "test-org"
		serverSID = "server-issued-sid"
	)

	// Primary credentials and an alternative set (different bytes → different hash).
	creds := makeCredJWT(t, "user-id-primary")
	altCreds := makeCredJWT(t, "user-id-alt")

	// Pre-compute the cache key for the primary credentials so tests that seed
	// the cache can inject an entry before calling getOrCreateSession.
	h := sha256.Sum256(creds)
	primaryCredHash := hex.EncodeToString(h[:])

	validSession := makeSessionJWT(t, time.Now().Add(1*time.Hour).Unix())
	nearExpirySession := makeSessionJWT(t, time.Now().Add(5*time.Minute).Unix()) // inside 10-min buffer
	noExpirySession := makeSessionJWT(t, 0)                                      // ExpiresAt == 0

	type args struct {
		activeCreds []byte
		// preSession, when non-empty, is seeded into the cache under the primary
		// credential key before getOrCreateSession is called.
		preSession string
	}
	type want struct {
		loginCalls int32
		// session is the expected profile.Session value. Empty string is
		// substituted with serverSID (the value the test server returns).
		session string
	}

	cases := map[string]struct {
		reason string
		args
		want
	}{
		"CacheMiss_TriggersLogin": {
			reason: "an empty cache must trigger a login and cache the returned session",
			args:   args{activeCreds: creds},
			want:   want{loginCalls: 1},
		},
		"CacheHit_ValidSession_SkipsLogin": {
			reason: "a cached session well within its TTL must be returned without a login",
			args:   args{activeCreds: creds, preSession: validSession},
			want:   want{loginCalls: 0, session: validSession},
		},
		"CacheHit_NearExpiry_TriggersProactiveRelogin": {
			reason: "a session within the 10-minute pre-expiry window must be replaced immediately",
			args:   args{activeCreds: creds, preSession: nearExpirySession},
			want:   want{loginCalls: 1},
		},
		"CacheHit_NoExpiryClaim_SkipsLogin": {
			reason: "ExpiresAt=0 means no expiry; the session must be reused without a login",
			args:   args{activeCreds: creds, preSession: noExpirySession},
			want:   want{loginCalls: 0, session: noExpirySession},
		},
		"CredentialRotation_NewHashCausesLogin": {
			reason: "different credential bytes hash to a different key and must trigger a fresh login",
			// Cache is seeded for the primary creds key — altCreds produces a
			// different hash, so it is a miss even though something is cached.
			args: args{activeCreds: altCreds, preSession: validSession},
			want: want{loginCalls: 1},
		},
		"CacheHit_InvalidJWT_FallsThrough": {
			reason: "a cached value that fails JWT parsing is not a usable session and must trigger re-login",
			args:   args{activeCreds: creds, preSession: "not-a-jwt"},
			want:   want{loginCalls: 1},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			resetCacheForTest()

			var loginCalls atomic.Int32
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				loginCalls.Add(1)
				http.SetCookie(w, &http.Cookie{Name: CookieName, Value: serverSID})
				w.WriteHeader(http.StatusOK)
			}))
			t.Cleanup(srv.Close)

			endpoint, _ := url.Parse(srv.URL)
			pcSpec := &pcv1alpha1common.ProviderConfigSpec{Organization: org}

			if tc.args.preSession != "" {
				key := sessionKey{
					Endpoint: endpoint.String(),
					Org:      org,
					CredHash: primaryCredHash,
				}
				cache.mu.Lock()
				cache.sessions[key] = Profile{Session: tc.args.preSession}
				cache.mu.Unlock()
			}

			profile, _, err := getOrCreateSession(context.Background(), tc.args.activeCreds, pcSpec, endpoint)
			if err != nil {
				t.Fatalf("\n%s\ngetOrCreateSession(...): unexpected error: %v", tc.reason, err)
			}

			if diff := cmp.Diff(tc.want.loginCalls, loginCalls.Load()); diff != "" {
				t.Errorf("\n%s\ngetOrCreateSession(...): -want loginCalls, +got loginCalls:\n%s", tc.reason, diff)
			}

			wantSession := tc.want.session
			if wantSession == "" {
				wantSession = serverSID
			}
			if diff := cmp.Diff(wantSession, profile.Session); diff != "" {
				t.Errorf("\n%s\ngetOrCreateSession(...): -want session, +got session:\n%s", tc.reason, diff)
			}
		})
	}
}

// stubTransport is an http.RoundTripper that returns a fixed status code
// without making a network call, used to drive sessionClearingTransport tests.
type stubTransport struct {
	statusCode int
}

func (s *stubTransport) RoundTrip(_ *http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: s.statusCode,
		Body:       io.NopCloser(strings.NewReader("")),
	}, nil
}

func TestSessionClearingTransport(t *testing.T) {
	key := sessionKey{Endpoint: "https://api.upbound.io", Org: "test-org", CredHash: "abc123"}

	type args struct {
		statusCode       int
		cachedSession    string // session stored in cache at call time
		transportSession string // session the transport was created with
	}
	type want struct {
		cacheHit bool
	}

	cases := map[string]struct {
		reason string
		args
		want
	}{
		"401_EvictsCacheEntry": {
			reason: "a 401 for the session this transport carries must evict the cache entry so the next reconcile re-logins",
			args:   args{statusCode: http.StatusUnauthorized, cachedSession: "s1", transportSession: "s1"},
			want:   want{cacheHit: false},
		},
		"200_RetainsCacheEntry": {
			reason: "a successful response must leave the cache untouched",
			args:   args{statusCode: http.StatusOK, cachedSession: "s1", transportSession: "s1"},
			want:   want{cacheHit: true},
		},
		"401_DoesNotEvictFresherSession": {
			reason: "if a concurrent reconcile already refreshed the session under the same key, a stale 401 must not evict the newer entry",
			args:   args{statusCode: http.StatusUnauthorized, cachedSession: "s2-refreshed", transportSession: "s1-stale"},
			want:   want{cacheHit: true},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			resetCacheForTest()
			cache.mu.Lock()
			cache.sessions[key] = Profile{Session: tc.args.cachedSession}
			cache.mu.Unlock()

			transport := &sessionClearingTransport{
				wrapped: &stubTransport{statusCode: tc.args.statusCode},
				key:     key,
				session: tc.args.transportSession,
			}

			req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "http://example.com", nil)
			resp, err := transport.RoundTrip(req)
			if err != nil {
				t.Fatalf("\n%s\nRoundTrip(...): unexpected error: %v", tc.reason, err)
			}
			_ = resp.Body.Close()

			cache.mu.Lock()
			_, hit := cache.sessions[key]
			cache.mu.Unlock()

			if diff := cmp.Diff(tc.want.cacheHit, hit); diff != "" {
				t.Errorf("\n%s\nRoundTrip(...): -want cacheHit, +got cacheHit:\n%s", tc.reason, diff)
			}
		})
	}
}

// TestGetOrCreateSession_LoginFailed verifies that an error from login() —
// here simulated by a server that returns no SID cookie — is propagated to
// the caller rather than swallowed.
func TestGetOrCreateSession_LoginFailed(t *testing.T) {
	resetCacheForTest()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Respond with 200 but omit the SID cookie — extractSession will fail.
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	endpoint, _ := url.Parse(srv.URL)
	pcSpec := &pcv1alpha1common.ProviderConfigSpec{Organization: "test-org"}
	creds := makeCredJWT(t, "user-id")

	_, _, err := getOrCreateSession(context.Background(), creds, pcSpec, endpoint)
	if err == nil {
		t.Fatal("getOrCreateSession(...): expected error when login server returns no session cookie, got nil")
	}
}

// Test401ThenRelogin_EndToEnd verifies the full Bug-B recovery chain (cached
// session not refreshed on 401): a 401 response from the Upbound API evicts the
// cached session via sessionClearingTransport, and the subsequent Connect() call
// — modelled here as a direct call to getOrCreateSession — re-logins successfully
// rather than reusing the now-invalid cached session.
func Test401ThenRelogin_EndToEnd(t *testing.T) {
	resetCacheForTest()

	const org = "test-org"
	creds := makeCredJWT(t, "user-id")

	var loginCalls atomic.Int32
	loginSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		loginCalls.Add(1)
		http.SetCookie(w, &http.Cookie{Name: CookieName, Value: "refreshed-session"})
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(loginSrv.Close)

	endpoint, _ := url.Parse(loginSrv.URL)

	// Step 1 — seed the cache so we start from an authenticated state.
	h := sha256.Sum256(creds)
	key := sessionKey{
		Endpoint: endpoint.String(),
		Org:      org,
		CredHash: hex.EncodeToString(h[:]),
	}
	seededSession := makeSessionJWT(t, time.Now().Add(1*time.Hour).Unix())
	cache.mu.Lock()
	cache.sessions[key] = Profile{Session: seededSession}
	cache.mu.Unlock()

	// Step 2 — simulate a 401 from the Upbound API; the transport must evict
	// the cache entry so the stale session is no longer used.
	transport := &sessionClearingTransport{
		wrapped: &stubTransport{statusCode: http.StatusUnauthorized},
		key:     key,
		session: seededSession,
	}
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "http://example.com", nil)
	resp, err := transport.RoundTrip(req)
	if err != nil {
		t.Fatalf("Test401ThenRelogin_EndToEnd: RoundTrip unexpected error: %v", err)
	}
	_ = resp.Body.Close()

	cache.mu.Lock()
	_, stillPresent := cache.sessions[key]
	cache.mu.Unlock()
	if stillPresent {
		t.Fatal("Test401ThenRelogin_EndToEnd: cache entry should have been evicted after 401")
	}

	// Step 3 — next Connect() cycle: getOrCreateSession finds no cache entry
	// and must perform a fresh login.
	pcSpec := &pcv1alpha1common.ProviderConfigSpec{Organization: org}
	profile, _, err := getOrCreateSession(context.Background(), creds, pcSpec, endpoint)
	if err != nil {
		t.Fatalf("Test401ThenRelogin_EndToEnd: re-login after 401 failed: %v", err)
	}

	if diff := cmp.Diff(int32(1), loginCalls.Load()); diff != "" {
		t.Errorf("Test401ThenRelogin_EndToEnd: -want loginCalls, +got loginCalls:\n%s", diff)
	}
	if diff := cmp.Diff("refreshed-session", profile.Session); diff != "" {
		t.Errorf("Test401ThenRelogin_EndToEnd: -want session, +got session:\n%s", diff)
	}
}
