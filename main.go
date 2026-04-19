package main

import (
	"log"

	"github.com/leezesi/usmp/internal/actor"
	"github.com/leezesi/usmp/internal/api"
	"github.com/leezesi/usmp/internal/cache"

	protoactor "github.com/asynkron/protoactor-go/actor"
)

func main() {
	// 初始化全局TTL+LRU缓存
	cache.InitGlobalCache()

	// 启动Actor系统
	system := protoactor.NewActorSystem()
	root := system.Root

	// 创建并启动ManagerActor
	managerProps := protoactor.PropsFromProducer(func() protoactor.Actor {
		return actor.NewManagerActor()
	})
	managerPID, err := root.SpawnNamed(managerProps, "manager")
	if err != nil {
		log.Fatalf("Failed to start ManagerActor: %v", err)
	}

	// 保存ManagerPID供API使用
	actor.SetManagerPID(managerPID)

	// 启动Gin API服务器
	server := api.NewServer(root, managerPID)
	log.Printf("Starting server on :8080")
	if err := server.Run(":8080"); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
