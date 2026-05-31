# Prerequisites

Required local tools:

- Go v1.24.6 or newer
- Docker or a compatible container tool
- `kubectl`
- `kind`
- `clusterctl`
- `mdbook` for this documentation

Required STACKIT inputs:

- STACKIT project
- STACKIT service-account JSON key
- Existing network in the target region
- Image ID for a cloud-init capable node OS
- Machine type
- Optional SSH key name
- Optional security group IDs

The provider currently expects existing infrastructure inputs. It does not
create networks, security groups, SSH keys, or images. See
[STACKIT Cloud Resources](cloud-resources.md) for the required properties of
those inputs.

Validated development values used during previous local e2e runs included region
`eu01`, a non-ARM Ubuntu 22.04 image, and small `c2i` machine types.
