# devices-api - 架构设计

## 请求处理流程

### GET /api/v1/devices
```
Request → DeviceHandler.ListDevices()
  → mu.RLock() 读锁
  → 遍历 devices map 组装列表
  → manager.GetClientPool().Stats() 获取连接池统计
  → mu.RUnlock()
  → Success(devices + stats)
```

### POST /api/v1/devices
```
Request → DeviceHandler.AddDevice()
  → ShouldBindJSON() 解析请求体
  → 校验必填字段 (ip/username/password)
  → port=0 时默认830
  → mu.Lock() 写锁
  → devices[ip] = DeviceInfo{...} （覆盖写）
  → mu.Unlock()
  → pool.Get(DeviceConnectionInfo) 尝试NETCONF连接
  → 连接成功 → Success
  → 连接失败 → Error(500) （设备信息已保存）
```

### DELETE /api/v1/devices/:ip
```
Request → DeviceHandler.RemoveDevice()
  → c.Param("ip") 获取路径参数
  → mu.Lock() 写锁
  → delete(devices, ip)
  → mu.Unlock()
  → pool.Release(ip) 释放连接池资源
  → Success
```

### GET /api/v1/devices/:ip/status
```
Request → DeviceHandler.GetStatus()
  → c.Param("ip") 获取路径参数
  → mu.RLock() 读锁
  → 检查设备是否存在
  → mu.RUnlock()
  → 不存在 → Error(404)
  → 存在 → pool.Get(DeviceConnectionInfo) 获取/创建连接
  → cli.IsConnected() 判断连接状态
  → Success({running:true, connected:bool})
```

## 依赖关系

| 依赖 | 用途 | 调用位置 |
|------|------|----------|
| manager.Manager | 获取ClientPool | 所有handler |
| ClientPool.Get() | 获取/创建设备连接 | AddDevice, GetStatus |
| ClientPool.Release() | 释放设备连接 | RemoveDevice |
| ClientPool.Stats() | 连接池统计 | ListDevices |
| sync.RWMutex | 设备map并发安全 | 所有handler |
| client.DeviceConnectionInfo | 设备连接参数结构 | AddDevice, GetStatus |

## 错误处理策略

- **统一HTTP 200**：所有响应的HTTP状态码均为200，业务错误通过JSON body的`code`字段区分
- **成功**：`code=0, success=true`
- **客户端错误**：`code=400 (参数校验), code=404 (设备不存在)`
- **服务端错误**：`code=500 (连接失败), code=503 (设备离线)`
- **静默处理**：删除不存在的设备不报错，map delete对不存在的key为no-op
- **部分成功**：添加设备时连接失败，设备信息已保存但返回错误

## 数据存储

- 设备信息存储在内存 `map[string]DeviceInfo`（非持久化）
- 并发保护：`sync.RWMutex`（读操作RLock，写操作Lock）
- 默认预置一个测试设备 `192.168.1.1`（hardcoded in NewDeviceHandler）
- 无JSON文件持久化（与CLAUDE.md描述的"本地JSON元信息文件"不一致）