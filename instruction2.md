# Aufgabe: Nächste Entwicklungsphase für `cluster-api-provider-stackit`

## Kontext

Es existiert bereits ein MVP eines Cluster API Infrastructure Providers für STACKIT Cloud.

Der Provider kann grundsätzlich:

- `StackitCluster` verwalten
- `StackitMachine` verwalten
- STACKIT VMs erzeugen
- STACKIT VMs löschen
- Bootstrap UserData aus CAPI Bootstrap Secrets verwenden
- mit `KubeadmControlPlane` und `MachineDeployment` zusammenarbeiten
- einen einfachen Workload-Cluster auf STACKIT provisionieren

Ziel dieser nächsten Entwicklungsphase ist **nicht**, sofort neue Features zu bauen.

Ziel ist, den Provider von einem funktionierenden MVP zu einem **contract-korrekten, reproduzierbar testbaren und nutzbaren Cluster API Provider** weiterzuentwickeln.

---

## Übergeordnetes Ziel

Der Provider soll folgende Eigenschaften erreichen:

```text
contract-compliant
providerID-kompatibel
clusterctl-nutzbar
e2e-getestet
leak-sicher
bereit für Feature-Ausbau
```

---

## Nicht-Ziele dieser Phase

In dieser Phase **nicht** implementieren:

- MachinePool
- Hosted Control Planes
- Kamaji Integration
- Gardener Integration
- Autoscaler Integration
- komplexe Multi-AZ-Logik
- vollständig automatisches Netzwerk-Provisioning
- produktionsreifes Addon-Management
- eigene Bootstrap-Logik
- eigener Control Plane Provider

---

## Priorisierte PR-Reihenfolge

Implementiere die nächsten Schritte in dieser Reihenfolge:

```text
PR 1: CAPI Contract Audit und Contract Fixes
PR 2: ProviderID-Kompatibilität mit cloud-provider-stackit verifizieren
PR 3: cloud-provider-stackit als Workload-Cluster Addon integrieren
PR 4: E2E Create/Delete Test inklusive Leak Cleanup
PR 5: clusterctl Release Packaging
PR 6: Worker Scale Test
PR 7: Kubernetes Upgrade Test
PR 8: FailureDomains / Availability Zones
PR 9: ClusterClass Support
```

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

## PR 1: CAPI Contract Audit und Fixes

### Ziel

Stelle sicher, dass der Provider dem aktuellen Cluster API Infrastructure Provider Contract entspricht.

### Zu prüfen

Prüfe insbesondere:

```text
StackitCluster Contract
StackitMachine Contract
Status-Felder
Spec-Felder
Conditions
ProviderID
Finalizer
OwnerReferences
Paused Handling
ObservedGeneration
```

### StackitMachine Contract

Prüfe, ob `StackitMachine` ein Feld für `spec.providerID` besitzt.

Falls aktuell nur folgendes existiert:

```yaml
status:
  providerID: ...
```

dann ändere die API so, dass zusätzlich oder primär folgendes existiert:

```yaml
spec:
  providerID: ...
```

### Zielzustand

```yaml
apiVersion: infrastructure.cluster.x-k8s.io/v1alpha1
kind: StackitMachine
spec:
  providerID: stackit://...
status:
  initialization:
    provisioned: true
  conditions: []
```

### Akzeptanzkriterien

- `StackitMachine.spec.providerID` wird gesetzt, sobald die VM existiert.
- `StackitMachine.status.initialization.provisioned` wird gesetzt.
- `StackitMachine.status.ready` darf aus Kompatibilitätsgründen zusätzlich gesetzt werden.
- `conditions` enthalten sinnvolle Ready-/Failure-Zustände.
- `observedGeneration` wird korrekt gepflegt.

### StackitCluster Contract

Prüfe, ob `StackitCluster` den Control Plane Endpoint korrekt bereitstellt.

### Zielzustand

```yaml
apiVersion: infrastructure.cluster.x-k8s.io/v1alpha1
kind: StackitCluster
spec:
  controlPlaneEndpoint:
    host: ...
    port: 6443
status:
  initialization:
    provisioned: true
  conditions: []
```

