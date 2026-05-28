/*
Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0
*/

package cloud

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/stackitcloud/stackit-sdk-go/core/config"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	iaas "github.com/stackitcloud/stackit-sdk-go/services/iaas/v2api"
	lb "github.com/stackitcloud/stackit-sdk-go/services/loadbalancer/v2api"
)

const (
	stackitNoAuthEnv        = "STACKIT_NO_AUTH"
	stackitTokenEndpointEnv = "STACKIT_TOKEN_BASEURL"
	stackitIAASEndpointEnv  = "STACKIT_IAAS_ENDPOINT"
	stackitLBEndpointEnv    = "STACKIT_LOADBALANCER_ENDPOINT"

	apiserverTargetPoolName = "apiserver"
	apiserverListenerName   = "apiserver"
	bootstrapTargetName     = "capi-bootstrap-placeholder"
	bootstrapTargetIP       = "10.0.0.1"
)

// SDKClient is the STACKIT SDK-backed implementation of Client.
type SDKClient struct {
	projectID string
	region    string

	iaasClient *iaas.APIClient
	lbClient   *lb.APIClient
}

// NewClient returns the real STACKIT cloud client.
//
// The credential secret format mirrors machine-controller-manager-provider-stackit:
// "project-id" plus "serviceaccount.json". Authentication uses the STACKIT
// service-account key flow.
func NewClient(_ context.Context, creds Credentials) (Client, error) {
	if creds.ProjectID == "" {
		return nil, fmt.Errorf("%w: project ID is empty", ErrInvalidInput)
	}
	if creds.Region == "" {
		return nil, fmt.Errorf("%w: region is empty", ErrInvalidInput)
	}
	if len(creds.ServiceAccountJSON) == 0 && os.Getenv(stackitNoAuthEnv) != "true" {
		return nil, fmt.Errorf("%w: service account JSON is empty", ErrInvalidInput)
	}

	iaasClient, err := iaas.NewAPIClient(stackitOptions(creds, os.Getenv(stackitIAASEndpointEnv))...)
	if err != nil {
		return nil, fmt.Errorf("%w: create IAAS SDK client: %v", ErrInvalidInput, err)
	}
	lbClient, err := lb.NewAPIClient(stackitOptions(creds, os.Getenv(stackitLBEndpointEnv))...)
	if err != nil {
		return nil, fmt.Errorf("%w: create load balancer SDK client: %v", ErrInvalidInput, err)
	}

	return &SDKClient{
		projectID:  creds.ProjectID,
		region:     creds.Region,
		iaasClient: iaasClient,
		lbClient:   lbClient,
	}, nil
}

func stackitOptions(creds Credentials, endpoint string) []config.ConfigurationOption {
	opts := []config.ConfigurationOption{
		config.WithUserAgent("cluster-api-provider-stackit"),
	}
	if os.Getenv(stackitNoAuthEnv) == "true" {
		opts = append(opts, config.WithoutAuthentication())
	} else {
		opts = append(opts, config.WithServiceAccountKey(string(creds.ServiceAccountJSON)))
	}
	if endpoint != "" {
		opts = append(opts, config.WithEndpoint(endpoint))
	}
	if tokenEndpoint := os.Getenv(stackitTokenEndpointEnv); tokenEndpoint != "" {
		opts = append(opts, config.WithTokenEndpoint(tokenEndpoint))
	}
	return opts
}

func (c *SDKClient) GetServer(ctx context.Context, id string) (*Server, error) {
	server, err := c.iaasClient.DefaultAPI.GetServer(ctx, c.projectID, c.region, id).Execute()
	if err != nil {
		return nil, classifySDKError("get server", err)
	}
	return c.serverFromSDK(ctx, server), nil
}

func (c *SDKClient) FindServerByTags(ctx context.Context, tags map[string]string) (*Server, error) {
	servers, err := c.ListServersByTags(ctx, tags)
	if err != nil {
		return nil, err
	}
	if len(servers) == 0 {
		return nil, fmt.Errorf("%w: no server matching tags", ErrNotFound)
	}
	if len(servers) > 1 {
		return nil, fmt.Errorf("%w: multiple servers matching tags", ErrConflict)
	}
	return servers[0], nil
}

