---
title: Quickstart
weight: 1
---
# Quickstart

This guide walks through the process to install Provider Upbound to a Kubernetes
cluster with Crossplane or to an Upbound Control Plane. 

To install and use this provider:
* Install the `Provider` and apply a `ProviderConfig`.
* Create a *managed resource* in Upbound with Kubernetes.

### Install the Provider

To install the provider, create a `Provider` object in the control plane. The
following command creates `Provider` object that installs Provider Upbound.

```bash
cat <<EOF | kubectl apply -f -
apiVersion: pkg.crossplane.io/v1
kind: Provider
metadata:
  name: provider-upbound
spec:
  package: xpkg.upbound.io/upbound/provider-upbound:v0.6.0
EOF
kubectl wait "providers.pkg.crossplane.io/provider-upbound" --for=condition=Installed --timeout=180s
kubectl wait "providers.pkg.crossplane.io/provider-upbound" --for=condition=Healthy --timeout=180s
```

### Configuration

Provider Upbound needs a valid Upbound personal access token (pat) to authenticate with Upbound.

Then, we need to create a `Secret` object that contains the token.
```bash
kubectl -n upbound-system create secret generic upbound-creds --from-literal=creds=${PersonalAccessToken}
```

Then, we need to create a `ProviderConfig` object that references the default`Organization` and the `Secret`
object we just created. The following command creates a `ProviderConfig` object
that references the `Secret` object we just created.

```bash
cat <<EOF | kubectl apply -f -
apiVersion: upbound.io/v1alpha1
kind: ProviderConfig
metadata:
  name: default
spec:
  organization: myorg
  credentials:
    source: Secret
    secretRef:
      namespace: upbound-system
      name: upbound-creds
      key: creds
EOF
```

### First Managed Resource

Let's create a `Robot` in our Upbound account. The following command creates a
`Robot` object that creates a `Robot` in Upbound. Note that you need to change
the `spec.owner.name` to your organization name.

```bash
cat <<EOF | kubectl apply -f -
apiVersion: iam.upbound.io/v1alpha1
kind: Robot
metadata:
  name: myrobot
spec:
  forProvider:
    name: myrobot
    description: my powerful robot
    owner:
      name: myorg
EOF
```

Once created, you can check its progress with the following command:
```bash
kubectl get robot myrobot --watch
```

You can also check the status of the `Robot` object in Upbound Console.

See the [Provider Upbound](https://marketplace.upbound.io/providers/upbound/provider-upbound/latest/crds)
in the Upbound Marketplace for more examples.

### Clean Up

You can delete your `Robot` with the following command:
```bash
kubectl delete robot myrobot
```