Falls der Endpoint aktuell nur in `status.apiServerEndpoint` steht, ergänze oder migriere auf das erwartete Contract-Feld.

### Akzeptanzkriterien

- `StackitCluster.spec.controlPlaneEndpoint` oder das vom aktuellen CAPI Contract erwartete Feld wird korrekt gesetzt.
- `StackitCluster.status.initialization.provisioned=true`, sobald Cluster-Infrastruktur bereit ist.
- `StackitCluster.status.ready` darf zusätzlich gesetzt werden.
- Conditions sind aussagekräftig.
- Paused Cluster werden nicht reconciled.

### Conditions

Implementiere oder korrigiere Conditions.

Mindestens:

```text
Ready
InfrastructureReady
LoadBalancerReady
NetworkReady
InstanceReady
BootstrapDataReady
```

### Regeln

- Conditions müssen `observedGeneration` setzen.
- Conditions müssen sinnvolle `Reason`-Werte haben.
- Conditions dürfen nicht nur generische Fehlermeldungen enthalten.
- Transiente Fehler sollen klar von terminalen Validierungsfehlern getrennt werden.

### Paused Handling

Wenn ein Cluster pausiert ist, darf der Provider nicht weiter reconciliieren.

Zu respektieren:

```yaml
metadata:
  annotations:
    cluster.x-k8s.io/paused: "true"
```

und/oder:

```yaml
spec:
  paused: true
```

### Akzeptanzkriterium

Ein pausierter Cluster führt zu keinem Cloud-API-Aufruf.

---

## PR 2: ProviderID-Kompatibilität mit `cloud-provider-stackit`

### Ziel

Stelle sicher, dass der ProviderID-Wert exakt kompatibel mit `cloud-provider-stackit` ist.

Das ist ein kritischer Integrationspunkt.

Wenn CAPI, STACKIT VM und Kubernetes Node unterschiedliche ProviderIDs verwenden, bleibt die Machine möglicherweise hängen oder bekommt keinen korrekten `NodeRef`.

### Aufgaben

Untersuche im Repository:

```text
https://github.com/stackitcloud/cloud-provider-stackit
```

folgende Punkte:

```text
Welches ProviderID-Format wird verwendet?
Wo wird ProviderID erzeugt?
Wo wird ProviderID gelesen?
Wie wird die Node zur VM gematched?
Welche Rolle spielen Projekt, Region, Server-ID?
```

### Implementierung

Implementiere in:

```text
pkg/cloud/providerid.go
```

folgende Funktionen:

```go
func NewProviderID(projectID string, region string, serverID string) string

func ParseProviderID(providerID string) (projectID string, region string, serverID string, err error)
```

Falls das echte ProviderID-Format keine Projekt- oder Regionsinformation enthält, passe die Signatur sinnvoll an, aber dokumentiere die Entscheidung.

### Tests

Erstelle Unit Tests:

```text
TestNewProviderID
TestParseProviderID
TestProviderIDRoundtrip
TestProviderIDMatchesCloudProviderStackitFormat
```

### E2E-Verifikation

In einem echten Workload-Cluster müssen diese Werte zusammenpassen:

```bash
kubectl get stackitmachine -o yaml
kubectl get machine -o yaml
kubectl get node -o yaml
```

Prüfe:

```text
StackitMachine.spec.providerID
Machine.spec.providerID / Machine.status.nodeRef
Node.spec.providerID
```

### Akzeptanzkriterium

Die CAPI Machine bekommt zuverlässig einen `NodeRef`.

---

## PR 3: `cloud-provider-stackit` als Workload-Cluster Addon integrieren

### Ziel

Der erzeugte Workload-Cluster soll nicht nur Nodes bootstrappen, sondern auch korrekt mit STACKIT als Cloud Provider interagieren.

Dazu soll `cloud-provider-stackit` als Addon integriert werden.

### Aufgaben

Erstelle:

```text
templates/addons/cloud-provider-stackit.yaml
```

oder eine vergleichbare Addon-Struktur.

Das Addon soll im Workload-Cluster installieren:

```text
STACKIT Cloud Controller Manager
ggf. CSI-Komponenten, falls für den Test notwendig
RBAC
ServiceAccount
Deployment/DaemonSet
Config/Secret Referenzen
```

