# Agentic-Coding-Spezifikation: Cluster API Infrastructure Provider für STACKIT

## 1. Ziel

Baue einen neuen Kubernetes Cluster API Infrastructure Provider für STACKIT Cloud.

Repository-Name:

```text
cluster-api-provider-stackit
```

Kurzname im Code und in clusterctl:

```text
stackit
```

Provider-Label für clusterctl:

```text
infrastructure-stackit
```

Der Provider soll als MVP einen Kubernetes-Workload-Cluster auf STACKIT-VMs erstellen und löschen können.

Der Management-Cluster läuft lokal, typischerweise via `kind`.

Der Workload-Cluster läuft auf STACKIT Compute Engine.

Für Bootstrap und Control Plane werden bestehende Cluster-API-Provider verwendet:

```text
Core Provider:          cluster-api
Bootstrap Provider:     kubeadm / CABPK
Control Plane Provider: kubeadm / KCP
Infrastructure Provider: cluster-api-provider-stackit
```

Dieser Provider ist also ausschließlich ein Infrastructure Provider.

---

## 2. Wichtige Referenzen

Der Agent soll diese Projekte als Referenzen verwenden:

```text
/Users/c.voigt/go/src/tangled.org/voigt.tngl.sh/cloud-provider-stackit
/Users/c.voigt/go/src/tangled.org/voigt.tngl.sh/machine-controller-manager-provider-stackit
/Users/c.voigt/go/src/tangled.org/voigt.tngl.sh/cluster-api
/Users/c.voigt/go/src/tangled.org/voigt.tngl.sh/stackit-sdk-go
```

Die bestehende STACKIT-MCM-Implementierung ist die wichtigste Referenz für:

```text
STACKIT SDK usage
VM lifecycle
server create/delete/get/list
credential handling
project-id handling
serviceaccount.json handling
networking fields
machineType
imageId
bootVolume
securityGroups
availabilityZone
labels/metadata
userData/cloud-init
```

Der STACKIT Cloud Provider ist die wichtigste Referenz für:

```text
providerID format
Cloud Controller Manager integration
Node identification
LoadBalancer behavior
CSI integration
Kubernetes minor-version compatibility
```

---

## 3. Nicht-Ziele für MVP

Nicht implementieren:

```text
eigener Bootstrap Provider
eigener Control Plane Provider
Hosted Control Planes
MachinePool
ClusterClass
Multi-AZ Scheduling
Autoscaler Integration
automatisches CNI Deployment
automatisches CSI Deployment
automatisches CCM Deployment
vollständiges Netzwerk-Provisioning
Cluster API Runtime Extensions
Managed Kubernetes / SKE Integration
```

Der MVP soll bewusst klein bleiben.

---

## 4. MVP-Erfolgskriterium

Folgender Ablauf soll am Ende funktionieren:

```bash
kind create cluster --name capi-stackit

clusterctl init \
  --bootstrap kubeadm \
  --control-plane kubeadm

make install
make run
```

Dann:

```bash
kubectl apply -f templates/cluster-template.yaml
```

Erwartung:

```bash
kubectl get clusters
kubectl get machines
kubectl get stackitclusters
kubectl get stackitmachines
```

zeigt sinngemäß:

```text
Cluster Ready
Machine Ready
StackitCluster Ready
StackitMachine Ready
```

Danach soll ein kubeconfig für den Workload-Cluster funktionieren:

```bash
kubectl --kubeconfig workload.kubeconfig get nodes
```

Erwartet:

```text
1 control-plane node
1 worker node
```

Delete-Test:

```bash
kubectl delete cluster <cluster-name>
```

Erwartung:

```text
alle vom Provider erzeugten STACKIT VMs gelöscht
API Server Load Balancer gelöscht, falls vom Provider erzeugt
keine dangling finalizers
keine orphaned STACKIT Ressourcen
```

---

## 5. Architektur

Der Provider besteht aus vier Hauptteilen:

