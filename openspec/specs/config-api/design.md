# config-api - 架构设计

## 请求处理流程

### GET /api/v1/config/:ip/*path
```
Request → ConfigHandler.GetConfig()
  → c.Param("ip"), c.Param("path")
  → c.Query("force_refresh") 解析但忽略
  → pool.Get(DeviceConnectionInfo{IP:ip})
  → 获取失败 → Error(500)
  → cli.IsConnected() == false → Error(503)
  → context.WithTimeout(10s)
  → cli.Get(path, WithDatastore("running"))
  → NETCONF失败 → Error(500)
  → Success({data: result.Data})
```

### POST /api/v1/config/:ip/*path
```
Request → ConfigHandler.SetConfig()
  → c.Param("ip"), c.Param("path")
  → ShouldBindJSON(&data) 解析请求body
  → 解析失败 → Error(400)
  → convertToTypedStruct(path, data)
    → path含"system:" → convertMapToHuaweiSystem
    → path含"ifm:ifm"+"interfaces" → convertMapToHuaweiIfm
    → path含"vlan:"+"vlan/vlans" → convertMapToHuaweiVlan
    → 其他 → 原始map回退
  → 转换失败 → Error(400)
  → configStore.Set(ip, path, desiredConfig)
  → 写入失败 → Error(500)
  → manager.TriggerReconcile(ip, path)
  → Success({status:"ACCEPTED", reconciliation:{triggered, message}})
```

## 依赖关系

| 依赖 | 用途 | 调用位置 |
|------|------|----------|
| manager.Manager | 获取ClientPool/ConfigStore | GetConfig, SetConfig |
| ClientPool.Get() | 获取设备NETCONF连接 | GetConfig |
| client.Device | NETCONF协议操作 | GetConfig |
| ConfigStore.Set() | 存储期望配置 | SetConfig |
| Manager.TriggerReconcile() | 触发异步Reconciliation | SetConfig |
| generated/huawei | YANG强类型结构体 | SetConfig(convertToTypedStruct) |
| context.WithTimeout | NETCONF操作10秒超时 | GetConfig |

## 错误处理策略

- **GET错误链**：pool.Get失败 → 500; 设备离线 → 503; NETCONF Get失败 → 500; 超时 → 500
- **POST错误链**：JSON解析失败 → 400; YANG类型转换失败 → 400; ConfigStore写入失败 → 500
- **声明式语义**：POST成功仅表示配置已接受，不代表设备已配置；实际配置由Reconciler异步完成
- **类型转换容错**：未知YANG路径回退到原始map，不阻断请求

## YANG类型转换路由

| 路径关键字 | 转换函数 | 目标结构体 |
|-----------|----------|-----------|
| `system:` | convertMapToHuaweiSystem | HuaweiSystem_System |
| `ifm:ifm` + `interfaces` | convertMapToHuaweiIfm | HuaweiIfm_Ifm_Interfaces |
| `vlan:` + `vlan`/`vlans` | convertMapToHuaweiVlan | HuaweiVlan_Vlan_Vlans |
| 其他 | 无转换（原始map） | map[string]interface{} |