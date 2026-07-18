# device-capability-negotiation — 每设备 YANG 能力协商

## Purpose

统一 YANG 对接多版本/多角色交换机：设备实际可配模块集由 NETCONF hello capabilities 实测协商（连接层缓存、重连刷新），北向以 `?device=` 按设备透出；SND blacklist 以注解形式随模块列表透出。协商永不使用 role/blacklist 做硬裁剪。

## Requirements


### Requirement: CN-01 hello 能力采集与缓存

NETCONF 连接建立后系统 SHALL 采集设备 hello capabilities 并按 DeviceID 缓存于连接层（内存，随连接生命周期；断线重连 SHALL 以新 hello 刷新）。SHALL NOT 将能力集持久化到 CRD/磁盘（R03）。

#### Scenario: 连接后能力可查
- **WHEN** 设备完成 NETCONF 会话建立
- **THEN** 该设备的 hello capabilities SHALL 可按 DeviceID 查询

#### Scenario: 重连刷新
- **WHEN** 设备断线重连且新 hello 能力集不同
- **THEN** 缓存 SHALL 反映新能力集

### Requirement: CN-02 按设备模块协商查询

`GET /api/v1/yang/modules?device=<id>` SHALL 返回 `NarrowModulesByCapabilities(该设备 hello caps, 已加载模块)` 的子集；无 `device` 参数 SHALL 保持全量返回（向后兼容）。响应 SHALL 携带 `negotiated` 布尔标记：协商成功 true；设备已注册但能力不可得（离线且建连失败）SHALL 降级返回全量模块并置 `negotiated:false`（R08 可用优先、诚实透出）。`device` 指向未注册设备 SHALL 返回 404 明确错误。

#### Scenario: 设备只声明部分模块
- **WHEN** 设备 hello 仅含 huawei-vlan/huawei-ifm 模块能力，请求 `?device=<id>`
- **THEN** 返回模块 SHALL 仅为 vlan/ifm（+无模块能力例外规则不适用），`negotiated:true`

#### Scenario: 离线降级（负路径）
- **WHEN** 设备已注册但离线且建连失败
- **THEN** SHALL 返回全量模块 + `negotiated:false`，SHALL NOT 5xx

#### Scenario: 未注册设备（负路径）
- **WHEN** `device` 参数指向未注册设备
- **THEN** SHALL 返回 404 明确错误

### Requirement: CN-03 blacklist 注解透出

系统 SHALL 在构建期从 `snd/ce6866p-yang/blacklist.xml` 生成模块屏蔽映射（生成物入库，运行期零 snd 文件依赖），`GET /api/v1/yang/modules` 对命中（模块名+revision 匹配生成映射）的模块项 SHALL 附 `blacklisted:true`（omitempty）。本能力 SHALL NOT 据此裁剪模块（仅注解；硬裁剪待黑名单语义澄清后另行决策）。

#### Scenario: 黑名单模块注解
- **WHEN** 已加载模块命中 blacklist 映射
- **THEN** 该模块项 SHALL 含 `blacklisted:true`，且 SHALL 仍出现在列表中

#### Scenario: 未命中省略
- **WHEN** 模块未命中映射
- **THEN** SHALL NOT 含 `blacklisted` 键