```text
API types / CRDs
Controllers
Cloud abstraction layer
clusterctl templates and release assets
```

Zielstruktur:

```text
cluster-api-provider-stackit/
├── api/
│   └── v1alpha1/
│       ├── groupversion_info.go
│       ├── stackitcluster_types.go
│       ├── stackitclustertemplate_types.go
│       ├── stackitmachine_types.go
│       └── stackitmachinetemplate_types.go
├── controllers/
│   ├── stackitcluster_controller.go
│   └── stackitmachine_controller.go
├── pkg/
│   ├── cloud/
│   │   ├── client.go
│   │   ├── errors.go
│   │   ├── loadbalancers.go
│   │   ├── networks.go
│   │   ├── providerid.go
│   │   ├── servers.go
│   │   ├── tags.go
│   │   └── userdata.go
│   ├── cloud/fake/
│   │   └── client.go
│   ├── scope/
│   │   ├── cluster_scope.go
│   │   └── machine_scope.go
│   └── util/
│       ├── conditions.go
│       └── ownerrefs.go
├── config/
│   ├── crd/
│   ├── manager/
│   ├── rbac/
│   └── default/
├── templates/
│   ├── cluster-template.yaml
│   └── clusterctl.yaml
├── test/
│   ├── unit/
│   ├── envtest/
│   └── e2e/
├── Dockerfile
├── Makefile
├── go.mod
└── main.go
```

---

## 6. API Group und Version

API group:

```text
infrastructure.cluster.x-k8s.io
```

Version:

```text
v1alpha1
```

Kinds:

```text
StackitCluster
StackitClusterTemplate
StackitMachine
StackitMachineTemplate
```

---

## 7. Cluster API Contract-Ziel

Der Provider muss die CAPI Infrastructure Provider Contracts erfüllen.

Insbesondere:

```text
StackitCluster ist ein InfraCluster
StackitMachine ist eine InfraMachine
StackitCluster wird von Cluster.spec.infrastructureRef referenziert
StackitMachine wird von Machine.spec.infrastructureRef referenziert
StackitMachine muss ProviderID liefern
StackitMachine muss Addresses liefern, sobald bekannt
StackitMachine muss Ready/Conditions setzen
StackitCluster muss API endpoint liefern
```

Wichtig: Nicht auf zufälliges Verhalten von CAPI-Core verlassen, sondern nur auf dokumentierte Contract-Felder.

---

## 8. CRD: StackitCluster

### 8.1 Zweck

`StackitCluster` beschreibt cluster-weite STACKIT-Infrastruktur.

Im MVP soll `StackitCluster` primär:

```text
Credentials referenzieren
Region/Projekt definieren
bestehendes Netzwerk referenzieren
API Server Endpoint bereitstellen
optional API Server Load Balancer erzeugen
```

### 8.2 Spec

```yaml
apiVersion: infrastructure.cluster.x-k8s.io/v1alpha1
kind: StackitCluster
metadata:
  name: example
  namespace: default
spec:
  projectID: "00000000-0000-0000-0000-000000000000"
  region: "eu01"

  credentialsSecretRef:
    name: stackit-credentials
    namespace: default

  network:
    id: "00000000-0000-0000-0000-000000000000"

  apiServerLoadBalancer:
    enabled: true
```

### 8.3 Go-Typen

```go
type StackitClusterSpec struct {
    ProjectID string `json:"projectID"`
    Region string `json:"region"`

    CredentialsSecretRef corev1.SecretReference `json:"credentialsSecretRef"`

    Network StackitClusterNetworkSpec `json:"network"`

    APIServerLoadBalancer StackitAPIServerLoadBalancerSpec `json:"apiServerLoadBalancer,omitempty"`
}

type StackitClusterNetworkSpec struct {
    ID string `json:"id"`
}

type StackitAPIServerLoadBalancerSpec struct {
    Enabled bool `json:"enabled,omitempty"`
}
```

### 8.4 Status

