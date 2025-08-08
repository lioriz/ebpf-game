package main

import (
	"fmt"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
)

// IPManager handles IP list operations
type IPManager struct {
	ips  []int
	mu   sync.RWMutex
	port string
}

// NewIPManager creates a new IP manager instance
func NewIPManager(port string) *IPManager {
	return &IPManager{
		ips:  make([]int, 0),
		port: port,
	}
}

// AddIPs adds IPs to the list
func (im *IPManager) AddIPs(newIPs []int) {
	im.mu.Lock()
	defer im.mu.Unlock()
	im.ips = append(im.ips, newIPs...)
}

// ClearIPList clears all IPs
func (im *IPManager) ClearIPList() {
	im.mu.Lock()
	defer im.mu.Unlock()
	im.ips = make([]int, 0)
}

// GetAllIPs returns all IPs
func (im *IPManager) GetAllIPs() []int {
	im.mu.RLock()
	defer im.mu.RUnlock()
	return append([]int{}, im.ips...)
}

// APIServer handles HTTP API requests
type APIServer struct {
	ipManager *IPManager
	router    *gin.Engine
	port      string
}

// NewAPIServer creates a new API server instance
func NewAPIServer(port string) *APIServer {
	ipManager := NewIPManager(port)
	router := gin.Default()
	
	server := &APIServer{
		ipManager: ipManager,
		router:    router,
		port:      port,
	}
	
	server.setupRoutes()
	return server
}

// setupRoutes configures all API routes
func (as *APIServer) setupRoutes() {
	// GET - List all available APIs
	as.router.GET("/apis", as.getAvailableAPIs)
	
	// POST - Add IPs to the list
	as.router.POST("/add_ips", as.addIPs)
	
	// POST - Clear IP list
	as.router.POST("/clear_ip_list", as.clearIPList)
	
	// POST - Print all IPs
	as.router.POST("/print_all_ips", as.printAllIPs)
}

// getAvailableAPIs returns all available API endpoints
func (as *APIServer) getAvailableAPIs(c *gin.Context) {
	apis := map[string]interface{}{
		"available_apis": []string{
			"GET /apis - Get all available APIs",
			"POST /add_ips - Add IPs to the list",
			"POST /clear_ip_list - Clear all IPs",
			"POST /print_all_ips - Print all IPs",
		},
		"usage": map[string]interface{}{
			"add_ips": map[string]interface{}{
				"method": "POST",
				"body":   `{"add_ips": [1,2,3]}`,
			},
			"clear_ip_list": map[string]interface{}{
				"method": "POST",
				"body":   `{"clear_ip_list": true}`,
			},
			"print_all_ips": map[string]interface{}{
				"method": "POST",
				"body":   `{"print_all_ips": true}`,
			},
		},
	}
	
	c.JSON(http.StatusOK, apis)
}

// addIPs handles adding IPs to the list
func (as *APIServer) addIPs(c *gin.Context) {
	var request struct {
		AddIPs []int `json:"add_ips"`
	}
	
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON format"})
		return
	}
	
	fmt.Printf("Received request: POST {add_ips: %v}\n", request.AddIPs)
	
	as.ipManager.AddIPs(request.AddIPs)
	
	c.JSON(http.StatusOK, gin.H{
		"message": "IPs added successfully",
		"added_ips": request.AddIPs,
		"total_ips": len(as.ipManager.GetAllIPs()),
	})
}

// clearIPList handles clearing the IP list
func (as *APIServer) clearIPList(c *gin.Context) {
	var request struct {
		ClearIPList bool `json:"clear_ip_list"`
	}
	
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON format"})
		return
	}
	
	fmt.Printf("Received request: POST {clear_ip_list: %v}\n", request.ClearIPList)
	
	as.ipManager.ClearIPList()
	
	c.JSON(http.StatusOK, gin.H{
		"message": "IP list cleared successfully",
		"total_ips": 0,
	})
}

// printAllIPs handles printing all IPs
func (as *APIServer) printAllIPs(c *gin.Context) {
	var request struct {
		PrintAllIPs bool `json:"print_all_ips"`
	}
	
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON format"})
		return
	}
	
	fmt.Printf("Received request: POST {print_all_ips: %v}\n", request.PrintAllIPs)
	
	allIPs := as.ipManager.GetAllIPs()
	
	c.JSON(http.StatusOK, gin.H{
		"message": "All IPs retrieved successfully",
		"ips": allIPs,
		"total_ips": len(allIPs),
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