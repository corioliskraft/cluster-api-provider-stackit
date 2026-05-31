# PR Plan

The current development phase is organized into small PR-sized changes:

| PR | Topic | Status |
| --- | --- | --- |
| PR 1 | Cluster API contract audit and fixes | done |
| PR 2 | ProviderID compatibility | real NodeRef alignment verified |
| PR 3 | `cloud-provider-stackit` addon | embedded in default template |
| PR 4 | create/delete e2e and cleanup | core flow implemented |
| PR 5 | clusterctl release packaging | done |
| PR 6 | worker scale e2e | infra and workload Node readiness flows done |
| PR 7 | Kubernetes upgrade e2e | worker and control-plane workload flows done |
| PR 8 | failure domains / availability zones | done |
| PR 9 | ClusterClass support | topology workload create/ready/delete done |
| PR 10 | ClusterTopology validation flow | local config and workload validation done |
| PR 11 | mdBook documentation | this documentation |

Keep future changes granular and bookmark each PR-sized change in jj.
