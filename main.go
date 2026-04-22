package main

import (
	"context"
	"log"
	"time"

	"github.com/leezesi/usmp/internal/api"
	"github.com/leezesi/usmp/pkg/yang-runtime/manager"
)

func main() {
	// Create and start the yang-controller-runtime Manager
	mgr := manager.New(
		manager.WithDefaultTimeout(10 * time.Second),
		// TODO: Set schema directory when we have dynamic schema loading
		// manager.WithSchemeDir("./yang-modules"),
	)

	// Start the manager - loads schema, starts all controllers
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := mgr.Start(ctx); err != nil {
		log.Fatalf("Failed to start Manager: %v", err)
	}
	log.Printf("YANG Controller Runtime started successfully")

	// 启动Gin API服务器
	server := api.NewServer(mgr)
	log.Printf("Starting server on :8080")
	if err := server.Run(":8080"); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}

	// Stop manager on exit
	mgr.Stop()
}
