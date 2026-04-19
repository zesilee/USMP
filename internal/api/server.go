package api

import (
	protoactor "github.com/asynkron/protoactor-go/actor"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// Server represents the API server
type Server struct {
	router      *gin.Engine
	root        *protoactor.RootContext
	managerPID  *protoactor.PID
}

// NewServer creates a new API server
func NewServer(root *protoactor.RootContext, managerPID *protoactor.PID) *Server {
	s := &Server{
		root:     root,
		router:     gin.Default(),
		managerPID: managerPID,
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
			deviceHandler := NewDeviceHandler(s.root, s.managerPID)
			deviceGroup.GET("", deviceHandler.ListDevices)
			deviceGroup.POST("", deviceHandler.AddDevice)
			deviceGroup.DELETE("/:ip", deviceHandler.RemoveDevice)
			deviceGroup.GET("/:ip/status", deviceHandler.GetStatus)
		}

		// Configuration endpoints
		configGroup := v1.Group("/config")
		{
			configHandler := NewConfigHandler(s.root, s.managerPID)
			configGroup.GET("/:ip/:path", configHandler.GetConfig)
			configGroup.POST("/:ip/:path", configHandler.SetConfig)
		}

		// YANG model endpoints
		yangGroup := v1.Group("/yang")
		{
			yangHandler := NewYangHandler()
			yangGroup.GET("/modules", yangHandler.ListModules)
		}
	}
}

// Run starts the server
func (s *Server) Run(addr string) error {
	return s.router.Run(addr)
}
