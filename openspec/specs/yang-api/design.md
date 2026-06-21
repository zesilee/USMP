# yang-api - 架构设计

## 请求处理流程

### GET /api/v1/yang/modules
```
Request → YangHandler.ListModules()
  → manager.GetSchema().Modules() 遍历已加载模块
  → 每个模块提取: Name, Root.Name, Root.Description, Root.Type
  → vendor固定为"huawei"
  → 无模块 → 返回硬编码示例列表(huawei-ifm, huawei-vlan)
  → Success(modules)
```

### GET /api/v1/yang/schema/:module
```
Request → YangHandler.GetSchema()
  → c.Param("module") 获取模块名
  → switch module:
    → "huawei-ifm" / "Interfaces" → 预定义IFM Schema
    → "huawei-vlan" / "VLANs" → 预定义VLAN Schema
    → default → 通用Schema(name + description)
  → Success(schema)
```

## 依赖关系

| 依赖 | 用途 | 调用位置 |
|------|------|----------|
| manager.Manager | 获取Schema | ListModules |
| Schema.Modules() | 遍历已加载YANG模块 | ListModules |

## 错误处理策略

- **无错误场景**：两个端点在任何输入下均返回200成功
- **未知模块回退**：请求不存在的模块名时返回通用Schema而非错误
- **空模块回退**：无已加载模块时返回硬编码示例

## 硬编码数据

当前Schema全部硬编码在handler中，未从YANG模型文件动态生成：
- ListModules: 无模块时返回2个示例
- GetSchema: IFM和VLAN各有一套预定义FieldDef，未知模块返回通用2字段Schema