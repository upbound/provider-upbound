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

package v1alpha1

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/crossplane/crossplane-runtime/v2/pkg/errors"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reference"
)

func (mg *Token) ResolveReferences(ctx context.Context, c client.Reader) error {
	r := reference.NewAPIResolver(c, mg)

	var ref reference.To
	switch mg.Spec.ForProvider.Owner.Type {
	case "robots":
		ref = reference.To{
			List:    &RobotList{},
			Managed: &Robot{},
		}
	default:
		return errors.New("only robots is supported as owner type for reference resolution")
	}

	rsp, err := r.Resolve(ctx, reference.ResolutionRequest{
		CurrentValue: reference.FromPtrValue(mg.Spec.ForProvider.Owner.ID),
		Extract:      reference.ExternalName(),
		Reference:    mg.Spec.ForProvider.Owner.IDRef,
		Selector:     mg.Spec.ForProvider.Owner.IDSelector,
		To:           ref,
	})
	if err != nil {
		return errors.Wrap(err, "mg.Spec.ForProvider.Owner.ID")
	}
	mg.Spec.ForProvider.Owner.ID = reference.ToPtrValue(rsp.ResolvedValue)
	mg.Spec.ForProvider.Owner.IDRef = rsp.ResolvedReference

	return nil
}
