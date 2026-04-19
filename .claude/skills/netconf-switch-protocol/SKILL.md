---
name: netconf-switch-protocol
description: SSH 830端口NETCONF连接，get-config/edit-config/commit，对接ygot结构体，断线重连异常容错
---

# 技能详情
## 一、激活时机（何时自动触发）
1.  当用户需求包含「NETCONF」「交换机对接」「配置获取」「配置下发」等关键词时，自动激活。
2.  开发流程中，涉及「设备连接」「RPC请求」「配置读写」「断线重连」时，自动启用。
3.  与 DeviceActor、ygot 技能联动，需要通过 NETCONF 读取/下发配置，并映射为 ygot 对象时，自动触发。

## 二、核心原则（底层设计逻辑）
1.  标准化原则：严格遵循 RFC6241 标准，确保与所有支持 NETCONF 的交换机兼容（华为/H3C/思科等）。
2.  可靠性原则：实现断线自动重连、超时重试、熔断机制，避免因设备离线导致程序异常。
3.  便捷性原则：直接对接 ygot 结构体，无需手动解析 XML 报文，降低开发成本。
4.  实时性原则：不持久化配置，所有配置均通过 NETCONF 实时从设备读取，确保数据最新。

## 三、使用样例（实操指令+输出效果）
### 样例1：触发技能指令
封装 NETCONF 客户端，实现连接、get-config 获取接口配置，对接 ygot 结构体
plaintext

### 样例2：技能输出效果（符合规则，≤500行）
```go
package netconf

import (
	"context"
	"fmt"
	"time"

	"github.com/scrapli/scrapligo/driver/netconf"
	"github.com/scrapli/scrapligo/transport"
	"你的项目路径/yang_models"
)

// NETCONF客户端
type NETCONFClient struct {
	deviceIP   string
	username   string
	password   string
	port       int
	driver     *netconf.Driver
	ctx        context.Context
}

// 初始化客户端
func NewNETCONFClient(ctx context.Context, deviceIP, username, password string, port int) *NETCONFClient {
	return &NETCONFClient{
		ctx:        ctx,
		deviceIP:   deviceIP,
		username:   username,
		password:   password,
		port:       port,
	}
}

// 连接设备（支持自动重连）
func (c *NETCONFClient) Connect() error {
	driver, err := netconf.NewDriver(
		c.deviceIP,
		netconf.WithPort(c.port),
		netconf.WithAuthNoStrictKey(),
		netconf.WithAuthUsername(c.username),
		netconf.WithAuthPassword(c.password),
		netconf.WithTransportType(transport.TransportSSH),
		netconf.WithTimeoutOps(30*time.Second),
	)
	if err != nil {
		return fmt.Errorf("初始化NETCONF驱动失败: %v", err)
	}

	// 连接设备
	if err := driver.Open(c.ctx); err != nil {
		return fmt.Errorf("连接设备失败: %v", err)
	}

	c.driver = driver
	// 启动重连协程
	go c.reconnectLoop()
	return nil
}

// 获取接口配置（对接ygot结构体）
func (c *NETCONFClient) GetInterfaceConfig(ifName string) (*yang_models.Interface, error) {
	// 构造get-config RPC请求
	filter := fmt.Sprintf(`<filter><interfaces xmlns="http://openconfig.net/yang/interfaces"><interface><name>%s</name></interface></interfaces></filter>`, ifName)
	resp, err := c.driver.GetConfig(c.ctx, netconf.WithSource("running"), netconf.WithFilter(filter))
	if err != nil {
		return nil, fmt.Errorf("获取配置失败: %v", err)
	}

	// XML转ygot结构体
	var iface yang_models.Interface
	if err := yang_models.Unmarshal([]byte(resp.Result), &iface); err != nil {
		return nil, fmt.Errorf("解析配置失败: %v", err)
	}

	return &iface, nil
}

// 断线重连循环
func (c *NETCONFClient) reconnectLoop() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		if c.driver == nil || !c.driver.IsOpen() {
			fmt.Printf("设备%s NETCONF连接断开，尝试重连...\n", c.deviceIP)
			if err := c.Connect(); err != nil {
				fmt.Printf("重连失败: %v\n", err)
				continue
			}
			fmt.Printf("设备%s NETCONF重连成功\n", c.deviceIP)
		}
	}
}
```

### 样例 3：联动其他技能
DeviceActor通过NETCONF客户端获取接口配置，写入TTL缓存，供前端查询