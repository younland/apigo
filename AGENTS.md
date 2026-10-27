# 全局 Agent 工作流

本文件约束自动化代理在本仓库中的默认工作方式：最短路径、风险分级、验证门禁与可验证交付。
框架知识见 skill `.agents/skills/mygo-maintenance`（MyGo 官方维护技能）；项目定位、技术栈、命令与目录结构见 `README.md`；README 写作规范见 `.agents/readme/SKILL.md`；提交信息规范见 `.agents/commit/SKILL.md`。

## 指令优先级

1. 当前会话中用户的明确要求
2. 仓库规则与本文件的工作流规则
3. 通用习惯

只读/分析类任务不进入实现流程，但结论必须有依据、可追溯。

## 默认原则

- **最短路径**：采用满足质量要求的最轻流程；能直接完成并验证的，不升级为更重流程。
- **自包含**：Debug 先收集证据、定位根因，再修改；Review 独立检查；完成前必须运行与风险相称的验证。
- **流程升级**：影响面超出初始判断、涉及共享逻辑/请求执行链路/持久化格式/需求不清或验证手段不足时，升级流程。
- **流程降级**：改动局部、边界清晰、验证简单时，及时降级，避免形式成本。

## 任务分流

| 类型 | 处理方式 |
|---|---|
| 只读（分析/解释/阅读） | 直接处理，给出可追溯结论 |
| 实现（新功能/修复/重构） | 先澄清目标、边界、验收标准与验证方式；小任务轻量 planning（目标/边界/风险/验证四要素），大任务再升级 |
| Debug | 不猜测式修补：先收集日志/复现/渲染证据，定位根因后再修，修复补贴近故障表面的回归保护 |
| Review | 检查规格符合性、正确性、边界与错误路径、回归风险、可维护性；反馈先验证成立性再处理 |
| 原生界面（ui 视图） | 明确布局对齐/主题（亮暗）/缩放（100%–150%）行为；改后跑 `CGO_ENABLED=0 go test ./...`（含 ui.Tester 视图测试），必要时真实窗口截图核验（项目经验见 `.agents/skills/mygo-maintenance/references/project-experiences.md`） |

## 验证与交付门禁（硬门禁）

- **不得虚构**已运行的命令、退出码或验证结果；关键验证无法执行时必须说明原因，并降低完成度表述。
- 本项目标准验证命令（MyGo 不需要 cgo），日常走 Task，CI 与本地同一入口：
  - 全量：`task check` = `go build ./...` + `go vet ./...` + `go tool mygo vet .`（元素值/goroutine 捕获）+ `CGO_ENABLED=0 go test ./...` + `go test -race ./...`（竞态检测，需要 cgo）
  - 并发或共享状态改动：`task test:race`
  - 环境自检：`task doctor`（`go tool mygo doctor`）
  - 打包冒烟：`task package`（Windows）、`task package:darwin`、`task package:linux ARCH=arm64`；浏览器版 `go tool mygo build -platform web`
  - 工作流校验：改动 `.github/workflows/**` 或 `Taskfile.yml` 后跑 `go run github.com/rhysd/actionlint/cmd/actionlint@latest .github/workflows/check.yml .github/workflows/release.yml`，并确认 CI 调用的任务名（`task check`、`task package:windows` 等）真实存在
- **GUI 验证**：界面变更优先用 `ui.NewTester` 离屏渲染 + 真实窗口截图；无人值守跑不了 GUI 时必须声明，不得声称"界面正常"。
- **交付门禁**：声明"完成/已修复/可提交"前必须完成上述相关验证并阅读实际输出。

## 测试策略（TDD 按风险判定，非默认全量）

- **A 直接修改 + 定向验证**：文案/样式微调、显然局部的小修复；不强制新增测试，但须跑相关既有测试。
- **B 修复后补回归测试**：中小 bug、局部行为变化；尽量补贴近问题表面的回归测试。
- **C 必须 TDD**（Red→Green→Refactor→Verify）：新功能、请求执行与集合/环境等契约变更、跨模块共享逻辑、并发/状态机相关改动。TDD 要求先观察到测试因预期原因失败。

质量门禁分层：L0 定向验证 < L1 回归测试 < L2 TDD < L3 Code Review < L4 完成前验证（所有声称完成的任务必须过 L4）。

## MyGo 原生 UI 约定