func (c *SDKClient) ListServersByTags(ctx context.Context, tags map[string]string) ([]*Server, error) {
	if len(tags) == 0 {
		return nil, fmt.Errorf("%w: empty tag selector", ErrInvalidInput)
	}
	resp, err := c.iaasClient.DefaultAPI.ListServers(ctx, c.projectID, c.region).
		LabelSelector(labelSelector(tags)).
		Details(true).
		Execute()
	if err != nil {
		return nil, classifySDKError("list servers", err)
	}
	matched := []*Server{}
	for _, server := range resp.GetItems() {
		if labelsContainTags(server.GetLabels(), tags) {
			server := server
			matched = append(matched, c.serverFromSDK(ctx, &server))
		}
	}
	return matched, nil
}

func (c *SDKClient) CreateServer(ctx context.Context, input CreateServerInput) (*Server, error) {
	if existing, err := c.FindServerByTags(ctx, input.Tags); err == nil {
		return existing, nil
	} else if !IsNotFound(err) {
		return nil, err
	}

	payload := &iaas.CreateServerPayload{}
	payload.SetName(input.Name)
	payload.SetMachineType(input.MachineType)
	useBootVolume := input.RootVolume.SizeGiB > 0 || input.RootVolume.PerformanceClass != ""
	if input.ImageID != "" && !useBootVolume {
		payload.SetImageId(input.ImageID)
	}
	if len(input.Tags) > 0 {
		payload.SetLabels(tagsToSDKLabels(input.Tags))
	}
	networking := iaas.NewCreateServerNetworking()
	if input.NetworkID != "" {
		networking.SetNetworkId(input.NetworkID)
	}
	payload.SetNetworking(iaas.CreateServerNetworkingAsCreateServerPayloadAllOfNetworking(networking))
	if len(input.SecurityGroups) > 0 {
		payload.SetSecurityGroups(input.SecurityGroups)
	}
	if len(input.UserData) > 0 {
		payload.SetUserData(base64.StdEncoding.EncodeToString(input.UserData))
		payload.SetConfigDrive(true)
	}
	if useBootVolume {
		bootVolume := iaas.NewBootVolume()
		if input.ImageID != "" {
			bootVolume.SetSource(*iaas.NewBootVolumeSource(input.ImageID, "image"))
		}
		if input.RootVolume.SizeGiB > 0 {
			bootVolume.SetSize(int64(input.RootVolume.SizeGiB))
		}
		if input.RootVolume.PerformanceClass != "" {
			bootVolume.SetPerformanceClass(input.RootVolume.PerformanceClass)
		}
		bootVolume.SetDeleteOnTermination(input.RootVolume.DeleteOnTermination)
		payload.SetBootVolume(*bootVolume)
	}
	if input.SSHKeyName != "" {
		payload.SetKeypairName(input.SSHKeyName)
	}
	if input.AvailabilityZone != "" {
		payload.SetAvailabilityZone(input.AvailabilityZone)
	}

	server, err := c.iaasClient.DefaultAPI.CreateServer(ctx, c.projectID, c.region).
		CreateServerPayload(*payload).
		Execute()
	if err != nil {
		return nil, classifySDKError("create server", err)
	}
	return c.serverFromSDK(ctx, server), nil
}

func (c *SDKClient) DeleteServer(ctx context.Context, id string) error {
	if err := c.iaasClient.DefaultAPI.DeleteServer(ctx, c.projectID, c.region, id).Execute(); err != nil {
		return classifySDKError("delete server", err)
	}
	return nil
}

func (c *SDKClient) GetNetwork(ctx context.Context, id string) (*Network, error) {
	network, err := c.iaasClient.DefaultAPI.GetNetwork(ctx, c.projectID, c.region, id).Execute()
	if err != nil {
		return nil, classifySDKError("get network", err)
	}
	ipv4 := network.GetIpv4()
	return &Network{
		ID:           network.GetId(),
		Name:         network.GetName(),
		IPv4Prefixes: ipv4.GetPrefixes(),
	}, nil
}

