package loadbalancer

import (
	"continuity/common"
	"errors"
	"log"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
)

type Pool struct {
	Hostname                string
	HealthCheckInterval     atomic.Uint64
	HealthCheckInitialDelay atomic.Uint64
	HealthCheckTimeout      atomic.Uint64
	HealthCheck_numOk       atomic.Uint32
	HealthCheck_numFail     atomic.Uint32
	ConditionalServers      []*ServerHost
	UnconditionalServers    []*ServerHost
	StickySessions          bool
	StickyMethod            StickyMethod
	StickySessionTimeout    time.Duration
	stickyCookieName        string
	stickySessionMap        map[string]Session
	stickySessionMutex      *sync.RWMutex
	serverListMutex         *sync.RWMutex
	client                  *http.Client
	RequestCounter          atomic.Uint64
}

type Session struct {
	ServerHost *ServerHost
	CreatedAt  time.Time
}

func (s Session) isExpired(sessionDuration time.Duration) bool {
	return time.Since(s.CreatedAt) > sessionDuration
}

const (
	StickyMethod_IP StickyMethod = iota
	StickyMethod_AppCookie
	StickyMethod_LBCookie
)

const LB_COOKIE_NAME = "x-continuity-sticky"

type StickyMethod int

var StickyMethodName = map[StickyMethod]string{
	StickyMethod_IP:        "IP",
	StickyMethod_AppCookie: "AppCookie",
	StickyMethod_LBCookie:  "LBCookie",
}

func (s StickyMethod) String() string {
	return StickyMethodName[s]
}

func GetStickyMethodFromString(method string) (StickyMethod, error) {
	for k, v := range StickyMethodName {
		if v == method {
			return k, nil
		}
	}
	return -1, errors.New("No StickyMethod exists for value " + method)
}

func NewPool(hostname string,
	healthCheckTimeout, interval, healthCheckInitialDelay time.Duration,
	numOk uint32, numFail uint32) *Pool {
	pool := &Pool{
		Hostname:             hostname,
		ConditionalServers:   []*ServerHost{},
		UnconditionalServers: []*ServerHost{},
		stickySessionMap:     map[string]Session{},
		stickySessionMutex:   &sync.RWMutex{},
		serverListMutex:      &sync.RWMutex{},
		client: &http.Client{
			Timeout: healthCheckTimeout,
		},
	}
	pool.HealthCheckTimeout.Store(uint64(healthCheckTimeout))
	pool.HealthCheckInterval.Store(uint64(interval))
	pool.HealthCheckInitialDelay.Store(uint64(healthCheckInitialDelay))
	pool.HealthCheck_numOk.Store(numOk)
	pool.HealthCheck_numFail.Store(numFail)
	return pool
}

/*
NewPoolWithStickySession
Creates a new Pool with Sticky Sessions enabled managed by Load Balancer using the LB_COOKIE_NAME cookie.
*/
func NewPoolWithStickySession(hostname string,
	healthCheckTimeout,
	interval,
	healthCheckInitialDelay,
	stickySessionTimeout time.Duration,
	numOk,
	numFail uint32) *Pool {
	pool := NewPool(hostname,
		healthCheckTimeout,
		interval,
		healthCheckInitialDelay,
		numOk,
		numFail)
	pool.StickySessions = true
	pool.StickyMethod = StickyMethod_LBCookie
	pool.stickyCookieName = LB_COOKIE_NAME
	pool.StickySessionTimeout = stickySessionTimeout
	return pool
}

/*
NewPoolWithStickySessionCustomCookie
Creates a new Pool with Sticky Sessions enabled using a cookie provided by the
server behind the load balancer.
*/
func NewPoolWithStickySessionCustomCookie(hostname string,
	healthCheckTimeout,
	interval,
	healthCheckInitialDelay,
	stickySessionTimeout time.Duration,
	numOk,
	numFail uint32,
	cookieName string) (*Pool, error) {
	if cookieName == "" {
		return nil, errors.New("cookie name cannot be empty")
	}
	pool := NewPoolWithStickySession(hostname,
		healthCheckTimeout,
		interval,
		healthCheckInitialDelay,
		stickySessionTimeout,
		numOk,
		numFail)
	pool.StickyMethod = StickyMethod_AppCookie
	pool.stickyCookieName = cookieName
	return pool, nil
}

/*
NewPoolWithIPStickySessions
Creates a new Pool with Sticky Sessions enabled managed by Load Balancer using Client IP.
Each IP will hit the same backend server as long as it's healthy.
*/
func NewPoolWithIPStickySessions(hostname string,
	healthCheckTimeout,
	interval,
	healthCheckInitialDelay,
	stickySessionTimeout time.Duration,
	numOk,
	numFail uint32) *Pool {
	pool := NewPoolWithStickySession(hostname,
		healthCheckTimeout,
		interval,
		healthCheckInitialDelay,
		stickySessionTimeout,
		numOk,
		numFail)
	pool.StickyMethod = StickyMethod_IP
	return pool
}

