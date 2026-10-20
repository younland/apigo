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
| 原生界面（ui 视图） | 明确布局对齐/主题（亮暗）/缩放（100%–150%）行为；改后跑 `CGO_ENABLED=0 go test ./...`（含 ui.Tester 视图测试），必要时真实窗口截图核验（见「已沉淀经验」） |

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

**快速上手顺序**：`.agents/skills/mygo-maintenance` → 本文件「已沉淀经验」→ `README.md` → `Taskfile.yml`（构建/打包/检查入口）→ 任务涉及的模块代码（视图 `*view.go`，服务 `*service.go`，窗口与入口 `main.go`）。

**文档与经验沉淀**：开发中反复证明有价值的经验（常见根因、排查顺序、易回归点、验证入口）应及时补充到本文件「已沉淀经验」，模板为：

- **标题**：一句话概括
- **触发信号**：什么现象说明又遇到同类问题
- **根因/约束**：为什么发生
- **正确做法**：以后优先怎么处理
- **验证方式**：如何确认处理正确
- **适用范围**：影响哪些模块/命令

#### 已沉淀经验

以下界面经验在 MyGo v0.5.5 上复现并核实过；升级 MyGo 后按 skill 要求对照 pinned 源码重新验证。

- **标题**：`ui.Row` 默认把子元素垂直居中，内容不足满高时会被"居中"到窗口中部
  - **触发信号**：窗口上部大片空白、内容出现在视觉中心（内容少的窗口最明显）。
  - **根因/约束**：`ui.Row` 的跨轴对齐默认 Center（"lays its children out from left to right, centered vertically"），侧栏 FillHeight 不受影响，右侧内容区（Scroll/Column）只等于内容高度。
  - **正确做法**：主布局 Row 显式 `.AlignItems(ui.Start)`，且**内容区再显式 `.FillHeight()`**——只靠 AlignItems(Start) 时 Scroll 拿不到有界高度，内容超长会溢出窗口而不是滚动。
  - **验证方式**：内容少的窗口靠顶、内容多的窗口出现滚动条且底部按钮不被覆盖。
  - **适用范围**：所有「侧栏 + 内容区」的布局（请求列表/工作区、设置页）。
- **标题**：`ui.Select` 触发器有主题最小宽度（`t.Space(35)`，约 140px），在固定宽列内会溢出并与相邻控件重叠
  - **触发信号**：两个下拉框画成一个长框、边框/箭头叠在一起。
  - **根因/约束**：Select 的 MinWidth 在 clamp 阶段最后生效，会顶掉显式 Width；子元素溢出固定宽父列后，后续兄弟仍按列宽排布 → 重叠。
  - **正确做法**：列内 Select 一律 `.MinWidth(0)`，再 FillWidth 或显式 Width。
  - **验证方式**：真实窗口截图核验（离屏渲染可能看不出，见下条）。
  - **适用范围**：所有窄列内的下拉（方法选择、协议、环境等）。
- **标题**：`ButtonBase` 默认 `.Center()` 水平居中内容，列表项要左对齐需显式改
  - **触发信号**：列表里短标签项的图标比长标签项更靠右，逐项错位。
  - **根因/约束**：`coreButtonBase = coreRow(c).Center()`。
  - **正确做法**：`.Justify(ui.Start)`（垂直居中保留）。
  - **验证方式**：树/导航项图标左缘对齐。
  - **适用范围**：集合树项、历史列表项、设置导航项等 FillWidth 的 ButtonBase。
- **标题**：离屏渲染（`ui.Render`/`ui.Tester`）与真实窗口可能有差异，关键界面要真实窗口截图核验
  - **触发信号**：Tester 渲染正常，真实窗口布局错乱。
  - **根因/约束**：真实窗口的字体回退、DPI、动画帧状态与 Tester 不完全一致。
  - **正确做法**：对运行中的真实窗口截图检查；临时调试可用环境变量钩子直接弹出目标窗口。
  - **验证方式**：截图中布局/控件与预期一致。
  - **适用范围**：所有涉及具体布局坐标、控件重叠的界面改动。
- **标题**：`.gitignore` 排除目录后无法反选其中子目录（"It is not possible to re-include a file if a parent directory is excluded"）
  - **触发信号**：`git add -A` 后想纳入版本控制的子目录文件始终未入库。
  - **正确做法**：写 `/build/*` + `!/build/windows/`（排除目录内容而非目录本身）。
  - **验证方式**：`git status --short` 能看到新文件。
  - **适用范围**：本仓库任何想纳入版本控制的构建目录子路径。
- **标题**：MyGo 生成的 Windows 安装包是用户级安装（`%LOCALAPPDATA%`、不提权）
  - **触发信号**：需要随机安装系统组件、驱动或写入 `Program Files`。
  - **根因/约束**：`mygo build` 的 NSIS 脚本是硬编码模板（用户级），没有配置注入点。
  - **正确做法**：先用默认安装包；确需系统级安装时自建 NSIS 脚本（照抄 mygo 模板改管理员级），`mygo build` 后用 makensis 产出同名安装包覆盖，且脚本保存为 UTF-8 with BOM。
  - **验证方式**：安装包文件名保持 `<name> Setup <version> <arch>.exe`，安装流程与 updater 的资产匹配仍成立。
  - **适用范围**：Windows 发布链路。
