#!/usr/bin/env bash
set -euo pipefail

workload_kubeconfig="${WORKLOAD_KUBECONFIG:-${KUBECONFIG:-}}"
cni="${STACKIT_WORKLOAD_CNI:-cilium}"
cilium_version="${CILIUM_VERSION:-1.19.4}"
cilium_values="${CILIUM_VALUES:-templates/addons/cilium-values.yaml}"
calico_manifest="${CALICO_MANIFEST:-https://raw.githubusercontent.com/projectcalico/calico/v3.30.0/manifests/calico.yaml}"
custom_manifest="${CNI_MANIFEST:-}"

if [[ -z "${workload_kubeconfig}" ]]; then
	echo "Set WORKLOAD_KUBECONFIG or KUBECONFIG to the workload-cluster kubeconfig path" >&2
	exit 1
fi

if [[ ! -f "${workload_kubeconfig}" ]]; then
	echo "Workload kubeconfig does not exist: ${workload_kubeconfig}" >&2
	exit 1
fi

wait_for_cilium() {
	kubectl --kubeconfig "${workload_kubeconfig}" -n kube-system rollout status deployment/cilium-operator --timeout=10m
	kubectl --kubeconfig "${workload_kubeconfig}" -n kube-system rollout status daemonset/cilium --timeout=10m
	if kubectl --kubeconfig "${workload_kubeconfig}" -n kube-system get daemonset/cilium-envoy >/dev/null 2>&1; then
		kubectl --kubeconfig "${workload_kubeconfig}" -n kube-system rollout status daemonset/cilium-envoy --timeout=10m
	fi
}

wait_for_calico() {
	kubectl --kubeconfig "${workload_kubeconfig}" -n kube-system rollout status daemonset/calico-node --timeout=10m
	kubectl --kubeconfig "${workload_kubeconfig}" -n kube-system rollout status deployment/calico-kube-controllers --timeout=10m
}

if [[ -n "${custom_manifest}" ]]; then
	kubectl --kubeconfig "${workload_kubeconfig}" apply -f "${custom_manifest}"
	exit 0
fi

case "${cni}" in
	cilium)
		command -v cilium >/dev/null 2>&1 || {
			echo "cilium CLI is required for STACKIT_WORKLOAD_CNI=cilium" >&2
			exit 1
		}
		cilium install \
			--kubeconfig "${workload_kubeconfig}" \
			--version "${cilium_version}" \
			--values "${cilium_values}"
		wait_for_cilium
		;;
	calico)
		kubectl --kubeconfig "${workload_kubeconfig}" apply -f "${calico_manifest}"
		wait_for_calico
		;;
	*)
		echo "Unsupported STACKIT_WORKLOAD_CNI=${cni}; use cilium, calico, or CNI_MANIFEST" >&2
		exit 1
		;;
esac
