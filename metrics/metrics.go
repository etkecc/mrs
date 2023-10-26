package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// ServersOnline - The total number of known matrix servers that are online and federateable
	ServersOnline = promauto.NewCounter(prometheus.CounterOpts{
		Name: "mrs_servers_online",
		Help: "The total number of known matrix servers that are online and federateable",
	})
	// ServersIndexable - The total number of online matrix server that serve public rooms directory over federation
	ServersIndexable = promauto.NewCounter(prometheus.CounterOpts{
		Name: "mrs_servers_indexable",
		Help: "The total number of online matrix server that serve public rooms directory over federation",
	})

	// RoomsParsed - The total number of rooms parsed from the indexable servers
	RoomsParsed = promauto.NewCounter(prometheus.CounterOpts{
		Name: "mrs_rooms_parsed",
		Help: "The total number of rooms parsed from the indexable servers",
	})
	// RoomsIndexed - The total number of rooms indexed from the indexable servers
	RoomsIndexed = promauto.NewCounter(prometheus.CounterOpts{
		Name: "mrs_rooms_indexed",
		Help: "The total number of rooms indexed from the indexable servers",
	})

	// SearchQueries - The total number of search queries done through MRS, by api (rest or matrix) and server
	SearchQueries = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "mrs_search_queries",
		Help: "The total number of search queries done through MRS, by api (rest or matrix) and server",
	}, []string{"api", "server"})
)
