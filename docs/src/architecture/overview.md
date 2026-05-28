# Architecture Overview

The provider has four main areas:

- API types and CRDs
- Controllers
- Cloud abstraction layer
- clusterctl templates and release assets

Key packages:

```text
api/v1alpha1/             Provider API types
internal/controller/      Reconciliation logic
pkg/cloud/                Cloud client interface and SDK implementation
pkg/cloud/fake/           In-memory fake for tests
pkg/util/                 Shared helpers
templates/                clusterctl templates
config/                   Kubebuilder manifests
```

The controllers do not call the STACKIT SDK directly. They use the cloud client
interface, which keeps reconciliation testable and allows fake-client envtest
coverage.
