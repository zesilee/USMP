package main

import (
	"context"
	"log"
	"time"

	"github.com/leezesi/usmp/backend/internal/api"
	"github.com/leezesi/usmp/backend/internal/controller/ifm"
	"github.com/leezesi/usmp/backend/internal/controller/system"
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

	// Create and register the Huawei VLAN controller
	// The VLAN controller reconciles VLAN configuration every 5 minutes
	cs := mgr.GetConfigStore()
	clientPool := mgr.GetClientPool()

	// Periodic source polls all configured devices for reconciliation
	// Pass nil for deviceIDs to indicate all devices that have desired config
	vlanCtrl := controller.ControllerManagedBy("huawei-vlan").
		WithReconciler(vlan.New(cs, clientPool)).
		WithSource(source.NewPeriodicSource(5 * time.Minute, nil, "/vlan:vlan/vlan:vlans")).
		WithPredicate(predicate.Prefix("/vlan:vlan/vlan:vlans")).
		WithWorkerCount(2).
		Build()

	mgr.AddController(vlanCtrl)
	log.Printf("Huawei VLAN controller registered successfully")

	// Create and register the Huawei IFM controller
	// The IFM controller reconciles interface configuration every 5 minutes
	ifmCtrl := controller.ControllerManagedBy("huawei-ifm").
		WithReconciler(ifm.New(cs, clientPool)).
		WithSource(source.NewPeriodicSource(5 * time.Minute, nil, "/ifm:ifm/ifm:interfaces")).
		WithPredicate(predicate.Prefix("/ifm:ifm/ifm:interfaces")).
		WithWorkerCount(2).
		Build()

	mgr.AddController(ifmCtrl)
	log.Printf("Huawei IFM controller registered successfully")

	// Create and register the Huawei System controller
	// The System controller reconciles system configuration every 5 minutes
	systemCtrl := controller.ControllerManagedBy("huawei-system").
		WithReconciler(system.New(cs, clientPool)).
		WithSource(source.NewPeriodicSource(5 * time.Minute, nil, "/system:system")).
		WithPredicate(predicate.Prefix("/system:system")).
		WithWorkerCount(2).
		Build()

	mgr.AddController(systemCtrl)
	log.Printf("Huawei System controller registered successfully")

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
