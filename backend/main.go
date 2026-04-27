package main

import (
	"context"
	"log"
	"time"

	"github.com/leezesi/usmp/backend/internal/api"
	"github.com/leezesi/usmp/backend/internal/controller/interfaces"
	"github.com/leezesi/usmp/backend/internal/controller/vlan"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/controller"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/manager"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/predicate"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/source"
)

func main() {
	// Create and start the yang-controller-runtime Manager
	mgr := manager.New(
		manager.WithDefaultTimeout(10 * time.Second),
		// TODO: Set schema directory when we have dynamic schema loading
		// manager.WithSchemeDir("./yang-modules"),
	)

	// Create and register the OpenConfig VLAN controller
	// The VLAN controller reconciles VLAN configuration every 5 minutes
	cs := mgr.GetConfigStore()
	clientPool := mgr.GetClientPool()

	// Periodic source polls all configured devices for reconciliation
	// Pass nil for deviceIDs to indicate all devices that have desired config
	vlanCtrl := controller.ControllerManagedBy("openconfig-vlan").
		WithReconciler(vlan.New(cs, clientPool)).
		WithSource(source.NewPeriodicSource(5 * time.Minute, nil, "/vlans")).
		WithPredicate(predicate.Prefix("/vlans")).
		WithWorkerCount(2).
		Build()

	mgr.AddController(vlanCtrl)
	log.Printf("OpenConfig VLAN controller registered successfully")

	// Create and register the OpenConfig Interfaces controller
	// The Interfaces controller reconciles interface configuration every 5 minutes
	interfacesCtrl := controller.ControllerManagedBy("openconfig-interfaces").
		WithReconciler(interfaces.New(cs, clientPool)).
		WithSource(source.NewPeriodicSource(5 * time.Minute, nil, "/interfaces")).
		WithPredicate(predicate.Prefix("/interfaces")).
		WithWorkerCount(2).
		Build()

	mgr.AddController(interfacesCtrl)
	log.Printf("OpenConfig Interfaces controller registered successfully")

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
