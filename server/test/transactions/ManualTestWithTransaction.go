package main

import (
	"continuity/server/loadbalancer"
	"log"
	"os"
	"time"
)

func main() {
	lb, err := loadbalancer.NewLoadBalancer("test", "0.0.0.0", 8080)
	log.Println("Creating load balancer")
	if err != nil {
		log.Println("Error creating load balancer:", err)
		os.Exit(1)
	}
	lb.AddPool(loadbalancer.NewPoolWithStickySession("localhost:8080",
		1*time.Second,
		1*time.Second,
		5*time.Second,
		1*time.Hour,
		3,
		3,
	))
	log.Println("Pool added to load balancer")
	pool, err := lb.GetPool("localhost:8080")
	if err != nil {
		log.Println("Error getting pool:", err)
		os.Exit(1)
	}
	server1, err := loadbalancer.NewServerHost("http://127.0.0.1:8081", "/check", loadbalancer.Condition{})
	if err != nil {
		log.Println("Error creating server host:", err)
		os.Exit(1)
	}
	server2, err := loadbalancer.NewServerHost("http://127.0.0.1:8082", "/check", loadbalancer.Condition{})
	if err != nil {
		log.Println("Error creating server host:", err)
		os.Exit(1)
	}
	server3, err := loadbalancer.NewServerHost("http://127.0.0.1:8083", "/check", loadbalancer.Condition{})
	if err != nil {
		log.Println("Error creating server host:", err)
		os.Exit(1)
	}
	log.Println("Adding servers")
	pool.AddServer(server1)
	pool.AddServer(server2)
	pool.AddServer(server3)
	log.Println("Load balancer ready")
	time.Sleep(20 * time.Second)
	log.Println("Removing server 8081 in 2 seconds")
	time.Sleep(2 * time.Second)
	_, err = pool.RemoveServer(server1.Id)
	log.Println("Removed. Removing server 8082 in 10 seconds")
	time.Sleep(10 * time.Second)
	_, err = pool.RemoveServer(server2.Id)
	log.Println("Removed. Continuing only with 8083...")
	log.Println("Adding fake server 8084 with transaction (should fail and don't remove 8083")
	server4, err := loadbalancer.NewServerHost("http://127.0.0.1:8084", "/check", loadbalancer.Condition{})
	if err != nil {
		log.Println("Error creating server host:", err)
		os.Exit(1)
	}
	err = pool.Transaction(server4, server3.Id)
	if err != nil {
		log.Println("(Expected) Error creating transaction:", err)
	} else {
		log.Println("Transaction succeeded unexpectedly")
		os.Exit(2)
	}
	log.Println("Adding back server 8081 and waiting 10 seconds")
	pool.AddServer(server1)
	time.Sleep(10 * time.Second)
	log.Println("Adding real server 8082 with transaction (should  remove 8083)")
	err = pool.Transaction(server2, server3.Id)
	if err != nil {
		log.Println("Error creating transaction:", err)
		os.Exit(1)
	} else {
		log.Println("Transaction succeeded as expected")
	}
	log.Println("Servers in pool:")
	for _, s := range pool.UnconditionalServers {
		log.Println("Server:", s.Address.String(), "Status:", s.ServerStatus.String())
	}
	for {
		time.Sleep(10 * time.Second)
		stats := pool.GetStats()
		log.Println("Current Pool Stats:")
		for addr, stat := range stats {
			log.Printf("Server: %s - OK Responses: %d - Not OK Responses: %d\n", addr, stat.OkResponses, stat.NotOkResponses)
		}
	}
}
