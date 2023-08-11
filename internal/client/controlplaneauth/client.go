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

package controlplaneauth

import (
	"context"
	"strings"

	v1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
	"sigs.k8s.io/controller-runtime/pkg/client"

	corev1 "k8s.io/api/core/v1"
)

const (
	errGetSecret            = "cannot get secret for token"
	UpboundKubeconfigKeyFmt = "upbound-"
	UpboundK8sResource      = "k8s"
	UpboundProxy            = "https://proxy.upbound.io/v1/controlPlanes/"
)

// GetSecretValue fetches the referenced input secret key reference
func GetSecretValue(ctx context.Context, kube client.Client, ref *v1.SecretKeySelector) (val string, err error) {
	secret, err := getSecret(ctx, kube, ref.SecretReference)
	if resource.IgnoreNotFound(err) != nil {
		return "", errors.Wrap(err, errGetSecret)
	}

	pwRaw := secret.Data[ref.Key]
	return string(pwRaw), nil
}

// getSecret to get secret
func getSecret(ctx context.Context, kube client.Client, ref v1.SecretReference) (*corev1.Secret, error) {
	secret := new(corev1.Secret)
	err := kube.Get(ctx, types.NamespacedName{Name: ref.Name, Namespace: ref.Namespace}, secret)
	return secret, err
}

// BuildControlPlaneKubeconfig build kubeconfig object
func BuildControlPlaneKubeconfig(organization, controlplane, token string) (string, error) {
	clusterName := UpboundKubeconfigKeyFmt + organization + "-" + controlplane
	serverURL := UpboundProxy + organization + "/" + controlplane + "/" + UpboundK8sResource

	config := &api.Config{
		Clusters: map[string]*api.Cluster{
			clusterName: {
				Server: serverURL,
			},
		},
		AuthInfos: map[string]*api.AuthInfo{
			clusterName: {
				Token: token,
			},
		},
		Contexts: map[string]*api.Context{
			clusterName: {
				Cluster:  clusterName,
				AuthInfo: clusterName,
			},
		},
		CurrentContext: clusterName,
	}

	configBytes, err := clientcmd.Write(*config)
	if err != nil {
		return "", err
	}
	configString := string(configBytes)
	configString = strings.TrimSpace(configString)

	return configString, nil
}
