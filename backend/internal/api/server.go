package api

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/manager"
)

// Server represents the API server
type Server struct {
	router  *gin.Engine
	manager manager.Manager
}

// NewServer creates a new API server
func NewServer(manager manager.Manager) *Server {
	s := &Server{
		router:  gin.Default(),
		manager: manager,
	}

	s.setupCORS()
	s.setupRoutes()

	return s
}

func (s *Server) setupCORS() {
	config := cors.DefaultConfig()
	config.AllowAllOrigins = true
	config.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
	config.AllowHeaders = []string{"Origin", "Content-Type", "Authorization"}
	s.router.Use(cors.New(config))
}

func (s *Server) setupRoutes() {
	v1 := s.router.Group("/api/v1")
	{
		// Device endpoints
		deviceGroup := v1.Group("/devices")
		{
			deviceHandler := NewDeviceHandler(s.manager)
			deviceGroup.GET("", deviceHandler.ListDevices)
			deviceGroup.POST("", deviceHandler.AddDevice)
			deviceGroup.DELETE("/:ip", deviceHandler.RemoveDevice)
			deviceGroup.GET("/:ip/status", deviceHandler.GetStatus)
		}

		// Configuration endpoints
		configGroup := v1.Group("/config")
		{
			configHandler := NewConfigHandler(s.manager)
			configGroup.GET("/:ip/*path", configHandler.GetConfig)
			configGroup.POST("/:ip/*path", configHandler.SetConfig)
		}

		// YANG model endpoints
		yangGroup := v1.Group("/yang")
		{
			yangHandler := NewYangHandler(s.manager)
			yangGroup.GET("/modules", yangHandler.ListModules)
			yangGroup.GET("/schema/:module", yangHandler.GetSchema)
		}
	}
}

// Run starts the server
func (s *Server) Run(addr string) error {
	return s.router.Run(addr)
}
