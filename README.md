# Temporal Provider

Provider Temporal is a [Crossplane](https://www.crossplane.io/) provider. It was build based on the [Crossplane Template](https://github.com/crossplane/provider-template). It is used to manage and configure [Temporal](https://temporal.io/). It uses the [Temporal Go SDK](https://github.com/temporalio/sdk-go)

# Using 
Provider Credentials:
```
{
  "HostPort": "temporal:7233"
}
```

Example:
```
apiVersion: pkg.crossplane.io/v1
kind: Provider
metadata:
  name: provider-temporal
spec:
  package: <packagepath>:<packagelabel>
  packagePullPolicy: IfNotPresent
  revisionActivationPolicy: Automatic
---
apiVersion: v1
kind: Secret
metadata:
  name: provider-temporal-config-creds
  namespace: crossplane-system
type: Opaque
stringData:
  credentials: |
    {
      "HostPort": "temporal:7233"
    }
---
apiVersion: temporal.crossplane.io/v1alpha1
kind: ProviderConfig
metadata:
  name: provider-temporal-config
spec: 
  credentials:
    source: Secret
    secretRef:
      namespace: crossplane-system
      name: provider-temporal-config-creds
      key: credentials  
```

# Covered Managed Resources
Currently covered Managed Resources
- [TemporalNamespace](#temporalnamespace)

## TemporalNamespace 
A Namespace is a unit of isolation within the Temporal Platform

[temporal docs](https://docs.temporal.io/namespaces) 

[temporal cli](https://docs.temporal.io/cli/operator#namespace)

Example:
```
apiVersion: core.temporal.crossplane.io/v1alpha1
kind: TemporalNamespace
metadata:
  name: ns1
spec:
  forProvider:
    name: "Test 1"
    description: "Desc 1"
    ownerEmail: "Test@test.local"
  providerConfigRef:
    name: provider-temporal-config
```

## Developing
1. Add new type by running the following command:
```shell
  export provider_name=temporal
  export group=core # lower case e.g. core, cache, database, storage, etc.
  export type=MyType # Camel casee.g. Bucket, Database, CacheCluster, etc.
  make provider.addtype provider=${provider_name} group=${group} kind=${type}
```
2. Replace the *core* group with your new group in apis/{provider}.go
3. Replace the *MyType* type with your new type in internal/controller/{provider}.go

4. Run `make reviewable` to run code generation, linters, and tests.
5. Run `make build` to build the provider.

Refer to Crossplane's [CONTRIBUTING.md] file for more information on how the
Crossplane community prefers to work. The [Provider Development][provider-dev]
guide may also be of use.

[CONTRIBUTING.md]: https://github.com/crossplane/crossplane/blob/master/CONTRIBUTING.md
[provider-dev]: https://github.com/crossplane/crossplane/blob/master/contributing/guide-provider-development.md
