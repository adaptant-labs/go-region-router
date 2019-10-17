package main

import (
	"github.com/adaptant-labs/go-region-router/api"
	"github.com/adaptant-labs/go-region-router/middleware"
	"github.com/gorilla/mux"
	"github.com/urfave/cli"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
)

type Router struct {
	Region		*region.RegionRouter
	mux			*mux.Router
	config		*api.ConsulConfiguration
	Refresh		chan bool
}

func (r Router) Start(host string, port int) error {
	addr := net.JoinHostPort(host, strconv.Itoa(port))
	log.Printf("Listening on %s ...", addr)
	return http.ListenAndServe(addr, region.CountryCodeHandler(r.Region.RegionHandler()(r.mux)))
}

func NewRouter(config *api.ConsulConfiguration) *Router {
	r := &Router{
		Region: region.NewRegionRouter(),
		mux: mux.NewRouter(),
		config: config,
		Refresh: make(chan bool, 1),
	}

	err := r.Region.UpdateRegionRoutesFromConsul(config)
	if err != nil {
		return nil
	}

	return r
}
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

	config := api.NewConsulConfiguration()

	app.Flags = []cli.Flag {
		cli.StringFlag{
			Name:			"consul-agent",
			Usage:			"Consul agent to connect to",
			Destination:	&config.Host,
		},

		cli.StringFlag{
			Name:			"consul-service",
			Usage:			"Name of Consul Service to look up",
			Value:			config.Service,
			Destination:	&config.Service,
		},

		cli.StringFlag{
			Name:			"consul-tag",
			Usage:			"Name of Consul tag to filter on",
			Value:			config.Tag,
			Destination:	&config.Tag,
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

	r := NewRouter(config)
	if r == nil {
		log.Fatal("Failed to initialize router")
	}

	app.Action = func(c *cli.Context) error {
		return r.Start(c.String("host"), c.Int("port"))
	}

	go func() {
		err := app.Run(os.Args)
		if err != nil {
			log.Fatal(err)
		}
	}()

	s := make(chan os.Signal, 1)
	signal.Notify(s, syscall.SIGUSR2)

	go func() {
		for {
			<-s
			r.Refresh <- true
			log.Println("Setting refresh needed")
		}
	}()

	for {
		<-r.Refresh
		_ = r.Region.UpdateRegionRoutesFromConsul(r.config)
		log.Println("Reloading server configuration...")
	}
}
