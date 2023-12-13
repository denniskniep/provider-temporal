# Temporal Provider

Provider Temporal is a [Crossplane](https://www.crossplane.io/) provider. It was build based on the [Crossplane Template](https://github.com/crossplane/provider-template). It is used to manage and configure [Temporal](https://temporal.io/)

https://github.com/crossplane/crossplane/blob/master/contributing/guide-provider-development.md

# temporal

`provider-temporal` is a minimal [Crossplane](https://crossplane.io/) Provider
that is meant to be used to configure [Temporal](https://temporal.io/). It uses the [Temporal Go SDK](https://github.com/temporalio/sdk-go)


Covered Resources
- Namespace

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
