package api

import (
	"continuity/common/requests"
	"continuity/common/responses"
	"continuity/server/loadbalancer"
	"continuity/server/version"
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type saveConfigFunc func(server *ApiServer)
type ApiServer struct {
	Address           string
	Port              int
	LoadBalancer      *loadbalancer.LoadBalancer
	saveConfig        chan bool
	transactions      map[uuid.UUID]*Transaction
	transactionsMutex sync.RWMutex
}

type Transaction struct {
	OldServerId uuid.UUID
	NewServer   *loadbalancer.ServerHost
	Completed   bool
	Error       error
	CreatedAdt  time.Time
	CompletedAt time.Time
}

func NewApiServer(address string,
	port int,
	loadBalancer *loadbalancer.LoadBalancer,
	saveChannel chan bool,
) *ApiServer {
	return &ApiServer{
		Address:           address,
		Port:              port,
		LoadBalancer:      loadBalancer,
		saveConfig:        saveChannel,
		transactions:      make(map[uuid.UUID]*Transaction),
		transactionsMutex: sync.RWMutex{},
	}
}

func (api *ApiServer) Start() {
	router := gin.Default()

	// Define API routes
	router.GET("/version", api.GetVersion)
	router.GET("/pools", api.GetPools)
	router.POST("/pools", api.CreatePool)
	router.DELETE("/pools/:hostname", api.DeletePool)
	router.GET("/pools/:hostname", api.GetPoolConfig)
	router.GET("/pools/:hostname/stats", api.GetPoolStats)
	router.POST("/pools/:hostname", api.UpdatePool)
	router.POST("/pools/:hostname/server", api.AddServer)
	router.DELETE("/pools/:hostname/:server", api.RemoveServer)
	router.POST("/pools/:hostname/transaction", api.AddTransaction)
	router.GET("/pools/transaction/:transaction", api.GetTransaction)

	addr := api.Address + ":" + fmt.Sprint(api.Port)
	log.Println("Starting API server on", addr)
	if err := router.Run(addr); err != nil {
		log.Fatal("Failed to start API server:", err)
	}

}

func (api *ApiServer) GetVersion(context *gin.Context) {
	context.JSON(http.StatusOK, responses.VersionResponse{
		Version: version.Version,
	})
}

func (api *ApiServer) GetPools(context *gin.Context) {
	pools := api.LoadBalancer.GetPools()
	responsePools := make([]string, 0, len(pools))
	for i := range pools {
		responsePools = append(responsePools, pools[i].Hostname)
	}
	context.JSON(http.StatusOK, responses.ListPoolResponse{Pools: responsePools})

}

func (api *ApiServer) CreatePool(context *gin.Context) {
	var req requests.CreatePoolRequest
	var pool *loadbalancer.Pool
	err := context.ShouldBindJSON(&req)
	if err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	pool, err = req.Validate()
	if err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	err = api.LoadBalancer.AddPool(pool)
	if err != nil {
		context.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	api.saveConfig <- true
}

func (api *ApiServer) DeletePool(context *gin.Context) {
	hostname, err := base64.RawURLEncoding.DecodeString(context.Param(("hostname")))
	if err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": "invalid hostname encoding"})
		return
	}
	err = api.LoadBalancer.RemovePool(string(hostname))
	if err != nil {
		context.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	api.saveConfig <- true
}

func (api *ApiServer) GetPoolConfig(context *gin.Context) {
	hostname, err := base64.RawURLEncoding.DecodeString(context.Param(("hostname")))
	if err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": "invalid hostname encoding"})
		return
	}
	log.Println("Getting config for pool:", hostname)
	pool, err := api.LoadBalancer.GetPool(string(hostname))
	if err != nil {
		context.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	context.JSON(http.StatusOK, responses.NewPoolResponse(pool))
}

func (api *ApiServer) GetPoolStats(context *gin.Context) {
	hostname, err := base64.RawURLEncoding.DecodeString(context.Param(("hostname")))
	if err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": "invalid hostname encoding"})
		return
	}
	pool, err := api.LoadBalancer.GetPool(string(hostname))
	if err != nil {
		context.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	context.JSON(http.StatusOK, responses.PoolStatsResponse{
		Stats: pool.GetStats(),
	})
}

func (api *ApiServer) UpdatePool(context *gin.Context) {
	var req requests.UpdatePoolRequest
	err := context.ShouldBindJSON(&req)
	if err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	hostname, err := base64.RawURLEncoding.DecodeString(context.Param(("hostname")))
	if err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": "invalid hostname encoding"})
		return
	}
	serverPool, err := api.LoadBalancer.GetPool(string(hostname))
	if err != nil {
		context.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	pool := &loadbalancer.Pool{
		Hostname: serverPool.Hostname,
	}
	pool.HealthCheck_numFail.Store(serverPool.HealthCheck_numFail.Load())
	pool.HealthCheck_numOk.Store(serverPool.HealthCheck_numOk.Load())
	pool.HealthCheckInterval.Store(serverPool.HealthCheckInterval.Load())
	pool.HealthCheckInitialDelay.Store(serverPool.HealthCheckInitialDelay.Load())
	pool.HealthCheckTimeout.Store(serverPool.HealthCheckTimeout.Load())

	if req.HealthCheck_numFail != 0 {
		pool.HealthCheck_numFail.Store(req.HealthCheck_numFail)
	}
	if req.HealthCheck_numOk != 0 {
		pool.HealthCheck_numOk.Store(req.HealthCheck_numOk)
	}
	if req.HealthCheckInterval != 0 {
		pool.HealthCheckInterval.Store(uint64(req.HealthCheckInterval * int64(time.Second)))
	}
	if req.HealthCheckInitialDelay != 0 {
		pool.HealthCheckInitialDelay.Store(uint64(req.HealthCheckInitialDelay * int64(time.Second)))
	}
	if req.HealthCheckTimeout != 0 {
		pool.HealthCheckTimeout.Store(uint64(req.HealthCheckTimeout * int64(time.Second)))

	}

	err = api.LoadBalancer.UpdatePool(pool)
	if err != nil {
		context.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	api.saveConfig <- true
}

func (api *ApiServer) AddServer(context *gin.Context) {
	var req requests.AddServerRequest
	err := context.ShouldBindJSON(&req)
	if err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	hostname, err := base64.RawURLEncoding.DecodeString(context.Param(("hostname")))
	if err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": "invalid hostname encoding"})
		return
	}
	pool, err := api.LoadBalancer.GetPool(string(hostname))
	if err != nil {
		context.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	server, err := req.Validate()
	if err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	pool.AddServer(server)
	api.saveConfig <- true
}

func (api *ApiServer) RemoveServer(context *gin.Context) {
	serverId := context.Param("server")
	hostname, err := base64.RawURLEncoding.DecodeString(context.Param(("hostname")))
	if err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": "invalid hostname encoding"})
		return
	}

	pool, err := api.LoadBalancer.GetPool(string(hostname))
	if err != nil {
		context.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	serverUUID, err := uuid.Parse(serverId)
	if err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": "invalid server ID"})
		return
	}
	_, err = pool.RemoveServer(serverUUID)
	if err != nil {
		context.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	api.saveConfig <- true
}

func (api *ApiServer) AddTransaction(context *gin.Context) {
	var req requests.TransactionRequest
	hostname, err := base64.RawURLEncoding.DecodeString(context.Param(("hostname")))
	if err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": "invalid hostname encoding"})
		return
	}
	err = context.ShouldBindJSON(&req)
	if err != nil {
		log.Printf("Error binding JSON: %v", err)
		context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	pool, err := api.LoadBalancer.GetPool(string(hostname))
	if err != nil {
		context.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	server, err := req.Validate()
	if err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	serverUUID, err := uuid.Parse(req.OldServerId)
	if err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": "invalid old server ID"})
		return
	}
	api.transactionsMutex.Lock()
	defer api.transactionsMutex.Unlock()
	txUUID := uuid.New()
	api.transactions[txUUID] = &Transaction{
		CreatedAdt:  time.Now(),
		OldServerId: serverUUID,
		NewServer:   server,
		Completed:   false,
		Error:       nil,
	}
	go func() {
		err = pool.Transaction(server, serverUUID)
		api.transactionsMutex.Lock()
		defer api.transactionsMutex.Unlock()
		tx := api.transactions[txUUID]
		tx.Completed = true
		tx.CompletedAt = time.Now()
		if err != nil {
			tx.Error = err
		}
		api.saveConfig <- true
	}()
	context.JSON(http.StatusOK, responses.TransactionResponse{
		TransactionId: txUUID.String(),
		Completed:     false,
	})
}

func (api *ApiServer) GetTransaction(context *gin.Context) {
	tx := context.Param("transaction")
	transactionUUID, err := uuid.Parse(tx)
	if err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": "invalid transaction ID"})
		return
	}
	api.transactionsMutex.RLock()
	defer api.transactionsMutex.RUnlock()
	transaction, ok := api.transactions[transactionUUID]
	if !ok {
		context.JSON(http.StatusNotFound, gin.H{"error": "transaction not found"})
		return
	}
	resp := responses.TransactionResponse{
		TransactionId: tx,
		Completed:     transaction.Completed,
		CompletedAt:   transaction.CompletedAt,
	}
	if transaction.Error != nil {
		resp.Error = transaction.Error.Error()
	}
	context.JSON(http.StatusOK, resp)
}
