package main

import (
	"continuity/common"
	"continuity/common/requests"
	"log"

	"github.com/spf13/cobra"
)

var poolName string
var serverAddress string
var serverUUID string
var serverPort int
var healthCheckPath string
var serverCondition string

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Manage pool servers",
}

var addServerCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a server to a pool",
	Run: func(cmd *cobra.Command, args []string) {
		condition, err := common.ParseCondition(serverCondition)
		if err != nil {
			log.Fatalf("Invalid condition: %v", err)
		}

		c.AddServer(poolName, requests.AddServerRequest{
			NewServerAddress: serverAddress,
			HealthCheckPath:  healthCheckPath,
			Condition:        condition,
		})
	},
}
var removeServerCmd = &cobra.Command{
	Use:   "del",
	Short: "Remove a server from a pool",
	Run: func(cmd *cobra.Command, args []string) {

		c.RemoveServer(poolName, serverUUID)
	},
}
var transactionCmd = &cobra.Command{
	Use:   "transaction",
	Short: "Add a server and remove another server transactionally",
	Run: func(cmd *cobra.Command, args []string) {
		condition, err := common.ParseCondition(serverCondition)
		if err != nil {
			log.Fatalf("Invalid condition: %v", err)
		}

		c.Transaction(poolName, requests.TransactionRequest{
			NewServerAddress:         serverAddress,
			NewServerHealthCheckPath: healthCheckPath,
			NewServerCondition:       condition,
			OldServerId:              serverUUID,
		})
	},
}

func init() {
	serverCmd.AddCommand(addServerCmd)
	serverCmd.AddCommand(removeServerCmd)
	serverCmd.AddCommand(transactionCmd)

	addServerCmd.Flags().StringVarP(&poolName, "pool", "p", "", "Name of the pool")
	addServerCmd.Flags().StringVarP(&serverAddress, "address", "a", "", "Address of the server to add. Must include protocol (http:// or https://)")
	addServerCmd.Flags().StringVarP(&healthCheckPath, "health-check", "c", "/health", "Health check path for the server")
	addServerCmd.Flags().StringVarP(&serverCondition, "condition", "", "", "Condition for adding the server in the format header=value")
	_ = addPoolCmd.MarkFlagRequired("pool")
	_ = addPoolCmd.MarkFlagRequired("address")

	removeServerCmd.Flags().StringVarP(&poolName, "pool", "", "", "Name of the pool")
	removeServerCmd.Flags().StringVarP(&serverUUID, "server", "s", "", "UUID of the server to remove")
	_ = removeServerCmd.MarkFlagRequired("pool")
	_ = removeServerCmd.MarkFlagRequired("server")

	transactionCmd.Flags().StringVarP(&poolName, "pool", "p", "", "Name of the pool")
	transactionCmd.Flags().StringVarP(&serverAddress, "address", "a", "", "Address of the server to add. Must include protocol (http:// or https://)")
	transactionCmd.Flags().StringVarP(&healthCheckPath, "health-check", "c", "/health", "Health check path for the server to add")
	transactionCmd.Flags().StringVarP(&serverUUID, "remove-server", "r", "", "UUID of the server to remove")
	_ = transactionCmd.MarkFlagRequired("pool")
	_ = transactionCmd.MarkFlagRequired("address")
}