func (p *Pool) AddServer(server *ServerHost) {
	if p.StickySessions && p.StickyMethod == StickyMethod_LBCookie {
		server.setLbCookie(p.stickyCookieName)
	}
	if p.StickySessions && p.StickyMethod == StickyMethod_AppCookie {
		server.setAppCookieInterceptor(p.stickyCookieName, func(cookieValue string) {
			p.createStickySession(nil, server, cookieValue)
		})
	}
	p.serverListMutex.Lock()
	defer p.serverListMutex.Unlock()
	if server.Condition == (common.Condition{}) {
		p.UnconditionalServers = append(p.UnconditionalServers, server)
	} else {
		p.ConditionalServers = append(p.ConditionalServers, server)
	}
}

func (p *Pool) RemoveServer(uuid uuid.UUID) (*ServerHost, error) {
	p.serverListMutex.Lock()
	defer p.serverListMutex.Unlock()
	for i, server := range p.ConditionalServers {
		if server.Id == uuid {
			p.ConditionalServers = append(p.ConditionalServers[:i], p.ConditionalServers[i+1:]...)
			return server, nil
		}
	}
	for i, server := range p.UnconditionalServers {
		if server.Id == uuid {
			p.UnconditionalServers = append(p.UnconditionalServers[:i], p.UnconditionalServers[i+1:]...)
			return server, nil
		}
	}
	return nil, errors.New("server not found in pool")
}

func (p *Pool) Transaction(serverToAdd *ServerHost, serverToRemove uuid.UUID) error {
	p.AddServer(serverToAdd)
	timeoutChan := time.After(time.Duration(p.HealthCheckInitialDelay.Load()) + time.Duration(p.HealthCheckTimeout.Load())*time.Duration(p.HealthCheck_numOk.Load()*2) + 1*time.Second)
	timedOut := false
	for {
		if serverToAdd.ServerStatus.Load() != uint32(Pending) {
			break
		}
		select {
		case <-timeoutChan:
			timedOut = true
			break
		default:
			time.Sleep(100 * time.Millisecond)
		}
	}
	if serverToAdd.ServerStatus.Load() == uint32(Healthy) {
		_, _ = p.RemoveServer(serverToRemove)
		return nil
	} else {
		_, _ = p.RemoveServer(serverToAdd.Id)
		if timedOut {
			return errors.New("new server is taking too long, transaction rolled back")
		} else {
			return errors.New("new server is not healthy, transaction rolled back. Server " + serverToAdd.String())
		}
	}
}

func (p *Pool) ChooseServer(req *http.Request) (*ServerHost, error) {
	if p.StickySessions {
		stickyServer := p.getStickyServer(req)
		if stickyServer != nil && stickyServer.ServerStatus.Load() == uint32(Healthy) {
			log.Println("Pool", p.Hostname, "- Sticky session hit for server", stickyServer.Address.String())
			return stickyServer, nil
		}
	}
	p.RequestCounter.Add(1)
	p.serverListMutex.RLock()
	defer p.serverListMutex.RUnlock()
	for _, server := range p.ConditionalServers {
		if server.ServerStatus.Load() == uint32(Healthy) && server.CheckCondition(req) {
			if p.StickySessions {
				p.createStickySession(req, server, "")
			}
			return server, nil
		}
	}

	//On unconditional servers we pick a random healthy one
	healtyServers := []*ServerHost{}

	for _, server := range p.UnconditionalServers {
		if server.ServerStatus.Load() == uint32(Healthy) {
			healtyServers = append(healtyServers, server)
		}
	}

	if len(healtyServers) > 0 {
		server := healtyServers[p.RequestCounter.Load()%uint64(len(healtyServers))]
		if p.StickySessions {
			p.createStickySession(req, server, "")
		}
		return server, nil
	}
	return nil, errors.New("no healthy servers available in pool")
}

func (p *Pool) getStickyServer(req *http.Request) *ServerHost {
	p.stickySessionMutex.RLock()
	defer p.stickySessionMutex.RUnlock()
	var stickySession Session
	switch p.StickyMethod {
	case StickyMethod_IP:
		stickySession = p.stickySessionMap[getHostFromRequest(req)]
		break
	case StickyMethod_LBCookie:
		fallthrough
	case StickyMethod_AppCookie:
		cookie, err := req.Cookie(p.stickyCookieName)
		if err == nil {
			stickySession = p.stickySessionMap[cookie.Value]
		}
		break
	}
	if stickySession != (Session{}) && (!p.checkIfServerExists(stickySession.ServerHost) ||
		stickySession.isExpired(p.StickySessionTimeout)) {
		return nil
	}
	return stickySession.ServerHost
}