### Wichtig

Der Provider soll das Addon zunächst nicht zwingend automatisch installieren müssen.

Für diese Phase reicht:

```text
Addon Manifest ist vorhanden
E2E Test kann es anwenden
Workload Cluster wird mit CCM Ready
```

### Cluster Template

Aktualisiere:

```text
templates/cluster-template.yaml
```

so, dass der Workload Cluster für externen Cloud Provider vorbereitet ist.

Insbesondere prüfen:

```text
kubelet cloud-provider external
controller-manager cloud-provider external, falls nötig
Node Taints
ProviderID Handling
```

### Akzeptanzkriterien

- Workload Cluster startet mit externem Cloud Provider.
- `cloud-provider-stackit` läuft im Workload Cluster.
- Nodes werden Ready.
- ProviderID wird korrekt verarbeitet.
- `kubectl get nodes` zeigt Control Plane und Worker Nodes als Ready.

---

## PR 4: E2E Create/Delete Test inklusive Leak Cleanup

### Ziel

Erstelle einen echten E2E-Test, der einen STACKIT Cluster erzeugt und wieder löscht.

Dieser Test ist wichtiger als neue Features.

### E2E-Szenario

Implementiere mindestens:

```text
create-cluster-1cp-1worker
delete-cluster-no-leaks
```

### Testablauf

Der Test soll ungefähr folgenden Flow ausführen:

```bash
kind create cluster --name capi-stackit-e2e

clusterctl init \
  --bootstrap kubeadm \
  --control-plane kubeadm \
  --infrastructure stackit

clusterctl generate cluster stackit-test \
  --kubernetes-version v1.31.0 \
  --control-plane-machine-count 1 \
  --worker-machine-count 1 \
  > /tmp/stackit-test.yaml

kubectl apply -f /tmp/stackit-test.yaml
```

Dann warten auf:

```text
Cluster Ready
KubeadmControlPlane Ready
MachineDeployment Available
1 Control Plane Machine Ready
1 Worker Machine Ready
2 Nodes Ready
```

Dann:

```bash
kubectl delete cluster stackit-test
```

Danach prüfen:

```text
keine StackitMachines
keine Machines
keine STACKIT VMs mit Test-Tags
kein API Load Balancer mit Test-Tags
keine hängen gebliebenen Finalizer
```

### Leak Cleanup

Implementiere ein separates Cleanup-Tool oder Make Target:

```bash
make cleanup-stackit
```

Dieses Cleanup darf nicht von Kubernetes-Objekten abhängig sein.

Es muss direkt über STACKIT APIs anhand von Tags/Labels Ressourcen finden und löschen können.

### Zu löschende Ressourcen

Mindestens:

```text
VMs
Load Balancer
ggf. Volumes
ggf. NICs
ggf. Security Group Regeln, falls vom Provider erzeugt
```

### Pflicht-Tags für Testressourcen

Alle E2E-Ressourcen müssen Tags enthalten:

```text
cluster-api-provider-stackit/e2e=true
cluster-api-provider-stackit/test-id=<unique-id>
cluster.x-k8s.io/cluster-name=<cluster-name>
cluster.x-k8s.io/cluster-namespace=<namespace>
```

### Akzeptanzkriterien

- E2E Create/Delete läuft reproduzierbar.
- Nach erfolgreichem Test bleiben keine Cloud-Ressourcen übrig.
- Nach fehlgeschlagenem Test kann `make cleanup-stackit` Ressourcen entfernen.
- Der Test kann mehrfach hintereinander laufen.

---

## PR 5: clusterctl Release Packaging

### Ziel

Der Provider soll über `clusterctl` nutzbar werden.

### Ziel-UX

Folgendes soll funktionieren:

```bash
clusterctl init --infrastructure stackit
```

und:

```bash
clusterctl generate cluster stackit-test \
  --infrastructure stackit \
  --kubernetes-version v1.31.0 \
  --control-plane-machine-count 1 \
  --worker-machine-count 1
```

### Benötigte Release-Artefakte

Erzeuge:

```text
infrastructure-components.yaml
metadata.yaml
cluster-template.yaml
cluster-template-development.yaml
```

### Provider Metadata

