package main

import (
	"github.com/adaptant-labs/go-region-router/middleware"
	"github.com/gorilla/mux"
	"github.com/urfave/cli"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
)

func main() {
	var port int
	var host string

	app := cli.NewApp()
	app.Name = "go-region-router"
	app.Usage = "A simple router for region endpoints"
	app.Version = "0.0.1"
	app.Author = "Adaptant Labs"
	app.Email = "labs@adaptant.io"
	app.Copyright = "(c) 2019 Adaptant Solutions AG"

	config := NewConsulConfiguration()

	app.Flags = []cli.Flag {
		cli.StringFlag{
			Name:			"consul-agent",
			Usage:			"Consul agent to connect to",
			Destination:	&config.host,
		},

		cli.StringFlag{
			Name:			"consul-service",
			Usage:			"Name of Consul Service to look up",
			Value:			config.service,
			Destination:	&config.service,
		},

		cli.StringFlag{
			Name:			"consul-tag",
			Usage:			"Name of Consul tag to filter on",
			Value:			config.tag,
			Destination:	&config.tag,
		},

		cli.StringFlag{
			Name:           "host",
			Usage:          "Host address to bind to",
			Value:          "",
			Destination:    &host,
		},

		cli.IntFlag{
			Name:			"port",
			Usage:			"Port to bind to",
			Value:			7000,
			Destination:	&port,
		},
	}

	app.Action = func(c *cli.Context) error {
		m := mux.NewRouter()
		r := region.NewRegionRouter()

		// Fetch the list of servers from Consul
		servers, err := ConsulRegionRoutes(config)
		if err != nil {
			return err
		}

		for _, srv := range servers {
			log.Printf("Setting up region routing for [%s] -> %s",
				strings.ToUpper(srv.country), srv.url.String())
			r.SetRegionServer(srv.country, srv.url.String())
		}

		addr := host + ":" + strconv.Itoa(port)
		log.Printf("Listening on %s ...", addr)

		return http.ListenAndServe(":7000", region.CountryCodeHandler(r.RegionHandler()(m)))
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
