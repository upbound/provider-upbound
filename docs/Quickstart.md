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
  package: xpkg.upbound.io/upbound/provider-upbound:v0.1.0
EOF
kubectl wait "providers.pkg.crossplane.io/provider-upbound" --for=condition=Installed --timeout=180s
kubectl wait "providers.pkg.crossplane.io/provider-upbound" --for=condition=Healthy --timeout=180s
```

### Configuration

Provider Upbound needs a valid Upbound token to authenticate with Upbound. There
are multiple ways to acquire one but the easiest one is to log in with `up` CLI
to get a session token and then use it.

```bash
# Once logged in, it will save token to ~/.up/config.json
up login
```

Then, we need to create a `Secret` object that contains the token.
```bash
kubectl -n crossplane-system create secret generic up-creds --from-file=creds=$HOME/.up/config.json
```

Then, we need to create a `ProviderConfig` object that references the `Secret`
object we just created. The following command creates a `ProviderConfig` object
that references the `Secret` object we just created.

```bash
cat <<EOF | kubectl apply -f -
apiVersion: upbound.io/v1alpha1
kind: ProviderConfig
metadata:
  name: default
spec:
  credentials:
    source: Secret
    secretRef:
      namespace: crossplane-system
      name: up-creds
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