Erstelle eine Provider-Metadata-Datei mit mindestens:

```yaml
apiVersion: clusterctl.cluster.x-k8s.io/v1alpha3
kind: Metadata
releaseSeries:
  - major: 0
    minor: 1
    contract: v1beta2
```

Passe den Contract an die tatsächlich unterstützte Cluster API Version an.

### cluster-template.yaml

Das Template muss folgende Ressourcen enthalten:

```text
Cluster
StackitCluster
KubeadmControlPlane
StackitMachineTemplate für Control Plane
MachineDeployment
StackitMachineTemplate für Worker
KubeadmConfigTemplate
StackitMachineTemplate für Worker
```

### Variablen

Nutze clusterctl-kompatible Variablen:

```text
${CLUSTER_NAME}
${NAMESPACE}
${KUBERNETES_VERSION}
${CONTROL_PLANE_MACHINE_COUNT}
${WORKER_MACHINE_COUNT}
${STACKIT_PROJECT_ID}
${STACKIT_REGION}
${STACKIT_NETWORK_ID}
${STACKIT_IMAGE_ID}
${STACKIT_MACHINE_TYPE}
${STACKIT_SSH_KEY_NAME}
${STACKIT_CREDENTIALS_SECRET_NAME}
```

### Akzeptanzkriterien

- Provider kann lokal via `clusterctl` installiert werden.
- Cluster-Template kann generiert werden.
- Generiertes Template kann angewendet werden.
- Bestehender manueller Flow wird durch clusterctl ersetzt.

---

## PR 6: Worker Scale Test

### Ziel

Stelle sicher, dass `MachineDeployment`-Scaling funktioniert.

### Szenario

Ausgangszustand:

```text
1 Control Plane Machine
1 Worker Machine
```

Skalieren auf:

```text
3 Worker Machines
```

Dann zurück auf:

```text
1 Worker Machine
```

### Testbefehle

```bash
kubectl scale machinedeployment <name> --replicas=3
```

Dann warten:

```text
3 Worker Machines Ready
3 Worker Nodes Ready
```

Dann:

```bash
kubectl scale machinedeployment <name> --replicas=1
```

Dann prüfen:

```text
nur noch 1 Worker Machine
nur noch 1 Worker Node
gelöschte STACKIT VMs sind entfernt
keine orphaned Volumes/NICs
```

### Akzeptanzkriterien

- Scale up erzeugt neue VMs.
- Scale down löscht VMs.
- Machines bekommen korrekte ProviderIDs.
- Nodes werden korrekt entfernt.
- Keine Ressourcen-Leaks.

---

## PR 7: Kubernetes Upgrade Test

### Ziel

Prüfe, ob einfache Kubernetes-Upgrades funktionieren.

### Szenario

Starte Cluster mit Version:

```text
v1.31.x
```

Upgrade auf:

```text
v1.32.x
```

oder eine andere aktuell unterstützte Version.

### Zu testen

```text
KubeadmControlPlane Upgrade
MachineDeployment Rolling Upgrade
neue Machines werden erzeugt
alte Machines werden gelöscht
Nodes joinen korrekt
ProviderID bleibt korrekt
```

### Akzeptanzkriterien

- Control Plane Upgrade läuft erfolgreich.
- Worker Upgrade läuft erfolgreich.
- Alte VMs werden gelöscht.
- Neue VMs werden Ready.
- Workload Cluster bleibt erreichbar.

---

## PR 8: FailureDomains / Availability Zones

### Ziel

Modelliere STACKIT Availability Zones als CAPI FailureDomains.

### StackitCluster Status

Ergänze:

```yaml
status:
  failureDomains:
    eu01-1:
      controlPlane: true
    eu01-2:
      controlPlane: true
    eu01-3:
      controlPlane: true
```

### StackitMachine

`StackitMachine.spec.availabilityZone` soll mit FailureDomains kompatibel sein.

### Verhalten

- Der Cluster Controller ermittelt oder validiert verfügbare Availability Zones.
- FailureDomains werden im Cluster Status veröffentlicht.
- Control Plane und Worker Machines können über FailureDomains verteilt werden.

### Akzeptanzkriterien

