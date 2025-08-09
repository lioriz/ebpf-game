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
	pidManager *PIDManager
	cmdCh      chan MonitorCommand
	app        *ReadWriteMonitorApp
	router     *gin.Engine
	port       string
}

// NewAPIServer creates a new API server instance
func NewAPIServer(port string, cmdCh chan MonitorCommand, app *ReadWriteMonitorApp) *APIServer {
	pidManager := NewPIDManager()
	router := gin.Default()

	server := &APIServer{
		pidManager: pidManager,
		cmdCh:      cmdCh,
		app:        app,
		router:     router,
		port:       port,
	}

	server.setupRoutes()
	return server
}

// setupRoutes configures all API routes
func (as *APIServer) setupRoutes() {
	// GET - List all available APIs
	as.router.GET("/apis", as.getAvailableAPIs)

	// POST - Add PIDs to the target list and set print_all to false
	as.router.POST("/add_pid", as.addPID)

	// POST - Clear PID list and set print_all to false
	as.router.POST("/clear_pid_list", as.clearPIDList)

	// POST - Set print_all flag to true
	as.router.POST("/set_print_all", as.setPrintAll)

	// GET - Get current target PIDs and print_all flag state
	as.router.GET("/target_pids", as.getTargetPIDs)
}

// getAvailableAPIs returns all available API endpoints
func (as *APIServer) getAvailableAPIs(c *gin.Context) {
	apis := map[string]interface{}{
		"available_apis": []string{
			"GET /apis - Get all available APIs",
			"POST /add_pid - Add PID to target list (sets print_all to false)",
			"POST /clear_pid_list - Clear all target PIDs (sets print_all to false)",
			"POST /set_print_all - Set print_all flag to true (monitor all PIDs except own)",
			"GET /target_pids - Get current target PIDs and print_all flag state",
		},
		"usage": map[string]interface{}{
			"add_pid": map[string]interface{}{
				"method": "POST",
				"body":   `{"pid": 1234}`,
			},
			"clear_pid_list": map[string]interface{}{
				"method": "POST",
				"body":   `{} (optional - can be omitted)`,
			},
			"set_print_all": map[string]interface{}{
				"method": "POST",
				"body":   `{} (optional - can be omitted)`,
			},
		},
	}

	c.JSON(http.StatusOK, apis)
}

// addPID handles adding a PID to the target list
func (as *APIServer) addPID(c *gin.Context) {
	var request struct {
		PID uint32 `json:"pid"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON format. Expected: {\"pid\": 1234}"})
		return
	}

	fmt.Printf("Received request: POST /add_pid {pid: %d}\n", request.PID)

	// Add PID to local manager
	as.pidManager.AddPIDs([]uint32{request.PID})

	// Enqueue add pid
	select {
	case as.cmdCh <- MonitorCommand{Kind: CommandAddPID, PID: request.PID}:
	default:
		fmt.Printf("Warning: command queue full, dropping AddPID(%d)\n", request.PID)
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "PID enqueued for add; print_all set to false",
		"added_pid": request.PID,
		"total_pids": len(as.pidManager.GetAllPIDs()),
		"print_all": false,
	})
}

// clearPIDList handles clearing the PID list
func (as *APIServer) clearPIDList(c *gin.Context) {
	fmt.Printf("Received request: POST /clear_pid_list\n")

	// Clear PIDs from local manager
	as.pidManager.ClearPIDList()

	// Enqueue clear
	select {
	case as.cmdCh <- MonitorCommand{Kind: CommandClearPIDs}:
	default:
		fmt.Printf("Warning: command queue full, dropping ClearPIDs\n")
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "PID list enqueued for clear; print_all set to false",
		"total_pids": 0,
		"print_all": false,
	})
}

// setPrintAll handles setting the print_all flag to true
func (as *APIServer) setPrintAll(c *gin.Context) {
	fmt.Printf("Received request: POST /set_print_all\n")

	// Enqueue set_print_all
	select {
	case as.cmdCh <- MonitorCommand{Kind: CommandSetPrintAll, PrintAll: true}:
	default:
		fmt.Printf("Warning: command queue full, dropping SetPrintAll(true)\n")
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Print all flag enqueued to true successfully",
		"print_all": true,
	})
}

// getTargetPIDs returns current target PIDs and print_all flag state
func (as *APIServer) getTargetPIDs(c *gin.Context) {
	// Use the app (which queries EBpfMonitor)
	pids, err := as.app.GetTargetPIDs()
	if err != nil {
		fmt.Printf("Warning: Failed to get target PIDs: %v\n", err)
		pids = []uint32{}
	}

	printAll, err := as.app.GetPrintAllState()
	if err != nil {
		fmt.Printf("Warning: Failed to get print_all state: %v\n", err)
		printAll = false
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Target PIDs and print_all flag state retrieved successfully",
		"pids": pids,
		"total_pids": len(pids),
		"print_all": printAll,
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