```yaml
status:
  ready: true
  apiServerEndpoint:
    host: "203.0.113.10"
    port: 6443
  conditions: []
```

### 8.5 Go-Typen

```go
type StackitClusterStatus struct {
    Ready bool `json:"ready,omitempty"`

    APIServerEndpoint clusterv1.APIEndpoint `json:"apiServerEndpoint,omitempty"`

    Conditions clusterv1.Conditions `json:"conditions,omitempty"`
}
```

### 8.6 Verhalten

Der `StackitClusterReconciler` muss:

```text
Owner Cluster finden
Finalizer setzen
Credentials Secret laden
STACKIT Client initialisieren
Netzwerk validieren
API Server Load Balancer sicherstellen, falls enabled
status.apiServerEndpoint setzen
status.ready setzen
Conditions setzen
bei Delete erzeugte Ressourcen löschen
Finalizer entfernen
```

### 8.7 MVP-Vereinfachung

Für MVP darf das Netzwerk bereits existieren.

Der Provider muss kein neues Netzwerk erzeugen.

---

## 9. CRD: StackitClusterTemplate

### 9.1 Zweck

`StackitClusterTemplate` wird für ClusterClass oder spätere Topology-Nutzung vorbereitet.

Für MVP reicht ein Standard-CAPI-Template.

### 9.2 Spec

```yaml
apiVersion: infrastructure.cluster.x-k8s.io/v1alpha1
kind: StackitClusterTemplate
metadata:
  name: example
  namespace: default
spec:
  template:
    spec:
      projectID: "00000000-0000-0000-0000-000000000000"
      region: "eu01"
      credentialsSecretRef:
        name: stackit-credentials
        namespace: default
      network:
        id: "00000000-0000-0000-0000-000000000000"
      apiServerLoadBalancer:
        enabled: true
```

---

## 10. CRD: StackitMachine

### 10.1 Zweck

`StackitMachine` beschreibt eine einzelne STACKIT VM, die eine Kubernetes Node werden soll.

### 10.2 Spec

```yaml
apiVersion: infrastructure.cluster.x-k8s.io/v1alpha1
kind: StackitMachine
metadata:
  name: example
  namespace: default
spec:
  imageID: "00000000-0000-0000-0000-000000000000"
  machineType: "c2i.2"
  availabilityZone: "eu01-1"
  sshKeyName: "my-key"

  rootVolume:
    sizeGiB: 50
    performanceClass: "standard"
    deleteOnTermination: true

  network:
    id: "00000000-0000-0000-0000-000000000000"

  securityGroups:
    - "00000000-0000-0000-0000-000000000000"

  additionalLabels:
    environment: "dev"
```

### 10.3 Go-Typen

```go
type StackitMachineSpec struct {
    ImageID string `json:"imageID"`
    MachineType string `json:"machineType"`
    AvailabilityZone string `json:"availabilityZone,omitempty"`
    SSHKeyName string `json:"sshKeyName,omitempty"`

    RootVolume StackitRootVolumeSpec `json:"rootVolume,omitempty"`

    Network StackitMachineNetworkSpec `json:"network"`

    SecurityGroups []string `json:"securityGroups,omitempty"`

    AdditionalLabels map[string]string `json:"additionalLabels,omitempty"`
}

type StackitRootVolumeSpec struct {
    SizeGiB int `json:"sizeGiB,omitempty"`
    PerformanceClass string `json:"performanceClass,omitempty"`
    DeleteOnTermination *bool `json:"deleteOnTermination,omitempty"`
}

type StackitMachineNetworkSpec struct {
    ID string `json:"id"`
}
```

### 10.4 Status

```yaml
status:
  ready: true
  providerID: "stackit://..."
  instanceID: "00000000-0000-0000-0000-000000000000"
  instanceState: "ACTIVE"
  addresses:
    - type: InternalIP
      address: "10.0.0.12"
  conditions: []
```

### 10.5 Go-Typen

