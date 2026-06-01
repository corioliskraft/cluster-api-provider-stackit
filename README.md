# Kubernetes Cluster API Provider STACKIT

<p align="center">
<img src="https://github.com/kubernetes/kubernetes/raw/master/logo/logo.png"  width="100x"><a href="https://stackit.com/"><img width="192x" src="https://www.startup-atlas.de/uploads/logos/12482.svg" alt="STACKIT - A Brand By Schwarz Digits"></a>
</p>

<p align="center">
<!-- go doc / reference card
<a href="https://godoc.org/sigs.k8s.io/cluster-api-provider-stackit">
<img src="https://godoc.org/sigs.k8s.io/cluster-api-provider-stackit?status.svg"></a>
 -->
<!-- goreportcard badge 
<a href="https://goreportcard.com/report/sigs.k8s.io/cluster-api-provider-stackit">
<img src="https://goreportcard.com/badge/sigs.k8s.io/cluster-api-provider-stackit"></a>
-->
<!-- join kubernetes slack channel for cluster-api-stackit-provider
<a href="http://slack.k8s.io/">
<img src="https://img.shields.io/badge/join%20slack-%23cluster--api--stackit-brightgreen"></a>
 -->
<!-- openssf badge 
<a href="https://bestpractices.coreinfrastructure.org/projects/5688">
<img src="https://bestpractices.coreinfrastructure.org/projects/5688/badge"></a>
-->
</p>

------

Kubernetes-native declarative infrastructure for STACKIT.

## What is the Cluster API Provider STACKIT

The [Cluster API][cluster_api] brings declarative, Kubernetes-style APIs to cluster creation, configuration and management.

The API itself is shared across multiple cloud providers allowing for true STACKIT hybrid deployments of Kubernetes.

## Documentation

Please see our [book](https://todo) for in-depth documentation.

## Launching a Kubernetes cluster on STACKIT

Check out the [Cluster API Quick Start](https://cluster-api.sigs.k8s.io/user/quick-start.html) for launching a cluster on STACKIT.

## Features

- [x] Native Kubernetes manifests and API
- [x] Doesn't use SSH for bootstrapping nodes.
- [x] Installs only the minimal components to bootstrap a control plane and workers.
- [x] Supports control planes on STACKIT VM instances.
- [ ] Manages the bootstrapping of networks, security groups and vm instances.
- [ ] Deploys Kubernetes control planes into private subnets with a separate bastion server.
- [ ] [SKE](https://stackit.com/de/produkte/runtime/stackit-kubernetes-engine) support
- [ ] Choice of Linux distribution using various Images.

------

## Compatibility with Cluster API and Kubernetes Versions

This provider's versions are compatible with the following versions of Cluster API
and support all Kubernetes versions that is supported by its compatible Cluster API version:

|                        | Cluster API v1alpha4 (v0.4) | Cluster API v1beta1 (v1.x) |
| ---------------------- | :-------------------------: | :------------------------: |
| CAPA v1alpha1 `(main)` |              x              |             ✓              |

(See [Kubernetes support matrix][https://cluster-api.sigs.k8s.io/reference/versions.html] of Cluster API versions).


------

## Getting involved and contributing

Are you interested in contributing to cluster-api-provider-stackit? We, the maintainers and community, would love your suggestions, contributions, and help! Also, the maintainers can be contacted at any time to learn more about how to get involved.

We also encourage ALL active community participants to act as if they are maintainers, even if you don't have "official" write permissions. This is a community effort, we are here to serve the Kubernetes community. If you have an active interest and you want to get involved, you have real power! Don't assume that the only people who can get things done around here are the "maintainers".

We also would love to add more "official" maintainers, so show us what you can do!

### Build the images locally

If you want to just build the CAPS containers locally, run

```shell
  REGISTRY=docker.io/my-reg make docker-build
```

### Tilt-based development environment

See [development][development] section for details.
