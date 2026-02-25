package conf

import (
	"continuity/common"
	"continuity/server/api"
	"continuity/server/loadbalancer"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func fakeLoadBalancer(address string, port int) (*loadbalancer.LoadBalancer, error) {
	return &loadbalancer.LoadBalancer{
		BindAddress: address,
		BindPort:    port,
		Pools:       make(map[string]*loadbalancer.Pool),
	}, nil
}

func TestCreateSampleConfig(t *testing.T) {
	tmp := filepath.Join(os.TempDir(), "sample_config.yaml")
	defer os.Remove(tmp)

	err := CreateSampleConfig(tmp)
	require.NoError(t, err)

	_, err = os.Stat(tmp)
	require.NoError(t, err)

	data, err := os.ReadFile(tmp)
	require.Equal(t, `address: 0.0.0.0
port: 80
managenentaddress: 127.0.0.1
managementport: 8090
pools: []
`, string(data))
}

func TestSaveAndLoadBaseConfig(t *testing.T) {
	loadbalancer.NewLoadBalancer = fakeLoadBalancer
	tmp := filepath.Join(os.TempDir(), "test_config.yaml")
	defer os.Remove(tmp)
	fakeChannel := make(chan bool, 10)
	lb, _ := loadbalancer.NewLoadBalancer("127.0.0.1", 8080)
	apiServer := api.NewApiServer("127.0.0.1", 8090, lb, fakeChannel, nil)

	err := SaveConfig(tmp, lb, apiServer)
	require.NoError(t, err)

	lb2, api2, err := LoadConfig(tmp)
	require.NoError(t, err)
	require.NotNil(t, lb2)
	require.NotNil(t, api2)
	require.Equal(t, lb, lb2)
	require.Equal(t, apiServer.Address, api2.Address)
	require.Equal(t, apiServer.Port, api2.Port)
}

func TestSaveAndLoadConfigWithPool(t *testing.T) {
	loadbalancer.NewLoadBalancer = fakeLoadBalancer
	tmp := filepath.Join(os.TempDir(), "test_config_with_pool.yaml")
	defer os.Remove(tmp)

	lb, _ := loadbalancer.NewLoadBalancer("127.0.0.1", 8080)
	pool := loadbalancer.NewPool(
		"test.example.com",
		5*time.Second,  // HealthCheckTimeoutSeconds
		10*time.Second, // HealthCheckIntervalSeconds
		2*time.Second,  // HealthCheckInitialDelaySeconds
		3,              // HealthCheck_numOk
		1,              // HealthCheck_numFail
	)
	err := lb.AddPool(pool)
	require.NoError(t, err)
	fakeChannel := make(chan bool, 10)
	apiServer := api.NewApiServer("127.0.0.1", 8090, lb, fakeChannel, nil)

	err = SaveConfig(tmp, lb, apiServer)
	require.NoError(t, err)

	lb2, api2, err := LoadConfig(tmp)
	require.NoError(t, err)
	require.NotNil(t, lb2)
	require.NotNil(t, api2)

	pool2, ok := lb2.Pools["test.example.com"]
	require.True(t, ok)
	require.Equal(t, pool.Hostname, pool2.Hostname)
	require.Equal(t, pool.HealthCheckTimeout.Load(), pool2.HealthCheckTimeout.Load())
	require.Equal(t, pool.HealthCheckInterval.Load(), pool2.HealthCheckInterval.Load())
	require.Equal(t, pool.HealthCheckInitialDelay.Load(), pool2.HealthCheckInitialDelay.Load())
	require.Equal(t, uint32(3), pool2.HealthCheck_numOk.Load())
	require.Equal(t, pool.HealthCheck_numFail.Load(), pool2.HealthCheck_numFail.Load())
}

func TestSaveAndLoadConfigWithConditionalAndUnconditionalServers(t *testing.T) {
	loadbalancer.NewLoadBalancer = fakeLoadBalancer
	tmp := filepath.Join(os.TempDir(), "test_config_with_servers.yaml")
	defer os.Remove(tmp)

	lb, _ := loadbalancer.NewLoadBalancer("127.0.0.1", 8080)
	pool := loadbalancer.NewPool(
		"test.example.com",
		5*time.Second,  // HealthCheckTimeoutSeconds
		10*time.Second, // HealthCheckIntervalSeconds
		2*time.Second,  // HealthCheckInitialDelaySeconds
		3,              // HealthCheck_numOk
		1,              // HealthCheck_numFail
	)

	uncondServer, _ := loadbalancer.NewServerHost("http://1.2.3.4:8081", "/health", common.Condition{})
	pool.AddServer(uncondServer)

	cond := common.Condition{Header: "X-Env", Value: "prod"}
	condServer, _ := loadbalancer.NewServerHost("http://5.6.7.8:8082", "/status", cond)
	pool.ConditionalServers = append(pool.ConditionalServers, condServer)
	fakeChannel := make(chan bool, 10)
	lb.Pools["test.example.com"] = pool
	apiServer := api.NewApiServer("127.0.0.1", 8090, lb, fakeChannel, nil)

	err := SaveConfig(tmp, lb, apiServer)
	require.NoError(t, err)

	lb2, api2, err := LoadConfig(tmp)
	require.NoError(t, err)
	require.NotNil(t, lb2)
	require.NotNil(t, api2)

	pool2, ok := lb2.Pools["test.example.com"]
	require.True(t, ok)

	require.Len(t, pool2.UnconditionalServers, 1)
	require.Equal(t, uncondServer.Address.String(), pool2.UnconditionalServers[0].Address.String())
	require.Equal(t, uncondServer.HealthCheckPath, pool2.UnconditionalServers[0].HealthCheckPath)

	require.Len(t, pool2.ConditionalServers, 1)
	require.Equal(t, condServer.Address.String(), pool2.ConditionalServers[0].Address.String())
	require.Equal(t, condServer.HealthCheckPath, pool2.ConditionalServers[0].HealthCheckPath)
	require.Equal(t, cond.Header, pool2.ConditionalServers[0].Condition.Header)
	require.Equal(t, cond.Value, pool2.ConditionalServers[0].Condition.Value)
}