- 视图每帧重建：元素与自定义部件在下一次构建后失效，应用状态里只存数据、`ui.Handle`，绝不存元素或部件；过期操作在开发/Tester 下 panic，生产下静默失效——修所有权，不要靠有效性判断兜底。
- `*ui.Context` 属于窗口：可存进 `Children` 闭包，不可存进应用状态或后台 goroutine；后台回调需要能力时先取 `c.Services()`。
- 持久身份与焦点：`.Bind(&handle)`、`c.Key(key)`、`.FocusBind(&a.field, target)`；容器 `.Key` 必须在子元素与局部状态之前；列表用 `.ItemKey` 保持行身份。
- 事件时序：绑定值在构建后更新，`Changed`/`Submitted` 在下一帧可读；`OnClick`/`OnShortcut` 在构建后执行；改集合用延迟动作。
- 后台工作与渲染：慢 IO（HTTP 请求）绝不放在视图里，放到 goroutine，结果经 `Window.Update` 回主线程；`Invalidate` 只请求重绘，不同步模型写入；来源变化或窗口关闭后丢弃过期结果。
- 绘制回调只负责画，可能反复执行；标准控件优先用 MyGo 组件，保留键盘、焦点、禁用与无障碍行为。

## 工程实践

**快速上手顺序**：`.agents/skills/mygo-maintenance` → `.agents/skills/mygo-maintenance/references/project-experiences.md`（按需查阅项目经验）→ `README.md` → `Taskfile.yml`（构建/打包/检查入口）→ 任务涉及的模块代码（入口 `main.go` → `internal/app/run.go`；视图 `internal/app/view_*.go`；持久化 `internal/store/service_*.go`；请求执行 `internal/req/*.go`；自绘控件 `internal/widgets/*.go`）。

**文档与经验沉淀**：开发中反复证明有价值的经验（常见根因、排查顺序、易回归点、验证入口）应及时补充到 `.agents/skills/mygo-maintenance/references/project-experiences.md`，模板为：

- **标题**：一句话概括
- **触发信号**：什么现象说明又遇到同类问题
- **根因/约束**：为什么发生
- **正确做法**：以后优先怎么处理
- **验证方式**：如何确认处理正确
- **适用范围**：影响哪些模块/命令

## 编码与重构规范

- SOLID/DRY/YAGNI；仅在关键或不直观逻辑加简短注释（解释"为什么"，不复述代码）；修改行为时顺手移除死代码；不隐藏失败。
- **硬指标（新代码）**：函数 ≤ 50 行、文件 ≤ 300 行（视图/服务文件超限时按职责拆分）、嵌套 ≤ 3 层、位置参数 ≤ 3、单函数圈复杂度 ≤ 10、禁止魔法数字。
- 断言约定 expected 在前、actual 在后；测试优先覆盖关键路径、边界与错误路径，避免脆弱测试（视图测试用 `ui.Tester`，不依赖真实窗口）。
- 视图代码遵循 ui 包惯用写法：builder 链式调用 + `Children` 闭包；元素值只在当帧有效，不进应用状态/后台 goroutine。
- 包按模块分：`internal/app`（界面与动作）、`internal/store`（持久化）、`internal/req`（HTTP 执行与变量解析）、`internal/widgets`（自绘控件）、`internal/design`、`internal/icons`、`internal/assets`（内嵌字体）。依赖方向单向：`main → app → {store, req, widgets, design, icons, assets}`，`req → widgets`，`store` 只依赖标准库与 yaml。
- 持久化（`internal/store`）不依赖 UI：状态变化走 `OnChange` 多播回调 + 视图 `Invalidate` 拉取，不反向持有窗口。
- 重构默认行为不变，在测试保护下分步推进。

## 安全规则

- 不运行破坏性命令（如 `git reset --hard`、强制推送），除非用户明确要求；不用非 Git 工具操作 `.git`。
- 密钥/凭据不硬编码：API Key、Authorization 头等只存在用户配置目录，不进仓库、不进日志、不进截图。
- 不终止非当前任务启动的进程；不修改参考项目 `E:\github\RcloneDrive` 的任何文件（只读参考）。
- 提权操作（UAC/runas）必须先给用户明确提示，禁止无提示静默提权。

## 沟通与子代理

- 执行类任务先说进度与下一步；分析类任务先给结论与依据。
- 子代理仅在可明确拆分、并行收益真实或需要独立审查时派发；派发时说清目标/范围/输入/预期输出/禁止区域；主代理必须复核子代理结论并对最终交付负责。