func (c *SDKClient) EnsureAPIServerLoadBalancer(ctx context.Context, input LoadBalancerInput) (*LoadBalancer, error) {
	if existing, err := c.findLoadBalancerByTags(ctx, input.Tags); err == nil {
		return existing, nil
	} else if !IsNotFound(err) && !isLoadBalancerServiceNotEnabled(err) {
		return nil, err
	}
	payload := lb.NewCreateLoadBalancerPayload()
	payload.SetName(input.Name)
	payload.SetRegion(input.Region)
	if len(input.Tags) > 0 {
		payload.SetLabels(loadBalancerLabels(input.Tags))
	}
	options := lb.NewLoadBalancerOptions()
	options.SetEphemeralAddress(true)
	payload.SetOptions(*options)

	network := lb.NewNetwork()
	network.SetNetworkId(input.NetworkID)
	network.SetRole("ROLE_LISTENERS_AND_TARGETS")
	payload.SetNetworks([]lb.Network{*network})

	targetPool := lb.NewTargetPool()
	targetPool.SetName(apiserverTargetPoolName)
	targetPool.SetTargetPort(input.Port)
	targets := make([]lb.Target, 0, len(input.Targets))
	if len(input.Targets) == 0 {
		target := lb.NewTarget()
		target.SetDisplayName(bootstrapTargetName)
		target.SetIp(bootstrapTargetIP)
		targets = append(targets, *target)
	} else {
		for _, targetInput := range input.Targets {
			target := lb.NewTarget()
			target.SetDisplayName(targetInput.Name)
			target.SetIp(targetInput.IP)
			targets = append(targets, *target)
		}
	}
	targetPool.SetTargets(targets)
	payload.SetTargetPools([]lb.TargetPool{*targetPool})

	listener := lb.NewListener()
	listener.SetDisplayName(apiserverListenerName)
	listener.SetPort(input.Port)
	listener.SetProtocol("PROTOCOL_TCP")
	listener.SetTargetPool(apiserverTargetPoolName)
	payload.SetListeners([]lb.Listener{*listener})

	lbResp, err := c.lbClient.DefaultAPI.CreateLoadBalancer(ctx, c.projectID, c.region).
		CreateLoadBalancerPayload(*payload).
		Execute()
	if err != nil {
		return nil, classifySDKError("create load balancer", err)
	}
	return loadBalancerFromSDK(lbResp), nil
}

func (c *SDKClient) DeleteAPIServerLoadBalancer(ctx context.Context, id string) error {
	if _, err := c.lbClient.DefaultAPI.DeleteLoadBalancer(ctx, c.projectID, c.region, id).Execute(); err != nil {
		return classifySDKError("delete load balancer", err)
	}
	return nil
}

func (c *SDKClient) ListAPIServerLoadBalancersByTags(ctx context.Context, tags map[string]string) ([]*LoadBalancer, error) {
	if len(tags) == 0 {
		return nil, fmt.Errorf("%w: empty tag selector", ErrInvalidInput)
	}
	resp, err := c.lbClient.DefaultAPI.ListLoadBalancers(ctx, c.projectID, c.region).Execute()
	if err != nil {
		err := classifySDKError("list load balancers", err)
		if isLoadBalancerServiceNotEnabled(err) {
			return nil, nil
		}
		return nil, err
	}
	matched := []*LoadBalancer{}
	labels := loadBalancerLabels(tags)
	for _, candidate := range resp.GetLoadBalancers() {
		if mapContains(candidate.GetLabels(), labels) {
			candidate := candidate
			matched = append(matched, loadBalancerFromSDK(&candidate))
		}
	}
	return matched, nil
}

