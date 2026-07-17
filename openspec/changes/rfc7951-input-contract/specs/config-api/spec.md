## MODIFIED Requirements

### Requirement: BR-05 声明式下发

`POST /api/v1/config/:ip/*path` 的请求体 SHALL 为**以 path 为根的 RFC7951 子树**（YANG 真名、枚举用名字字符串）；服务端 SHALL 将其解码为强类型 ygot 结构 → 存入 ConfigStore → 触发对账，返回 `status="ACCEPTED"`。下发即接受语义：配置**存储成功即返回**，实际对齐设备由异步对账完成。ConfigStore SHALL 只存类型化结构，SHALL NOT 存原始 map。

#### Scenario: 下发被接受
- **WHEN** 提交合法 YANG 路径 + RFC7951 子树
- **THEN** SHALL 存入 ConfigStore、触发对账，返回 `ACCEPTED` + `reconciliation.triggered`

#### Scenario: 表单子路径扁平载荷
- **WHEN** 对 `/system:system/system:system-info` 提交 `{"sys-name": "sw-01"}`
- **THEN** SHALL 解码为对应容器的类型化结构并接受（锚点相对包裹语义）

### Requirement: BR-06 类型转换路由

下发解码 SHALL 经 driver 描述符注册表单一路径：按 path 查得编码描述符 → 按其编码锚点（DR-05）把请求体机械包裹为锚点相对 RFC7951 → `ygot.Unmarshal` 根级解码。未注册路径、path 与锚点不构成前缀、path 段含 list 谓词、或解码失败时 SHALL 返回信封码 400（错误信息透出原因），SHALL NOT 回退手写转换器、SHALL NOT 静默存原始 map。

#### Scenario: 注册表命中并解码
- **WHEN** path 含 `ifm:ifm/ifm:interfaces` 且请求体为 `{"interface":[...]}`
- **THEN** SHALL 经 ifm 描述符解码为 `HuaweiIfm_Ifm_Interfaces`

#### Scenario: 未注册路径显式拒绝（负路径）
- **WHEN** 提交无编码描述符覆盖的路径
- **THEN** SHALL 返回信封码 400 并说明路径未注册，SHALL NOT 存入 ConfigStore

#### Scenario: 旧形状拒绝（负路径）
- **WHEN** 提交 legacy 形状（如复数键 `{"vlans":[...]}` 或 camelCase 叶名）
- **THEN** SHALL 返回信封码 400 透出 ygot 解码错误，SHALL NOT 下发
