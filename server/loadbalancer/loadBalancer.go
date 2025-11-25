package loadbalancer

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
)

type LoadBalancer struct {
	BindAddress string
	BindPort    int
	Pools       map[string]*Pool
	poolMutex   sync.RWMutex
}

func newLoadBalancer(bindAddress string, bindPort int) (*LoadBalancer, error) {
	lb := &LoadBalancer{
		BindAddress: bindAddress,
		BindPort:    bindPort,
		Pools:       make(map[string]*Pool),
	}
	http.HandleFunc("/", lb.ServeRequest)
	log.Println("Starting load balancer on", bindAddress+":"+fmt.Sprint(bindPort))
	go func() {
		_ = http.ListenAndServe(bindAddress+":"+fmt.Sprint(bindPort), nil)
	}()

	log.Println("Load balancer is listening on", bindAddress+":"+fmt.Sprint(bindPort))
	go lb.healthCheckLoop()
	return lb, nil
}

var NewLoadBalancer = newLoadBalancer

func (lb *LoadBalancer) healthCheckLoop() {
	log.Println("Starting health-checks loop for load balancer")

	for {
		lb.poolMutex.RLock()
		//TODO better handling to not lock for too long
		for _, pool := range lb.Pools {
			pool.RunHealthChecks()
		}
		lb.poolMutex.RUnlock()
		time.Sleep(1 * time.Second)
	}
}

func (lb *LoadBalancer) AddPool(pool *Pool) error {
	lb.poolMutex.Lock()
	defer lb.poolMutex.Unlock()
	_, ok := lb.Pools[pool.Hostname]
	if ok {
		return errors.New("Pool already exists")
	}
	lb.Pools[pool.Hostname] = pool
	return nil
}

func (lb *LoadBalancer) GetPool(hostname string) (*Pool, error) {
	lb.poolMutex.RLock()
	defer lb.poolMutex.RUnlock()
	if pool, exists := lb.Pools[hostname]; exists {
		return pool, nil
	}
	return &Pool{}, errors.New("pool not found")
}

func (lb *LoadBalancer) UpdatePool(pool *Pool) error {
	lb.poolMutex.Lock()
	defer lb.poolMutex.Unlock()
	existingPool, exists := lb.Pools[pool.Hostname]
	if !exists {
		return errors.New("Pool does not exist")
	}
	existingPool.HealthCheckInitialDelay.Store(pool.HealthCheckInitialDelay.Load())
	existingPool.HealthCheckTimeout.Store(pool.HealthCheckTimeout.Load())
	existingPool.HealthCheckInterval.Store(pool.HealthCheckInterval.Load())
	existingPool.HealthCheck_numOk.Store(pool.HealthCheck_numOk.Load())
	existingPool.HealthCheck_numFail.Store(pool.HealthCheck_numFail.Load())
	existingPool.client.Timeout = time.Duration(pool.HealthCheckTimeout.Load())
	return nil
}

func (lb *LoadBalancer) RemovePool(hostname string) error {
	lb.poolMutex.Lock()
	defer lb.poolMutex.Unlock()
	if _, exists := lb.Pools[hostname]; exists {
		delete(lb.Pools, hostname)
		return nil
	}
	return errors.New("pool not found")
}

func (lb *LoadBalancer) ServeRequest(rw http.ResponseWriter, r *http.Request) {
	lb.poolMutex.RLock()
	pool, exists := lb.Pools[r.Host]
	lb.poolMutex.RUnlock()
	if !exists {
		log.Println("No pool found for host:", r.Host)
		return
	}
	server, err := pool.ChooseServer(r)
	if err != nil {
		log.Println("No server available for request to host:", r.Host)
		http.Error(rw, "Service Unavailable", http.StatusServiceUnavailable)
		return
	}
	server.ServeHTTP(rw, r)
}

func (lb *LoadBalancer) GetPools() []*Pool {
	lb.poolMutex.RLock()
	defer lb.poolMutex.RUnlock()
	pools := make([]*Pool, 0, len(lb.Pools))
	for _, pool := range lb.Pools {
		pools = append(pools, pool)
	}
	return pools
}
