package loadbalancer

import (
	"continuity/common"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
)

type ServerStatus int32

const (
	Healthy ServerStatus = iota
	Unhealthy
	Pending
	Draining
)

var serverStatusName = map[ServerStatus]string{
	Healthy:   "Healthy",
	Unhealthy: "Unhealthy",
	Pending:   "Pending",
	Draining:  "Draining",
}

type ServerStats struct {
	OkResponses    uint64
	NotOkResponses uint64
}

func (ss ServerStatus) String() string {
	return serverStatusName[ss]
}

type InterceptAppCookieCallback func(cookieValue string)
type ServerHost struct {
	Id                         uuid.UUID
	Address                    *url.URL
	Condition                  common.Condition
	ServerStatus               atomic.Uint32
	HealthCheckPath            string
	LastChecked                atomic.Int64
	HealthyResponses           atomic.Uint32
	UnHealthyResponses         atomic.Uint32
	OkResponsesStats           atomic.Uint64
	NotOkResponsesStats        atomic.Uint64
	proxy                      *httputil.ReverseProxy
	CreatedAt                  int64
	lbCookieName               string
	appCookieName              string
	interceptAppCookieCallback InterceptAppCookieCallback
}

func NewServerHost(address string, healtCheckPath string, condition common.Condition) (*ServerHost, error) {
	parsed, err := url.Parse(address)
	if err != nil {
		return nil, err
	}
	server := &ServerHost{
		Id:              uuid.New(),
		Address:         parsed,
		Condition:       condition,
		HealthCheckPath: healtCheckPath,
		CreatedAt:       time.Now().Unix(),
	}
	server.createProxy(parsed)
	server.ServerStatus.Store(uint32(Pending))
	return server, nil
}

func (sh *ServerHost) createProxy(parsed *url.URL) {
	newProxy := httputil.NewSingleHostReverseProxy(parsed)
	newProxy.ErrorHandler = func(writer http.ResponseWriter, request *http.Request, e error) {
		log.Println("Error proxying request to", request.Host, ":", e)
		writer.WriteHeader(http.StatusBadGateway)
		sh.NotOkResponsesStats.Add(1)
	}
	newProxy.ModifyResponse = func(response *http.Response) error {
		if sh.lbCookieName != "" {
			cookie := &http.Cookie{
				Name:  sh.lbCookieName,
				Value: sh.Id.String(),
			}
			response.Header.Add("Set-Cookie", cookie.String())
		}
		if sh.interceptAppCookieCallback != nil {
			for _, cookie := range response.Cookies() {
				if cookie.Name == sh.appCookieName {
					sh.interceptAppCookieCallback(cookie.Value)
					break
				}
			}
		}
		sh.OkResponsesStats.Add(1)
		return nil
	}
	sh.proxy = newProxy
}

func (sh *ServerHost) CheckCondition(req *http.Request) bool {
	return req.Header.Get(sh.Condition.Header) == sh.Condition.Value
}

func (sh *ServerHost) SetHealty() {
	sh.ServerStatus.Store(uint32(Healthy))
	sh.UnHealthyResponses.Store(0)
	sh.HealthyResponses.Store(0)
}

func (sh *ServerHost) SetUnHealty() {
	sh.ServerStatus.Store(uint32(Unhealthy))
	sh.UnHealthyResponses.Store(0)
	sh.HealthyResponses.Store(0)
}

func (sh *ServerHost) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	sh.proxy.ServeHTTP(rw, r)
}

func (sh *ServerHost) isReady(initialDelay time.Duration) bool {
	return !(sh.ServerStatus.Load() == (uint32)(Pending) && time.Since(time.Unix(sh.CreatedAt, 0)) < initialDelay)
}

func (sh *ServerHost) setLbCookie(name string) {
	sh.lbCookieName = name
}

func (sh *ServerHost) setAppCookieInterceptor(cookie string, callback InterceptAppCookieCallback) {
	sh.interceptAppCookieCallback = callback
	sh.appCookieName = cookie
}
