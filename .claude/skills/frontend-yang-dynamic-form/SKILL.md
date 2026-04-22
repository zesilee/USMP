---
name: frontend-yang-dynamic-form
description: 根据ygot生成的YANG模型结构，前端自动渲染表单、树形菜单、输入框、下拉框、开关、批量提交、实时预览
---

# 技能规范：激活时机 + 核心原则 + 使用样例

## 一、激活时机
1. 用户需求包含「前端」「界面」「表单」「YANG页面」「动态表单」自动激活
2. 后端YANG模型、ygot结构体、controller/reconciler完成后自动生成对应前端
3. 需要展示设备配置、编辑配置、下发配置时自动触发
4. 与后端所有技能联动：yang-controller-runtime、NETCONF、缓存、TDD

## 二、核心原则
1. **模型驱动UI**：YANG = 前端结构，不手写任何配置页面
2. **实时性**：每次打开表单自动从 controller reconciler 拉取最新配置
3. **无状态**：前端不保存任何配置，所有数据来自后端 yang-controller-runtime
4. **类型自动映射**：
   - boolean → 开关
   - enum → 下拉选择框
   - string → 输入框
   - int/uint → 数字输入框
   - list → 表格 + 新增行
   - container → 分组面板
5. **一键下发**：提交后直接发给 controller reconciler → NETCONF 下发设备
6. **故障可见**：下发成功/失败、设备离线、超时、缓存状态全部展示