func (c *SDKClient) EnsureAPIServerLoadBalancerTarget(ctx context.Context, input LoadBalancerTargetInput) error {
	if input.LoadBalancerID == "" || input.Name == "" || input.IP == "" {
		return fmt.Errorf("%w: load balancer ID, target name, and target IP are required", ErrInvalidInput)
	}
	loadBalancer, err := c.lbClient.DefaultAPI.GetLoadBalancer(ctx, c.projectID, c.region, input.LoadBalancerID).Execute()
	if err != nil {
		return classifySDKError("get load balancer", err)
	}
	targetPool := apiServerTargetPool(loadBalancer)
	if targetPool == nil {
		return fmt.Errorf("%w: load balancer %q has no %q target pool", ErrNotFound, input.LoadBalancerID, apiserverTargetPoolName)
	}

	targets := withoutBootstrapTarget(targetPool.GetTargets())
	for i := range targets {
		if targets[i].GetDisplayName() == input.Name || targets[i].GetIp() == input.IP {
			targets[i].SetDisplayName(input.Name)
			targets[i].SetIp(input.IP)
			return c.updateAPIServerTargetPool(ctx, input.LoadBalancerID, targetPool, targets, input.Port)
		}
	}

	target := lb.NewTarget()
	target.SetDisplayName(input.Name)
	target.SetIp(input.IP)
	targets = append(targets, *target)
	return c.updateAPIServerTargetPool(ctx, input.LoadBalancerID, targetPool, targets, input.Port)
}

func withoutBootstrapTarget(targets []lb.Target) []lb.Target {
	out := targets[:0]
	for _, target := range targets {
		if target.GetDisplayName() == bootstrapTargetName {
			continue
		}
		out = append(out, target)
	}
	return out
}

func (c *SDKClient) DeleteAPIServerLoadBalancerTarget(ctx context.Context, input LoadBalancerTargetInput) error {
	if input.LoadBalancerID == "" || input.Name == "" {
		return fmt.Errorf("%w: load balancer ID and target name are required", ErrInvalidInput)
	}
	loadBalancer, err := c.lbClient.DefaultAPI.GetLoadBalancer(ctx, c.projectID, c.region, input.LoadBalancerID).Execute()
	if err != nil {
		return classifySDKError("get load balancer", err)
	}
	targetPool := apiServerTargetPool(loadBalancer)
	if targetPool == nil {
		return nil
	}

	targets := targetPool.GetTargets()
	out := make([]lb.Target, 0, len(targets))
	for _, target := range targets {
		if target.GetDisplayName() == input.Name {
			continue
		}
		out = append(out, target)
	}
	if len(out) == len(targets) {
		return nil
	}
	if len(out) == 0 {
		// STACKIT NLB target pools must contain at least one target. Leave the
		// last target in place; deleting the load balancer removes it.
		return nil
	}
	return c.updateAPIServerTargetPool(ctx, input.LoadBalancerID, targetPool, out, input.Port)
}

func (c *SDKClient) findLoadBalancerByTags(ctx context.Context, tags map[string]string) (*LoadBalancer, error) {
	matched, err := c.ListAPIServerLoadBalancersByTags(ctx, tags)
	if err != nil {
		return nil, err
	}
	if len(matched) == 0 {
		return nil, fmt.Errorf("%w: no load balancer matching tags", ErrNotFound)
	}
	if len(matched) > 1 {
		return nil, fmt.Errorf("%w: multiple load balancers matching tags", ErrConflict)
	}
	return matched[0], nil
}

func (c *SDKClient) updateAPIServerTargetPool(
	ctx context.Context,
	loadBalancerID string,
	current *lb.TargetPool,
	targets []lb.Target,
	port int32,
) error {
	payload := lb.NewUpdateTargetPoolPayload()
	payload.SetName(apiserverTargetPoolName)
	if port > 0 {
		payload.SetTargetPort(port)
	} else if current.GetTargetPort() > 0 {
		payload.SetTargetPort(current.GetTargetPort())
	}
	payload.SetTargets(targets)
	if _, err := c.lbClient.DefaultAPI.UpdateTargetPool(ctx, c.projectID, c.region, loadBalancerID, apiserverTargetPoolName).
		UpdateTargetPoolPayload(*payload).
		Execute(); err != nil {
		return classifySDKError("update load balancer target pool", err)
	}
	return nil
}