```go
type StackitMachineStatus struct {
    Ready bool `json:"ready,omitempty"`

    ProviderID string `json:"providerID,omitempty"`

    InstanceID string `json:"instanceID,omitempty"`

    InstanceState string `json:"instanceState,omitempty"`

    Addresses []clusterv1.MachineAddress `json:"addresses,omitempty"`

    Conditions clusterv1.Conditions `json:"conditions,omitempty"`
}
```

### 10.6 Wichtige Regel: Kein UserData in StackitMachineSpec

`StackitMachine.spec` darf kein direktes `userData` Feld für reguläres Bootstrap enthalten.

Richtig:

```text
KubeadmConfig erzeugt Bootstrap Secret
Machine.spec.bootstrap.dataSecretName zeigt auf Secret
StackitMachineReconciler liest Secret
StackitMachineReconciler übergibt UserData an STACKIT CreateServer
```

Falsch:

```yaml
spec:
  userData: ...
```

### 10.7 Verhalten

Der `StackitMachineReconciler` muss:

```text
Owner Machine finden
Owner Cluster finden
StackitCluster laden
Bootstrap Secret aus Machine.spec.bootstrap.dataSecretName lesen
UserData extrahieren
Credentials laden
STACKIT Client initialisieren
Finalizer setzen
bestehende VM anhand Tags finden
VM erzeugen, falls sie nicht existiert
ProviderID berechnen
status.providerID setzen
status.instanceID setzen
status.instanceState setzen
status.addresses setzen
status.ready setzen
Conditions setzen
bei Delete VM löschen
Finalizer entfernen
```

---

## 11. CRD: StackitMachineTemplate

### 11.1 Zweck

Template für `MachineDeployment` und `KubeadmControlPlane`.

### 11.2 Spec

```yaml
apiVersion: infrastructure.cluster.x-k8s.io/v1alpha1
kind: StackitMachineTemplate
metadata:
  name: example
  namespace: default
spec:
  template:
    spec:
      imageID: "00000000-0000-0000-0000-000000000000"
      machineType: "c2i.2"
      availabilityZone: "eu01-1"
      sshKeyName: "my-key"
      rootVolume:
        sizeGiB: 50
        performanceClass: "standard"
        deleteOnTermination: true
      network:
        id: "00000000-0000-0000-0000-000000000000"
      securityGroups:
        - "00000000-0000-0000-0000-000000000000"
```

---

## 12. Credentials

### 12.1 MVP Secret Format

Verwende das Secret-Format aus dem bestehenden STACKIT MCM Provider als Vorlage.

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: stackit-credentials
  namespace: default
type: Opaque
stringData:
  project-id: "00000000-0000-0000-0000-000000000000"
  serviceaccount.json: |
    {
      "credentials": {
        "iss": "service-account@sa.stackit.cloud",
        "sub": "00000000-0000-0000-0000-000000000000",
        "aud": "stackit"
      },
      "privateKey": "-----BEGIN PRIVATE KEY-----\n...\n-----END PRIVATE KEY-----"
    }
