# EviewUI 闸门离线验证工具包（方案 B 留档）

> 背景：EviewUI 实现包不出内网（用户合规拍板），闸门 1.3/1.4 的运行时行为
> 验证改为「工具包去离线机跑、纯文本报告带回」。本目录是 run.js 的仓库留档；
> **可执行发货物**在线侧构建为 `eview-gate-kit.tgz`（本 run.js + 自带
> node_modules：openinula@1.0.0 / inula-intl@1.0.35（cjs 目录已补
> type:commonjs patch）/ happy-dom@12），交付路径见 change tasks.md 组 1。

## 离线机用法

```bash
tar xzf eview-gate-kit.tgz && cd eview-gate-kit
node run.js --selftest                              # 先自检（不碰 EviewUI）
node run.js <前端工程目录> > gate-report.txt 2>&1    # 工程目录=含 node_modules/@nce/eview-react
```

把 `gate-report.txt` 全文带回即可。要求 Node ≥16（报告首行会回传实际版本）。

## 机制备忘（在线侧演练已验证）

- CJS `Module._resolveFilename` 钩子：react/react-dom/react-intl/@cloudsop/horizon* → 工具包自带 openinula/inula-intl；样式与静态资源扩展名空模块化。
- happy-dom 全局注册必须**批量挂全部大写构造器**——inula 事件系统引用 `HTMLInputElement` 等任意全局类，缺一个就静默吞事件（踩坑实录）。
- 场景 V0~V7 相互隔离（单场景崩溃不影响其余），失败时输出 DOM 快照供远程迭代。
- happy-dom 定时器使进程不退出：结束显式 `process.exit`；管道 `| tail` 会因此看似挂死。
