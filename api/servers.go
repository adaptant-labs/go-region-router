package api

import "net/url"

type ServerDefinition struct {
	// Constructed scheme://Host:port URL for region redirection - note that this excludes both the path specification
	// and query parameters, as these will be lazily inserted by the middleware
	URL		url.URL

	// ISO 3166-1 alpha-2 country code - e.g. de
	CountryCode		string

	// Is the server designated as the default handler for unmatched country codes?
	DefaultServer	bool
}
