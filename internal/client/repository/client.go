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

package repository

import (
	"github.com/upbound/up-sdk-go/service/repositories"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1alpha1 "github.com/upbound/provider-upbound/apis/repository/v1alpha1"
)

// StatusFromResponse set status from response
func StatusFromResponse(resp repositories.Repository) v1alpha1.RepositoryObservation {

	status := v1alpha1.RepositoryObservation{}

	status.AccountID = resp.AccountID
	status.CreatedAt = metav1.Time{Time: resp.CreatedAt}
	status.CurrentVersion = resp.CurrentVersion
	status.Name = resp.Name
	status.Official = resp.Official
	status.Public = resp.Public
	status.RepositoryID = resp.RepositoryID
	status.Type = resp.Type
	if resp.UpdatedAt != nil {
		status.UpdatedAt = &metav1.Time{Time: *resp.UpdatedAt}
	}

	return status
}
