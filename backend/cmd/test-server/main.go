// Package main E2E 测试服务器 — 为前端 Playwright 提供内存 REST 桩。
//
// 只提供前端要的 REST 接口，从不走 NETCONF。历史上误用 netsim 假「模拟器」
// 作后端，现改为名副其实的内存 VLAN store（fixture.go）。
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/beego/beego/v2/server/web"
	beecontext "github.com/beego/beego/v2/server/web/context"
	"github.com/beego/beego/v2/server/web/filter/cors"
)

// 内存 VLAN store（替代原 netsim 假模拟器）
var store *vlanStore

// 前端展示用的固定设备描述（本服务不经 NETCONF，仅供设备列表展示）。
const (
	fixtureDeviceIP   = "192.168.1.1"
	fixtureDevicePort = 830
	fixtureUsername   = "admin"
	fixturePassword   = "admin"
)

// ApiResponse 是 REST 响应的统一信封。
type ApiResponse[T any] struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	Data    T      `json:"data,omitempty"`
}

// VLANInfo VLAN 信息
type VLANInfo struct {
	ID            int      `json:"id"`
	Name          string   `json:"name"`
	AdminStatus   string   `json:"adminStatus"`
	OperStatus    string   `json:"operStatus"`
	TaggedPorts   []string `json:"taggedPorts"`
	UntaggedPorts []string `json:"untaggedPorts"`
}

