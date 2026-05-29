#!/usr/bin/env sh
set -eu

kubernetes_version="${KUBERNETES_VERSION:?KUBERNETES_VERSION is required}"
ccm_image="${STACKIT_CLOUD_CONTROLLER_MANAGER_IMAGE:?STACKIT_CLOUD_CONTROLLER_MANAGER_IMAGE is required}"

minor_from_version() {
  version="${1#v}"
  major="${version%%.*}"
  rest="${version#*.}"
  minor="${rest%%.*}"
  if [ -z "$major" ] || [ -z "$minor" ] || [ "$major" = "$version" ]; then
    return 1
  fi
  printf '%s.%s\n' "$major" "$minor"
}

kubernetes_minor="$(minor_from_version "$kubernetes_version")"
case "$kubernetes_minor" in
  1.33|1.34|1.35|1.36) ;;
  *)
    echo "unsupported Kubernetes minor v${kubernetes_minor}.x; supported minors are v1.33.x, v1.34.x, v1.35.x, and v1.36.x" >&2
    exit 1
    ;;
esac

ccm_tag="${ccm_image##*:}"
ccm_tag="${ccm_tag%%@*}"
ccm_minor="$(minor_from_version "$ccm_tag")"

if [ "$ccm_minor" != "$kubernetes_minor" ]; then
  echo "cloud-provider-stackit image minor v${ccm_minor}.x must match Kubernetes minor v${kubernetes_minor}.x" >&2
  exit 1
fi