```

### 12.2 Regeln

`StackitCluster.spec.projectID` und Secret-Key `project-id` müssen übereinstimmen, falls beide gesetzt sind.

Wenn nur `project-id` im Secret gesetzt ist, darf der Controller diesen Wert verwenden.

Wenn beide gesetzt sind und nicht übereinstimmen:

```text
kein Provisioning
Condition AuthReady=False oder CredentialsValid=False
Error loggen
nicht paniken
```

---

## 13. Bootstrap User Data

### 13.1 Quelle

UserData kommt aus dem Secret, auf das `Machine.spec.bootstrap.dataSecretName` zeigt.

### 13.2 Unterstützte Keys

In dieser Reihenfolge lesen:

```text
value
userData
```

### 13.3 Verhalten bei fehlendem Secret

Wenn `Machine.spec.bootstrap.dataSecretName` leer ist:

```text
keine VM erzeugen
requeue
Condition BootstrapReady=False
Reason: BootstrapDataSecretMissing
```

Wenn das Secret nicht existiert:

```text
keine VM erzeugen
requeue
Condition BootstrapReady=False
Reason: BootstrapDataSecretNotFound
```

Wenn das Secret existiert, aber keinen unterstützten Key enthält:

```text
keine VM erzeugen
Condition BootstrapReady=False
Reason: BootstrapDataInvalid
```

---

## 14. ProviderID

### 14.1 Ziel

ProviderID ist kritisch.

CAPI kopiert ProviderID von `StackitMachine.status.providerID` nach `Machine.spec.providerID/status`, und der Kubernetes Node muss später denselben ProviderID-Wert bekommen.

### 14.2 Package

Implementiere:

```text
pkg/cloud/providerid.go
```

### 14.3 API

```go
func NewProviderID(projectID, region, serverID string) string
func ParseProviderID(providerID string) (projectID string, region string, serverID string, err error)
```

### 14.4 Format

Das finale Format muss mit `cloud-provider-stackit` kompatibel sein.

Bis das exakt verifiziert ist, verwende ein explizites TODO und einen roten Test:

```go
// TODO: verify providerID format against cloud-provider-stackit.
// Current assumption:
stackit://<project-id>/<region>/<server-id>
```

### 14.5 Test

Schreibe Tests für:

```text
valid providerID generation
valid providerID parsing
invalid providerID parsing
empty values
roundtrip
```

---

## 15. Labels / Tags / Metadata

Jede erzeugte Cloud-Ressource muss eindeutige Tags/Labels/Metadata erhalten.

Minimal:

```text
cluster.x-k8s.io/cluster-name
cluster.x-k8s.io/cluster-namespace
cluster.x-k8s.io/machine-name
cluster.x-k8s.io/machine-uid
cluster.x-k8s.io/managed-by=cluster-api-provider-stackit
```

Für Cluster-weite Ressourcen:

```text
cluster.x-k8s.io/cluster-name
cluster.x-k8s.io/cluster-namespace
cluster.x-k8s.io/managed-by=cluster-api-provider-stackit
```

Regel:

```text
Controller müssen Ressourcen über diese Tags wiederfinden können.
Nicht ausschließlich auf lokal gespeicherte Status-IDs verlassen.
```

---

## 16. Cloud Client Abstraktion

### 16.1 Ziel

Controller dürfen keine direkten STACKIT SDK Calls enthalten.

SDK-Zugriff gehört in `pkg/cloud`.

### 16.2 Interface

```go
type Client interface {
    GetServer(ctx context.Context, id string) (*Server, error)

    FindServerByTags(ctx context.Context, tags map[string]string) (*Server, error)

    CreateServer(ctx context.Context, input CreateServerInput) (*Server, error)

    DeleteServer(ctx context.Context, id string) error

    GetNetwork(ctx context.Context, id string) (*Network, error)

    EnsureAPIServerLoadBalancer(ctx context.Context, input LoadBalancerInput) (*LoadBalancer, error)

    DeleteAPIServerLoadBalancer(ctx context.Context, id string) error
}
```

### 16.3 Domain Types

```go
type Server struct {
    ID string
    Name string
    State string
    ProviderID string
    Addresses []Address
}

type Address struct {
    Type string
    Address string
}

type Network struct {
    ID string
    Name string
}

type LoadBalancer struct {
    ID string
    Name string
    IP string
    DNSName string
    Port int32
}

type CreateServerInput struct {
    Name string
    ProjectID string
    Region string
    ImageID string
    MachineType string
    AvailabilityZone string
    SSHKeyName string
    NetworkID string
    SecurityGroups []string
    UserData []byte
    Tags map[string]string
    RootVolume RootVolumeInput
}

type RootVolumeInput struct {
    SizeGiB int
    PerformanceClass string
    DeleteOnTermination bool
}

