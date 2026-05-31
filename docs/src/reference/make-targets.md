# Make Targets

Common targets:

| Target | Purpose |
| --- | --- |
| `make test` | Run generated checks, formatting, vet, and non-e2e tests |
| `make manifests` | Regenerate CRDs and RBAC |
| `make generate` | Regenerate DeepCopy code |
| `make install` | Install CRDs into the current cluster |
| `make uninstall` | Remove CRDs from the current cluster |
| `make docker-build` | Build the controller image |
| `make deploy` | Deploy the controller into the current cluster |
| `make undeploy` | Remove the controller deployment |
| `make build-installer` | Generate `dist/install.yaml` |
| `make clusterctl-release` | Generate local clusterctl release assets |
| `make cleanup-stackit` | Delete tagged e2e resources through STACKIT APIs |
| `make install-workload-cni` | Install Cilium, Calico, or a custom CNI manifest into a workload cluster |

Documentation targets inside `docs/`:

| Target | Purpose |
| --- | --- |
| `make build` | Build the mdBook |
| `make serve` | Serve the mdBook locally |
| `make clean` | Remove generated book output |
