/*
Copyright 2025 Upbound Inc.

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
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/crossplane/crossplane-runtime/pkg/errors"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	"github.com/golang-jwt/jwt"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/json"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/upbound/up-sdk-go"

	"github.com/upbound/provider-upbound/apis/v1alpha1"
)

const (
	// UserAgent is the default user agent to use to make requests to the
	// Upbound API.
	UserAgent = "provider-upbound"
	// CookieName is the default cookie name used to identify a session token.
	CookieName = "SID"

	errNoIDInToken         = "no user id in personal access token"
	errLoginFailed         = "unable to login"
	loginPath              = "/v1/login"
	errReadBody            = "unable to read response body"
	errParseCookieFmt      = "unable to parse session cookie: %s"
	errSessionTokenParse   = "failed to parse session token"
	errSessionTokenExpired = "session token has expired"
)

var (
	DefaultAPIEndpoint, _ = url.Parse("https://api.upbound.io")
	profileMemory         = Profile{}
	mu                    sync.Mutex
)

func NewConfig(ctx context.Context, kube client.Client, mg resource.Managed) (*up.Config, Profile, error) {
	pc, err := GetProviderConfig(ctx, kube, mg)
	if err != nil {
		return nil, Profile{}, errors.Wrapf(err, "cannot get provider config %s", mg.GetProviderConfigReference().Name)
	}

	data, err := getCredentials(ctx, kube, pc)
	if err != nil {
		return nil, Profile{}, errors.Wrap(err, "cannot get credentials")
	}

	profile, err := createOrUpdateProfile(ctx, data, pc)
	if err != nil {
		return nil, Profile{}, err
	}

	apiEndpoint, err := getAPIEndpoint(pc)
	if err != nil {
		return nil, Profile{}, err
	}

	cl := createUpClient(apiEndpoint, profile.Session)

	return up.NewConfig(func(conf *up.Config) {
		conf.Client = cl
	}), *profile, nil
}

func GetProviderConfig(ctx context.Context, kube client.Client, mg resource.Managed) (*v1alpha1.ProviderConfig, error) {
	pc := &v1alpha1.ProviderConfig{}
	if err := kube.Get(ctx, types.NamespacedName{Name: mg.GetProviderConfigReference().Name}, pc); err != nil {
		return nil, err
	}
	return pc, nil
}

func getCredentials(ctx context.Context, kube client.Client, pc *v1alpha1.ProviderConfig) ([]byte, error) {
	return resource.CommonCredentialExtractor(ctx, pc.Spec.Credentials.Source, kube, pc.Spec.Credentials.CommonCredentialSelectors)
}

func createOrUpdateProfile(ctx context.Context, data []byte, pc *v1alpha1.ProviderConfig) (*Profile, error) { //nolint:gocyclo
	// use this shared to avoid get new session-token for each reconcile
	mu.Lock()
	defer mu.Unlock()

	if profileMemory.Session != "" {
		// Check the expiration of the profileMemory.Session token
		p := jwt.Parser{}
		claims := &jwt.StandardClaims{}
		_, _, err := p.ParseUnverified(profileMemory.Session, claims)
		if err != nil {
			return nil, errors.Wrap(err, errSessionTokenParse)
		}

		// Check if the token expiration time (claims.ExpiresAt) is greater than 0
		// and if the current Unix time (time.Now().Unix()) is greater than 10 minutes
		// before the token expires (claims.ExpiresAt - 10 minutes). This condition is
		// used to determine if the token is close to expiration and requires refreshing.
		if claims.ExpiresAt > 0 && time.Now().Unix() > claims.ExpiresAt-10*60 {
			profileMemory.Session = ""
			return nil, errors.New(errSessionTokenExpired)
		}

		return &profileMemory, nil
	}

	cliConfig := &CLIConfig{}
	profile := cliConfig.Upbound.Profiles[cliConfig.Upbound.Default]

	auth, err := constructAuth(string(data))
	if err != nil {
		return nil, errors.Wrap(err, errLoginFailed)
	}

	jsonStr, err := json.Marshal(auth)
	if err != nil {
		return nil, errors.Wrap(err, errLoginFailed)
	}

	loginURL := createLoginURL(DefaultAPIEndpoint)
	req, err := createLoginRequest(ctx, loginURL, jsonStr)
	if err != nil {
		return nil, errors.Wrap(err, errLoginFailed)
	}

	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, errLoginFailed)
	}
	defer res.Body.Close() // nolint:gosec,errcheck

	session, err := extractSession(res, CookieName)
	if err != nil {
		return nil, errors.Wrap(err, errLoginFailed)
	}

	profile.Type = TokenProfileType
	profile.ID = auth.ID
	if len(session) != 0 {
		profile.Session = session
	}
	profile.Account = pc.Spec.Organization
	profileMemory = profile

	return &profile, nil
}

func createLoginURL(apiEndpoint *url.URL) *url.URL {
	loginURL := &url.URL{
		Scheme: apiEndpoint.Scheme,
		Host:   apiEndpoint.Host,
		Path:   loginPath,
	}
	return loginURL
}

func createLoginRequest(ctx context.Context, loginURL *url.URL, jsonStr []byte) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, loginURL.String(), bytes.NewReader(jsonStr))
	if err != nil {
		return nil, err
	}

	return req, nil
}

func getAPIEndpoint(pc *v1alpha1.ProviderConfig) (*url.URL, error) {
	if pc.Spec.Endpoint == nil {
		// Use a default API endpoint when not specified in the provider config
		apiEndpoint := DefaultAPIEndpoint
		return apiEndpoint, nil
	}

	endpointURL, err := url.Parse(*pc.Spec.Endpoint)
	if err != nil {
		return nil, err
	}

	// If the user provided only the host, assume HTTPS scheme by default
	if endpointURL.Scheme == "" {
		endpointURL.Scheme = "https"
	}

	return endpointURL, nil
}

func createUpClient(apiEndpoint *url.URL, session string) up.Client {
	// Create a cookie jar and set the session cookie
	cj, _ := cookiejar.New(nil)
	cj.SetCookies(apiEndpoint, []*http.Cookie{
		{
			Name:  CookieName,
			Value: session,
		},
	})

	// Create the Up client configuration
	cl := up.NewClient(func(u *up.HTTPClient) {
		u.BaseURL = apiEndpoint
		u.HTTP = &http.Client{
			Jar: cj,
		}
		u.UserAgent = UserAgent
	})

	return cl
}

// ExtractUserIDFromToken extracts userId from SessionToken
func ExtractUserIDFromToken(sToken string) (string, error) {
	token := strings.Split(sToken, ".")
	if len(token) != 3 {
		return "", errors.New("invalid token format")
	}

	claimsData, _ := base64.StdEncoding.WithPadding(base64.NoPadding).DecodeString(token[1])

	var claims map[string]interface{}
	err := json.Unmarshal(claimsData, &claims)
	if err != nil {
		return "", errors.Wrap(err, "unable to unmarshal token claims")
	}

	return fmt.Sprintf("%d", claims["id"]), nil
}

// constructAuth constructs the body of an Upbound Cloud authentication request
// given the provided credentials.
func constructAuth(token string) (*auth, error) {
	id, err := ParseID(token)
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
func ParseID(token string) (string, error) {
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
