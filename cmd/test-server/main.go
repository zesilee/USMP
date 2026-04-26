// Package main E2E 测试服务器 - 启动后端服务连接到模拟网元
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/leezesi/usmp/test/netsim"
)

// 全局模拟服务器
var sim *netsim.Simulator

// API 响应结构
type ApiResponse[T any] struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	Data    T      `json:"data,omitempty"`
}

// VLANInfo VLAN 信息
type VLANInfo struct {
	ID             int      `json:"id"`
	Name           string   `json:"name"`
	AdminStatus    string   `json:"adminStatus"`
	OperStatus     string   `json:"operStatus"`
	TaggedPorts    []string `json:"taggedPorts"`
	UntaggedPorts  []string `json:"untaggedPorts"`
}

func main() {
	gin.SetMode(gin.TestMode)
	r := gin.Default()

	// CORS 配置 - 允许前端访问
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:5173", "http://127.0.0.1:5173"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// 启动 NETCONF 模拟服务器
	sim = netsim.NewSimulator()
	if err := sim.Start(); err != nil {
		log.Fatalf("Failed to start NETCONF simulator: %v", err)
	}
	defer sim.Stop()

	log.Printf("NETCONF Simulator started on %s:%d", sim.Addr(), sim.Port())

	// API 路由
	api := r.Group("/api/v1")
	{
		// 设备 API
		api.GET("/devices", listDevices)
		api.GET("/devices/:ip/status", getDeviceStatus)

		// VLAN 配置 API
		api.GET("/config/:ip/vlans", getVLANConfig)
		api.POST("/config/:ip/vlans", createVLAN)
		api.PUT("/config/:ip/vlans/:id", updateVLAN)
		api.DELETE("/config/:ip/vlans/:id", deleteVLAN)
	}

	// 启动 HTTP 服务器
	srv := &http.Server{
		Addr:    ":8080",
		Handler: r,
	}

	// 优雅关闭
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
	log.Println("NETCONF Simulator on port:", sim.Port())
	log.Println("=")
	log.Println("Run E2E tests: cd web && npm run e2e")
	log.Println("=")

	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Failed to start server: %v", err)
	}
}

// listDevices 返回设备列表
func listDevices(c *gin.Context) {
	c.JSON(http.StatusOK, ApiResponse[[]map[string]interface{}]{
		Success: true,
		Data: []map[string]interface{}{
			{
				"ip":       "192.168.1.1",
				"port":     sim.Port(),
				"username": sim.Username(),
				"password": sim.Password(),
				"status":   "online",
			},
		},
	})
}

// getDeviceStatus 获取设备状态
func getDeviceStatus(c *gin.Context) {
	c.JSON(http.StatusOK, ApiResponse[map[string]bool]{
		Success: true,
		Data: map[string]bool{
			"running":   true,
			"connected": true,
		},
	})
}

// getVLANConfig 获取 VLAN 配置
func getVLANConfig(c *gin.Context) {
	forceRefresh := c.Query("force_refresh") == "true"

	vlans := sim.GetAllVLANs()
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

	c.JSON(http.StatusOK, ApiResponse[map[string]interface{}]{
		Success: true,
		Data: map[string]interface{}{
			"vlans":       result,
			"fromCache":   !forceRefresh,
			"lastSync":    time.Now().Format(time.RFC3339),
		},
	})
}

// createVLAN 创建 VLAN
func createVLAN(c *gin.Context) {
	var vlan VLANInfo
	if err := c.ShouldBindJSON(&vlan); err != nil {
		c.JSON(http.StatusBadRequest, ApiResponse[any]{
			Success: false,
			Message: fmt.Sprintf("Invalid request: %v", err),
		})
		return
	}

	if vlan.ID < 1 || vlan.ID > 4094 {
		c.JSON(http.StatusBadRequest, ApiResponse[any]{
			Success: false,
			Message: "VLAN ID must be between 1 and 4094",
		})
		return
	}

	sim.AddVLAN(&netsim.VLANConfig{
		ID:             vlan.ID,
		Name:           vlan.Name,
		AdminState:     vlan.AdminStatus,
		TaggedPorts:    vlan.TaggedPorts,
		UntaggedPorts:  vlan.UntaggedPorts,
	})

	c.JSON(http.StatusOK, ApiResponse[any]{
		Success: true,
		Message: "VLAN created successfully",
	})
}

// updateVLAN 更新 VLAN
func updateVLAN(c *gin.Context) {
	idStr := c.Param("id")
	var id int
	if _, err := fmt.Sscanf(idStr, "%d", &id); err != nil {
		c.JSON(http.StatusBadRequest, ApiResponse[any]{
			Success: false,
			Message: "Invalid VLAN ID",
		})
		return
	}

	var vlan VLANInfo
	if err := c.ShouldBindJSON(&vlan); err != nil {
		c.JSON(http.StatusBadRequest, ApiResponse[any]{
			Success: false,
			Message: fmt.Sprintf("Invalid request: %v", err),
		})
		return
	}

	existing := sim.GetVLAN(id)
	if existing == nil {
		c.JSON(http.StatusNotFound, ApiResponse[any]{
			Success: false,
			Message: "VLAN not found",
		})
		return
	}

	existing.Name = vlan.Name
	existing.AdminState = vlan.AdminStatus
	existing.TaggedPorts = vlan.TaggedPorts
	existing.UntaggedPorts = vlan.UntaggedPorts

	sim.AddVLAN(existing)

	c.JSON(http.StatusOK, ApiResponse[any]{
		Success: true,
		Message: "VLAN updated successfully",
	})
}

// deleteVLAN 删除 VLAN
func deleteVLAN(c *gin.Context) {
	idStr := c.Param("id")
	var id int
	if _, err := fmt.Sscanf(idStr, "%d", &id); err != nil {
		c.JSON(http.StatusBadRequest, ApiResponse[any]{
			Success: false,
			Message: "Invalid VLAN ID",
		})
		return
	}

	sim.DeleteVLAN(id)

	c.JSON(http.StatusOK, ApiResponse[any]{
		Success: true,
		Message: "VLAN deleted successfully",
	})
}

// helper: JSON 响应
func jsonResponse(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
