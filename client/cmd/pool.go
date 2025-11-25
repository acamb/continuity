package main

import (
	"continuity/common/requests"

	"github.com/spf13/cobra"
)

var hostname string
var healthCheckInterval *int64
var healthCheckInitialDelay *int64
var healthCheckTimeout *int64
var healthCheckNumOk *uint32
var healthCheckNumFail *uint32
var stickySessions bool
var stickyMethod string
var StickySessionTimeout int64
var cookieName string
var healthCheckIntervalUpdate *int64
var healthCheckInitialDelayUpdate *int64
var healthCheckTimeoutUpdate *int64
var healthCheckNumOkUpdate *uint32
var healthCheckNumFailUpdate *uint32
var printJson bool
var poolCmd = &cobra.Command{
	Use:   "pool",
	Short: "Manage load balancer pools",
}

var addPoolCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a new pool to the load balancer",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		hostname = args[0]
		c.AddPool(requests.CreatePoolRequest{
			Hostname:                hostname,
			HealthCheckInterval:     *healthCheckInterval,
			HealthCheckInitialDelay: *healthCheckInitialDelay,
			HealthCheckTimeout:      *healthCheckTimeout,
			HealthCheck_numOk:       *healthCheckNumOk,
			HealthCheck_numFail:     *healthCheckNumFail,
			StickySessions:          stickySessions,
			StickyMethod:            stickyMethod,
			StickySessionTimeout:    StickySessionTimeout,
			StickySessionCookieName: cookieName,
		})
	},
}

var removePoolCmd = &cobra.Command{
	Use:   "del POOL_NAME",
	Short: "Remove a pool from the load balancer",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		hostname = args[0]

		c.RemovePool(hostname)
	},
}

var listPoolsCmd = &cobra.Command{
	Use:   "list",
	Short: "List all pools in the load balancer",
	Run: func(cmd *cobra.Command, args []string) {

		c.ListPools()
	},
}

var poolConfigCmd = &cobra.Command{
	Use:   "config POOL_NAME",
	Short: "Get configuration of a specific pool",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		hostname = args[0]

		c.GetPoolConfig(hostname, printJson)
	},
}

var poolStatsCmd = &cobra.Command{
	Use:   "stats POOL_NAME",
	Short: "Get statistics of a specific pool",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		hostname = args[0]

		c.GetPoolStats(hostname, printJson)
	},
}

var updatePoolCmd = &cobra.Command{
	Use:   "update POOL_NAME",
	Short: "Update configuration of a specific pool",
	Run: func(cmd *cobra.Command, args []string) {
		hostname = args[0]

		c.UpdatePool(requests.UpdatePoolRequest{
			Hostname:                hostname,
			HealthCheckInterval:     *healthCheckIntervalUpdate,
			HealthCheckInitialDelay: *healthCheckInitialDelayUpdate,
			HealthCheckTimeout:      *healthCheckTimeoutUpdate,
			HealthCheck_numOk:       *healthCheckNumOkUpdate,
			HealthCheck_numFail:     *healthCheckNumFailUpdate,
		})
	},
}

func init() {
	poolCmd.AddCommand(addPoolCmd)
	poolCmd.AddCommand(removePoolCmd)
	poolCmd.AddCommand(listPoolsCmd)
	poolCmd.AddCommand(poolConfigCmd)
	poolCmd.AddCommand(poolStatsCmd)
	poolCmd.AddCommand(updatePoolCmd)
	poolConfigCmd.Flags().BoolVarP(&printJson, "json", "j", false, "Print output in JSON format")
	poolStatsCmd.Flags().BoolVarP(&printJson, "json", "j", false, "Print output in JSON format")

	healthCheckInterval = addPoolCmd.Flags().Int64P("health-check-interval", "i", 10, "Health check interval in seconds")
	healthCheckInitialDelay = addPoolCmd.Flags().Int64P("health-check-initial-delay", "d", 20, "Health check initial delay in seconds")
	healthCheckNumOk = addPoolCmd.Flags().Uint32P("health-ok", "", 3, "Number of consecutive OK responses required to mark a server healthy")
	healthCheckNumFail = addPoolCmd.Flags().Uint32P("health-fail", "", 3, "Number of consecutive failed responses required to mark a server unhealthy")
	healthCheckTimeout = addPoolCmd.Flags().Int64P("health-check-timeout", "t", 5, "Health check timeout in seconds")
	addPoolCmd.Flags().BoolVarP(&stickySessions, "sticky-sessions", "s", false, "Enable sticky sessions")
	addPoolCmd.Flags().StringVarP(&stickyMethod, "sticky-method", "", "LBCookie", "Sticky session method (IP, AppCookie, LBCookie)")
	addPoolCmd.Flags().StringVarP(&cookieName, "cookie-name", "", "", "Cookie name for AppCookie sticky method")

	healthCheckIntervalUpdate = updatePoolCmd.Flags().Int64P("health-check-interval", "i", 10, "Health check interval in seconds")
	healthCheckInitialDelayUpdate = updatePoolCmd.Flags().Int64P("health-check-initial-delay", "d", 20, "Health check initial delay in seconds")
	healthCheckNumOkUpdate = updatePoolCmd.Flags().Uint32P("health-ok", "", 3, "Number of consecutive OK responses required to mark a server healthy")
	healthCheckNumFailUpdate = updatePoolCmd.Flags().Uint32P("health-fail", "", 3, "Number of consecutive failed responses required to mark a server unhealthy")
	healthCheckTimeoutUpdate = updatePoolCmd.Flags().Int64P("health-check-timeout", "t", 5, "Health check timeout in seconds")
}
