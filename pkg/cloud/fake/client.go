/*
Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0
*/

// Package fake provides an in-memory cloud.Client used by unit and envtest
// tests. It supports failure injection so callers can exercise error paths.
package fake

import (
	"context"
	"fmt"
	"sync"

	"voigt.tngl.sh/cluster-api-provider-stackit/pkg/cloud"
)

// Client is an in-memory implementation of cloud.Client.
type Client struct {
	mu sync.Mutex

	servers       map[string]*serverEntry
	loadBalancers map[string]*lbEntry
	networks      map[string]*cloud.Network

	nextID int

	// failure injection: if non-nil, the next call returns this error and the
	// field is cleared.
	FailNextCreateServer error
	FailNextDeleteServer error
	FailNextGetServer    error
	FailNextFindServer   error
	FailNextEnsureLB     error
	FailNextDeleteLB     error
	FailNextEnsureTarget error
	FailNextDeleteTarget error
	FailNextGetNetwork   error

	// CreateServerCalls counts successful CreateServer calls (for idempotency
	// assertions).
	CreateServerCalls int
}

type serverEntry struct {
	server *cloud.Server
	tags   map[string]string
}

type lbEntry struct {
	lb      *cloud.LoadBalancer
	tags    map[string]string
	targets map[string]string
}

// New returns a Client preconfigured with the given networks.
func New(networks ...cloud.Network) *Client {
	c := &Client{
		servers:       make(map[string]*serverEntry),
		loadBalancers: make(map[string]*lbEntry),
		networks:      make(map[string]*cloud.Network),
	}
	for i := range networks {
		n := networks[i]
		c.networks[n.ID] = &n
	}
	return c
}

func consume(p *error) error {
	if *p == nil {
		return nil
	}
	err := *p
	*p = nil
	return err
}

func (c *Client) genID() string {
	c.nextID++
	return fmt.Sprintf("fake-%d", c.nextID)
}

func (c *Client) GetServer(_ context.Context, id string) (*cloud.Server, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if err := consume(&c.FailNextGetServer); err != nil {
		return nil, err
	}
	entry, ok := c.servers[id]
	if !ok {
		return nil, fmt.Errorf("server %q: %w", id, cloud.ErrNotFound)
	}
	return cloneServer(entry.server), nil
}

func (c *Client) FindServerByTags(_ context.Context, tags map[string]string) (*cloud.Server, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if err := consume(&c.FailNextFindServer); err != nil {
		return nil, err
	}
	if len(tags) == 0 {
		return nil, fmt.Errorf("empty tags: %w", cloud.ErrNotFound)
	}
	for _, entry := range c.servers {
		if mapContains(entry.tags, tags) {
			return cloneServer(entry.server), nil
		}
	}
	return nil, fmt.Errorf("no server matching tags: %w", cloud.ErrNotFound)
}

// CreateServer creates a server. To satisfy spec section 20 (idempotency),
// it returns the existing server if one with matching tags already exists.
func (c *Client) CreateServer(_ context.Context, input cloud.CreateServerInput) (*cloud.Server, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if err := consume(&c.FailNextCreateServer); err != nil {
		return nil, err
	}
	for _, entry := range c.servers {
		if len(input.Tags) > 0 && mapContains(entry.tags, input.Tags) {
			return cloneServer(entry.server), nil
		}
	}

	id := c.genID()
	server := &cloud.Server{
		ID:    id,
		Name:  input.Name,
		State: "ACTIVE",
		Addresses: []cloud.Address{
			{Type: "InternalIP", Address: "10.0.0.10"},
		},
	}
	c.servers[id] = &serverEntry{server: server, tags: copyTags(input.Tags)}
	c.CreateServerCalls++
	return cloneServer(server), nil
}

// DeleteServer removes a server. It returns nil if the server is already gone.
func (c *Client) DeleteServer(_ context.Context, id string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if err := consume(&c.FailNextDeleteServer); err != nil {
		return err
	}
	delete(c.servers, id)
	return nil
}

func (c *Client) GetNetwork(_ context.Context, id string) (*cloud.Network, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if err := consume(&c.FailNextGetNetwork); err != nil {
		return nil, err
	}
	n, ok := c.networks[id]
	if !ok {
		return nil, fmt.Errorf("network %q: %w", id, cloud.ErrNotFound)
	}
	out := *n
	return &out, nil
}

func (c *Client) EnsureAPIServerLoadBalancer(_ context.Context, input cloud.LoadBalancerInput) (*cloud.LoadBalancer, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if err := consume(&c.FailNextEnsureLB); err != nil {
		return nil, err
	}
	for _, entry := range c.loadBalancers {
		if len(input.Tags) > 0 && mapContains(entry.tags, input.Tags) {
			out := *entry.lb
			return &out, nil
		}
	}
	id := c.genID()
	lb := &cloud.LoadBalancer{
		ID:   id,
		Name: input.Name,
		IP:   "203.0.113.10",
		Port: input.Port,
	}
	targets := map[string]string{}
	for _, target := range input.Targets {
		targets[target.Name] = target.IP
	}
	c.loadBalancers[id] = &lbEntry{lb: lb, tags: copyTags(input.Tags), targets: targets}
	out := *lb
	return &out, nil
}

func (c *Client) EnsureAPIServerLoadBalancerTarget(_ context.Context, input cloud.LoadBalancerTargetInput) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if err := consume(&c.FailNextEnsureTarget); err != nil {
		return err
	}
	entry, ok := c.loadBalancers[input.LoadBalancerID]
	if !ok {
		return fmt.Errorf("load balancer %q: %w", input.LoadBalancerID, cloud.ErrNotFound)
	}
	entry.targets[input.Name] = input.IP
	return nil
}

func (c *Client) DeleteAPIServerLoadBalancerTarget(_ context.Context, input cloud.LoadBalancerTargetInput) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if err := consume(&c.FailNextDeleteTarget); err != nil {
		return err
	}
	entry, ok := c.loadBalancers[input.LoadBalancerID]
	if !ok {
		return fmt.Errorf("load balancer %q: %w", input.LoadBalancerID, cloud.ErrNotFound)
	}
	delete(entry.targets, input.Name)
	return nil
}

func (c *Client) DeleteAPIServerLoadBalancer(_ context.Context, id string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if err := consume(&c.FailNextDeleteLB); err != nil {
		return err
	}
	delete(c.loadBalancers, id)
	return nil
}

// ServerCount returns the number of currently tracked servers (test helper).
func (c *Client) ServerCount() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.servers)
}

// LoadBalancerCount returns the number of currently tracked load balancers
// (test helper).
func (c *Client) LoadBalancerCount() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.loadBalancers)
}

// LoadBalancerTargetCount returns the number of targets in one load balancer
// target pool (test helper).
func (c *Client) LoadBalancerTargetCount(id string) int {
	c.mu.Lock()
	defer c.mu.Unlock()
	entry, ok := c.loadBalancers[id]
	if !ok {
		return 0
	}
	return len(entry.targets)
}

func mapContains(haystack, needle map[string]string) bool {
	for k, v := range needle {
		if haystack[k] != v {
			return false
		}
	}
	return true
}

func copyTags(in map[string]string) map[string]string {
	if in == nil {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func cloneServer(s *cloud.Server) *cloud.Server {
	out := *s
	if s.Addresses != nil {
		out.Addresses = make([]cloud.Address, len(s.Addresses))
		copy(out.Addresses, s.Addresses)
	}
	return &out
}
