package responses

import (
	"continuity/server/loadbalancer"
	"fmt"
	"time"
)

type PoolResponse struct {
	Hostname                string                `json:"hostname"`
	HealthCheckInterval     uint64                `json:"health_check_interval"`
	HealthCheckInitialDelay uint64                `json:"health_check_initial_delay"`
	HealthCheckTimeout      uint64                `json:"health_check_timeout"`
	HealthCheck_numOk       uint32                `json:"health_check_num_ok"`
	HealthCheck_numFail     uint32                `json:"health_check_num_fail"`
	ConditionalServers      []*ServerHostResponse `json:"conditional_servers"`
	UnconditionalServers    []*ServerHostResponse `json:"unconditional_servers"`
	StickySessions          bool                  `json:"sticky_sessions"`
	StickyMethod            string                `json:"sticky_method"`
	StickySessionTimeout    uint64                `json:"sticky_session_timeout"`
	stickyCookieName        string                `json:"sticky_cookie_name"`
	requestCounter          uint64                `json:"request_counter"`
}

func NewPoolResponse(pool *loadbalancer.Pool) *PoolResponse {
	resp := &PoolResponse{
		Hostname:                pool.Hostname,
		HealthCheckInterval:     uint64(time.Duration(pool.HealthCheckInterval.Load()).Seconds()),
		HealthCheckInitialDelay: uint64(time.Duration(pool.HealthCheckInitialDelay.Load()).Seconds()),
		HealthCheckTimeout:      uint64(time.Duration(pool.HealthCheckTimeout.Load()).Seconds()),
		HealthCheck_numOk:       pool.HealthCheck_numOk.Load(),
		HealthCheck_numFail:     pool.HealthCheck_numFail.Load(),
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
	resp += fmt.Sprintf("\tRequestCounter=%d\n", pr.requestCounter)
	return resp
}