type LoadBalancerInput struct {
    Name string
    ProjectID string
    Region string
    NetworkID string
    Tags map[string]string
    Port int32
}
```

### 16.4 Error Types

Implementiere klassifizierbare Fehler:

```go
var ErrNotFound = errors.New("not found")
var ErrUnauthorized = errors.New("unauthorized")
var ErrInvalidInput = errors.New("invalid input")
var ErrConflict = errors.New("conflict")
var ErrTransient = errors.New("transient")
```

Controller-Verhalten:

```text
ErrNotFound bei Get während Reconcile -> create
ErrNotFound bei Delete -> success
ErrUnauthorized -> Condition False, kein aggressives Requeue
ErrInvalidInput -> Condition False, kein aggressives Requeue
ErrConflict -> requeue
ErrTransient -> requeue
```

---

## 17. Fake Cloud Client

Implementiere:

```text
pkg/cloud/fake/client.go
```

Der Fake Client muss unterstützen:

```text
CreateServer
GetServer
FindServerByTags
DeleteServer
GetNetwork
EnsureAPIServerLoadBalancer
DeleteAPIServerLoadBalancer
failure injection
```

Failure Injection Beispiel:

```go
fakeClient.FailNextCreateServer = cloud.ErrTransient
```

Der Fake Client muss für Unit- und Envtest-Tests verwendet werden.

---

## 18. StackitClusterReconciler Details

### 18.1 Reconcile Ablauf

Pseudocode:

```text
get StackitCluster
if not found: return

get owner Cluster
if owner Cluster missing: return

if deleting:
  delete load balancer if owned
  remove finalizer
  return

add finalizer

load credentials
create cloud client

validate network exists

if apiServerLoadBalancer.enabled:
  ensure API server load balancer
  set status.apiServerEndpoint.host
  set status.apiServerEndpoint.port = 6443
else:
  require endpoint from spec or fail condition

set LoadBalancerReady condition
set NetworkReady condition
set Ready condition
set status.ready = true

patch status
```

### 18.2 Finalizer

```text
stackitcluster.infrastructure.cluster.x-k8s.io
```

### 18.3 Conditions

Mindestens:

```text
Ready
NetworkReady
LoadBalancerReady
CredentialsReady
```

---

## 19. StackitMachineReconciler Details

### 19.1 Reconcile Ablauf

Pseudocode:

```text
get StackitMachine
if not found: return

get owner Machine
if owner Machine missing: return

get owner Cluster
get StackitCluster from Cluster.spec.infrastructureRef

if deleting:
  find server by status.instanceID or tags
  delete server
  remove finalizer
  return

add finalizer

if StackitCluster not ready:
  requeue
  condition InfrastructureReady=False
  return

load bootstrap data from Machine.spec.bootstrap.dataSecretName
if missing:
  condition BootstrapReady=False
  requeue
  return

load credentials
create cloud client

find existing server by:
  1. status.instanceID if set
  2. tags

if server not found:
  create server with:
    imageID
    machineType
    availabilityZone
    networkID
    securityGroups
    rootVolume
    sshKeyName
    userData
    tags

compute providerID

set status:
  instanceID
  instanceState
  providerID
  addresses
  ready=true when server is created and providerID known

set conditions:
  BootstrapReady=True
  InstanceReady=True
  Ready=True

