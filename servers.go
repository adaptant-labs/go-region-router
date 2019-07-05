package main

import "net/url"

type ServerDefinition struct {
	// Constructed scheme://host:port URL for region redirection - note that this excludes both the path specification
	// and query parameters, as these will be lazily inserted by the middleware
	url		url.URL

	// ISO 3166-1 alpha-2 country code - e.g. de
	country	string
}
