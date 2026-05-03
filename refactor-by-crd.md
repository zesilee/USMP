一、整体架构规则
采用前端直连 K8s API Server 架构，不新增任何中间转发服务。
前端原生支持 List + Watch CRD 资源，长连接实时监听变更，无前端轮询。
后端仅作为 K8s Controller，不提供自定义 REST 接口，只负责 CR 监听、配置翻译、NETCONF 下发、设备状态回写。
整套系统仅两类资源：1 个原生 CRD + 4 个业务 CRD，所有模型必须包含 spec（期望配置）+ status（设备真实状态）。
南向统一基于 Scrapligo + NETCONF Base11 优先、Base10 自动降级，适配真实交换机 / KNE / GNS3 模拟设备。
二、CRD 定义约束
1. 原生配置（仅唯一 1 个 CRD）
CRD：NativeDeviceConfig，组 core.usmp.io/v1
能力：承载设备全量原始配置，支持 yang/xml/cli，纯透传、不翻译、不解析。
spec：deviceId、configType、content、encrypt
status：同步状态、同步时间、错误信息、设备真实配置回显
2. 业务配置（固定 4 个 CRD）
组 biz.usmp.io/v1
BusinessSwitch：交换机基础管理、厂商、管理 IP、运维状态
BusinessVlan：VLAN 业务参数，屏蔽厂商差异
BusinessInterface：接口模式、Access/Trunk、VLAN 绑定
BusinessRoute：静态路由业务配置
所有业务 CRD 统一结构：spec 业务期望字段，status 由控制器回填设备真实运行状态、同步结果、错误信息。
三、后端 Controller 强制要求
统一监听 5 个 CRD 全生命周期事件：增 / 改 / 删，走标准 Reconcile 调和逻辑。
双路径下发隔离：
业务 CRD：CR Spec → 翻译引擎 → 标准 IETF YANG → 厂商适配 YANG → NETCONF 下发
原生 CRD：直接透传 content 原文下发，不做任何字段修改
状态闭环：配置下发后主动从交换机拉取真实配置，回填到 CR status，供前端 Watch 感知。
内置：失败重试、错误日志、状态标记、配置同步失败信息回写。
翻译引擎模块化：支持 Cisco / 华为 适配，可后续扩展厂商。
南向封装 Scrapligo 连接池，自动兼容 NETCONF base10/base11。
四、前端强制规范
直接调用 K8s 原生 API，携带 SA Token、RBAC 鉴权。
必须实现：
List：分页、按 deviceId 筛选 CR 资源
Watch：长连接监听 ADDED/MODIFIED/DELETED，自动刷新界面
增删改 CR 资源，分别操作业务 CRD / 原生 CRD
双界面并存：
业务配置页：展示 spec 期望配置 + status 设备真实状态
原生配置页：编辑查看 NativeDeviceConfig 全量原始配置
不做后端接口适配，完全基于 K8s CRD 驱动页面数据与状态。
五、权限与部署约束
使用 K8s 原生 RBAC + ServiceAccount，给前端最小 CRD 读写 List/Watch 权限。
敏感配置支持加密存储，控制器自动加解密。
整体架构无多余中间服务，最小侵入、云原生原生架构。
六、代码输出要求
基于 Kubebuilder 生成 5 套 CRD Go 结构体 + CRD 部署 YAML。
完整 后端 Controller：监听逻辑、Reconcile、下发、状态回填。
业务→IETF YANG→厂商 YANG 翻译引擎完整实现。
Scrapligo 南向驱动封装、连接池、NETCONF 版本自适应。
前端 List/Watch/CRUD 调用 K8s API 示例代码。
全套部署 YAML：Controller、RBAC、SA、CRD 资源。