package conf

import (
	"continuity/common"
	"continuity/server/api"
	"continuity/server/loadbalancer"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/google/uuid"
	"gopkg.in/yaml.v2"
)

var ConfigPath = "config.yaml"
var SaveConfigChan = make(chan bool, 10)
var autosaveStarted = false

type Configuration struct {
	Address           string
	Port              int
	ManagenentAddress string
	ManagementPort    int
	Pools             []PoolConfig
}

type PoolConfig struct {
	Hostname                       string
	HealthCheckIntervalSeconds     uint64
	HealthCheckInitialDelaySeconds uint64
	HealthCheckTimeoutSeconds      uint64
	HealthCheck_numOk              uint32
	HealthCheck_numFail            uint32
	ConditionalServers             []*ServerHostConfig
	UnconditionalServers           []*ServerHostConfig
	StickySessions                 bool
	StickyMethod                   string
	StickySessionTimeoutSeconds    uint32
	stickyCookieName               string
}

type ServerHostConfig struct {
	Id              uuid.UUID
	Address         string
	Condition       common.Condition
	HealthCheckPath string
}

func LoadConfig(path string) (*loadbalancer.LoadBalancer, *api.ApiServer, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, nil, err
	}
	defer func() { _ = file.Close() }()

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, nil, err
	}
	configuration := &Configuration{}
	err = yaml.Unmarshal(data, configuration)
	if err != nil {
		return nil, nil, err
	}
	lb, err := loadbalancer.NewLoadBalancer(configuration.Address, configuration.Port)
	if err != nil {
		return nil, nil, err
	}
	for _, poolConf := range configuration.Pools {
		var pool *loadbalancer.Pool
		if poolConf.StickySessions {
			stickyMethod, err := loadbalancer.GetStickyMethodFromString(poolConf.StickyMethod)
			if err != nil {
				return nil, nil, err
			}
			switch stickyMethod {
			case loadbalancer.StickyMethod_AppCookie:
				pool, err = loadbalancer.NewPoolWithStickySessionCustomCookie(
					poolConf.Hostname,
					time.Second*time.Duration(poolConf.HealthCheckTimeoutSeconds),
					time.Second*time.Duration(poolConf.HealthCheckIntervalSeconds),
					time.Second*time.Duration(poolConf.HealthCheckInitialDelaySeconds),
					time.Second*time.Duration(poolConf.StickySessionTimeoutSeconds),
					poolConf.HealthCheck_numOk,
					poolConf.HealthCheck_numFail,
					poolConf.stickyCookieName,
				)
				if err != nil {
					return nil, nil, err
				}
				break
			case loadbalancer.StickyMethod_IP:
				pool = loadbalancer.NewPoolWithIPStickySessions(
					poolConf.Hostname,
					time.Second*time.Duration(poolConf.HealthCheckTimeoutSeconds),
					time.Second*time.Duration(poolConf.HealthCheckIntervalSeconds),
					time.Second*time.Duration(poolConf.HealthCheckInitialDelaySeconds),
					time.Second*time.Duration(poolConf.StickySessionTimeoutSeconds),
					poolConf.HealthCheck_numOk,
					poolConf.HealthCheck_numFail,
				)
				break
			default:
				pool = loadbalancer.NewPoolWithStickySession(
					poolConf.Hostname,
					time.Second*time.Duration(poolConf.HealthCheckTimeoutSeconds),
					time.Second*time.Duration(poolConf.HealthCheckIntervalSeconds),
					time.Second*time.Duration(poolConf.HealthCheckInitialDelaySeconds),
					time.Second*time.Duration(poolConf.StickySessionTimeoutSeconds),
					poolConf.HealthCheck_numOk,
					poolConf.HealthCheck_numFail,
				)
			}
		} else {
			pool = loadbalancer.NewPool(
				poolConf.Hostname,
				time.Second*time.Duration(poolConf.HealthCheckTimeoutSeconds),
				time.Second*time.Duration(poolConf.HealthCheckIntervalSeconds),
				time.Second*time.Duration(poolConf.HealthCheckInitialDelaySeconds),
				poolConf.HealthCheck_numOk,
				poolConf.HealthCheck_numFail,
			)
		}

		for _, serverConf := range poolConf.ConditionalServers {
			serverHost, err := loadbalancer.NewServerHost(serverConf.Address, serverConf.HealthCheckPath, serverConf.Condition)
			serverHost.Id = serverConf.Id
			if err != nil {
				return nil, nil, err
			}
			pool.AddServer(serverHost)
		}
		for _, serverConf := range poolConf.UnconditionalServers {
			serverHost, err := loadbalancer.NewServerHost(serverConf.Address, serverConf.HealthCheckPath, common.Condition{})
			serverHost.Id = serverConf.Id
			if err != nil {
				return nil, nil, err
			}
			pool.AddServer(serverHost)
		}
		err = lb.AddPool(pool)
		if err != nil {
			return nil, nil, err
		}
	}
	apiServer := api.NewApiServer(configuration.ManagenentAddress,
		configuration.ManagementPort,
		lb,
		SaveConfigChan)
	StartAutoSaveConfig(path, lb, apiServer)
	return lb, apiServer, nil
}

