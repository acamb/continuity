package api

import (
	"bytes"
	"continuity/server/loadbalancer"
	"encoding/base64"
	"encoding/json"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func fakeLoadBalancer(address string, port int) (*loadbalancer.LoadBalancer, error) {
	log.Println("Executing fakeLoadBalancer")
	return &loadbalancer.LoadBalancer{
		BindAddress: address,
		BindPort:    port,
		Pools:       make(map[string]*loadbalancer.Pool),
	}, nil
}

func setupTestServer() *ApiServer {
	log.Println("Executing setupTestServer")
	gin.SetMode(gin.TestMode)
	lb, _ := fakeLoadBalancer("127.0.0.1", 8080)
	fakeSaveChan := make(chan bool, 10) //large enough to avoid blocking in tests
	return NewApiServer("127.0.0.1", 8080, lb, fakeSaveChan)
}

func performRequest(r http.Handler, method, path string, body []byte) *httptest.ResponseRecorder {
	req, _ := http.NewRequest(method, path, bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestGetPools(t *testing.T) {
	log.Println("Executing ", t.Name())
	api := setupTestServer()
	router := gin.Default()
	router.GET("/pools", api.GetPools)

	w := performRequest(router, "GET", "/pools", nil)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestCreatePool_BadRequest(t *testing.T) {
	log.Println("Executing ", t.Name())
	api := setupTestServer()
	router := gin.Default()
	router.POST("/pools", api.CreatePool)

	// Corpo non valido
	w := performRequest(router, "POST", "/pools", []byte(`{}`))
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestCreatePool_Success(t *testing.T) {
	log.Println("Executing ", t.Name())
	api := setupTestServer()
	router := gin.Default()
	router.POST("/pools", api.CreatePool)

	body := map[string]interface{}{
		"hostname":                   "testpool",
		"health_check_interval":      1,
		"health_check_initial_delay": 1,
		"health_check_timeout":       1,
		"health_check_num_ok":        1,
		"health_check_num_fail":      1,
		"sticky_sessions":            true,
		"sticky_method":              "AppCookie",
		"sticky_session_timeout":     10,
		"sticky_session_cookie_name": "testcookie",
	}
	jsonBody, _ := json.Marshal(body)
	w := performRequest(router, "POST", "/pools", jsonBody)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	assert.Len(t, api.LoadBalancer.Pools, 1)
	pool, ok := api.LoadBalancer.Pools["testpool"]
	assert.True(t, ok)
	assert.Equal(t, "testpool", pool.Hostname)
	assert.Equal(t, uint64(1), pool.HealthCheckInterval.Load()/uint64(time.Second))
	assert.Equal(t, uint64(1), pool.HealthCheckInitialDelay.Load()/uint64(time.Second))
	assert.Equal(t, uint64(1), pool.HealthCheckTimeout.Load()/uint64(time.Second))
	assert.Equal(t, uint32(1), pool.HealthCheck_numOk.Load())
	assert.Equal(t, uint32(1), pool.HealthCheck_numFail.Load())
	assert.True(t, pool.StickySessions)
	assert.Equal(t, loadbalancer.StickyMethod_AppCookie, pool.StickyMethod)
	assert.Equal(t, "testcookie", pool.GetStickyCookieName())
	assert.Equal(t, 10, int(pool.StickySessionTimeout.Seconds()))
}

func TestDeletePool_NotFound(t *testing.T) {
	log.Println("Executing ", t.Name())
	api := setupTestServer()
	router := gin.Default()
	router.DELETE("/pools/:hostname", api.DeletePool)

	w := performRequest(router, "DELETE", "/pools/inesistente", nil)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestDeletePool_Success(t *testing.T) {
	log.Println("Executing ", t.Name())
	api := setupTestServer()
	// crea pool
	p := &loadbalancer.Pool{
		Hostname: "delpool",
	}
	api.LoadBalancer.Pools[p.Hostname] = p

	router := gin.Default()
	router.DELETE("/pools/:hostname", api.DeletePool)

	w := performRequest(router, "DELETE", "/pools/"+base64.RawURLEncoding.EncodeToString([]byte("delpool")), nil)
	assert.Equal(t, http.StatusOK, w.Code)
	_, ok := api.LoadBalancer.Pools["delpool"]
	assert.False(t, ok)
}

func TestGetPoolConfig_NotFound(t *testing.T) {
	log.Println("Executing ", t.Name())
	api := setupTestServer()
	router := gin.Default()
	router.GET("/pools/:hostname", api.GetPoolConfig)

	w := performRequest(router, "GET", "/pools/none", nil)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestGetPoolConfig_Success(t *testing.T) {
	log.Println("Executing ", t.Name())
	api := setupTestServer()
	p := loadbalancer.NewPool("test",
		5*time.Second,
		10*time.Second,
		2*time.Second,
		3,
		1,
	)
	api.LoadBalancer.AddPool(p)

	router := gin.Default()
	router.GET("/pools/:hostname", api.GetPoolConfig)

	w := performRequest(router, "GET", "/pools/"+base64.RawURLEncoding.EncodeToString([]byte("test")), nil)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestGetPoolStats_NotFound(t *testing.T) {
	log.Println("Executing ", t.Name())
	api := setupTestServer()
	router := gin.Default()
	router.GET("/pools/:hostname/stats", api.GetPoolStats)

	w := performRequest(router, "GET", "/pools/nostat/stats", nil)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestGetPoolStats_Success(t *testing.T) {
	log.Println("Executing ", t.Name())
	api := setupTestServer()
	p := loadbalancer.NewPool("test",
		5*time.Second,
		10*time.Second,
		2*time.Second,
		3,
		1,
	)
	api.LoadBalancer.AddPool(p)

	router := gin.Default()
	router.GET("/pools/:hostname/stats", api.GetPoolStats)

	w := performRequest(router, "GET", "/pools/"+base64.RawURLEncoding.EncodeToString([]byte("test"))+"/stats", nil)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestUpdatePool_NotFound(t *testing.T) {
	log.Println("Executing ", t.Name())
	api := setupTestServer()
	router := gin.Default()
	router.POST("/pools/:hostname", api.UpdatePool)

	body := []byte(`{"hostname":"missing","health_check_interval":2}`)
	w := performRequest(router, "POST", "/pools/missing", body)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestUpdatePool_Success(t *testing.T) {
	log.Println("Executing ", t.Name())
	api := setupTestServer()
	p := loadbalancer.NewPool("test",
		5*time.Second,
		10*time.Second,
		2*time.Second,
		3,
		1,
	)
	// valori iniziali
	p.HealthCheckInterval.Store(uint64(1 * time.Second))
	api.LoadBalancer.AddPool(p)

	router := gin.Default()
	router.POST("/pools/:hostname", api.UpdatePool)

	body := []byte(`{"hostname":"test","health_check_interval":3}`)
	w := performRequest(router, "POST", "/pools/"+base64.RawURLEncoding.EncodeToString([]byte("test")), body)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, uint64(3), p.HealthCheckInterval.Load()/uint64(time.Second))
}

func TestAddServer_PoolNotFound(t *testing.T) {
	log.Println("Executing ", t.Name())
	api := setupTestServer()
	router := gin.Default()
	router.POST("/pools/:hostname/server", api.AddServer)

	body := []byte(`{"pool":"noexist","new_server_address":"127.0.0.1", "health_check_path":"/check"}`)
	w := performRequest(router, "POST", "/pools/noexist/server", body)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestAddServer_BadRequest(t *testing.T) {
	log.Println("Executing ", t.Name())
	api := setupTestServer()
	router := gin.Default()
	router.POST("/pools/:hostname/server", api.AddServer)

	w := performRequest(router, "POST", "/pools/any/server", []byte(`{}`))
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAddServer_Success(t *testing.T) {
	log.Println("Executing ", t.Name())
	api := setupTestServer()
	p := loadbalancer.NewPool("test",
		5*time.Second,
		10*time.Second,
		2*time.Second,
		3,
		1,
	)
	api.LoadBalancer.AddPool(p)

	router := gin.Default()
	router.POST("/pools/:hostname/server", api.AddServer)

	body := []byte(`{"new_server_address":"127.0.0.1","health_check_path":"/check"}`)
	w := performRequest(router, "POST", "/pools/"+base64.RawURLEncoding.EncodeToString([]byte("test"))+"/server", body)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRemoveServer_NotFoundPool(t *testing.T) {
	log.Println("Executing ", t.Name())
	api := setupTestServer()
	router := gin.Default()
	router.DELETE("/pools/:hostname/:server", api.RemoveServer)

	w := performRequest(router, "DELETE", "/pools/"+base64.RawURLEncoding.EncodeToString([]byte("nopool"))+"/00000000-0000-0000-0000-000000000000", nil)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestRemoveServer_BadUUID(t *testing.T) {
	log.Println("Executing ", t.Name())
	api := setupTestServer()
	p := loadbalancer.NewPool("test",
		5*time.Second,
		10*time.Second,
		2*time.Second,
		3,
		1,
	)
	api.LoadBalancer.Pools[p.Hostname] = p

	router := gin.Default()
	router.DELETE("/pools/:hostname/:server", api.RemoveServer)

	w := performRequest(router, "DELETE", "/pools/"+base64.RawURLEncoding.EncodeToString([]byte("test"))+"/not-a-uuid", nil)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestRemoveServer_Success(t *testing.T) {
	log.Println("Executing ", t.Name())
	api := setupTestServer()
	p := loadbalancer.NewPool("test",
		5*time.Second,
		10*time.Second,
		2*time.Second,
		3,
		1,
	)
	api.LoadBalancer.AddPool(p)

	addRouter := gin.Default()
	addRouter.POST("/pools/:hostname/server", api.AddServer)
	addBody := []byte(`{"new_server_address":"127.0.0.1","health_check_path":"/check"}`)
	wAdd := performRequest(addRouter, "POST", "/pools/"+base64.RawURLEncoding.EncodeToString([]byte("test"))+"/server", addBody)
	assert.Equal(t, http.StatusOK, wAdd.Code)

	server := p.UnconditionalServers[0]

	addRouter.DELETE("/pools/:hostname/:server", api.RemoveServer)
	wRem := performRequest(addRouter, "DELETE", "/pools/"+base64.RawURLEncoding.EncodeToString([]byte("test"))+"/"+server.Id.String(), nil)
	assert.Equal(t, http.StatusOK, wRem.Code)
}

func TestAddTransaction_PoolNotFound(t *testing.T) {
	log.Println("Executing ", t.Name())
	api := setupTestServer()
	router := gin.Default()
	router.POST("/pools/:hostname/transaction", api.AddTransaction)

	body := []byte(`{"old_server_id":"00000000-0000-0000-0000-000000000000","new_server_address":"127.0.0.1","new_server_health_check_path":"/check"}`)
	w := performRequest(router, "POST", "/pools/"+base64.RawURLEncoding.EncodeToString([]byte("ghost"))+"/transaction", body)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestAddTransaction_BadRequest(t *testing.T) {
	log.Println("Executing ", t.Name())
	api := setupTestServer()
	router := gin.Default()
	router.POST("/pools/:hostname/transaction", api.AddTransaction)

	w := performRequest(router, "POST", "/pools/any/transaction", []byte(`{}`))
	assert.Equal(t, http.StatusBadRequest, w.Code)
}
