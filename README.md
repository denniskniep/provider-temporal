# Temporal Provider

Temporal Provider is a [Crossplane](https://www.crossplane.io/) provider. It was build based on the [Crossplane Template](https://github.com/crossplane/provider-template). It is used to manage and configure [Temporal](https://temporal.io/). It uses the [Temporal Go SDK](https://github.com/temporalio/sdk-go)

# How to use 
Repository and package:
```
xpkg.upbound.io/denniskniep/provider-temporal:<version>
```

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
  package: xpkg.upbound.io/denniskniep/provider-temporal:v1.2.0
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
# Troubleshooting
Create a DeploymentRuntimeConfig and set the arg `--debug` on the package-runtime container:

```
apiVersion: pkg.crossplane.io/v1beta1
kind: DeploymentRuntimeConfig
metadata:
  name: debug-config
spec:
  deploymentTemplate:
    spec:
      selector: {}
      template:
        spec:
          containers:
            - name: package-runtime
              args:
                - --debug
---
apiVersion: pkg.crossplane.io/v1
kind: Provider
metadata:
  name: provider-temporal
spec:
  package: xpkg.upbound.io/denniskniep/provider-temporal:v1.2.0
  packagePullPolicy: IfNotPresent
  revisionActivationPolicy: Automatic
  runtimeConfigRef:
    name: debug-config
```

# Covered Managed Resources
Currently covered Managed Resources:
- [TemporalNamespace](#temporalnamespace)
- [SearchAttribute](#searchattribute)

## TemporalNamespace 
A Namespace is a unit of isolation within the Temporal Platform

[temporal docs](https://docs.temporal.io/namespaces) 

[temporal cli](https://docs.temporal.io/cli/operator#namespace)

Hint: Currently its not possible to name this managed resource simply `Namespace`, because of [this](https://github.com/kubernetes/kubernetes/pull/108382) and [this](https://github.com/crossplane/terrajet/issues/234).

Example:
```
apiVersion: core.temporal.crossplane.io/v1alpha1
kind: TemporalNamespace
metadata:
  name: namespace1
spec:
  forProvider:
    name: "Test1"
    description: "Desc 1"
    ownerEmail: "Test@test.local"
    workflowExecutionRetentionDays: 30
    data:
      - key1: value1
      - key2: value2
    historyArchivalState: "Disabled"
    historyArchivalUri: ""
    visibilityArchivalState: "Disabled"
    visibilityArchivalUri: ""
  providerConfigRef:
    name: provider-temporal-config
```

## SearchAttribute
Search Attributes enable complex and business-logic-focused search queries for Workflow Executions. These are often queried through the Temporal Web UI, but you can also query from within your Workflow code. For more debugging and monitoring, you might want to add your own domain-specific Search Attributes, such as customerId or numItems, that can serve as useful search filters.

[temporal docs](https://docs.temporal.io/visibility#custom-search-attributes) 

[temporal cli](https://docs.temporal.io/cli/operator#search-attribute)


Example 1:
```
apiVersion: core.temporal.crossplane.io/v1alpha1
kind: SearchAttribute
metadata:
  name: searchattr1
spec:
  forProvider:
    name: "Test1"
    type: "Keyword"
    temporalNamespaceName: "Test1"
  providerConfigRef:
    name: local-temporal-instance-config
```


Example 2:
```
apiVersion: core.temporal.crossplane.io/v1alpha1
kind: SearchAttribute
metadata:
  name: searchattr1
spec:
  forProvider:
    name: "Test1"
    type: "Keyword"
    temporalNamespaceNameRef:
      name: "namespace1"
  providerConfigRef:
    name: local-temporal-instance-config
```

# Contribute
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
4. Run `make reviewable` to run code generation, linters, and tests. (`make generate` to only run code generation)
5. Run `make build` to build the provider.

Refer to Crossplane's [CONTRIBUTING.md] file for more information on how the
Crossplane community prefers to work. The [Provider Development][provider-dev]
guide may also be of use.

[CONTRIBUTING.md]: https://github.com/crossplane/crossplane/blob/master/CONTRIBUTING.md
[provider-dev]: https://github.com/crossplane/crossplane/blob/master/contributing/guide-provider-development.md

## Start Debug with local cluster
* `make dev` starts a fresh KIND cluster
*  `sudo docker-compose -f tests/docker-compose.yaml up -d` starts temporal environment
*  debug source code with `.vscode/launch.json`
*  Apply the CRDs `kubectl apply -f examples` 

## Stop Debug with local cluster
*  `make dev-clean` shutdown the earlier started KIND cluster
*  `sudo docker-compose -f tests/docker-compose.yaml down -v`

## Tests
Start temporal environment for tests
```
sudo docker-compose -f tests/docker-compose.yaml up 
```