# 项目记忆索引

- [状态读<get>通道](state-read-get-channel.md) — 做状态相关功能/真机接入前必读：config=false 显示链路已交付(PR#219/#220)、四坑(config壳归一化/payload深层剥除/scrapligo filter双包/vlan无status叶)、覆盖率棘轮新值(后端71.0、前端84/78/77/84)

- [K8s PaaS部署约束](k8s-paas-deployment-constraints.md) — 任何存储/HA/意图层/CRD设计前必读：USMP=K8s内PaaS底座组件、必须多实例、禁本地存储、持久化=CRD(当载体不当架构)；business-network-config 四拍板(公共契约CRD+YANG→CRD codegen、跨设备2PC、意图层多实例就绪、首能力VLAN打通+软归属)；**已全量交付归档(2026-07-16, PR#164-#173)**；**全局HA也已交付(PR#175-#180)**：Device CRD store+Secret凭据、ResolveConn收敛、leader.Gate统一选主、AuditRecord CRD；follow-up全清零(2026-07-17)：旧桥接退役PR#186-#191(Stack A载体归零+纯删除门禁豁免)、硬锁二期PR#192-#193(409拒绝+force审计留痕+前端阻断确认)

- [BGP 2b 波次](bgp-2b-wave.md) — 做 BGP 2b 前必读：策略属性门控目标模型集成的 DAG(tnlm→xpl→rtp→acl→BGP属性)；目标 struct 一期全闭包已生成故①②③零 codegen(仅④补 time-range/l3vpn)；①②③④都改同一 huawei.go 故必串行(TM03)；集成 playbook 6 步；**波次① tunnel-management=PR#155(2026-07-14待merge)**，下一步 merge 后开波次② xpl

- [worktree钩子陷阱](worktree-hooks-gotcha.md) — 在worktree改.githooks后钩子仍跑旧版：相对core.hooksPath解析到主仓库副本；改门禁钩子的提交用 git -c core.hooksPath=$PWD/.githooks；commit-msg与pr-size排除清单两处同步
- [gh监控坑](gh-cli-monitor-gotcha.md) — 本机 gh 不支持 pr checks --json，监控循环会静默死循环；用纯文本 grep 解析
- [合入授权](merge-authorization.md) — CI 全绿的 PR 直接自助 merge（2026-07-18 显式授权），契约破坏/范围扩张仍先确认
- [snd融合四期计划](snd-integration-program.md) — 做YANG基线/左树/i18n/能力协商前必读：snd整包融合拍板(全树+渐进生成、CRD人工role+能力实测、①基线→②角色协商→③左树→④i18n)、基线漂移实测、yang-models依赖触点清单
- [SND驱动注册表](snd-driver-registry.md) — 加新厂商/新YANG模块前必读：注册表(#134)+通用XML编解码引擎(#136-138)+manifest生成管线(#140)已交付、**P5-4剪出(不接非华为设备,能力零成本待命)**、加模块=注册一条描述符+gen.conf加模块名、R04已regen-and-diff化勿手改generated/、namespace须显式登记、注册可达性靠空白导入、golden方法论、超限PR三段拆法
- [scrapligo并发坑](scrapligo-concurrency-pitfalls.md) — 改NETCONF client/排查「持续500 EOF」/合堆叠PR前必读：scrapligo v1.4.0非并发安全(messageID++/Write无锁)+死连接Close死锁+被强杀时内部竞态；opMu串行化+isTransportError自愈已修(#131)；测试注入用优雅关闭别杀sim；**堆叠PR在base先合后再合=合进死分支不到main(#129翻车)**
- [删除语义](config-delete-semantics.md) — 改写链路/删除/netconfsim前必读：声明式通道刻意删不了(walkMap subset)→DELETE命令通道、先移desired再下发、sim已接RFC edit-config(整树替换退役)、opMu写事务串行化、华为vlan key是id；**PR#145「内置接口删不掉」根因=写链路漏发模块顶层容器`<ifm>/<vlan>`发扁平根→设备嵌套树匹配不到,已修为嵌套+Decode锚定容器解vlan同名歧义**
- [呈现元数据收割](ext-ui-annotations.md) — 扩展注解词汇/改FieldDef前必读：模块级扩展不存活运行期schema(task-name走构建期codegen)、3模块仅4种存量扩展、自造ui-*已决策推迟、R04门禁禁提交generated/、readonly-must门禁死锁模式
- [通用模块控制台](generic-module-console.md) — 改 /module 控制台/FieldDef/呈现扩展前必读：交付状态、路径派生规律、presence/删除债、rebase 注意
- [iMaster NCE洞察](imaster-nce-ux-insights.md) — 优化前端/YANG自动渲染前必读：华为AOC把R05做成框架(YANG→四件套)、用render-*扩展把渲染意图编码进模型、YANG类型→ElementPlus控件映射表、NCE设计系统；全文+截图在 docs/research/imaster-nce-ux-insights.md

- [test-server进程泄漏](test-server-process-leak.md) — 「8080被占/后端起不来」先查：cmd/test-server 编译成 /tmp/tsrv 后台跑，worktree删了进程不清理、长期霸占8080；ss+/proc定位，sudo kill

- [YANG约束引擎](yang-constraint-engine.md) — 改约束/加渲染前必读：when/must/pattern/range/leaf-list 数据驱动已交付(PR#116 P1/P2/P4)；choice待办需构建期codegen(yang-models是仅构建期submodule、运行镜像不含)；元数据都存活于ygot内嵌goyang Entry.Extra/Exts(choice拍平旧判断已被实测推翻)

- [测试军规](test-governance-military-rules.md) — 改测试/加用例前必读：改动类型→必补层映射(§5.6)、覆盖率棘轮基线值(后端57.8/前端74/71/67/74)、加用例后要上调阈值、2个follow-up

- [对账收敛三根因](reconcile-convergence-3rootcauses.md) — 设备配置「一直漂移」= diff无Map分支/edit-config序列化漏配/回读解析恒空 三叠加；IFM+VLAN均已修(分支 worktree-fix-ifm-interface-drift 三提交)

- [双栈半迁移架构](dual-stack-migration.md) — 历史背景（双栈并存期）；2026-07-17 起 Stack A 代码载体已归零(actor/crdsource/translator/api-biz-v1 全物理删,SC-01 禁重引)，现状看 arch-optimization-roadmap
- [架构优化路线](arch-optimization-roadmap.md) — **全部债务清零(2026-07-18)**：单栈单进程、Stack A载体清零(SC-01禁重引)、D3 plugin/D5 gNMI空壳已删(gNMI=规划能力,工厂显式未实现错误)。历史脉络参考
- [OpenSpec CLI](openspec-cli.md) — 装 CLI 用 @fission-ai/openspec（非裸 openspec 空壳）；需 Node≥20
- [前端 CI 坑](frontend-ci-gotchas.md) — Actions 设 local_only 会让所有工作流 startup_failure（像卡住）；vitest4 需 Node≥20.12（本机 Node18 跑不了）；lockfile 漂移已修(#26)
- [后端 CI flaky](backend-ci-flaky-tests.md) — compliance -race 时序测试偶发挂（TestDelayingQueueAddAfter 等）；空提交重触发即可，勿调试
- [CI/CD 自托管](cicd-self-hosted.md) — 仓库已转 PRIVATE；基础门禁留 GitHub，部署+全栈E2E+常驻staging 上 Mac 自托管 Runner(macos-staging)；含 E2E v2 债(旧规格断言虚构契约)
- [spec格式标准](spec-format-standard.md) — 改/写 spec 前必读：主 spec 对齐 OpenSpec CLI 标准(Purpose/Requirements/Requirement/Scenario)、delta 新增用 ADDED；**军规:刷 spec 先于开发**；CI 只卡改动的 spec；渐进逐能力迁移+借机修正与代码不一致
- [对账连接信息债](reconcile-conninfo-debt.md) — 协议+凭据无共享device store,多子系统各自残缺建连;致「新增接口」下发失败(unsupported protocol + ssh none auth);PR#100最小兜底(ProtocolAUTO+admin/admin兜底)解封,根治=加共享device store另开PR
- [kind部署坑](kind-deploy-gotchas.md) — kind/真集群部署前必读：一键脚本 scripts/kind-deploy.sh、compose 掩盖的两坑（旧桥接CRD缺失崩溃#182、非root绑830要sysctl#183）
- [WSL华为内网部署](deploy-wsl-huawei-proxy.md) — 在华为内网 WSL 上 docker compose 部署 uSMP 跑通：TLS拦截(关校验/GOINSECURE/apk-http)+代理用宿主IP:3131+docker0挪到172.28腾出192.168.1.0/24；hack禁commit main、WSL重启后重建要换代理IP
- [BGP接入依赖](bgp-onboarding-deps.md) — 接BGP或规划其依赖模型前必读：peers全在network-instance augment下(公网邻居也不独立)→ni是peering唯一硬前置(103行极易);routing软must可摘除(甩掉bgp/routing/ethernet重分支);强制字段零跨模型依赖、策略引用全可选;分期DAG(本期公网BGP依赖0→ni→2a基础邻居→2b策略门控tnlm/xpl/rtp/acl);change huawei-bgp-public-config已就绪可apply
- [设备配置·StackB](vlan-config-stackb.md) — 华为VLAN+接口(IFM)全属性配置已交付,泛化为通用流；链路唯一=Stack B直连(/yang/schema+/config)；**前端CRD死路已整链退役(#143,2026-07-13)**；概念分层:原生配置=模块控制台、业务网络配置=未来编排层(openspec/tasks/business-network-config.md,business*标识符已清场)
- [前端契约生成](frontend-contract-gen.md) — 前端测试能力改造**全交付**：0层锁Node22 / 1a契约生成+漂移门禁 / 1b全API面 / 1c typecheck门禁+清34错 / 层2 Vitest Browser Mode / 层4 Storybook；配套体积门禁排除生成物(#60,加重依赖前必做)。踩坑：swag2.0、git--exit-status、ref-UnwrapRef、playwright双版本歧义
- [前端重设计](frontend-redesign.md) — 商用级高保真原型**已批准并合入main**(#78+#79)；浅色iMaster NCE气质+收敛台账+新鲜度环+模型驱动YANG树；落地计划见 docs/superpowers/plans/2026-07-05-frontend-design-landing.md + openspec/tasks/同名
- [落地迭代工作方式](frontend-landing-workflow.md) — 每PR用独立agent检视设计/测试/开发；CI过后自助merge；关键风险记risklog供次日规划；认真测试设计；长迭代注意上下文压缩保留关键信息
- [落地风险台账](frontend-landing-risklog.md) — PR-B1对账态(drifted=已纠正非持久/status无TTL/desired 5min过期/fleet不含unknown)；PR-B2(config GET不缓存、force_refresh是TODO、与§8矛盾)
