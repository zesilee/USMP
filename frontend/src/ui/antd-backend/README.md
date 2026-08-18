# antd 测试后端镜像（窗口期基建，非生产代码）

生产真身 = `src/ui/eview/*`（EviewUI 桥）。但 EviewUI 实现包不出内网（方案 B），
外网测试环境没有真组件——业务 F1/F2 若打到桥会渲染空 stub，行为断言全灭。

故 vitest 的 app 工程（非桥测试）把 `src/ui/eview/components/*`、`src/ui/eview/feedback`
与 `src/ui/provider` 别名到本目录：**纯 re-export antd**（对外形态与桥一致=antd 形态），
业务测试拿到真实组件行为。桥行为等价性由内网校准套件（eview-real）+组 6 F3+组 7 E2E 兜底。

- 别名清单在 vitest.config.ts（app 工程）；storybook viteFinal 同套。
- 桥测试工程（bridges）不经过本目录，打真桥（@nce 子路径 stub）。
- typecheck 全局走真身桥类型（本目录不参与类型面）。
- 回收点：波 C 或外网可得真包时退役。
