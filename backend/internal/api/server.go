package api

import (
	"log"
	"net/http"
	"time"

	"github.com/beego/beego/v2/server/web"
	beecontext "github.com/beego/beego/v2/server/web/context"
	"github.com/beego/beego/v2/server/web/filter/cors"
	"github.com/leezesi/usmp/backend/internal/intent"
	"github.com/leezesi/usmp/backend/pkg/yang-runtime/manager"
)

// Server represents the API server
type Server struct {
	router  *web.ControllerRegister
	manager manager.Manager
}

// NewServer creates a new API server
func NewServer(manager manager.Manager) *Server {
	s := &Server{
		router:  web.NewControllerRegister(),
		manager: manager,
	}

	s.setupCORS()
	s.setupRoutes()

	return s
}

func (s *Server) setupCORS() {
	_ = s.router.InsertFilter("*", web.BeforeRouter, cors.Allow(&cors.Options{
		AllowAllOrigins: true,
		AllowMethods:    []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:    []string{"Origin", "Content-Type", "Authorization"},
		// gin-contrib/cors DefaultConfig 自带 12h 预检缓存，保持对等避免预检风暴
		MaxAge: 12 * time.Hour,
	}))
	// 访问日志：承接 gin.Default() 的 Logger 中间件（运维排查依赖请求日志）
	_ = s.router.InsertFilter("*", web.FinishRouter, func(ctx *beecontext.Context) {
		log.Printf("%s %s -> %d", ctx.Input.Method(), ctx.Request.URL.Path, ctx.ResponseWriter.Status)
	}, web.WithReturnOnOutput(false))
}

func (s *Server) setupRoutes() {
	// Device endpoints
	reconcileHandler := NewReconcileHandler(s.manager)
	deviceHandler := NewDeviceHandler(s.manager)
	s.router.Get("/api/v1/devices", deviceHandler.ListDevices)
	s.router.Post("/api/v1/devices", deviceHandler.AddDevice)
	s.router.Delete("/api/v1/devices/:ip", deviceHandler.RemoveDevice)
	s.router.Get("/api/v1/devices/:ip/status", deviceHandler.GetStatus)
	// hello capabilities 原文（CN-06：诊断 + deviations 侦察）
	s.router.Get("/api/v1/devices/:ip/capabilities", deviceHandler.GetCapabilities)
	// Per-device reconcile outcome (desired↔actual convergence)
	s.router.Get("/api/v1/devices/:ip/reconcile", reconcileHandler.GetDeviceReconcile)

	// Fleet-wide reconcile aggregate (for the convergence dashboard)
	s.router.Get("/api/v1/reconcile/status", reconcileHandler.GetFleetReconcile)

	// Operation audit log (config-delivery records + live reconcile outcome)
	s.router.Get("/api/v1/logs", NewAuditHandler(s.manager).ListLogs)

	// Configuration endpoints
	configHandler := NewConfigHandler(s.manager)
	// 攒批变更集（config-changeset）：静态段 "changeset" 与 ":ip" 参数段
	// 在 beego 路由树共存（静态优先，行为由 beego_router_equiv_test 锁死），
	// 先注册以示意优先级。
	changesetHandler := NewChangesetHandler(s.manager)
	s.router.Post("/api/v1/config/changeset/preview", changesetHandler.Preview)
	s.router.Post("/api/v1/config/changeset/commit", changesetHandler.Commit)
	s.router.Get("/api/v1/config/:ip/*", configHandler.GetConfig)
	s.router.Post("/api/v1/config/:ip/*", configHandler.SetConfig)
	s.router.Delete("/api/v1/config/:ip/*", configHandler.DeleteConfig)

	// rpc 执行（RPC-03）：触发模块运维操作，不写缓存/不对账（§8/D4）
	s.router.Post("/api/v1/rpc/:ip/:module/:rpc", NewRPCHandler(s.manager).Execute)

	// Soft-ownership query (BIO-07：原生控制台徽标/手改提示数据面)
	s.router.Get("/api/v1/ownership/:device", NewOwnershipHandler().Query)

	// 业务网络配置（意图 CR 代理，design D7：前端不直连 apiserver）
	bizHandler := NewBusinessHandler(intent.APIClient, intent.Namespace())
	s.router.Get("/api/v1/business/vlan-services", bizHandler.List)
	s.router.Get("/api/v1/business/vlan-services/:name", bizHandler.Get)
	s.router.Post("/api/v1/business/vlan-services", bizHandler.Apply)
	s.router.Delete("/api/v1/business/vlan-services/:name", bizHandler.Delete)

	// YANG model endpoints
	yangHandler := NewYangHandler(s.manager)
	s.router.Get("/api/v1/yang/modules", yangHandler.ListModules)
	s.router.Get("/api/v1/yang/left-tree", yangHandler.LeftTree)
	s.router.Get("/api/v1/yang/schema/:module", yangHandler.GetSchema)
}

// Run starts the server
func (s *Server) Run(addr string) error {
	return http.ListenAndServe(addr, s.router)
}
