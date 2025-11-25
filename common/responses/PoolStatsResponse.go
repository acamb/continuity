package responses

import (
	"continuity/server/loadbalancer"
	"fmt"
)

type PoolStatsResponse struct {
	Stats map[string]loadbalancer.ServerStats
}

func (psr *PoolStatsResponse) String() string {
	resp := "Pool Stats:\n"
	for server, stats := range psr.Stats {
		resp += " Server " + server + ":\n"
		resp += "  TotalRequests: " + fmt.Sprintf("%d", stats.OkResponses+stats.NotOkResponses) + "\n"
		resp += "  SuccessfulRequests: " + fmt.Sprintf("%d", stats.OkResponses) + "\n"
		resp += "  FailedRequests: " + fmt.Sprintf("%d", stats.NotOkResponses) + "\n"
	}
	return resp
}
