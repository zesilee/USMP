## MODIFIED Requirements

### Requirement: UI-01 语言切换与持久化

界面 SHALL 支持 zh-cn/en-us 双语：Header 或设置页提供切换入口；选择 SHALL 持久化于 localStorage 并在下次加载生效；默认 zh-cn。切换 SHALL 即时生效（无需刷新），且 SHALL 同步 **UI 适配层导出的组件库 locale**（由适配层内部联动，业务代码 SHALL NOT 直接依赖具体组件库的 locale 机制）。

#### Scenario: 切换即时生效并持久化
- **WHEN** 用户切换到 en-us
- **THEN** 导航/按钮/**适配层控件**文案 SHALL 即时变为英文；刷新后 SHALL 保持 en-us

#### Scenario: 组件库替换不影响语言联动（换库锚点）
- **WHEN** 底层组件库实现发生替换
- **THEN** 语言切换与控件 locale 联动行为 SHALL 保持不变，改动 SHALL 限于适配层

### Requirement: UI-02 界面文案全覆盖

全部界面 chrome 文案（导航、布局、视图标题、按钮、提示、空态、校验消息模板）SHALL 经 i18n locale 文件提供 zh-cn/en-us 两份；**前端源码目录** SHALL NOT 残留硬编码界面中文（用户业务数据除外；以中文字符扫描清点为完成判据）。左树与模块双语名 SHALL 按当前语言取 zh/en 字段。词表**键名 SHALL 保持稳定**，SHALL NOT 因 i18n 框架更替而重命名。

#### Scenario: 左树随语言切换
- **WHEN** 语言为 en-us
- **THEN** 左树分组 SHALL 显示 en-us 名（如 Ethernet Switching），zh-cn 时显示中文

#### Scenario: 无残留硬编码（负路径）
- **WHEN** 扫描前端源码目录中的中文字符（排除 locales/assets/测试）
- **THEN** SHALL 为零残留

#### Scenario: 词表键名跨框架稳定
- **WHEN** i18n 框架实现发生更替
- **THEN** 词表键名与取值结果 SHALL 保持一致，界面文案 SHALL NOT 出现缺档回退
