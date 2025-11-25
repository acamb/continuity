package responses

import (
	"continuity/server/loadbalancer"
	"fmt"
	"time"
)

type PoolResponse struct {
	Hostname                string
	HealthCheckInterval     uint64
	HealthCheckInitialDelay uint64
	HealthCheckTimeout      uint64
	HealthCheck_numOk       uint64
	HealthCheck_numFail     uint64
	ConditionalServers      []*ServerHostResponse
	UnconditionalServers    []*ServerHostResponse
	StickySessions          bool
	StickyMethod            string
	StickySessionTimeout    uint64
	stickyCookieName        string
	requestCounter          uint64
}

func NewPoolResponse(pool *loadbalancer.Pool) *PoolResponse {
	resp := &PoolResponse{
		Hostname:                pool.Hostname,
		HealthCheckInterval:     uint64(time.Duration(pool.HealthCheckInterval.Load()).Seconds()),
		HealthCheckInitialDelay: uint64(time.Duration(pool.HealthCheckInitialDelay.Load()).Seconds()),
		HealthCheckTimeout:      uint64(time.Duration(pool.HealthCheckTimeout.Load()).Seconds()),
		HealthCheck_numOk:       uint64(time.Duration(pool.HealthCheck_numOk.Load()).Seconds()),
		HealthCheck_numFail:     uint64(time.Duration(pool.HealthCheck_numFail.Load()).Seconds()),
		StickySessions:          pool.StickySessions,
		StickyMethod:            pool.StickyMethod.String(),
		StickySessionTimeout:    uint64(pool.StickySessionTimeout.Seconds()),
		stickyCookieName:        pool.GetStickyCookieName(),
		requestCounter:          pool.RequestCounter.Load(),
	}
	resp.ConditionalServers = []*ServerHostResponse{}
	resp.UnconditionalServers = []*ServerHostResponse{}
	for _, server := range pool.ConditionalServers {
		resp.ConditionalServers = append(resp.ConditionalServers, NewServerHostResponse(server))
	}
	for _, server := range pool.UnconditionalServers {
		resp.UnconditionalServers = append(resp.UnconditionalServers, NewServerHostResponse(server))
	}
	return resp
}

func (pr *PoolResponse) String() string {
	resp := fmt.Sprintf("Pool %s:\n"+
		"\tHealthCheckInterval=%ds,\n"+
		"\tHealthCheckInitialDelay=%ds,\n"+
		"\tHealthCheckTimeout=%ds,\n"+
		"\tHealthCheck_numOk=%d,\n"+
		"\tHealthCheck_numFail=%d,\n"+
		"\tStickySessions=%t", pr.Hostname,
		pr.HealthCheckInterval,
		pr.HealthCheckInitialDelay,
		pr.HealthCheckTimeout,
		pr.HealthCheck_numOk,
		pr.HealthCheck_numFail,
		pr.StickySessions)
	if pr.StickySessions {
		resp += fmt.Sprintf(",\n\tStickyMethod=%s,\n\tStickySessionTimeout=%ds,\n\tStickyCookieName=%s",
			pr.StickyMethod,
			pr.StickySessionTimeout,
			pr.stickyCookieName)
	}
	if len(pr.ConditionalServers) > 0 {
		resp += "\n\tConditional Servers:\n"
		for _, server := range pr.ConditionalServers {
			resp += "\t\t" + server.String() + "\n"
		}
	}
	if len(pr.UnconditionalServers) > 0 {
		resp += "\n\tUnconditional Servers:\n"
		for _, server := range pr.UnconditionalServers {
			resp += "\t\t" + server.String() + "\n"
		}
	}
	return resp
}