patch status
```

### 19.2 Finalizer

```text
stackitmachine.infrastructure.cluster.x-k8s.io
```

### 19.3 Conditions

Mindestens:

```text
Ready
BootstrapReady
CredentialsReady
InstanceReady
```

---

## 20. Idempotenz-Regeln

Alle Reconcile-Schritte müssen idempotent sein.

Regeln:

```text
CreateServer darf keine zweite VM erzeugen, wenn eine VM mit passenden Tags existiert.
EnsureAPIServerLoadBalancer darf keinen zweiten LB erzeugen, wenn einer mit passenden Tags existiert.
DeleteServer muss erfolgreich sein, wenn die VM schon weg ist.
DeleteLoadBalancer muss erfolgreich sein, wenn der LB schon weg ist.
Status darf jederzeit aus Cloud-State rekonstruiert werden.
```

---

## 21. Webhooks / Validation

Für MVP minimale Validierung per Kubebuilder markers oder einfache Validating Webhook.

Validieren:

```text
region nicht leer
machineType nicht leer
imageID nicht leer
network.id nicht leer
credentialsSecretRef.name nicht leer
rootVolume.sizeGiB >= 0
availabilityZone optional, aber falls gesetzt: Format grob prüfen
sshKeyName optional, aber max length 127
securityGroups optional
```

Kein Overengineering.

---

## 22. cluster-template.yaml

Erzeuge:

```text
templates/cluster-template.yaml
```

Es muss enthalten:

```text
Cluster
StackitCluster
KubeadmControlPlane
StackitMachineTemplate für Control Plane
MachineDeployment
StackitMachineTemplate für Worker
KubeadmConfigTemplate
```

### 22.1 Variablen

Nutze clusterctl-style Variablen:

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
${STACKIT_AVAILABILITY_ZONE}
${STACKIT_SSH_KEY_NAME}
${STACKIT_SECURITY_GROUP_ID}
```

### 22.2 Beispielstruktur

```yaml
apiVersion: cluster.x-k8s.io/v1beta1
kind: Cluster
metadata:
  name: ${CLUSTER_NAME}
  namespace: ${NAMESPACE}
spec:
  clusterNetwork:
    pods:
      cidrBlocks:
        - 192.168.0.0/16
    services:
      cidrBlocks:
        - 10.128.0.0/12
  infrastructureRef:
    apiVersion: infrastructure.cluster.x-k8s.io/v1alpha1
    kind: StackitCluster
    name: ${CLUSTER_NAME}
  controlPlaneRef:
    apiVersion: controlplane.cluster.x-k8s.io/v1beta1
    kind: KubeadmControlPlane
    name: ${CLUSTER_NAME}-control-plane
---
apiVersion: infrastructure.cluster.x-k8s.io/v1alpha1
kind: StackitCluster
metadata:
  name: ${CLUSTER_NAME}
  namespace: ${NAMESPACE}
spec:
  projectID: ${STACKIT_PROJECT_ID}
  region: ${STACKIT_REGION}
  credentialsSecretRef:
    name: stackit-credentials
    namespace: ${NAMESPACE}
  network:
    id: ${STACKIT_NETWORK_ID}
  apiServerLoadBalancer:
    enabled: true
```

Der Agent soll die restlichen Objekte vollständig ergänzen.

---

## 23. clusterctl Packaging

Erzeuge später Release Assets:

```text
metadata.yaml
infrastructure-components.yaml
cluster-template.yaml
```

Für lokale Entwicklung soll `clusterctl` mit Overrides funktionieren.

Beispiel:

```yaml
providers:
  - name: stackit
    url: ~/local-repository/infrastructure-stackit/v0.1.0/infrastructure-components.yaml
    type: InfrastructureProvider
```

---

## 24. Makefile Targets

Implementiere mindestens:

```text
make generate
make manifests
make test
make test-envtest
make docker-build
make docker-push
make install
make uninstall
make run
make deploy
make undeploy
```

---

## 25. Tests

### 25.1 Unit Tests

Implementiere Tests für:

```text
ProviderID generation
ProviderID parsing
Bootstrap Secret extraction
Credentials Secret parsing
Tag generation
Cloud error classification
Spec validation helpers
```

### 25.2 Envtest Controller Tests

Implementiere Tests für:

```text
StackitMachine ohne bootstrap.dataSecretName erzeugt keine VM
StackitMachine mit fehlendem Bootstrap Secret erzeugt keine VM
StackitMachine mit Bootstrap Secret erzeugt VM
StackitMachine Delete löscht VM
StackitMachine Delete entfernt Finalizer
StackitCluster mit gültigem Network setzt ready
StackitCluster setzt apiServerEndpoint
StackitCluster Delete löscht Load Balancer
```

