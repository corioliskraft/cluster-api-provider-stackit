//go:build e2e
// +build e2e

/*
Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package e2e

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"voigt.tngl.sh/cluster-api-provider-stackit/pkg/cloud"
	"voigt.tngl.sh/cluster-api-provider-stackit/test/utils"
)

// namespace where the project is deployed in
const namespace = "cluster-api-provider-stackit-system"

// serviceAccountName created for the project
const serviceAccountName = "cluster-api-provider-stackit-controller-manager"

// metricsServiceName is the name of the metrics service of the project
const metricsServiceName = "cluster-api-provider-stackit-controller-manager-metrics-service"

// metricsRoleBindingName is the name of the RBAC that will be created to allow get the metrics data
const metricsRoleBindingName = "cluster-api-provider-stackit-metrics-binding"

var _ = Describe("Manager", Ordered, func() {
	var controllerPodName string

	// Before running the tests, set up the environment by creating the namespace,
	// enforce the restricted security policy to the namespace, installing CRDs,
	// and deploying the controller.
	BeforeAll(func() {
		By("ensuring manager namespace exists")
		cmd := exec.Command("kubectl", "get", "ns", namespace)
		var err error
		if _, err := utils.Run(cmd); err != nil {
			cmd = exec.Command("kubectl", "create", "ns", namespace)
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to create namespace")
		}

		By("labeling the namespace to enforce the restricted security policy")
		cmd = exec.Command("kubectl", "label", "--overwrite", "ns", namespace,
			"pod-security.kubernetes.io/enforce=restricted")
		_, err = utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred(), "Failed to label namespace with restricted policy")

		By("installing CRDs")
		cmd = exec.Command("make", "install")
		_, err = utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred(), "Failed to install CRDs")

		By("deploying the controller-manager")
		cmd = exec.Command("make", "deploy", fmt.Sprintf("IMG=%s", managerImage))
		_, err = utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred(), "Failed to deploy the controller-manager")
	})

	// After all tests have been executed, clean up by undeploying the controller, uninstalling CRDs,
	// and deleting the namespace.
	AfterAll(func() {
		By("cleaning up the curl pod for metrics")
		cmd := exec.Command("kubectl", "delete", "pod", "curl-metrics", "-n", namespace)
		_, _ = utils.Run(cmd)

		By("undeploying the controller-manager")
		cmd = exec.Command("make", "undeploy")
		_, _ = utils.Run(cmd)

		By("uninstalling CRDs")
		cmd = exec.Command("make", "uninstall")
		_, _ = utils.Run(cmd)

		By("removing manager namespace")
		cmd = exec.Command("kubectl", "delete", "ns", namespace)
		_, _ = utils.Run(cmd)
	})

	// After each test, check for failures and collect logs, events,
	// and pod descriptions for debugging.
	AfterEach(func() {
		specReport := CurrentSpecReport()
		if specReport.Failed() {
			By("Fetching controller manager pod logs")
			cmd := exec.Command("kubectl", "logs", controllerPodName, "-n", namespace)
			controllerLogs, err := utils.Run(cmd)
			if err == nil {
				_, _ = fmt.Fprintf(GinkgoWriter, "Controller logs:\n %s", controllerLogs)
			} else {
				_, _ = fmt.Fprintf(GinkgoWriter, "Failed to get Controller logs: %s", err)
			}

			By("Fetching Kubernetes events")
			cmd = exec.Command("kubectl", "get", "events", "-n", namespace, "--sort-by=.lastTimestamp")
			eventsOutput, err := utils.Run(cmd)
			if err == nil {
				_, _ = fmt.Fprintf(GinkgoWriter, "Kubernetes events:\n%s", eventsOutput)
			} else {
				_, _ = fmt.Fprintf(GinkgoWriter, "Failed to get Kubernetes events: %s", err)
			}

			By("Fetching curl-metrics logs")
			cmd = exec.Command("kubectl", "logs", "curl-metrics", "-n", namespace)
			metricsOutput, err := utils.Run(cmd)
			if err == nil {
				_, _ = fmt.Fprintf(GinkgoWriter, "Metrics logs:\n %s", metricsOutput)
			} else {
				_, _ = fmt.Fprintf(GinkgoWriter, "Failed to get curl-metrics logs: %s", err)
			}

			By("Fetching controller manager pod description")
			cmd = exec.Command("kubectl", "describe", "pod", controllerPodName, "-n", namespace)
			podDescription, err := utils.Run(cmd)
			if err == nil {
				fmt.Println("Pod description:\n", podDescription)
			} else {
				fmt.Println("Failed to describe controller pod")
			}
		}
	})

	SetDefaultEventuallyTimeout(2 * time.Minute)
	SetDefaultEventuallyPollingInterval(time.Second)

	Context("Manager", func() {
		It("should run successfully", func() {
			By("validating that the controller-manager pod is running as expected")
			verifyControllerUp := func(g Gomega) {
				By("getting the name of the controller-manager pod")
				cmd := exec.Command("kubectl", "get",
					"pods", "-l", "control-plane=controller-manager",
					"-o", "go-template={{ range .items }}"+
						"{{ if not .metadata.deletionTimestamp }}"+
						"{{ .metadata.name }}"+
						"{{ \"\\n\" }}{{ end }}{{ end }}",
					"-n", namespace,
				)

				podOutput, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred(), "Failed to retrieve controller-manager pod information")
				podNames := utils.GetNonEmptyLines(podOutput)
				g.Expect(podNames).To(HaveLen(1), "expected 1 controller pod running")
				controllerPodName = podNames[0]
				g.Expect(controllerPodName).To(ContainSubstring("controller-manager"))

				By("validating the pod's status")
				cmd = exec.Command("kubectl", "get",
					"pods", controllerPodName, "-o", "jsonpath={.status.phase}",
					"-n", namespace,
				)
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(output).To(Equal("Running"), "Incorrect controller-manager pod status")
			}
			Eventually(verifyControllerUp).Should(Succeed())
		})

		It("should ensure the metrics endpoint is serving metrics", func() {
			By("creating a ClusterRoleBinding for the service account to allow access to metrics")
			cmd := exec.Command("kubectl", "create", "clusterrolebinding", metricsRoleBindingName,
				"--clusterrole=cluster-api-provider-stackit-metrics-reader",
				fmt.Sprintf("--serviceaccount=%s:%s", namespace, serviceAccountName),
			)
			_, err := utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to create ClusterRoleBinding")

			By("validating that the metrics service is available")
			cmd = exec.Command("kubectl", "get", "service", metricsServiceName, "-n", namespace)
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Metrics service should exist")

			By("getting the service account token")
			token, err := serviceAccountToken()
			Expect(err).NotTo(HaveOccurred())
			Expect(token).NotTo(BeEmpty())

			By("ensuring the controller pod is ready")
			verifyControllerPodReady := func(g Gomega) {
				cmd := exec.Command("kubectl", "get", "pod", controllerPodName, "-n", namespace,
					"-o", "jsonpath={.status.conditions[?(@.type=='Ready')].status}")
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(output).To(Equal("True"), "Controller pod not ready")
			}
			Eventually(verifyControllerPodReady, 3*time.Minute, time.Second).Should(Succeed())

			By("verifying that the controller manager is serving the metrics server")
			verifyMetricsServerStarted := func(g Gomega) {
				cmd := exec.Command("kubectl", "logs", controllerPodName, "-n", namespace)
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(output).To(ContainSubstring("Serving metrics server"),
					"Metrics server not yet started")
			}
			Eventually(verifyMetricsServerStarted, 3*time.Minute, time.Second).Should(Succeed())

			// +kubebuilder:scaffold:e2e-metrics-webhooks-readiness

			By("creating the curl-metrics pod to access the metrics endpoint")
			cmd = exec.Command("kubectl", "run", "curl-metrics", "--restart=Never",
				"--namespace", namespace,
				"--image=curlimages/curl:latest",
				"--overrides",
				fmt.Sprintf(`{
					"spec": {
						"containers": [{
							"name": "curl",
							"image": "curlimages/curl:latest",
							"command": ["/bin/sh", "-c"],
							"args": [
								"for i in $(seq 1 30); do curl -v -k -H 'Authorization: Bearer %s' https://%s.%s.svc.cluster.local:8443/metrics && exit 0 || sleep 2; done; exit 1"
							],
							"securityContext": {
								"readOnlyRootFilesystem": true,
								"allowPrivilegeEscalation": false,
								"capabilities": {
									"drop": ["ALL"]
								},
								"runAsNonRoot": true,
								"runAsUser": 1000,
								"seccompProfile": {
									"type": "RuntimeDefault"
								}
							}
						}],
						"serviceAccountName": "%s"
					}
				}`, token, metricsServiceName, namespace, serviceAccountName))
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to create curl-metrics pod")

			By("waiting for the curl-metrics pod to complete.")
			verifyCurlUp := func(g Gomega) {
				cmd := exec.Command("kubectl", "get", "pods", "curl-metrics",
					"-o", "jsonpath={.status.phase}",
					"-n", namespace)
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(output).To(Equal("Succeeded"), "curl pod in wrong status")
			}
			Eventually(verifyCurlUp, 5*time.Minute).Should(Succeed())

			By("getting the metrics by checking curl-metrics logs")
			verifyMetricsAvailable := func(g Gomega) {
				metricsOutput, err := getMetricsOutput()
				g.Expect(err).NotTo(HaveOccurred(), "Failed to retrieve logs from curl pod")
				g.Expect(metricsOutput).NotTo(BeEmpty())
				g.Expect(metricsOutput).To(ContainSubstring("< HTTP/1.1 200 OK"))
			}
			Eventually(verifyMetricsAvailable, 2*time.Minute).Should(Succeed())
		})

		// +kubebuilder:scaffold:e2e-webhooks-checks

		It("should create and delete a real STACKIT VM for a workload Cluster Machine", func() {
			if os.Getenv("STACKIT_E2E_CREATE_VMS") != "true" {
				Skip("set STACKIT_E2E_CREATE_VMS=true to run the real STACKIT VM lifecycle e2e test")
			}

			cfg := stackitVMConfigFromEnv()
			ctx := context.Background()
			cloudClient := stackitCloudClientFromCredentialsSecret(ctx, cfg)
			clusterName := fmt.Sprintf("stackit-e2e-%d", time.Now().Unix())
			machineName := clusterName + "-machine-0"

			By("applying a workload Cluster and StackitMachine fixture")
			fixture := renderStackitVMFixture(clusterName, machineName, cfg)
			fixturePath := writeTempManifest("stackit-vm-e2e-*.yaml", fixture)
			defer func() {
				cleanupStackitVMFixture(clusterName, machineName, cfg.Namespace)
				_ = os.Remove(fixturePath)
			}()

			cmd := exec.Command("kubectl", "apply", "-f", fixturePath)
			_, err := utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to apply STACKIT VM lifecycle fixture")

			By("waiting for the StackitCluster to validate credentials and network")
			Eventually(func(g Gomega) {
				output := kubectlOutput(g, "get", "stackitcluster", clusterName, "-n", cfg.Namespace, "-o", "jsonpath={.status.ready}")
				g.Expect(output).To(Equal("true"))
			}, 10*time.Minute, 10*time.Second).Should(Succeed())

			By("waiting for the StackitMachine to provision a VM")
			var instanceID string
			Eventually(func(g Gomega) {
				ready := kubectlOutput(g, "get", "stackitmachine", machineName, "-n", cfg.Namespace, "-o", "jsonpath={.status.ready}")
				g.Expect(ready).To(Equal("true"))
				instanceID = kubectlOutput(g, "get", "stackitmachine", machineName, "-n", cfg.Namespace, "-o", "jsonpath={.status.instanceID}")
				g.Expect(instanceID).NotTo(BeEmpty())
			}, 25*time.Minute, 15*time.Second).Should(Succeed())

			By("verifying the VM exists in STACKIT")
			Eventually(func(g Gomega) {
				server, err := cloudClient.GetServer(ctx, instanceID)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(server.ID).To(Equal(instanceID))
			}, 5*time.Minute, 15*time.Second).Should(Succeed())

			By("deleting the StackitMachine to trigger VM cleanup")
			cmd = exec.Command("kubectl", "delete", "stackitmachine", machineName, "-n", cfg.Namespace, "--wait=true", "--timeout=20m")
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to delete StackitMachine")

			By("verifying the VM was deleted from STACKIT")
			Eventually(func(g Gomega) {
				_, err := cloudClient.GetServer(ctx, instanceID)
				g.Expect(cloud.IsNotFound(err)).To(BeTrue(), "expected server %s to be deleted, got %v", instanceID, err)
			}, 20*time.Minute, 15*time.Second).Should(Succeed())
		})
	})
})

type stackitVMConfig struct {
	Namespace             string
	ProjectID             string
	Region                string
	NetworkID             string
	ImageID               string
	MachineType           string
	AvailabilityZone      string
	SSHKeyName            string
	SecurityGroupIDs      []string
	CredentialsSecretName string
	CredentialsSecretNS   string
	RootVolumeSizeGiB     string
	RootVolumePerformance string
}

func stackitVMConfigFromEnv() stackitVMConfig {
	return stackitVMConfig{
		Namespace:             envDefault("STACKIT_E2E_NAMESPACE", "default"),
		ProjectID:             requiredEnv("STACKIT_PROJECT_ID"),
		Region:                envDefault("STACKIT_REGION", "eu01"),
		NetworkID:             requiredEnv("STACKIT_NETWORK_ID"),
		ImageID:               requiredEnv("STACKIT_IMAGE_ID"),
		MachineType:           requiredEnv("STACKIT_MACHINE_TYPE"),
		AvailabilityZone:      requiredEnv("STACKIT_AVAILABILITY_ZONE"),
		SSHKeyName:            os.Getenv("STACKIT_SSH_KEY_NAME"),
		SecurityGroupIDs:      splitCSV(os.Getenv("STACKIT_SECURITY_GROUP_IDS")),
		CredentialsSecretName: envDefault("STACKIT_CREDENTIALS_SECRET_NAME", "stackit-credentials"),
		CredentialsSecretNS:   envDefault("STACKIT_CREDENTIALS_SECRET_NAMESPACE", envDefault("STACKIT_E2E_NAMESPACE", "default")),
		RootVolumeSizeGiB:     envDefault("STACKIT_ROOT_VOLUME_SIZE_GIB", "50"),
		RootVolumePerformance: envDefault("STACKIT_ROOT_VOLUME_PERFORMANCE_CLASS", "storage_premium_perf6"),
	}
}

func stackitCloudClientFromCredentialsSecret(ctx context.Context, cfg stackitVMConfig) cloud.Client {
	secret := stackitCredentialsSecret(ctx, cfg.CredentialsSecretName, cfg.CredentialsSecretNS)
	serviceAccountJSON, ok := secret["serviceaccount.json"]
	Expect(ok).To(BeTrue(), "credentials Secret is missing serviceaccount.json")
	Expect(serviceAccountJSON).NotTo(BeEmpty(), "credentials Secret serviceaccount.json is empty")

	client, err := cloud.NewClient(ctx, cloud.Credentials{
		ProjectID:          cfg.ProjectID,
		Region:             cfg.Region,
		ServiceAccountJSON: serviceAccountJSON,
	})
	Expect(err).NotTo(HaveOccurred(), "Failed to create STACKIT cloud client")
	return client
}

func stackitCredentialsSecret(_ context.Context, name, namespace string) map[string][]byte {
	cmd := exec.Command("kubectl", "get", "secret", name, "-n", namespace, "-o", "json")
	output, err := utils.Run(cmd)
	Expect(err).NotTo(HaveOccurred(), "Failed to read STACKIT credentials Secret")

	var secret struct {
		Data map[string]string `json:"data"`
	}
	Expect(json.Unmarshal([]byte(output), &secret)).To(Succeed())
	out := map[string][]byte{}
	for key, value := range secret.Data {
		decoded, err := base64.StdEncoding.DecodeString(value)
		Expect(err).NotTo(HaveOccurred(), "Failed to decode Secret key %s", key)
		out[key] = decoded
	}
	return out
}

func renderStackitVMFixture(clusterName, machineName string, cfg stackitVMConfig) string {
	securityGroups := ""
	for _, securityGroupID := range cfg.SecurityGroupIDs {
		securityGroups += fmt.Sprintf("\n        - %s", securityGroupID)
	}
	if securityGroups != "" {
		securityGroups = "\n      securityGroups:" + securityGroups
	}

	sshKeyName := ""
	if cfg.SSHKeyName != "" {
		sshKeyName = fmt.Sprintf("\n      sshKeyName: %s", cfg.SSHKeyName)
	}

	return fmt.Sprintf(`apiVersion: cluster.x-k8s.io/v1beta2
kind: Cluster
metadata:
  name: %[1]s
  namespace: %[3]s
spec:
  infrastructureRef:
    apiGroup: infrastructure.cluster.x-k8s.io
    kind: StackitCluster
    name: %[1]s
  clusterNetwork:
    pods:
      cidrBlocks:
        - 192.168.0.0/16
    services:
      cidrBlocks:
        - 10.128.0.0/12
---
apiVersion: infrastructure.cluster.x-k8s.io/v1alpha1
kind: StackitCluster
metadata:
  name: %[1]s
  namespace: %[3]s
spec:
  projectID: %[4]s
  region: %[5]s
  credentialsSecretRef:
    name: %[6]s
    namespace: %[7]s
  network:
    id: %[8]s
  apiServerLoadBalancer:
    enabled: false
  controlPlaneEndpoint:
    host: 203.0.113.10
    port: 6443
---
apiVersion: v1
kind: Secret
metadata:
  name: %[2]s-bootstrap
  namespace: %[3]s
type: Opaque
data:
  value: IyEvYmluL3NoCmVjaG8gc3RhY2tpdC1lMmUK
---
apiVersion: cluster.x-k8s.io/v1beta2
kind: Machine
metadata:
  name: %[2]s
  namespace: %[3]s
  labels:
    cluster.x-k8s.io/cluster-name: %[1]s
spec:
  clusterName: %[1]s
  bootstrap:
    dataSecretName: %[2]s-bootstrap
  infrastructureRef:
    apiGroup: infrastructure.cluster.x-k8s.io
    kind: StackitMachine
    name: %[2]s
---
apiVersion: infrastructure.cluster.x-k8s.io/v1alpha1
kind: StackitMachine
metadata:
  name: %[2]s
  namespace: %[3]s
spec:
  imageID: %[9]s
  machineType: %[10]s
  availabilityZone: %[11]s%[12]s
  rootVolume:
    sizeGiB: %[13]s
    performanceClass: %[14]s
    deleteOnTermination: true
  network:
    id: %[8]s%[15]s
`, clusterName, machineName, cfg.Namespace, cfg.ProjectID, cfg.Region, cfg.CredentialsSecretName, cfg.CredentialsSecretNS,
		cfg.NetworkID, cfg.ImageID, cfg.MachineType, cfg.AvailabilityZone, sshKeyName, cfg.RootVolumeSizeGiB,
		cfg.RootVolumePerformance, securityGroups)
}

func cleanupStackitVMFixture(clusterName, machineName, namespace string) {
	for _, args := range [][]string{
		{"delete", "stackitmachine", machineName, "-n", namespace, "--ignore-not-found", "--wait=true", "--timeout=20m"},
		{"delete", "machine", machineName, "-n", namespace, "--ignore-not-found", "--wait=true", "--timeout=5m"},
		{"delete", "stackitcluster", clusterName, "-n", namespace, "--ignore-not-found", "--wait=true", "--timeout=5m"},
		{"delete", "cluster", clusterName, "-n", namespace, "--ignore-not-found", "--wait=true", "--timeout=5m"},
		{"delete", "secret", machineName + "-bootstrap", "-n", namespace, "--ignore-not-found"},
	} {
		cmd := exec.Command("kubectl", args...)
		if _, err := utils.Run(cmd); err != nil {
			_, _ = fmt.Fprintf(GinkgoWriter, "cleanup warning: %v\n", err)
		}
	}
}

func writeTempManifest(pattern, content string) string {
	file, err := os.CreateTemp("", pattern)
	Expect(err).NotTo(HaveOccurred(), "Failed to create temporary manifest")
	defer func() {
		Expect(file.Close()).To(Succeed())
	}()
	_, err = file.WriteString(content)
	Expect(err).NotTo(HaveOccurred(), "Failed to write temporary manifest")
	return file.Name()
}

func kubectlOutput(g Gomega, args ...string) string {
	cmd := exec.Command("kubectl", args...)
	output, err := utils.Run(cmd)
	g.Expect(err).NotTo(HaveOccurred())
	return strings.TrimSpace(output)
}

func requiredEnv(name string) string {
	value := os.Getenv(name)
	if value == "" {
		Skip(fmt.Sprintf("%s is required for STACKIT VM e2e tests", name))
	}
	return value
}

func envDefault(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}

func splitCSV(value string) []string {
	if value == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

// serviceAccountToken returns a token for the specified service account in the given namespace.
// It uses the Kubernetes TokenRequest API to generate a token by directly sending a request
// and parsing the resulting token from the API response.
func serviceAccountToken() (string, error) {
	const tokenRequestRawString = `{
		"apiVersion": "authentication.k8s.io/v1",
		"kind": "TokenRequest"
	}`

	By("creating temporary file to store the token request")
	secretName := fmt.Sprintf("%s-token-request", serviceAccountName)
	tokenRequestFile := filepath.Join("/tmp", secretName)
	err := os.WriteFile(tokenRequestFile, []byte(tokenRequestRawString), os.FileMode(0o644))
	if err != nil {
		return "", err
	}

	var out string
	verifyTokenCreation := func(g Gomega) {
		By("executing kubectl command to create the token")
		cmd := exec.Command("kubectl", "create", "--raw", fmt.Sprintf(
			"/api/v1/namespaces/%s/serviceaccounts/%s/token",
			namespace,
			serviceAccountName,
		), "-f", tokenRequestFile)

		output, err := cmd.CombinedOutput()
		g.Expect(err).NotTo(HaveOccurred())

		By("parsing the JSON output to extract the token")
		var token tokenRequest
		err = json.Unmarshal(output, &token)
		g.Expect(err).NotTo(HaveOccurred())

		out = token.Status.Token
	}
	Eventually(verifyTokenCreation).Should(Succeed())

	return out, err
}

// getMetricsOutput retrieves and returns the logs from the curl pod used to access the metrics endpoint.
func getMetricsOutput() (string, error) {
	By("getting the curl-metrics logs")
	cmd := exec.Command("kubectl", "logs", "curl-metrics", "-n", namespace)
	return utils.Run(cmd)
}

// tokenRequest is a simplified representation of the Kubernetes TokenRequest API response,
// containing only the token field that we need to extract.
type tokenRequest struct {
	Status struct {
		Token string `json:"token"`
	} `json:"status"`
}