- **标题**：mygo 的 deb / install.sh 为 webview 应用硬编码了 GTK 与 WebKitGTK 依赖，native UI 包里必须剥掉
  - **触发信号**：apt 安装 apigo 时连带拖入整套 `libwebkit2gtk`；或 `install.sh` 报缺少 WebKitGTK 而拒绝安装。
  - **根因/约束**：`cmd/mygo/deb.go` 里 `Depends: libgtk-3-0 | libgtk-3-0t64, libwebkit2gtk-4.1-0` 是硬编码（`linux.depends` 只能追加），`install.sh` 的 `has_webkit` 同样按 webview 应用假设；native UI 走 OpenGL（缺失时 CPU 渲染），运行时 dlopen，不链接这两个库。
  - **正确做法**：发布前 `dpkg-deb -R` 取出 control 删除 `Depends:` 行后重打包；tar.gz 里的 `install.sh` 用 `sed` 把 `has_webkit` 的 `grep -qF -e libwebkit2gtk…` 行替换为 `return 0`（见 `.github/workflows/release.yml` 的「收集产物」步骤）。
  - **验证方式**：`ar p` 取 control.tar.gz 确认无 GTK/WebKit 依赖；tar.gz 内 install.sh 中该 grep 行已消失。
  - **适用范围**：Linux 发布链路（deb、tar.gz）。
- **标题**：`mygo build` 的产物名与目录布局固定，发布脚本按它取名即可
  - **触发信号**：CI 里猜产物文件名导致 `cp` 失败。
  - **正确做法**：`build/<os>-<arch>/` 下 Windows 为 `<name>.exe` 与 `<name> Setup <version> <arch>.exe`，Linux 为 `<name>`、`<name>_<version>_<arch>.deb`、`<name>-<version>-linux-<arch>.tar.gz`、`install.sh`，macOS 为 `build/darwin-universal/<name>.app`（**dmg 只能在 macOS 上生成**，其他平台会跳过）。
  - **验证方式**：本地 `go tool mygo build -platform <os>/<arch>` 后 `ls build/<os>-<arch>`。
  - **适用范围**：`.github/workflows/release.yml` 与任何发布脚本。
- **标题**：`-race` 需要 cgo，机器上 `go env CGO_ENABLED=0` 时竞态任务会直接失败
  - **触发信号**：`go test -race ./...` 报 `-race requires cgo; enable cgo by setting CGO_ENABLED=1`。
  - **根因/约束**：本机（mise 管道的 Go）默认 `CGO_ENABLED=0`，与"发布链路不需要 cgo"是两件事；竞态检测本身依赖 cgo。
  - **正确做法**：Taskfile 里给竞态任务显式 `env: CGO_ENABLED: '1'`（`task test:race`），其余测试任务保持 `CGO_ENABLED=0`。
  - **验证方式**：`task check` 全绿，且日志里竞态步骤为 `go test -race ./...`。
  - **适用范围**：`Taskfile.yml`、`.github/workflows/check.yml`。
- **标题**：Task 在 Windows 上会借用 `sh`，但内联 `VAR=x cmd` 不可靠，环境变量要用 `env:`
  - **触发信号**：任务在开发者机器上通过、换台机器或换 CI 就报"找不到命令"或变量为空。
  - **正确做法**：`CGO_ENABLED`、`GOFLAGS` 等一律写在任务的 `env:` 块；版本号这类参数通过 Task 变量（`task package:windows APP_VERSION=1.2.3`）传入，再交给 `tools/syncversion` 落盘。
  - **验证方式**：`task --list` 有该任务，且 `task <name> APP_VERSION=1.2.3` 后 `grep version mygo.json` 为新值。
  - **适用范围**：`Taskfile.yml` 全部任务。

## 编码与重构规范

- SOLID/DRY/YAGNI；仅在关键或不直观逻辑加简短注释（解释"为什么"，不复述代码）；修改行为时顺手移除死代码；不隐藏失败。
- **硬指标（新代码）**：函数 ≤ 50 行、文件 ≤ 300 行（视图/服务文件超限时按职责拆分）、嵌套 ≤ 3 层、位置参数 ≤ 3、单函数圈复杂度 ≤ 10、禁止魔法数字。
- 断言约定 expected 在前、actual 在后；测试优先覆盖关键路径、边界与错误路径，避免脆弱测试（视图测试用 `ui.Tester`，不依赖真实窗口）。
- 视图代码遵循 ui 包惯用写法：builder 链式调用 + `Children` 闭包；元素值只在当帧有效，不进应用状态/后台 goroutine。
- 服务（`*service.go`）不依赖 UI：状态变化走 `OnChange` 多播回调 + 视图 `Invalidate` 拉取，不反向持有窗口。
- 重构默认行为不变，在测试保护下分步推进。

## 安全规则

- 不运行破坏性命令（如 `git reset --hard`、强制推送），除非用户明确要求；不用非 Git 工具操作 `.git`。
- 密钥/凭据不硬编码：API Key、Authorization 头等只存在用户配置目录，不进仓库、不进日志、不进截图。
- 不终止非当前任务启动的进程；不修改参考项目 `E:\github\RcloneDrive` 的任何文件（只读参考）。
- 提权操作（UAC/runas）必须先给用户明确提示，禁止无提示静默提权。

## 沟通与子代理

- 执行类任务先说进度与下一步；分析类任务先给结论与依据。
- 子代理仅在可明确拆分、并行收益真实或需要独立审查时派发；派发时说清目标/范围/输入/预期输出/禁止区域；主代理必须复核子代理结论并对最终交付负责。
