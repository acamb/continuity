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
	pool, err := loadbalancer.NewPoolWithStickySessionCustomCookie("localhost:8080",
		1*time.Second,
		1*time.Second,
		5*time.Second,
		1*time.Hour,
		3,
		3,
		"MY_CUSTOM_COOKIE",
	)
	lb.AddPool(pool)
	if err != nil {
		log.Println("Error creating pool with sticky session and custom cookie:", err)
		os.Exit(1)
	}
	log.Println("Pool added to load balancer")
	pool, err = lb.GetPool("localhost:8080")
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

	for {
		time.Sleep(10 * time.Second)
		stats := pool.GetStats()
		log.Println("Current Pool Stats:")
		for addr, stat := range stats {
			log.Printf("Server: %s - OK Responses: %d - Not OK Responses: %d\n", addr, stat.OkResponses, stat.NotOkResponses)
		}
	}
}
