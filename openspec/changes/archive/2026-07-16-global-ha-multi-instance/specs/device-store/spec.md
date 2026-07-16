# device-store delta — CRD 持久化共享 + 兜底收敛

## MODIFIED Requirements

### Requirement: DS-01 共享连接信息注册表

系统 SHALL 提供 Manager 级 `DeviceStore`，键为 DeviceID（裸 IP，与 ConfigStore desired 同键），值为完整 `DeviceConnectionInfo`（含 `Vendor` 厂商标识）。`Manager` 接口 SHALL 暴露 `GetDeviceStore()`。集群模式下实现 SHALL 以 K8s `Device` CR 为持久载体（SC-02：CRD 仅当载体）、进程内 watch 镜像承接读路径（`Get`/`List` SHALL NOT 逐次访问 apiserver）；无可达集群时 SHALL 自动降级为纯内存实现（R08，现行为）。实现 SHALL 无自管数据库（R03），并发 `Put/Get/Delete/List` SHALL 无数据竞态（R09）。

#### Scenario: 注册后解析完整连接信息
- **WHEN** `Put(id, info)` 后 `Get(id)`
- **THEN** SHALL 返回完整 `DeviceConnectionInfo`（含凭据与协议），`ok=true`

#### Scenario: 写穿后本地即时可读（无陈旧窗口）
- **WHEN** 集群模式下 `Put(id, info)` 成功返回后立即 `Get(id)`
- **THEN** SHALL 返回该设备（write-through 先行更新镜像，不等 watch 回流）

#### Scenario: Vendor 随连接信息透传
- **WHEN** `Put(id, info)` 时 `info.Vendor="huawei"`，随后 `Get(id)`
- **THEN** 返回的 `DeviceConnectionInfo.Vendor` SHALL 为 `"huawei"`

#### Scenario: Vendor 零值缺省语义（边界）
- **WHEN** 存入的 `DeviceConnectionInfo.Vendor` 为空字符串（存量数据）
- **THEN** 消费方 SHALL 按 `huawei` 缺省解读并 SHALL NOT 失败（R08 降级）

#### Scenario: 未注册设备
- **WHEN** `Get` 一个未注册 DeviceID
- **THEN** SHALL 返回 `ok=false`（调用方据此降级，SHALL NOT panic，R08）

#### Scenario: 并发读写安全
- **WHEN** 多协程并发 `Put`/`Get`/`Delete`/`List`
- **THEN** SHALL 无数据竞态（`-race` 通过，R09）

#### Scenario: 无集群自动降级内存实现
- **WHEN** 启动时无可达 kubeconfig
- **THEN** DeviceStore SHALL 降级为纯内存实现并记日志提示，全部接口行为与现状一致，进程 SHALL NOT 崩溃（R08）

### Requirement: DS-03 设备注册写入库

DeviceHandler SHALL 把 `AddDevice`/`RemoveDevice` 写入/删除 DeviceStore（含 `Vendor`，未指定时缺省 `huawei`）；`/api/devices`（含在线探活）与配置回读 SHALL 从库读取。设备连接信息 SHALL NOT 再散落于各处私有结构。种子设备 SHALL NOT 硬编码于代码：仅内存降级模式下经可选环境变量 `USMP_SEED_DEVICE` 注入（供本地开发/E2E）；集群模式下设备集合完全来自 `Device` CR。

#### Scenario: 环境变量种子设备进库（降级模式）
- **WHEN** 无集群降级模式启动且设置了 `USMP_SEED_DEVICE`
- **THEN** DeviceStore SHALL 含该种子设备（Port 缺省 830、Protocol=AUTO、Vendor 缺省 `huawei`）

#### Scenario: 未设种子则空库启动
- **WHEN** 无集群降级模式启动且未设置 `USMP_SEED_DEVICE`
- **THEN** DeviceStore SHALL 为空，`/api/v1/devices` 返回空列表，SHALL NOT 崩溃

#### Scenario: 集群模式忽略种子变量
- **WHEN** 集群模式启动且设置了 `USMP_SEED_DEVICE`
- **THEN** 设备集合 SHALL 仅来自 `Device` CR，种子变量 SHALL 被忽略并记日志说明

## ADDED Requirements

### Requirement: DS-04 凭据 Secret 引用（不明文进 etcd）

集群模式下设备凭据（Username/Password）SHALL 存放于同 namespace K8s Secret，`Device` CR spec SHALL 仅存 `credentialsSecretRef` 引用，SHALL NOT 在 CR 中承载明文凭据。写入顺序 SHALL 先 upsert Secret 再 upsert CR；删除 SHALL 先删 CR 再删 Secret。存在性以 CR 为权威：有 CR 无 Secret 时 `Get` SHALL 返回空凭据的连接信息并记日志（下游 clean fail，R08），SHALL NOT panic。

#### Scenario: Put 落两资源
- **WHEN** 集群模式 `Put(id, info)`（含凭据）成功
- **THEN** apiserver 中 SHALL 同时存在该设备的 `Device` CR（无明文凭据字段）与凭据 Secret，`Get(id)` SHALL 还原完整凭据

#### Scenario: Secret 缺失降级空凭据
- **WHEN** `Device` CR 存在但其引用的 Secret 被外部删除
- **THEN** `Get(id)` SHALL 返回 `ok=true` 且凭据为空、记警告日志；下游建连按空凭据 clean fail，SHALL NOT panic

#### Scenario: apiserver 不可达时写入失败可见
- **WHEN** 集群模式下 `Put`/`Delete` 期间 apiserver 不可达
- **THEN** SHALL 向调用方返回错误（API 层呈现 5xx 信封），本地镜像 SHALL NOT 先行变更

### Requirement: DS-05 跨副本共享与重启恢复

集群模式下任一副本对设备的增删 SHALL 经 CR watch 反映到所有副本的 `Get`/`List`（最终一致）；任一副本重建后 SHALL 从 CR 完整恢复设备集合（SC-06 实例无状态可替换）。

#### Scenario: 副本间可见
- **WHEN** 副本 A `Put(id, info)` 成功
- **THEN** 副本 B 的 `Get(id)`/`List()` SHALL 在 watch 收敛后含该设备

#### Scenario: 重启恢复
- **WHEN** 已注册 N 个设备后实例重建
- **THEN** 新实例 DeviceStore SHALL 从 CR 恢复全部 N 个设备（含经 Secret 还原的凭据）

### Requirement: DS-06 建连解析统一 helper（兜底收敛）

`device` 包 SHALL 提供唯一的 `ResolveConn(store, deviceID)` helper：已注册返回库中完整连接信息；未注册返回 `{IP, Protocol: AUTO}`+空凭据并记统一日志。全部消费方（各模块 reconciler、intent 2PC、config 读/删路径、device API 探活）SHALL 经此 helper 解析建连参数，SHALL NOT 各自内联未注册兜底逻辑。

#### Scenario: 已注册走库
- **WHEN** 设备已注册且 reconciler 经 `ResolveConn` 解析
- **THEN** SHALL 返回库中完整连接信息（协议/凭据/厂商）

#### Scenario: 未注册统一兜底
- **WHEN** 未注册设备经 `ResolveConn` 解析
- **THEN** SHALL 返回 `Protocol=AUTO`+空凭据并记统一格式日志，SHALL NOT panic（行为与收敛前各内联副本一致）
