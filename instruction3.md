# Aufgabe: Nächste Entwicklungsphase für `cluster-api-provider-stackit`

  What is already verified:

  - StackitMachine.spec.providerID and status.providerID are set.
  - CAPI Machine.spec.providerID is populated.
  - Format matches cloud-provider-stackit: stackit://<server-id>.
  - E2E code already checks Machine and StackitMachine provider IDs.

  What is still missing:

  - A real workload cluster Node must appear.
  - cloud-provider-stackit must run inside that workload cluster.
  - The Node must get spec.providerID=stackit://<server-id>.
  - CAPI must match that Node to the Machine and set Machine.status.nodeRef.

  To finish it, we need a real e2e flow that:

  1. Creates a 1 CP / 1 worker workload cluster.
  2. Gets the workload-cluster kubeconfig.
  3. Installs required workload addons:
      - CNI, so Nodes can become Ready. Use cilium as cni.
      - templates/addons/cloud-provider-stackit.yaml.
  4. Waits until Nodes exist and have spec.providerID.
  5. Asserts alignment:
      - StackitMachine.spec.providerID
      - StackitMachine.status.providerID
      - Machine.spec.providerID
      - Node.spec.providerID
        all equal stackit://<server-id>.
  6. Asserts Machine.status.nodeRef.name points to that Node.
  7. Deletes the cluster and verifies cleanup.

So the open task belongs after/with PR3 completion. We have the static compatibility and addon manifest, but not yet the full workload-cluster CCM + NodeRef validation.
  
Once this is done, check the relevant todos in `todo.md`.  
  
---

Wichtig: nutze jj bookmarks um die Entwicklungschritte sauber voneinander abzugrenzen. Mache keine riesen commits, lieber kleinere. Achte auf aussagekräftige change descriptions in englisch.

---

## 2. Wichtige Referenzen

Der Agent soll diese Projekte als Referenzen verwenden:

```text
/Users/c.voigt/go/src/tangled.org/voigt.tngl.sh/cloud-provider-stackit
/Users/c.voigt/go/src/tangled.org/voigt.tngl.sh/machine-controller-manager-provider-stackit
/Users/c.voigt/go/src/tangled.org/voigt.tngl.sh/cluster-api
/Users/c.voigt/go/src/tangled.org/voigt.tngl.sh/stackit-sdk-go
```

---
