# yanggen 生成约定规格（冻结自 ygot@v0.29.20 生成物实测，2026-08-11）

> 自研生成器必须复刻的结构约定（YN-01「字节级冻结」的展开）。来源：对
> `internal/generated/huawei/`（3068 struct/819 enum/21 union）的独立 agent 全量研究。
> 「零消费符号」章节列出可以不生成的部分（按 backend 非测试代码 grep 实证）。

## 1. 命名

- **struct 类型名**：`CamelCase(根模块名)_CamelCase(路径段)...`，`_` 连接；不含 choice/case 段；augment 子树用**宿主路径**命名（module tag 才体现来源）。fakeroot=`Device`（路径 `/device`）。
- **字段名** = goyang `yang.CamelCase`（构建期直接调用，勿自写）：`-`/`.`→`_` 归一；首分隔符→`X`；`_`+小写→丢弃并大写；`_`+数字/大写→**保留 `_`**；数字后小写视为新词首；`Ietf`→`IETF`。例证：`ce-vlan-value-8021p`→`CeVlanValue_8021P`、`dot1q-vid`→`Dot1QVid`、`m-lag`→`MLag`、`group6s`→`Group6S`。Go 关键字无需特判（首字母大写天然避开）。
- **重名兜底**：尾部追加 `_`（当前闭包零命中）。

## 2. 类型映射与 tag

| YANG | Go |
|---|---|
| 标量 leaf | 指针 `*string/*uintN/*intN/*bool` |
| enumeration/identityref | 非指针 `E_Xxx` |
| empty | 非指针 `object.Empty`（原 YANGEmpty，消费方按 reflect bool kind 识别） |
| binary | `object.Binary`（当前闭包 0 处） |
| 嵌套容器 | 指针 `*T` |
| 单键 list | `map[K]*T`（K=值类型/E_*/union 接口） |
| 复合键 list | `map[T_Key]*T` |
| ordered-by user list | `*T_OrderedMap`（仅 2 处，tunnel-policy） |
| 无 key list | `[]*T`（当前闭包 0 处，可延后） |
| leaf-list | `[]T`（元素非指针；含 `[]E_*`、`[]X_Union`） |
| -ignore_unsupported 节点 | 非指针 `interface{}`（68 处） |

- **tag**：`` `path:"<yang名>" module:"<belonging module>"` `` 顺序固定、单元素、无 `/`；augment 时 module=来源模块；**`_Key` struct 字段只有 path 无 module**。
- **字段顺序**：普通 struct 按 **Go 字段名字典序**；`_Key` struct 按 **YANG key 语句顺序**。

## 3. 方法（自研最小面）

- 每 struct：`IsYangObject()`（替 IsYANGGoStruct，空实现指针接收者）。
- list 成员：`ListKeyMap() (map[string]interface{}, error)`（替 ΛListKeyMap）——指针型 key 先 nil 检查（错误文案冻结 `"nil value for key %s"`，%s=Go 字段名）再解引用；非指针 key（enum/union/interface{}）直接取值；返回 map key=YANG 叶名、按 key 语句顺序。
- `_Key` struct：命名 `<父List类型>_Key`，值接收者 `ListKeyMap()`（无 nil 检查无 error 分支）。
- **不生成**（零非测试消费实证）：`Validate/ΛValidate`、`ΛEnumTypeMap/ΛEnumTypes`、`ΛBelongingModule`、`New<List>` 构造器、`To_*_Union` helper、`IsYANGGoKeyStruct`、OrderedMap 的 Append/GetOrCreate 族（保留类型+init/Get/Keys/Values/Len/Append 最小集使字段可编译）。
- huawei 包 `Validate` 零消费；business 包 `Device.Validate` 被 `intent/cr.go` 用（阶段5 换 IR 校验器）。

## 4. 枚举

- 类型名：typedef→`E_<CamelCase(定义模块)>_<CamelCase(typedef名)>`；内联→`E_<所在struct类型名>_<CamelCase(叶名)>`；union 内 typedef enum 追加 `_Enum`。底层 `int64`。
- 常量：`<去E_类型名>_<safe(值名)>`，`..._UNSET=0`，数值=**YANG value+1**。
- 值名净化：`. - / 空格`→`_`，`+`→`_PLUS`，`,`→`_COMMA`，`@`→`_AT`，`$`→`_DOLLAR`，`*`→`_ASTERISK`，`:`→`_COLON`，**`|`→`_OR_`（genfix 行为内建）**；大小写原样。
- 映射表：`map[string]map[int64]object.EnumDefinition`，外层 key **带 `E_` 前缀**（xmlcodec 按 `fv.Type().Name()` 查表，失配即静默解码错）；`Name` 存**未净化原始值**（`"50|100GE"`）。仅 identityref 填 `DefiningModule`（identity 名不做 CamelCase）。
- 方法：`IsYangEnum()`、`EnumMaps()`、`String(){ object.EnumLogString(e, int64(e), "E_<全名>") }`。

## 5. union（21 类型/6 接口）

- 接口名：`<目标叶struct路径>_<CamelCase(叶名)>_Union`（leafref 指向 union 时取**目标叶**命名）；唯一方法 `Is_<接口名>()`（指针接收者）。
- 包装类型：`<接口名>_<成员Go类型CamelCase>`，唯一字段名=字段类型名本身（`type X_Union_String struct { String string }`）。
- 字段声明非指针接口；leaf-list union=`[]X_Union`；union 作 list key 时 `map[X_Union]*T` 且 ListKeyMap 无 nil 检查。

## 6. 包级导出（自研版）

- 必须：全部 struct/enum/union 类型、`Device`、类型注册表（Scheme 数据，替 SchemaTree 的「按类型名索引」用途——注意 SchemaTree 在 drivers 里被用来取 `*yang.Entry` 给 xmlcodec，阶段4 改读 Schema IR）、RFC7951 `Unmarshal`（阶段3 生成式方法承载）。
- 退役：`ySchema` gzip blob、`UnzipSchema`、`Schema()`（IR 化后 schemagen 直读 YANG 源）。
- 文件布局：structs 按类型名字典序 `ceil(total/N)` 均分到 `structs-0..N-1.go`；`_Key`/OrderedMap 随父容器同文件；enum.go/enum_map.go/union.go 独立。

## 7. 生成管线不变式

- gen.conf 键：`yang_path`（逗号多目录，snd 源+deviations）、`modules`（空格分隔）、`generate_fakeroot`、`compress_paths`、`split_count`。
- goyang 解析：等价 `-ignore_unsupported=true`；deviation 模块（usmp-deviations）列在 modules 里由 goyang 自动应用。
- 确定性内建：模块/struct/enum/常量全排序；无机器路径头注释。
- 延期模块勿加回：huawei-pic（跨模块 submodule typedef）。
