package main

import (
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
	logger    Logger
	pidManager *PIDManager
	cmdCh      chan MonitorCommand
	ebpfController        *EBpfController
	router     *gin.Engine
	port       string
}

// NewAPIServer creates a new API server instance
func NewAPIServer(port string, logger Logger, cmdCh chan MonitorCommand, ebpfController *EBpfController) *APIServer {
	pidManager := NewPIDManager()
	router := gin.Default()

	server := &APIServer{
		logger:    logger,
		pidManager: pidManager,
		cmdCh:      cmdCh,
		ebpfController:        ebpfController,
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
	as.router.POST("/add_pids", as.addPIDs)

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
			"POST /add_pids - Add PIDs to target list (sets print_all to false)",
			"POST /clear_pid_list - Clear all target PIDs (sets print_all to false)",
			"POST /set_print_all - Set print_all flag to true (monitor all PIDs except own)",
			"GET /target_pids - Get current target PIDs and print_all flag state",
		},
		"usage": map[string]interface{}{
			"add_pids": map[string]interface{}{
				"method": "POST",
				"body":   `{"pids": [1234, 5678]}`,
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

// addPIDs handles adding PIDs to the target list
func (as *APIServer) addPIDs(c *gin.Context) {
	var request struct {
		PIDs []uint32 `json:"pids"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON format. Expected: {\"pids\": [1234, 5678]}"})
		return
	}
	if len(request.PIDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Field 'pids' must contain at least one PID"})
		return
	}

	as.logger.Infof("Received request: POST /add_pids {pids: %v}", request.PIDs)

	// Add PIDs to local manager
	as.pidManager.AddPIDs(request.PIDs)

	// Enqueue each PID
	for _, pid := range request.PIDs {
		select {
		case as.cmdCh <- MonitorCommand{Kind: CommandAddPID, PID: pid}:
		default:
			as.logger.Warnf("command queue full, dropping AddPID(%d)", pid)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "PIDs enqueued for add; print_all set to false",
		"added_pids": request.PIDs,
		"total_pids": len(as.pidManager.GetAllPIDs()),
		"print_all": false,
	})
}

// clearPIDList handles clearing the PID list
func (as *APIServer) clearPIDList(c *gin.Context) {
	as.logger.Infof("Received request: POST /clear_pid_list")

	// Clear PIDs from local manager
	as.pidManager.ClearPIDList()

	// Enqueue clear
	select {
	case as.cmdCh <- MonitorCommand{Kind: CommandClearPIDs}:
	default:
		as.logger.Warnf("command queue full, dropping ClearPIDs")
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "PID list enqueued for clear; print_all set to false",
		"total_pids": 0,
		"print_all": false,
	})
}

// setPrintAll handles setting the print_all flag to true
func (as *APIServer) setPrintAll(c *gin.Context) {
	as.logger.Infof("Received request: POST /set_print_all")

	// Enqueue set_print_all
	select {
	case as.cmdCh <- MonitorCommand{Kind: CommandSetPrintAll, PrintAll: true}:
	default:
		as.logger.Warnf("command queue full, dropping SetPrintAll(true)")
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Print all flag enqueued to true successfully",
		"print_all": true,
	})
}

// getTargetPIDs returns current target PIDs and print_all flag state
func (as *APIServer) getTargetPIDs(c *gin.Context) {
	// Use the ebpfController (which queries EBpfProbe)
	pids, err := as.ebpfController.GetTargetPIDs()
	if err != nil {
		as.logger.Warnf("Failed to get target PIDs: %v", err)
		pids = []uint32{}
	}

	printAll, err := as.ebpfController.GetPrintAllState()
	if err != nil {
		as.logger.Warnf("Failed to get print_all state: %v", err)
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
	as.logger.Infof("API Server starting on localhost:%s", as.port)
	return as.router.Run(":" + as.port)
}

// GetRouter returns the router for testing purposes
func (as *APIServer) GetRouter() *gin.Engine {
	return as.router
} 