//TODO evict expired sessions periodically

func (p *Pool) checkIfStickySessionExists(req *http.Request, server *ServerHost, appCookieValue string) bool {
	p.stickySessionMutex.RLock()
	defer p.stickySessionMutex.RUnlock()
	switch p.StickyMethod {
	case StickyMethod_IP:
		v, ok := p.stickySessionMap[getHostFromRequest(req)]
		return ok && !v.isExpired(p.StickySessionTimeout)
	case StickyMethod_LBCookie:
		v, ok := p.stickySessionMap[server.Id.String()]
		return ok && !v.isExpired(p.StickySessionTimeout)
	case StickyMethod_AppCookie:
		v, ok := p.stickySessionMap[appCookieValue]
		return ok && !v.isExpired(p.StickySessionTimeout)
	}
	return false
}

func (p *Pool) createStickySession(req *http.Request, server *ServerHost, appCookieValue string) {
	if p.checkIfStickySessionExists(req, server, appCookieValue) {
		return
	}
	p.stickySessionMutex.Lock()
	defer p.stickySessionMutex.Unlock()
	switch p.StickyMethod {
	case StickyMethod_IP:

		p.stickySessionMap[getHostFromRequest(req)] = Session{
			ServerHost: server,
			CreatedAt:  time.Now(),
		}

		break
	case StickyMethod_LBCookie:
		p.stickySessionMap[server.Id.String()] = Session{
			ServerHost: server,
			CreatedAt:  time.Now(),
		}
	case StickyMethod_AppCookie:
		p.stickySessionMap[appCookieValue] = Session{
			ServerHost: server,
			CreatedAt:  time.Now(),
		}
	}
}

func (p *Pool) GetStats() map[string]ServerStats {
	stats := make(map[string]ServerStats)
	log.Printf("Pool %s Stats:\n", p.Hostname)
	p.serverListMutex.RLock()
	defer p.serverListMutex.RUnlock()
	for _, server := range append(p.ConditionalServers, p.UnconditionalServers...) {
		stats[server.Address.String()] = ServerStats{
			OkResponses:    server.OkResponsesStats.Load(),
			NotOkResponses: server.NotOkResponsesStats.Load(),
		}
	}
	return stats
}

func (p *Pool) RunHealthChecks() {
	//TODO better handling to not lock for too long
	p.serverListMutex.RLock()
	defer p.serverListMutex.RUnlock()
	for _, server := range append(p.ConditionalServers, p.UnconditionalServers...) {
		if time.Since(time.Unix(server.LastChecked.Load(), 0)) >= time.Duration(p.HealthCheckInterval.Load()) && server.isReady(time.Duration(p.HealthCheckInitialDelay.Load())) {

			go p.check(server)
		}
	}
}

func (p *Pool) check(server *ServerHost) {
	resp, err := p.client.Get(server.Address.String() + server.HealthCheckPath)
	defer func() {
		if err == nil {
			_ = resp.Body.Close()
		}
	}()
	serverStatus := (ServerStatus)(server.ServerStatus.Load())
	if err != nil || resp.StatusCode != http.StatusOK {
		if serverStatus == Healthy || serverStatus == Pending {
			server.UnHealthyResponses.Add(1)
			if server.UnHealthyResponses.Load() >= p.HealthCheck_numFail.Load() {
				server.SetUnHealty()
				log.Printf("Pool %s - Server %s marked as Unhealthy\n", p.Hostname, server.Address.String())
			}
		}
	} else {
		if serverStatus == Unhealthy || serverStatus == Pending {
			server.HealthyResponses.Add(1)
			if server.HealthyResponses.Load() >= p.HealthCheck_numOk.Load() {
				server.SetHealty()
				log.Printf("Pool %s - Server %s is Healty\n", p.Hostname, server.Address.String())
			}
		}
	}
	server.LastChecked.Store(time.Now().Unix())
}

func (p *Pool) checkIfServerExists(host *ServerHost) bool {
	p.serverListMutex.RLock()
	defer p.serverListMutex.RUnlock()
	for _, server := range append(p.ConditionalServers, p.UnconditionalServers...) {
		if server.Id == host.Id {
			return true
		}
	}
	return false
}

func getHostFromRequest(req *http.Request) string {
	return strings.Split(req.RemoteAddr, ":")[0]
}

func (p *Pool) GetStickyCookieName() string {
	return p.stickyCookieName
}
