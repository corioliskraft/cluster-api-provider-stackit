# Developer Guide

Install CRDs into the current cluster:

```sh
make install
```

Run the controller locally against the current kubeconfig context:

```sh
make run
```

Build and deploy the controller image:

```sh
export IMG=<registry>/cluster-api-provider-stackit:<tag>
make docker-build docker-push IMG="$IMG"
make deploy IMG="$IMG"
```

For a local kind management cluster, build and load the image instead of pushing it:

```sh
export IMG=cluster-api-provider-stackit:dev
make docker-build IMG="$IMG"
kind load docker-image "$IMG" --name capi-stackit
make deploy IMG="$IMG"
```

The local development cluster used during validation is `kind-capi-stackit`.