### 25.3 E2E Tests

Später:

```text
kind management cluster erstellen
CAPI installieren
CAPSTACKIT installieren
Cluster manifest anwenden
warten bis Control Plane Ready
warten bis Worker Ready
kubeconfig holen
kubectl get nodes
Cluster löschen
Cloud-Ressourcen prüfen
```

---

## 26. Entwicklungsreihenfolge

Der Agent soll exakt in dieser Reihenfolge arbeiten; bereits erledigte steps auslassen.

```text
1. Repository initialisieren
2. Kubebuilder Projekt erzeugen
3. Go Module setzen
4. CAPI Dependencies hinzufügen
5. API Types erzeugen
6. CRD Markers ergänzen
7. DeepCopy/Manifests generieren
8. Fake Cloud Client implementieren
9. ProviderID Package implementieren
10. Bootstrap Secret Reader implementieren
11. Credentials Reader implementieren
12. Tag Helper implementieren
13. StackitMachineReconciler mit Fake Client implementieren
14. StackitMachineReconciler Tests schreiben
15. StackitClusterReconciler mit Fake Client implementieren
16. StackitClusterReconciler Tests schreiben
17. Real STACKIT Client Skeleton implementieren
18. VM Create anbinden
19. VM Get/List/Find anbinden
20. VM Delete anbinden
21. Network Get anbinden
22. Load Balancer Ensure/Delete anbinden
23. cluster-template.yaml schreiben
24. kind-basierter manueller Test
25. clusterctl Packaging
26. E2E Test Skeleton
```

---

## 27. Coding-Regeln

```text
Go verwenden
controller-runtime verwenden
Kubebuilder Patterns verwenden
CAPI util packages verwenden, wo sinnvoll
keine Panics
context.Context überall durchreichen
structured logging verwenden
Status nur per status patch/update ändern
Spec nicht im Controller mutieren
Finalizer sauber behandeln
OwnerReferences respektieren
keine Cloud-Ressourcen ohne Tags erzeugen
keine direkten STACKIT SDK Calls im Controller
transiente Fehler requeue
Auth-/Validation-Fehler als Conditions surfacen
Tests vor echter Cloud-Anbindung schreiben
```

---

## 28. Offene Fragen

Diese Punkte muss der Agent explizit als TODO markieren, falls nicht eindeutig aus den STACKIT-Repos ableitbar:

```text
exaktes providerID format aus cloud-provider-stackit
exakte STACKIT Server State Namen
exakte SDK CreateServer Input-Struktur
exakte Load Balancer API für Kubernetes API Server
ob STACKIT LB L4/TCP für Port 6443 unterstützt
ob Security Groups am Server, NIC oder Network hängen
ob UserData base64-encoded oder plain übergeben wird
ob server metadata/tags/labels getrennte Konzepte sind
ob VM addresses zuverlässig direkt nach Create verfügbar sind
```

---

## 29. Erwartetes Ergebnis des ersten Agent-Laufs

Der erste Agent-Lauf soll nicht versuchen, alles produktionsreif zu machen.

Er soll liefern:

```text
kompilierendes Repository
CRDs
Controller Skeletons
Fake Cloud Client
ProviderID Tests
Bootstrap Secret Tests
StackitMachineReconciler mit Fake Client
StackitClusterReconciler mit Fake Client
initiale cluster-template.yaml
TODOs für echte STACKIT SDK Details
```

Definition of Done für ersten Lauf:

```bash
go test ./...
make manifests
make generate
```

laufen erfolgreich.

---

## 30. Danach

Nach dem ersten stabilen Skeleton folgt der zweite Agent-Lauf:

```text
echten STACKIT SDK Client anbinden
MCM Provider Code gezielt auswerten
ProviderID endgültig klären
VM Create/Delete gegen echte STACKIT Cloud testen
kind Management Cluster verwenden
ersten 1-CP/1-Worker Cluster booten
```
