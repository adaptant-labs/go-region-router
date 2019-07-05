package main

import (
	"fmt"
	consul "github.com/hashicorp/consul/api"
	"strings"
)

type ConsulConfiguration struct {
	host			string
	service			string
	tag				string
}

func NewConsulConfiguration() *ConsulConfiguration {
	return &ConsulConfiguration{service: "api", tag: "v1"}
}

// ServerDefinitionFromServiceEntry creates a routing definition for a region from a Consul Catalog Service entry
func ServerDefinitionFromServiceEntry(entry *consul.CatalogService) *ServerDefinition {
	var address string

	srv := &ServerDefinition{}

	// If the Service Address is provided, this will be the target of the route.
	if entry.ServiceAddress != "" {
		address = entry.ServiceAddress
	} else {
		// If undefined, however, fall back on the node address.
		address = entry.Address
	}

	// Consul, somewhat inexplicably, does not provide protocol hinting out of the box for a service definition. We
	// therefore have no direct means of decoding the target scheme without manually hinting through service-specific
	// metadata.
	scheme := entry.ServiceMeta["protocol"]
	if scheme == "" {
		// If no protocol has been defined, assume HTTPS - override for plain HTTP, WebSockets, etc.
		scheme = "https"
	}

	srv.url.Scheme = scheme
	srv.url.Host = fmt.Sprintf("%s:%d", address, entry.ServicePort)

	// Extract a region-<code> identifier from the tags
	for _, tag := range entry.ServiceTags {
		if strings.HasPrefix(tag, "region-") {
			srv.country = strings.ToLower(strings.TrimPrefix(tag, "region-"))
			break
		}
	}

	return srv
}

// ConsulRegionRoutes returns a list of servers and regions for a specific service and tag
func ConsulRegionRoutes(config *ConsulConfiguration) ([]*ServerDefinition, error) {
	var servers []*ServerDefinition

	consulConfig := consul.DefaultConfig()
	if config.host != "" {
		consulConfig.Address = config.host
	}
	client, err := consul.NewClient(consulConfig)
	if err != nil {
		return nil, err
	}

	results, _, err := client.Catalog().Service(config.service, config.tag, nil)
	if err != nil {
		return nil, err
	}

	for _, res := range results {
		srv := ServerDefinitionFromServiceEntry(res)
		servers = append(servers, srv)
	}

	return servers, nil
}