func main() {
	r := web.NewControllerRegister()

	// 放行前端开发服务器 origin 的跨域请求。
	_ = r.InsertFilter("*", web.BeforeRouter, cors.Allow(&cors.Options{
		AllowOrigins:     []string{"http://localhost:3000", "http://127.0.0.1:3000", "http://localhost:5173", "http://127.0.0.1:5173"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	store = newVLANStore()

	r.Get("/api/v1/devices", listDevices)
	r.Get("/api/v1/devices/:ip/status", getDeviceStatus)

	r.Get("/api/v1/config/:ip/vlans", getVLANConfig)
	r.Post("/api/v1/config/:ip/vlans", createVLAN)
	r.Put("/api/v1/config/:ip/vlans/:id", updateVLAN)
	r.Delete("/api/v1/config/:ip/vlans/:id", deleteVLAN)

	srv := &http.Server{
		Addr:    ":8080",
		Handler: r,
	}

	// 优雅关闭：收到终止信号后停止收新请求，等在途请求收尾再退出。
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		log.Println("Shutting down server...")

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := srv.Shutdown(ctx); err != nil {
			log.Fatalf("Server shutdown failed: %v", err)
		}
	}()

	log.Println("=")
	log.Println("E2E Test Server started on http://localhost:8080")
	log.Println("In-memory VLAN REST fixture (no NETCONF)")
	log.Println("=")
	log.Println("Run E2E tests: cd frontend && npm run e2e")
	log.Println("=")

	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Failed to start server: %v", err)
	}
}

// bindJSON 对齐 gin ShouldBindJSON：直读请求体解码到 v，空体/坏 JSON 返回错误。
func bindJSON(c *beecontext.Context, v interface{}) error {
	if c.Request == nil || c.Request.Body == nil {
		return errors.New("request body is empty")
	}
	return json.NewDecoder(c.Request.Body).Decode(v)
}

func listDevices(c *beecontext.Context) {
	_ = c.Output.JSON(ApiResponse[[]map[string]interface{}]{
		Success: true,
		Data: []map[string]interface{}{
			{
				"ip":       fixtureDeviceIP,
				"port":     fixtureDevicePort,
				"username": fixtureUsername,
				"password": fixturePassword,
				"status":   "online",
			},
		},
	}, false, false)
}

func getDeviceStatus(c *beecontext.Context) {
	_ = c.Output.JSON(ApiResponse[map[string]bool]{
		Success: true,
		Data: map[string]bool{
			"running":   true,
			"connected": true,
		},
	}, false, false)
}

func getVLANConfig(c *beecontext.Context) {
	forceRefresh := c.Input.Query("force_refresh") == "true"

	vlans := store.all()
	result := make([]VLANInfo, 0, len(vlans))

	for _, v := range vlans {
		operStatus := "ACTIVE"
		if v.AdminState == "DOWN" {
			operStatus = "SUSPENDED"
		}

		result = append(result, VLANInfo{
			ID:            v.ID,
			Name:          v.Name,
			AdminStatus:   v.AdminState,
			OperStatus:    operStatus,
			TaggedPorts:   v.TaggedPorts,
			UntaggedPorts: v.UntaggedPorts,
		})
	}

	_ = c.Output.JSON(ApiResponse[map[string]interface{}]{
		Success: true,
		Data: map[string]interface{}{
			"vlans":     result,
			"fromCache": !forceRefresh,
			"lastSync":  time.Now().Format(time.RFC3339),
		},
	}, false, false)
}

func createVLAN(c *beecontext.Context) {
	var vlan VLANInfo
	if err := bindJSON(c, &vlan); err != nil {
		c.Output.SetStatus(http.StatusBadRequest)
		_ = c.Output.JSON(ApiResponse[any]{
			Success: false,
			Message: fmt.Sprintf("Invalid request: %v", err),
		}, false, false)
		return
	}

	if vlan.ID < 1 || vlan.ID > 4094 {
		c.Output.SetStatus(http.StatusBadRequest)
		_ = c.Output.JSON(ApiResponse[any]{
			Success: false,
			Message: "VLAN ID must be between 1 and 4094",
		}, false, false)
		return
	}

	store.put(&VLAN{
		ID:            vlan.ID,
		Name:          vlan.Name,
		AdminState:    vlan.AdminStatus,
		TaggedPorts:   vlan.TaggedPorts,
		UntaggedPorts: vlan.UntaggedPorts,
	})

	_ = c.Output.JSON(ApiResponse[any]{
		Success: true,
		Message: "VLAN created successfully",
	}, false, false)
}

func updateVLAN(c *beecontext.Context) {
	idStr := c.Input.Param(":id")
	var id int
	if _, err := fmt.Sscanf(idStr, "%d", &id); err != nil {
		c.Output.SetStatus(http.StatusBadRequest)
		_ = c.Output.JSON(ApiResponse[any]{
			Success: false,
			Message: "Invalid VLAN ID",
		}, false, false)
		return
	}

	var vlan VLANInfo
	if err := bindJSON(c, &vlan); err != nil {
		c.Output.SetStatus(http.StatusBadRequest)
		_ = c.Output.JSON(ApiResponse[any]{
			Success: false,
			Message: fmt.Sprintf("Invalid request: %v", err),
		}, false, false)
		return
	}

	existing := store.get(id)
	if existing == nil {
		c.Output.SetStatus(http.StatusNotFound)
		_ = c.Output.JSON(ApiResponse[any]{
			Success: false,
			Message: "VLAN not found",
		}, false, false)
		return
	}

	existing.Name = vlan.Name
	existing.AdminState = vlan.AdminStatus
	existing.TaggedPorts = vlan.TaggedPorts
	existing.UntaggedPorts = vlan.UntaggedPorts

	store.put(existing)

	_ = c.Output.JSON(ApiResponse[any]{
		Success: true,
		Message: "VLAN updated successfully",
	}, false, false)
}

func deleteVLAN(c *beecontext.Context) {
	idStr := c.Input.Param(":id")
	var id int
	if _, err := fmt.Sscanf(idStr, "%d", &id); err != nil {
		c.Output.SetStatus(http.StatusBadRequest)
		_ = c.Output.JSON(ApiResponse[any]{
			Success: false,
			Message: "Invalid VLAN ID",
		}, false, false)
		return
	}

	store.remove(id)

	_ = c.Output.JSON(ApiResponse[any]{
		Success: true,
		Message: "VLAN deleted successfully",
	}, false, false)
}

func jsonResponse(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
