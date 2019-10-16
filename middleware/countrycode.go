package region

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
)

type GeocoderResponse struct {
	Country string `json:"country_code"`
}

func checkCountryCode(ip string) (string, error) {
	geocoderHost := os.Getenv("REVERSE_GEOCODER_URL")
	if geocoderHost == "" {
		geocoderHost = "127.0.0.1:4041"
	}

	geocoderURL := "http://" + geocoderHost + "/georeverse/" + ip

	resp, err := http.Post(geocoderURL, "application/json", nil)
	if err != nil {
		log.Println("Failed POSTing IP address", err)
		return "", err
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("Failed reading response body", err)
		return "", err
	}

	var response GeocoderResponse

	err = json.Unmarshal(body, &response)
	if err != nil {
		log.Println("Failed unmarshalling response body", err)
		return "", err
	}

	return response.Country, nil
}

func CountryCodeHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		destRegion := r.Header.Get("X-Country-Code")
		if destRegion != "" {
			next.ServeHTTP(w, r)
			return
		}

		var ipAddr string

		fwd := r.Header.Get("X-Forwarded-For")
		if fwd != "" {
			s := strings.Index(fwd, ", ")
			if s == -1 {
				s = len(fwd)
			}
			ipAddr = fwd[:s]
		} else {
			ipAddr = r.RemoteAddr
		}

		ip := net.ParseIP(ipAddr)
		if ip == nil {
			next.ServeHTTP(w, r)
			return
		}

		country, err := checkCountryCode(ip.String())
		if err != nil {
			next.ServeHTTP(w, r)
			return
		}

		r.Header.Set("X-Country-Code", strings.ToLower(country))
		next.ServeHTTP(w, r)
	})
}