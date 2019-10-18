package main

import (
	"github.com/adaptant-labs/go-region-router/api"
	"github.com/adaptant-labs/go-region-router/middleware"
	"github.com/gorilla/mux"
	consul "github.com/hashicorp/consul/api"
	"github.com/hashicorp/consul/api/watch"
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
	Region			*region.RegionRouter
	mux				*mux.Router
	config			*api.ConsulConfiguration
	signalRefresh	chan bool
	serviceUpdates	chan []*consul.ServiceEntry
	serviceParams	map[string]interface{}
	plan			*watch.Plan
}

func (r Router) Start(host string, port int) error {
	addr := net.JoinHostPort(host, strconv.Itoa(port))
	log.Printf("Listening on %s ...", addr)
	return http.ListenAndServe(addr, region.CountryCodeHandler(r.Region.RegionHandler()(r.mux)))
}

func NewRouter(config *api.ConsulConfiguration) *Router {
	var err error

	r := &Router{
		Region: region.NewRegionRouter(),
		mux: mux.NewRouter(),
		config: config,
		signalRefresh: make(chan bool, 1),
		serviceUpdates: make(chan []*consul.ServiceEntry, 1),
	}

	r.serviceParams = make(map[string]interface{})
	r.serviceParams["type"] = "service"
	r.serviceParams["service"] = config.Service
	r.serviceParams["tag"] = config.Tag

	r.plan, err = watch.Parse(r.serviceParams)
	if err != nil {
		log.Fatal(err)
	}

	r.plan.Handler = func(index uint64, result interface{}) {
		if entries, ok := result.([]*consul.ServiceEntry); ok {
			r.serviceUpdates <- entries
		}
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
		go func() {
			if err := r.plan.Run(c.String("consul-agent")); err != nil {
				log.Fatal(err)
			}
		}()

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

	// Handle signal-based refresh
	go func() {
		for {
			<-s
			r.signalRefresh <- true
			log.Println("Triggering refresh from signal handler")
		}
	}()

	for {
		select {
		case <-r.signalRefresh:
			log.Println("Received reload signal")
			_ = r.Region.UpdateRegionRoutesFromConsul(r.config)
			log.Println("Reloaded server configuration by signal")
		case updates := <-r.serviceUpdates:
			log.Println("Received service updates from Consul")
			servers := api.ServerDefinitionsFromServiceEntries(updates)
			_ = r.Region.UpdateRegionRoutesFromServerDefinitions(servers)
			log.Println("Reloaded server configuration by Consul Watch")
		}
	}
}
