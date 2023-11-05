package metrics

import (
	"fmt"
	"net/http"

	"github.com/VictoriaMetrics/metrics"
)

var (
	// ServersOnline - The total number of known matrix servers that are online and federateable
	ServersOnline = metrics.NewCounter("mrs_servers_online")
	// ServersIndexable - The total number of online matrix server that serve public rooms directory over federation
	ServersIndexable = metrics.NewCounter("mrs_servers_indexable")

	// RoomsParsed - The total number of rooms parsed from the indexable servers
	RoomsParsed = metrics.NewCounter("mrs_rooms_parsed")
	// RoomsIndexed - The total number of rooms indexed from the indexable servers
	RoomsIndexed = metrics.NewCounter("mrs_rooms_indexed")
)

// IncSearchQueries increments search queries counter with labels
func IncSearchQueries(api, server string) {
	metrics.GetOrCreateCounter(fmt.Sprintf("mrs_search_queries{api=%q,server=%q}", api, server))
}

// Handler for metrics
type Handler struct{}

func (h *Handler) ServeHTTP(w http.ResponseWriter, _ *http.Request) {
	metrics.WritePrometheus(w, false)
}
