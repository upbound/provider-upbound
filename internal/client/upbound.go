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
	"context"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/url"

	"github.com/crossplane/crossplane-runtime/pkg/errors"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
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
)

var (
	DefaultAPIEndpoint, _ = url.Parse("https://api.upbound.io")
)

func NewConfig(ctx context.Context, kube client.Client, mg resource.Managed) (*up.Config, Profile, error) {
	pc := &v1alpha1.ProviderConfig{}
	profile := Profile{}
	if err := kube.Get(ctx, types.NamespacedName{Name: mg.GetProviderConfigReference().Name}, pc); err != nil {
		return nil, Profile{}, errors.Wrapf(err, "cannot get provider config %s", mg.GetProviderConfigReference().Name)
	}

	data, err := resource.CommonCredentialExtractor(ctx, pc.Spec.Credentials.Source, kube, pc.Spec.Credentials.CommonCredentialSelectors)
	if err != nil {
		return nil, Profile{}, errors.Wrap(err, "cannot get credentials")
	}
	cliConfig := &CLIConfig{}
	if err := json.Unmarshal(data, cliConfig); err != nil {
		return nil, Profile{}, errors.Wrap(err, "cannot unmarshal credentials")
	}
	profile = cliConfig.Upbound.Profiles[cliConfig.Upbound.Default]
	if len(profile.Session) == 0 {
		return nil, Profile{}, errors.New("no session found")
	}
	apiEndpoint := DefaultAPIEndpoint
	if pc.Spec.Endpoint != nil {
		// NOTE(muvaf): We expect full endpoint instead of `api.upbound.io`
		// because 10/10 times I gave the wrong input, and it was hard to debug.
		endpoint, err := url.Parse(*pc.Spec.Endpoint)
		if err != nil {
			return nil, Profile{}, errors.Wrapf(err, "cannot parse apiEndpoint %s", *pc.Spec.Endpoint)
		}
		a := fmt.Sprintf("%s://api.%s", endpoint.Scheme, endpoint.Host)
		apiEndpoint, err = url.Parse(a)
		if err != nil {
			return nil, Profile{}, errors.Wrapf(err, "cannot parse constructed api endpoint %s", a)
		}
	}
	cj, _ := cookiejar.New(nil)
	cj.SetCookies(apiEndpoint, []*http.Cookie{
		{
			Name:  CookieName,
			Value: profile.Session,
		},
	})
	cl := up.NewClient(func(u *up.HTTPClient) {
		u.BaseURL = apiEndpoint
		u.HTTP = &http.Client{
			Jar: cj,
		}
		u.UserAgent = UserAgent
	})
	return up.NewConfig(func(conf *up.Config) {
		conf.Client = cl
	}), profile, nil
}