func apiServerTargetPool(loadBalancer *lb.LoadBalancer) *lb.TargetPool {
	if loadBalancer == nil {
		return nil
	}
	for _, targetPool := range loadBalancer.GetTargetPools() {
		if targetPool.GetName() == apiserverTargetPoolName {
			return &targetPool
		}
	}
	return nil
}

func (c *SDKClient) serverFromSDK(ctx context.Context, server *iaas.Server) *Server {
	if server == nil {
		return nil
	}
	out := &Server{
		ID:    server.GetId(),
		Name:  server.GetName(),
		State: server.GetStatus(),
	}
	nics, err := c.iaasClient.DefaultAPI.ListServerNICs(ctx, c.projectID, c.region, out.ID).Execute()
	if err == nil {
		for _, nic := range nics.GetItems() {
			if ipv4 := nic.GetIpv4(); ipv4 != "" {
				out.Addresses = append(out.Addresses, Address{Type: "InternalIP", Address: ipv4})
			}
			if ipv6 := nic.GetIpv6(); ipv6 != "" {
				out.Addresses = append(out.Addresses, Address{Type: "InternalIP", Address: ipv6})
			}
		}
	}
	return out
}

func loadBalancerFromSDK(in *lb.LoadBalancer) *LoadBalancer {
	if in == nil {
		return nil
	}
	return &LoadBalancer{
		ID:   in.GetName(),
		Name: in.GetName(),
		IP:   firstNonEmpty(in.GetExternalAddress(), in.GetPrivateAddress()),
		Port: listenerPort(in.GetListeners()),
	}
}

func listenerPort(listeners []lb.Listener) int32 {
	for _, listener := range listeners {
		if port := listener.GetPort(); port > 0 {
			return port
		}
	}
	return 0
}

func labelSelector(tags map[string]string) string {
	parts := make([]string, 0, len(tags))
	for k, v := range tags {
		parts = append(parts, k+"="+v)
	}
	return strings.Join(parts, ",")
}

func labelsContainTags(labels map[string]interface{}, tags map[string]string) bool {
	for k, want := range tags {
		got, ok := labels[k]
		if !ok || fmt.Sprint(got) != want {
			return false
		}
	}
	return true
}

func tagsToSDKLabels(tags map[string]string) map[string]interface{} {
	out := make(map[string]interface{}, len(tags))
	for k, v := range tags {
		out[k] = v
	}
	return out
}

func mapContains(haystack, needle map[string]string) bool {
	for k, v := range needle {
		if haystack[k] != v {
			return false
		}
	}
	return true
}

func loadBalancerLabels(tags map[string]string) map[string]string {
	out := make(map[string]string, len(tags))
	for k, v := range tags {
		out[strings.ReplaceAll(k, "/", ".")] = v
	}
	return out
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func classifySDKError(op string, err error) error {
	var oapiErr *oapierror.GenericOpenAPIError
	if errors.As(err, &oapiErr) {
		switch oapiErr.StatusCode {
		case http.StatusNotFound:
			return fmt.Errorf("%w: %s: %v", ErrNotFound, op, err)
		case http.StatusUnauthorized, http.StatusForbidden:
			return fmt.Errorf("%w: %s: %v", ErrUnauthorized, op, err)
		case http.StatusConflict:
			return fmt.Errorf("%w: %s: %v", ErrConflict, op, err)
		case http.StatusBadRequest, http.StatusUnprocessableEntity:
			return fmt.Errorf("%w: %s: %v", ErrInvalidInput, op, err)
		case http.StatusTooManyRequests:
			return fmt.Errorf("%w: %s: %v", ErrTransient, op, err)
		default:
			if oapiErr.StatusCode >= 500 {
				return fmt.Errorf("%w: %s: %v", ErrTransient, op, err)
			}
		}
	}
	return fmt.Errorf("%w: %s: %v", ErrTransient, op, err)
}

func isLoadBalancerServiceNotEnabled(err error) bool {
	return IsUnauthorized(err) && strings.Contains(err.Error(), "Service not enabled")
}
