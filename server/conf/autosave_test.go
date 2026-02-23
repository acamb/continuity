package conf

import (
	"bytes"
	"continuity/server/api"
	"continuity/server/loadbalancer"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestAutoSaveServersAndReloadConfig(t *testing.T) {
	fmt.Println("Executing TestAutoSaveServersAndReloadConfig")
	loadbalancer.NewLoadBalancer = fakeLoadBalancer
	defer os.Remove(ConfigPath)
	gin.SetMode(gin.TestMode)
	lb, _ := loadbalancer.NewLoadBalancer("127.0.0.1", 8080)
	apiServer := api.NewApiServer("127.0.0.1", 8090, lb, SaveConfigChan, "")

	//Create pool request
	reqBody := map[string]interface{}{
		"hostname":                   "test",
		"health_check_interval":      6,
		"health_check_initial_delay": 7,
		"health_check_timeout":       8,
		"health_check_num_ok":        9,
		"health_check_num_fail":      10,
		"sticky_sessions":            false,
	}
	body, _ := json.Marshal(reqBody)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("POST", "/pools", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	apiServer.CreatePool(c)
	require.Equal(t, http.StatusOK, w.Code)

	//Add unconditional server request
	reqBody = map[string]interface{}{
		"new_server_address": "http://127.0.0.1:8081",
		"health_check_path":  "/health",
	}
	body, _ = json.Marshal(reqBody)
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("POST", "/pools/"+base64.RawURLEncoding.EncodeToString([]byte("test"))+"/server", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{
		{Key: "hostname", Value: base64.RawURLEncoding.EncodeToString([]byte("test"))},
	}

	apiServer.AddServer(c)
	require.Equal(t, http.StatusOK, w.Code)

	//Add conditional server request
	reqBody = map[string]interface{}{
		"new_server_address": "http://127.0.0.1:8081",
		"health_check_path":  "/health",
		"condition": map[string]string{
			"header": "X-Env",
			"value":  "prod",
		},
	}
	body, _ = json.Marshal(reqBody)
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("POST", "/pools/"+base64.RawURLEncoding.EncodeToString([]byte("test"))+"/server", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{
		{Key: "hostname", Value: base64.RawURLEncoding.EncodeToString([]byte("test"))},
	}

	//Manually start the autosave routine (it's normally called in LoadConfig() )
	StartAutoSaveConfig(ConfigPath, lb, apiServer)

	apiServer.AddServer(c)
	require.Equal(t, http.StatusOK, w.Code)

	pool, ok := lb.Pools["test"]
	require.True(t, ok)

	require.Len(t, pool.ConditionalServers, 1)
	require.Len(t, pool.UnconditionalServers, 1)

	time.Sleep(1 * time.Second) //Wait for the autosave to complete
	lb2, api2, err := LoadConfig(ConfigPath)
	require.NoError(t, err)
	require.NotNil(t, lb2)
	require.NotNil(t, api2)

	pool2, ok := lb2.Pools["test"]
	require.True(t, ok)
	require.Equal(t, pool.Hostname, pool2.Hostname)
	require.Equal(t, pool.HealthCheckTimeout.Load(), pool2.HealthCheckTimeout.Load())
	require.Equal(t, pool.HealthCheckInterval.Load(), pool2.HealthCheckInterval.Load())
	require.Equal(t, pool.HealthCheckInitialDelay.Load(), pool2.HealthCheckInitialDelay.Load())
	require.Equal(t, pool.HealthCheck_numOk.Load(), pool2.HealthCheck_numOk.Load())
	require.Equal(t, pool.HealthCheck_numFail.Load(), pool2.HealthCheck_numFail.Load())

	require.Len(t, pool2.UnconditionalServers, 1)

	uncondServer := pool.UnconditionalServers[0]
	require.Equal(t, uncondServer.Address.String(), pool2.UnconditionalServers[0].Address.String())
	require.Equal(t, uncondServer.HealthCheckPath, pool2.UnconditionalServers[0].HealthCheckPath)

	require.Len(t, pool2.ConditionalServers, 1)
	condServer := pool.ConditionalServers[0]
	require.Equal(t, condServer.Address.String(), pool2.ConditionalServers[0].Address.String())
	require.Equal(t, condServer.HealthCheckPath, pool2.ConditionalServers[0].HealthCheckPath)
	require.Equal(t, condServer.Condition.Header, pool2.ConditionalServers[0].Condition.Header)
	require.Equal(t, condServer.Condition.Value, pool2.ConditionalServers[0].Condition.Value)
}
