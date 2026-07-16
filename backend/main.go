package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/leezesi/usmp/backend/internal/api"
	"github.com/leezesi/usmp/backend/internal/controller/bgp"
	"github.com/leezesi/usmp/backend/internal/controller/ifm"
	"github.com/leezesi/usmp/backend/internal/controller/networkinstance"
	"github.com/leezesi/usmp/backend/internal/controller/system"
	"github.com/leezesi/usmp/backend/internal/controller/vlan"
	"github.com/leezesi/usmp/backend/internal/crdsource"
	"github.com/leezesi/usmp/backend/internal/intent"
	"github.com/leezesi/usmp/backend/internal/yangschema"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/controller"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/device"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/manager"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/predicate"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/source"
	ctrlcfg "sigs.k8s.io/controller-runtime/pkg/client/config"
)

// @title           USMP 交换机设备管理平台 API
// @version         1.0
// @description     无数据库、模型驱动的交换机配置管理 REST API（NETCONF/gNMI）。
// @description     响应统一信封 {code,message,data,success}；此规格是前端 TS 类型的唯一真源（勿手改生成物）。
// @BasePath        /api/v1
func main() {
	// Build the YANG schema tree from generated ygot models (huawei + openconfig)
	// so the manager's schema tree is populated (fixes the empty-schema gap).
	// Device NETCONF capabilities narrow the usable module set at runtime.
	yangSchema, err := yangschema.Load()
	if err != nil {
		log.Fatalf("failed to load YANG schema: %v", err)
	}

	// 根 context：CRD store watch 与 Manager 同生命周期。
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create and start the yang-controller-runtime Manager
	mgr := manager.New(
		manager.WithDefaultTimeout(10*time.Second),
		manager.WithSchema(yangSchema),
		// 操作审计日志持久化到本地 JSON（§8，可用 USMP_AUDIT_FILE 覆盖）。
		manager.WithAuditFile(auditFilePath()),
		// DS-01/DS-05: 集群可达时设备注册表走 Device CRD（跨副本共享+重启
		// 恢复+凭据 Secret 引用），否则降级进程内存实现（R08）。
		manager.WithDeviceStore(buildDeviceStore(ctx)),
	)

	// Create and register the Huawei VLAN controller
	// The VLAN controller reconciles VLAN configuration every 5 minutes
	cs := mgr.GetConfigStore()
	clientPool := mgr.GetClientPool()

	// Periodic source polls all configured devices for reconciliation
	// Pass nil for deviceIDs to indicate all devices that have desired config
	vlanCtrl := controller.ControllerManagedBy("huawei-vlan").
		WithReconciler(vlan.New(cs, clientPool, mgr.GetDeviceStore())).
		WithSource(source.NewPeriodicSourceWithLister(5*time.Minute, mgr.GetDeviceStore(), "/vlan:vlan/vlan:vlans")).
		WithPredicate(predicate.Prefix("/vlan:vlan/vlan:vlans")).
		WithWorkerCount(2).
		Build()

	mgr.AddController(vlanCtrl)
	log.Printf("Huawei VLAN controller registered successfully")

	// Create and register the Huawei IFM controller
	// The IFM controller reconciles interface configuration every 5 minutes
	ifmCtrl := controller.ControllerManagedBy("huawei-ifm").
		WithReconciler(ifm.New(cs, clientPool, mgr.GetDeviceStore())).
		WithSource(source.NewPeriodicSourceWithLister(5*time.Minute, mgr.GetDeviceStore(), "/ifm:ifm/ifm:interfaces")).
		WithPredicate(predicate.Prefix("/ifm:ifm/ifm:interfaces")).
		WithWorkerCount(2).
		Build()

	mgr.AddController(ifmCtrl)
	log.Printf("Huawei IFM controller registered successfully")

	// Create and register the Huawei System controller
	// The System controller reconciles system configuration every 5 minutes
	systemCtrl := controller.ControllerManagedBy("huawei-system").
		WithReconciler(system.New(cs, clientPool, mgr.GetDeviceStore())).
		WithSource(source.NewPeriodicSourceWithLister(5*time.Minute, mgr.GetDeviceStore(), "/system:system")).
		WithPredicate(predicate.Prefix("/system:system")).
		WithWorkerCount(2).
		Build()

	mgr.AddController(systemCtrl)
	log.Printf("Huawei System controller registered successfully")

	// Create and register the Huawei 公网 BGP controller（容器根模块，/bgp:bgp）。
	// Name 含 "bgp" → manager.TriggerReconcile 按 ControllerToken="bgp" 路由命中。
	bgpCtrl := controller.ControllerManagedBy("huawei-bgp").
		WithReconciler(bgp.New(cs, clientPool, mgr.GetDeviceStore())).
		WithSource(source.NewPeriodicSourceWithLister(5*time.Minute, mgr.GetDeviceStore(), bgp.BgpPath)).
		WithPredicate(predicate.Prefix(bgp.BgpPath)).
		WithWorkerCount(2).
		Build()

	mgr.AddController(bgpCtrl)
	log.Printf("Huawei BGP controller registered successfully")

	// Create and register the Huawei network-instance controller（容器根 + 嵌套 list，
	// /ni:network-instance）——BGP 二期 peering 唯一硬前置。Name 含 "network-instance"
	// → manager.TriggerReconcile 按 ControllerToken="network-instance" 路由命中。
	niCtrl := controller.ControllerManagedBy("huawei-network-instance").
		WithReconciler(networkinstance.New(cs, clientPool, mgr.GetDeviceStore())).
		WithSource(source.NewPeriodicSourceWithLister(5*time.Minute, mgr.GetDeviceStore(), networkinstance.NetworkInstancePath)).
		WithPredicate(predicate.Prefix(networkinstance.NetworkInstancePath)).
		WithWorkerCount(2).
		Build()

	mgr.AddController(niCtrl)
	log.Printf("Huawei network-instance controller registered successfully")

	// Register the BusinessVlan CRD intent source (场景② 意图面收编 Stack B),
	// parallel to the legacy Actor path. Degrades gracefully without a K8s cluster.
	crdCache, err := crdsource.RegisterIntentSources(mgr)
	if err != nil {
		log.Printf("Failed to register CRD intent source: %v", err)
	}

	// 业务网络配置意图控制器（business-network-config）：BusinessVlanService CR
	// watch → 校验/展开/status 回写；旧 BusinessVlan 桥接并行保留（渐进替换）。
	// 无 K8s 集群时优雅降级（BIO-01）。
	intentCache, err := intent.Register(mgr)
	if err != nil {
		log.Printf("Failed to register business intent controller: %v", err)
	}

	// Start the manager - loads schema, starts all controllers
	go crdsource.StartCache(ctx, crdCache)
	go crdsource.StartCache(ctx, intentCache)

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

// auditFilePath 返回操作审计日志的本地 JSON 路径，可用 USMP_AUDIT_FILE 覆盖，
// 默认 data/audit.json（§8 本地元信息，非数据库 R03）。
func auditFilePath() string {
	if p := os.Getenv("USMP_AUDIT_FILE"); p != "" {
		return p
	}
	return "data/audit.json"
}

// buildDeviceStore 按集群可达性选择设备注册表后端（DS-01/DS-05）：可达走
// Device CRD store（namespace 复用 USMP_INTENT_NAMESPACE），任一步失败降级
// 进程内存实现并记日志（R08，不崩溃）。
func buildDeviceStore(ctx context.Context) device.Store {
	cfg, err := ctrlcfg.GetConfig()
	if err != nil {
		log.Printf("device store: no reachable cluster, using in-memory store (devices lost on restart): %v", err)
		return device.NewStore()
	}
	s, err := device.NewCRDStore(ctx, cfg, intent.Namespace())
	if err != nil {
		log.Printf("device store: CRD store unavailable, degrading to in-memory: %v", err)
		return device.NewStore()
	}
	log.Printf("device store: CRD-backed (namespace %q)", intent.Namespace())
	return s
}
