---
name: go-protoactor-device
description: 一台交换机=一个DeviceActor；设备内每个YANG配置对象=独立子Actor；树形父子Actor架构
---

# 技能规范（激活时机 + 核心原则 + 使用样例）

## 一、激活时机
1. 用户需求包含「DeviceActor」「YANG Actor」「子Actor」「配置隔离」「树形Actor」时自动激活
2. 创建设备、加载YANG模型、生成配置对象、同步NETCONF时自动触发
3. 与 ygot、NETCONF、缓存技能联动时自动创建双层Actor结构

## 二、核心原则（架构铁律）
1. **设备级隔离**：一台设备 = 一个DeviceActor，互不干扰
2. **YANG配置隔离**：一个YANG模型/配置节点 = 一个独立子Actor
3. **父子树形结构**：DeviceActor 管理所有 YANG Actor
4. **状态自治**：每个YANG Actor独立管理自己的配置、缓存、NETCONF状态
5. **异步无阻塞**：Actor之间仅通过消息通信，无直接调用
6. **自动生命周期**：DeviceActor退出 → 所有YANG Actor自动销毁
7. **ygot驱动**：YANG Actor类型由ygot生成结构体自动决定

## 三、使用样例（真实可运行代码样例）
### 样例1：创建设备Actor + 自动创建YANG子Actor
指令：
创建一个设备 Actor，并自动为 openconfig-interfaces 生成 YANG 子 Actor

输出代码（≤500行）：
```go
// 设备Actor（顶层）
type DeviceActor struct {
	deviceIP string
	yangActors map[string]*actor.PID // YANG 子Actor注册表
	mutex      sync.RWMutex
}

// YANG 配置对象Actor（每个YANG节点独立一个）
type YangObjectActor struct {
	deviceIP  string
	yangPath  string    // YANG路径，如 /interfaces/interface
	config    ygot.ValidatedGoStruct // ygot生成的强类型对象
	cache     *cache.TTLCache
	netconf   *netconf.Client
}

// 启动设备Actor → 自动创建YANG子Actor
func (d *DeviceActor) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case *CreateDeviceRequest:
		d.deviceIP = msg.DeviceIP

		// 自动创建 YANG 子Actor（interface）
		ifaceActor := NewYangObjectActor(
			d.deviceIP,
			"/interfaces/interface",
			yang_models.NewInterface(), // ygot自动生成
			d.cache,
			d.netconf,
		)
		pid, _ := ctx.SpawnNamed(actor.PropsFromProducer(func() actor.Actor {
			return ifaceActor
		}), "interface")

		// 注册到设备Actor
		d.mutex.Lock()
		d.yangActors["/interfaces/interface"] = pid
		d.mutex.Unlock()

		ctx.Respond(&DeviceCreatedResponse{Success: true})
	}
}
```

样例 2：向 YANG 子 Actor 发送获取配置消息
指令：
给接口YANG Actor发送GetConfigRequest消息，从缓存/NETCONF获取配置
输出代码：
```go
// 从YANG子Actor获取接口配置
func (d *DeviceActor) GetInterfaceConfig(ctx actor.Context, ifName string) {
	d.mutex.RLock()
	pid, ok := d.yangActors["/interfaces/interface"]
	d.mutex.RUnlock()

	if !ok {
		ctx.Respond(&ErrorResponse{Err: "YANG Actor not found"})
		return
	}

	// 转发消息给YANG子Actor
	ctx.Forward(pid)
}

// YANG子Actor处理GetConfig
func (y *YangObjectActor) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case *GetConfigRequest:
		// 先查缓存
		if val, ok := y.cache.Get(y.deviceIP + y.yangPath); ok {
			ctx.Respond(&GetConfigResponse{Config: val})
			return
		}

		// 缓存未命中 → NETCONF获取
		conf, err := y.netconf.GetConfig(y.yangPath)
		if err != nil {
			ctx.Respond(&ErrorResponse{Err: err.Error()})
			return
		}

		// 写入缓存
		y.cache.Set(y.deviceIP+y.yangPath, conf)
		ctx.Respond(&GetConfigResponse{Config: conf})
	}
}
```

### 样例 3：设备销毁 → YANG Actor 自动全部销毁
```go
case *actor.Stopping:
	// 设备Actor停止 → 自动销毁所有YANG子Actor
	d.mutex.Lock()
	for _, pid := range d.yangActors {
		ctx.Poison(pid)
	}
	d.mutex.Unlock()
```