---
name: view-ui-insights
description: "iView/View UI 组件库调研结论 — 不换库,可借鉴表格导出/LoadingBar/Split 三个模式;全文在 docs/research/view-ui-insights.md"
metadata: 
  node_type: memory
  type: reference
  originSessionId: 8bc7d355-8e51-4654-be90-104c05535f53
  modified: 2026-08-10T02:12:39.909Z
---

2026-08-10 调研 iView/View UI(用户拼写 "eivew-ui",实为 iView;Vue3 版=view-ui-plus)。

核心结论:
- **不迁移组件库**:View UI Plus 430 star vs Element Plus 27.7k(差 60 倍)、近乎单人维护(2026-03 仍发版但间隔数月);F3 测试资产全在 Element Plus 上,迁移=纯损耗+撞 R10。
- npm 包谱系防坑:`iview`(Vue2 已死 2019 停更)→`view-design`(Vue2)→`view-ui-plus`(Vue3 存活);网上大量教程指向死包。
- 设计再印证:iView(#2d8cf0 蓝/浅色/高密度)与 NCE 同族审美,14px 字号+32px 默认控件高=企业后台事实标准,我们设计系统不用改(见 [[imaster-nce-ux-insights]] 对应记忆 imaster-nce-ux-insights)。
- 可借鉴三模式(未立项):①台账导出 CSV(嵌套字段先拍平;大表走后端快照出口配合 [[list-server-pagination]],前端导出只限当前页/小结果集) ②全局 LoadingBar 细进度条(下发→回读在途反馈,与新鲜度环互补,自实现几十行不引库) ③左树宽度可拖拽(Split 模式)。多页签 tag-nav 拍板默认不做。

全文+证据分级:docs/research/view-ui-insights.md
