package main

import (
	"fmt"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
)

// PIDManager handles PID list operations
type PIDManager struct {
	pids []uint32
	mu   sync.RWMutex
}

// NewPIDManager creates a new PID manager instance
func NewPIDManager() *PIDManager {
	return &PIDManager{
		pids: make([]uint32, 0),
	}
}

// AddPIDs adds PIDs to the list
func (pm *PIDManager) AddPIDs(newPIDs []uint32) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.pids = append(pm.pids, newPIDs...)
}

// ClearPIDList clears all PIDs
func (pm *PIDManager) ClearPIDList() {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.pids = make([]uint32, 0)
}

// GetAllPIDs returns all PIDs
func (pm *PIDManager) GetAllPIDs() []uint32 {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	return append([]uint32{}, pm.pids...)
}

// APIServer handles HTTP API requests
type APIServer struct {
	pidManager  *PIDManager
	ebpfMonitor *EBpfMonitor
	router      *gin.Engine
	port        string
}

// NewAPIServer creates a new API server instance
func NewAPIServer(port string, ebpfMonitor *EBpfMonitor) *APIServer {
	pidManager := NewPIDManager()
	router := gin.Default()
	
	server := &APIServer{
		pidManager:  pidManager,
		ebpfMonitor: ebpfMonitor,
		router:      router,
		port:        port,
	}
	
	server.setupRoutes()
	return server
}

// setupRoutes configures all API routes
func (as *APIServer) setupRoutes() {
	// GET - List all available APIs
	as.router.GET("/apis", as.getAvailableAPIs)
	
	// POST - Add PIDs to the target list
	as.router.POST("/add_pids", as.addPIDs)
	
	// POST - Clear PID list
	as.router.POST("/clear_pid_list", as.clearPIDList)
	
	// POST - Print all PIDs
	as.router.POST("/print_all_pids", as.printAllPIDs)
	
	// POST - Set print_all flag
	as.router.POST("/set_print_all", as.setPrintAll)
	
	// GET - Get current target PIDs
	as.router.GET("/target_pids", as.getTargetPIDs)
}

// getAvailableAPIs returns all available API endpoints
func (as *APIServer) getAvailableAPIs(c *gin.Context) {
	apis := map[string]interface{}{
		"available_apis": []string{
			"GET /apis - Get all available APIs",
			"POST /add_pids - Add PIDs to the target list",
			"POST /clear_pid_list - Clear all target PIDs",
			"POST /print_all_pids - Get all target PIDs",
			"POST /set_print_all - Set print_all flag (monitor all PIDs except own)",
			"GET /target_pids - Get current target PIDs",
		},
		"usage": map[string]interface{}{
			"add_pids": map[string]interface{}{
				"method": "POST",
				"body":   `{"add_pids": [1234, 5678]}`,
			},
			"clear_pid_list": map[string]interface{}{
				"method": "POST",
				"body":   `{"clear_pid_list": true}`,
			},
			"print_all_pids": map[string]interface{}{
				"method": "POST",
				"body":   `{"print_all_pids": true}`,
			},
			"set_print_all": map[string]interface{}{
				"method": "POST",
				"body":   `{"print_all": true}`,
			},
		},
	}
	
	c.JSON(http.StatusOK, apis)
}

// addPIDs handles adding PIDs to the target list
func (as *APIServer) addPIDs(c *gin.Context) {
	var request struct {
		AddPIDs []uint32 `json:"add_pids"`
	}
	
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON format"})
		return
	}
	
	fmt.Printf("Received request: POST {add_pids: %v}\n", request.AddPIDs)
	
	// Add PIDs to both local manager and eBPF map
	as.pidManager.AddPIDs(request.AddPIDs)
	
	for _, pid := range request.AddPIDs {
		if err := as.ebpfMonitor.AddTargetPID(pid); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to add PID %d to eBPF map: %v", pid, err)})
			return
		}
	}
	
	c.JSON(http.StatusOK, gin.H{
		"message": "PIDs added successfully",
		"added_pids": request.AddPIDs,
		"total_pids": len(as.pidManager.GetAllPIDs()),
	})
}

// clearPIDList handles clearing the PID list
func (as *APIServer) clearPIDList(c *gin.Context) {
	var request struct {
		ClearPIDList bool `json:"clear_pid_list"`
	}
	
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON format"})
		return
	}
	
	fmt.Printf("Received request: POST {clear_pid_list: %v}\n", request.ClearPIDList)
	
	// Clear PIDs from both local manager and eBPF map
	as.pidManager.ClearPIDList()
	
	if err := as.ebpfMonitor.ClearTargetPIDs(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to clear PIDs from eBPF map: %v", err)})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"message": "PID list cleared successfully",
		"total_pids": 0,
	})
}

// printAllPIDs handles printing all PIDs
func (as *APIServer) printAllPIDs(c *gin.Context) {
	var request struct {
		PrintAllPIDs bool `json:"print_all_pids"`
	}
	
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON format"})
		return
	}
	
	fmt.Printf("Received request: POST {print_all_pids: %v}\n", request.PrintAllPIDs)
	
	allPIDs := as.pidManager.GetAllPIDs()
	
	c.JSON(http.StatusOK, gin.H{
		"message": "All PIDs retrieved successfully",
		"pids": allPIDs,
		"total_pids": len(allPIDs),
	})
}

// setPrintAll handles setting the print_all flag
func (as *APIServer) setPrintAll(c *gin.Context) {
	var request struct {
		PrintAll bool `json:"print_all"`
	}
	
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON format"})
		return
	}
	
	fmt.Printf("Received request: POST {print_all: %v}\n", request.PrintAll)
	
	// Set the print_all flag in eBPF map
	if err := as.ebpfMonitor.SetPrintAll(request.PrintAll); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to set print_all flag: %v", err)})
		return
	}
	
	status := "disabled"
	if request.PrintAll {
		status = "enabled"
	}
	
	c.JSON(http.StatusOK, gin.H{
		"message": fmt.Sprintf("Print all flag %s successfully", status),
		"print_all": request.PrintAll,
	})
}

// getTargetPIDs returns current target PIDs
func (as *APIServer) getTargetPIDs(c *gin.Context) {
	pids, err := as.ebpfMonitor.GetTargetPIDs()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to get target PIDs: %v", err)})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"message": "Target PIDs retrieved successfully",
		"pids": pids,
		"total_pids": len(pids),
	})
}

// Start starts the API server
func (as *APIServer) Start() error {
	fmt.Printf("API Server starting on localhost:%s\n", as.port)
	return as.router.Run(":" + as.port)
}

// GetRouter returns the router for testing purposes
func (as *APIServer) GetRouter() *gin.Engine {
	return as.router
} 