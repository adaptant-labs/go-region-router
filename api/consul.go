package api

import (
	"fmt"
	consul "github.com/hashicorp/consul/api"
	"strings"
)

type ConsulConfiguration struct {
	Host    string
	Service string
	Tag     string
}

func NewConsulConfiguration() *ConsulConfiguration {
	return &ConsulConfiguration{Service: "api", Tag: "v1"}
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

	srv.URL.Scheme = scheme
	srv.URL.Host = fmt.Sprintf("%s:%d", address, entry.ServicePort)
	srv.DefaultServer = false

	// Extract a region-<code> identifier from the tags
	for _, tag := range entry.ServiceTags {
		if tag == "default" {
			srv.DefaultServer = true
			continue
		}

		if strings.HasPrefix(tag, "region-") {
			srv.CountryCode = strings.ToLower(strings.TrimPrefix(tag, "region-"))
			break
		}
	}

	return srv
}

// ConsulRegionRoutes returns a list of servers and regions for a specific service and tag
func ConsulRegionRoutes(config *ConsulConfiguration) ([]*ServerDefinition, error) {
	var servers []*ServerDefinition

	consulConfig := consul.DefaultConfig()
	if config.Host != "" {
		consulConfig.Address = config.Host
	}
	client, err := consul.NewClient(consulConfig)
	if err != nil {
		return nil, err
	}

	results, _, err := client.Catalog().Service(config.Service, config.Tag, nil)
	if err != nil {
		return nil, err
	}

	for _, res := range results {
		srv := ServerDefinitionFromServiceEntry(res)
		servers = append(servers, srv)
	}

	return servers, nil
}