- FailureDomains erscheinen im Status.
- Machines können in verschiedenen AZs erstellt werden.
- Ungültige AZs führen zu klaren Conditions.
- Bestehende Single-AZ Templates funktionieren weiter.

---

## PR 9: ClusterClass Support

### Ziel

Füge ClusterClass-Support hinzu, nachdem Create/Delete, Scale und Upgrade stabil funktionieren.

### Artefakte

Erzeuge:

```text
templates/clusterclass.yaml
templates/cluster-template-topology.yaml
```

### ClusterClass soll unterstützen

```text
Kubernetes Version
Control Plane Machine Count
Worker Machine Count
Machine Type
Image ID
Region
Network ID
SSH Key Name
Credentials Secret
```

### Akzeptanzkriterien

Folgender Flow funktioniert:

```bash
kubectl apply -f templates/clusterclass.yaml
kubectl apply -f templates/cluster-template-topology.yaml
```

Dann:

```text
Cluster wird erzeugt
Control Plane wird Ready
Worker werden Ready
Delete funktioniert
```

---

## Make Targets

Ergänze oder überprüfe folgende Make Targets:

```makefile
make generate
make manifests
make install
make uninstall
make run
make docker-build
make docker-push
make deploy
make undeploy

make test
make test-unit
make test-envtest
make test-e2e-create
make test-e2e-delete
make test-e2e-scale
make test-e2e-upgrade

make clusterctl-release
make cleanup-stackit
```

---

## Qualitätstore

Vor Feature-Ausbau müssen folgende Gates grün sein:

```text
go test ./...
envtest grün
providerID tests grün
create/delete e2e grün
cleanup-stackit funktioniert
clusterctl generate cluster funktioniert
cloud-provider-stackit Addon funktioniert
keine Ressourcen-Leaks
```

---

## Wichtige Designregeln

### Idempotenz

Jeder Reconcile muss wiederholbar sein.

Keine Cloud-Ressource darf doppelt erzeugt werden, wenn der Reconcile erneut läuft.

### Tags zuerst

Keine STACKIT Ressource ohne eindeutige Tags erzeugen.

Pflicht-Tags:

```text
cluster.x-k8s.io/cluster-name
cluster.x-k8s.io/cluster-namespace
cluster.x-k8s.io/machine-name
cluster.x-k8s.io/machine-uid
cluster-api-provider-stackit/managed=true
```

### Keine SDK Calls direkt im Reconciler

Reconciler dürfen nur gegen interne Interfaces sprechen.

Erlaubt:

```go
cloudClient.CreateServer(...)
```

Nicht erlaubt:

```go
stackitSDK.ServersAPI.CreateServer(...)
```

### Fehlerbehandlung

Klassifiziere Fehler:

```text
NotFound
AlreadyExists
Transient
RateLimited
Unauthorized
InvalidConfiguration
Terminal
```

Verhalten:

```text
Transient -> requeue
RateLimited -> requeue with backoff
Unauthorized -> Condition setzen, nicht aggressiv requeue
InvalidConfiguration -> Condition setzen
NotFound bei Delete -> Erfolg
AlreadyExists bei Create -> adoptieren oder finden
```

### Finalizer

Finalizer dürfen nicht hängen bleiben.

Delete-Reconcile muss robust sein gegen:

```text
bereits gelöschte VM
bereits gelöschter Load Balancer
fehlende Credentials
teilweise gelöschte Ressourcen
Cloud API transient failures
```

### Observability

Logs müssen strukturierte Felder enthalten:

```text
cluster
namespace
machine
stackitMachine
serverID
providerID
reconcileID
```

---

## Wichtigster nächster Meilenstein

Der wichtigste nächste Meilenstein lautet:

```text
clusterctl generate cluster erzeugt einen STACKIT Cluster,
der inklusive cloud-provider-stackit Ready wird,
Worker skalieren kann
und beim Löschen garantiert keine STACKIT Ressourcen liegen lässt.
```

---

## Reihenfolge strikt einhalten

Wichtig:

Erst:

```text
Contract
ProviderID
CCM Addon
E2E Delete/Leak Safety
clusterctl Packaging
```

Dann erst:

```text
FailureDomains
ClusterClass
MachinePool
Autoscaler
Hosted Control Planes
```