func SaveConfig(path string, lb *loadbalancer.LoadBalancer, api *api.ApiServer) error {
	configuration := &Configuration{
		Address:           lb.BindAddress,
		Port:              lb.BindPort,
		ManagenentAddress: api.Address,
		ManagementPort:    api.Port,
		Pools:             []PoolConfig{},
	}
	for _, pool := range lb.GetPools() {
		poolConf := PoolConfig{
			Hostname:                       pool.Hostname,
			HealthCheckIntervalSeconds:     pool.HealthCheckInterval.Load() / uint64(time.Second),
			HealthCheckInitialDelaySeconds: pool.HealthCheckInitialDelay.Load() / uint64(time.Second),
			HealthCheckTimeoutSeconds:      pool.HealthCheckTimeout.Load() / uint64(time.Second),
			HealthCheck_numOk:              uint32(pool.HealthCheck_numOk.Load()),
			HealthCheck_numFail:            uint32(pool.HealthCheck_numFail.Load()),
			ConditionalServers:             []*ServerHostConfig{},
			UnconditionalServers:           []*ServerHostConfig{},
			StickySessions:                 pool.StickySessions,
		}
		if pool.StickySessions {
			poolConf.StickyMethod = pool.StickyMethod.String()
			poolConf.StickySessionTimeoutSeconds = uint32(pool.StickySessionTimeout.Seconds())
			poolConf.stickyCookieName = pool.GetStickyCookieName()
		}
		for _, server := range pool.ConditionalServers {
			serverConf := &ServerHostConfig{
				Id:              server.Id,
				Address:         server.Address.String(),
				Condition:       server.Condition,
				HealthCheckPath: server.HealthCheckPath,
			}
			poolConf.ConditionalServers = append(poolConf.ConditionalServers, serverConf)
		}
		for _, server := range pool.UnconditionalServers {
			serverConf := &ServerHostConfig{
				Id:              server.Id,
				Address:         server.Address.String(),
				HealthCheckPath: server.HealthCheckPath,
			}
			poolConf.UnconditionalServers = append(poolConf.UnconditionalServers, serverConf)
		}
		configuration.Pools = append(configuration.Pools, poolConf)
	}
	data, err := yaml.Marshal(configuration)
	if err != nil {
		return err
	}
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			fmt.Println("Error closing file:", err)
		}
	}(file)
	_, err = file.Write(data)
	return err
}

func CreateSampleConfig(path string) error {
	configuration := &Configuration{
		Address:           "0.0.0.0",
		Port:              443,
		ManagenentAddress: "127.0.0.1",
		ManagementPort:    8090,
		Pools:             []PoolConfig{},
	}
	data, err := yaml.Marshal(configuration)
	if err != nil {
		return err
	}
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			fmt.Println("Error closing file:", err)
		}
	}(file)
	_, err = file.Write(data)
	return err
}

func StartAutoSaveConfig(path string, lb *loadbalancer.LoadBalancer, api *api.ApiServer) {
	if autosaveStarted {
		return
	}
	autosaveStarted = true
	go func() {
		for {
			<-SaveConfigChan
			err := SaveConfig(path, lb, api)
			if err != nil {
				fmt.Println("Error saving configuration:", err)
			}
		}
	}()
}
