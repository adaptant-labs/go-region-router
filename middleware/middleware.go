package region

import (
	"net/http"
	"net/url"
	"strings"
	"sync"
)

type RegionRouter struct {
	h		http.Handler

	// HTTP Status code for redirection (defaults to http.StatusFound - 302)
	StatusCode	int

	mapLock	sync.RWMutex

	// A string map keyed with the ISO 3166-1 alpha-2 2-character country code and a target host. Scheme is defined by
	// the destination, while the path is inherited from the originating request.
	m		map[string]string
}

// NewRegionRouter returns a new region router instance
func NewRegionRouter() *RegionRouter {
	return &RegionRouter{m: make(map[string]string), StatusCode: http.StatusFound}
}

// Handler provides a Region Routing middleware for enabling regional server redirection.
// Example:
//
// import (
//		"github.com/adaptant-labs/go-region-router/middleware"
//		"github.com/gorilla/mux"
//		"net/http"
// )
//
// func main() {
//		m := mux.NewRouter()
//		r := region.NewRegionRouter()
//		r.SetRegionServer("de", "https://de.api.xxx.com")
//		r.SetRegionServer("at", "https://at.api.xxx.com")
//
//		// Apply the region routing middleware with default settings
//		http.ListenAndServer(":8080", r.RegionHandler()(m))
// }
func (reg *RegionRouter) RegionHandler() func(h http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		reg.h = h
		return reg
	}
}

// GetRegionServer returns a registered server handling requests for a specific region, or an empty string if none
// is defined.
func (reg *RegionRouter) GetRegionServer(countryCode string) string {
	reg.mapLock.RLock()
	s := reg.m[countryCode]
	reg.mapLock.RUnlock()

	return s
}

// SetRegionServer defines a registered server for handling requests in a specific region. It returns a boolean
// value indicating whether the registration of the server for the designated country code succeeded or not - this
// may fail in case where a server has already been defined.
func (reg *RegionRouter) SetRegionServer(countryCode string, server string) bool {
	set := true

	reg.mapLock.Lock()

	if reg.m[countryCode] == "" {
		reg.m[countryCode] = server
	} else {
		set = false
	}

	reg.mapLock.Unlock()

	return set
}

// DeleteRegionServer unregisters the server handling requests for a specific region.
func (reg *RegionRouter) DeleteRegionServer(countryCode string) {
	reg.mapLock.Lock()
	delete(reg.m, countryCode)
	reg.mapLock.Unlock()
}

func (reg RegionRouter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	destRegion := r.Header.Get("X-Country-Code")
	if destRegion == "" {
		// If no region is specified, pass through to the next middleware
		reg.h.ServeHTTP(w, r)
		return
	}

	target := reg.GetRegionServer(strings.ToLower(destRegion))
	if target == "" {
		// If there is no specific server to handle this region, pass through to the next middleware
		reg.h.ServeHTTP(w, r)
		return
	}

	dest, err := url.Parse(target)
	if err != nil {
		// If the region server specified can not be decoded, pass through to the next middleware
		reg.h.ServeHTTP(w, r)
		return
	}

	if dest.Host == "" || dest.Scheme == "" {
		reg.h.ServeHTTP(w, r)
		return
	}

	if !strings.EqualFold(r.Host, dest.Host) {
		// Re-build the destination URL
		destUrl := dest.Scheme + "://" + dest.Host + r.URL.Path
		if r.URL.RawQuery != "" {
			destUrl += "?" + r.URL.RawQuery
		}

		// Redirect
		http.Redirect(w, r, destUrl, reg.StatusCode)
		return
	}

	reg.h.ServeHTTP(w, r)
}