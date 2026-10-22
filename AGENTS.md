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

**快速上手顺序**：`.agents/skills/mygo-maintenance` → 本文件「已沉淀经验」→ `README.md` → `Taskfile.yml`（构建/打包/检查入口）→ 任务涉及的模块代码（入口 `main.go` → `internal/app/run.go`；视图 `internal/app/view_*.go`；持久化 `internal/store/service_*.go`；请求执行 `internal/req/*.go`；自绘控件 `internal/widgets/*.go`）。

**文档与经验沉淀**：开发中反复证明有价值的经验（常见根因、排查顺序、易回归点、验证入口）应及时补充到本文件「已沉淀经验」，模板为：

- **标题**：一句话概括
- **触发信号**：什么现象说明又遇到同类问题
- **根因/约束**：为什么发生
- **正确做法**：以后优先怎么处理
- **验证方式**：如何确认处理正确
- **适用范围**：影响哪些模块/命令

#### 已沉淀经验

以下界面经验在 MyGo v0.6.0 上复现并核实过（0.5.5 → 0.6.0 升级只带来刻度级的绘制差异：文字实测宽度变化、鼠标滚轮滚动带缓动）；再升级 MyGo 时按 skill 要求对照 pinned 源码重新验证。

模块化之后代码分在 `internal/{app,store,req,widgets,design,icons,assets}`：`store` 与 `req` 的类型、方法一律首字母大写；条目里若还写着旧的小写名（`savedRequest`、`folder`、`bodyFormat`…），那是 `internal/app` 里 `store_alias.go` / `req_alias.go` 的类型别名，仍然有效。

- **标题**：拆分内部包要靠「别名兜底 + 编译器驱动导出」，一次只让一个包编译通过
  - **触发信号**：把 40 多个根目录文件搬进 `internal/` 时，几百处类型/方法引用要跟着改名；手工扫一遍几乎必然漏。
  - **根因/约束**：Go 的 `main` 包不能被 import，功能要下沉就得成包；跨包就只能用导出的名字，而 `folder`/`collection`/`response` 这类名字在 app 里遍地都是。类型别名（`type folder = store.Folder`）能让 app 侧的 `folder{...}`、`e.Folder` 一行都不用改，被挡住的只有「未导出方法/字段」和「包级函数」。
  - **正确做法**：分批下沉，每批结束都要 `go build ./...` 绿：①`widgets` 直接搬；②`internal/store`（13 个 `service_*.go`）——包内用「整词重命名」把类型/方法导出，app 侧加 `store_alias.go` 别名 + 只改 `a.store.X` 这类方法调用；③`internal/req`（`http.go`/`formats.go`/`vars.go`）+ `req_alias.go`；④其余全进 `internal/app`，根目录只留 `main.go`（`package main` 里 `app.Run()`）。`//go:embed` 不能跨目录，字体单独成 `internal/assets`（`assets` 由 `run.go` 空导入触发 `init()` 注册）。测试按被测包落位：纯 store 的进 `internal/store`，req 的进 `internal/req`，视图测试留 `internal/app`。
  - **验证方式**：`task check` 绿（`mygo vet` 改成 `./...`，否则只看根包）；`APIGO_SHOTS=... go test -run TestShots ./internal/app/` 与重构前的图逐张 `cmp`，应**逐字节一致**（纯搬家不改行为）；字体没注册会立刻体现在截图的字形上。
  - **适用范围**：`internal/**`、根 `main.go`、`Taskfile.yml`（`mygo vet ./...`）、`README.md` 目录结构、`AGENTS.md` 的经验条目路径。

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
  - **适用范围**：集合树项、设置导航项等 FillWidth 的 ButtonBase。
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
- **标题**：native UI 的 Linux 包要用 `"linux": {"webview": false}` 声明，否则 deb 会拖入整套 WebKitGTK
  - **触发信号**：apt 安装 apigo 时连带拖入 `libwebkit2gtk-4.1-0`；或 `install.sh` 报缺少 WebKitGTK 而拒绝安装。
  - **根因/约束**：`mygo.json` 不写 `linux.webview` 时按 webview 应用处理——deb 的 `Depends` 带 `libwebkit2gtk-4.1-0`，`install.sh` 的 `has_webkit` 也按 webview 假设。native UI 走 OpenGL（缺失时 CPU 渲染），运行时 dlopen，不链接 GTK/WebKit。
  - **正确做法**：`mygo.json` 里写 `"linux": {"webview": false}`（0.6.0 起官方支持），deb 只依赖 `libgtk-3-0 | libgtk-3-0t64`、`install.sh` 不再检查 WebKitGTK；发布流程不再需要 `dpkg-deb -R` 剥 `Depends:` 与 `sed` 改 `has_webkit`（0.6.0 之前只能靠这两个补丁）。
  - **验证方式**：`go tool mygo build -platform linux/amd64` 后 `ar x apigo_<版本>_amd64.deb && tar -xzf control.tar.gz && cat control`，`Depends` 行里没有 `libwebkit2gtk`；`go tool mygo doctor`（Linux）打印 `webkit not needed`。
  - **适用范围**：Linux 发布链路（deb、tar.gz）、`mygo.json`、`.github/workflows/release.yml`。
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
- **标题**：`ui.Tester.Find` 按无障碍标签匹配，同屏重复文案会命中先构建的那个
  - **触发信号**：断言某区域的 `Rect` 却落在别的区域（顶部栏的“REST”抢了图标栏的“REST”）。
  - **根因/约束**：`Find` 遍历标签列表返回首个同名项，文本与 `.Label()` 共用同一匹配空间。
  - **正确做法**：同屏不要复用同一文案；容器用 `.Label()` 给唯一名字再断言，或把断言目标换成该区域独有的文本。
  - **验证方式**：测试通过且 `Rect` 落在预期区域内（X/Y 边界断言）。
  - **适用范围**：所有 `ui.Tester` 布局断言。
- **标题**：窗口主题必须由 app 自己下发；「跟随系统」只能在启动时取一次
  - **触发信号**：真实窗口里 `PrimaryButton` 是 MyGo 默认蓝、输入框是默认粗边，而 `ui.Tester` 截图里是绿色细边——两套观感对不上。
  - **根因/约束**：`ui.Tester` 的截图都在固定外观（浅色/深色/纯黑）下跑，会 `c.SetTheme(design.Theme(...))`；而「跟随系统」分支若直接不调用 `SetTheme`，窗口就停在 MyGo 默认主题上，自绘配色（`design.Palette`）与控件配色（theme）分裂。反过来，`SetTheme` 一调用就再也交不还给桌面（MyGo 没有 reset），`c.Theme().Dark` 反映的是自己的主题。
  - **正确做法**：`themeFor` 永远返回 `design.Theme(resolvedMode())` 并 `c.SetTheme`；`System` 模式在第一帧（`applied == Mode(-1)`）用 `c.Theme().Dark` 记录桌面亮暗，之后会话内固定，设置页注明「跟随系统在窗口打开时取一次」。
  - **验证方式**：`TestWindowThemeFollowsPalette`（跟随系统时 `theme.Accent/Border/Background/Font` 与 palette 一致）、`TestSettingsModal`（外观循环）。
  - **适用范围**：`internal/app/app.go` 的 `themeFor`/`palette`/`resolvedMode`、`internal/app/view.go`、设置页外观分区。
- **标题**：延迟增删用的索引字段拿零值当「无操作」，会每帧删掉第一项
  - **触发信号**：键值列表每帧少一行、Query 参数在发送时消失（`internal/widgets/kvlist.go` 的 `remove`）。
  - **根因/约束**：`remove int` 的零值是 0，和「删除第 0 行」无法区分，`if remove >= 0` 的守卫恒真。
  - **正确做法**：索引用 1-based 哨兵（0 = 无操作），或额外配一个 `bool`；延迟动作在构建前统一 `Flush`，并给「无操作」路径写用例。
  - **验证方式**：`go test ./widgets`；填入参数后发送，服务端收到该参数。
  - **适用范围**：所有把增删/重排推迟到下一帧的列表状态。
- **标题**：`ui.Checkbox` 的第三个参数是可见标签文本，不是无障碍名
  - **触发信号**：键值表格每行都多出一段“启用”文字，与原版（只有方框）不一致。
  - **正确做法**：`ui.Checkbox(c, &v, "")` 传空标签，再用 `.Label(fmt.Sprintf("启用第 %d 行", i+1))` 补无障碍名；行内控件需要唯一名字时优先 `.Label()` 而不是可见文本。
  - **验证方式**：截图里没有多余文字；`ui.Tester` 仍能按 `Label` 找到控件。
  - **适用范围**：`internal/widgets/kvlist.go` 及任何表格/列表行内的勾选框、按钮。
- **标题**：`editor.State` 的 `LineNumbers` 零值是 `LineNumbersOn`，关行号要显式写 `LineNumbersOff`
  - **触发信号**：只读响应正文多出列行号，或注释里写着“关行号”却不生效。
  - **根因/约束**：枚举把 `LineNumbersOn` 放在 0 位，零值即开启；`bool` 式的直觉判断会失效。
  - **正确做法**：需要关行号时显式赋 `editor.LineNumbersOff`；新增 `editor` 选项前先读 `ui/editor/state.go` 的常量顺序。
  - **验证方式**：离屏截图确认正文无行号；`TestResponseRenderedAsReadOnlyJSON` 之类的状态断言。
  - **适用范围**：`internal/app/app.go` 的 `bodyEdit`/`respEdit` 配置。
- **标题**：条件分支里重复构造同一个控件，两份都会挂到父节点
  - **触发信号**：紧凑模式下列表仍按宽松宽度渲染，另一个输入框被挤出面板（侧栏环境变量行）。
  - **根因/约束**：`ui.TextInput(c, ...)` 一构造就挂到当前父节点；`if compact { x = ui.TextInput(...) }` 只换了变量，先构造的那份仍在树上。
  - **正确做法**：控件只构造一次，再按条件改尺寸；布局差异大时拆成两个互斥函数（`row` / `compactRow`），各自构造各自的全部子元素。
  - **验证方式**：离屏截图确认窄面板内两个输入框都可见；`ui.Tester.Find` 的宽度断言。
  - **适用范围**：`internal/widgets/kvlist.go` 及所有在分支里建控件的视图。
- **标题**：用 `TextInput.Changed()` 当脏标记会在没编辑时误报
  - **触发信号**：切换标签页后，未改动的标签也显示“有未保存内容”的圆点；用 `t.touch()` 挂在 `Changed()` 上尤其明显。
  - **根因/约束**：`Changed()` 报告的是本次构建里绑定值发生了变化，聚焦/失焦与外部写入（切换标签时重新绑定）都会命中，和“用户编辑过”不是一回事。
  - **正确做法**：脏标记改为可判定的内容判定（URL/body/键值行是否有值），或自行比较上一帧快照；`Changed()` 只用于需要“值变了就做事”的场景（如切语言）。
  - **验证方式**：新建多个标签、只编辑其中一个，未编辑项不出现圆点。
  - **适用范围**：`internal/app/view_tabs.go` 的圆点、任何需要“用户是否改过”的界面状态。
- **标题**：集合拥有工作区，请求标签不能放回全局 app
  - **触发信号**：切换集合后仍看到上一个集合的请求标签；关闭集合再打开，旧标签又出现。
  - **根因/约束**：标签属于集合的 `workspace`（每个集合一份），不是窗口全局状态；主窗口没有工作区。
  - **正确做法**：`app.spaces map[collectionID]*workspace`，所有标签操作都带 `ws` 参数；`current()` 在 `active == ""` 时返回 nil，视图只在集合激活时构建请求区；集合关闭时删除对应 workspace。
  - **验证方式**：开两个集合各建标签，来回切换互不串；关闭集合后重开没有标签、直接见「还没有打开接口」。
  - **适用范围**：`internal/app/app.go`、`internal/app/view.go`、`internal/app/view_tabs.go`、`internal/app/view_requests.go`。
- **标题**：`store` 的查询函数返回切片元素副本，改它不会写回
  - **触发信号**：删除激活环境后想把剩下的第一个设为激活却无效；变量改了文件没变。
  - **根因/约束**：`environmentsFor`/类似函数返回 `[]environment` 值拷贝，改元素只改拷贝。
  - **正确做法**：需要修改时按下标遍历 `s.environments`，或用 `environment(id) *environment` 取指针再改，最后统一 `saveEnvironments()`。
  - **验证方式**：`TestEnvironmentLifecycle` 删除激活环境后另一个自动激活并落盘。
  - **适用范围**：`internal/store/service_store.go` 及所有返回存储切片的查询。
- **标题**：集合是一个目录，接口与环境的名字就是文件名，同名必须编号
  - **触发信号**：建两个叫 `Untitled` 的接口，树里只剩一个；改名后旧文件还在，列表里出现"幽灵"接口。
  - **根因/约束**：磁盘布局是「目录=文件夹、`<名字>.yaml`=接口/环境」，身份是路径而不是 id；同名会写到同一个文件，改名不删旧文件就会读回两个。
  - **正确做法**：写盘前用 `uniqueName(taken, name)` 编号（`safeFileName` 去掉平台非法字符）；改名/删除走「写新文件 + `pruneEnvDir` 清旧文件」，不要 `os.Rename` 到未校验的名字；`collection.ID` 用目录名（`uniqueCollectionID` 保证唯一）。
  - **验证方式**：`TestCollectionLayoutOnDisk`、`TestNewInterfaceWritesFile`、`TestEnvironmentRenameMovesFile`。
  - **适用范围**：`internal/store/service_store.go`、`internal/store/service_tree.go`、`internal/store/service_env.go`、所有新增写盘路径。
- **标题**：`folder.Path` 的空串是「根目录」，不能拿空串当"没有文件夹"
  - **触发信号**：根目录折叠状态或它的新建接口菜单作用到了别的文件夹，或者根目录无法折叠。
  - **根因/约束**：根目录是集合自己的目录，路径就是空串；`foldersClosed`、`folderMenus` 都以 Path 为键，空串是合法键。
  - **正确做法**：树里所有位置判断用 `findFolder(&col.Root, path)`（找不到才是 nil），折叠/菜单状态直接以 Path 为键，别用"key 为空就是无操作"的约定。
  - **验证方式**：`TestFolderTreeAddsAndNavigates` 折叠「文件夹 根目录」后整棵树隐藏。
  - **适用范围**：`app.foldersClosed`、`app.folderMenus`、`internal/app/view_requests.go`、`internal/store/service_tree.go`。
- **标题**：接口的编辑只在离开标签时写回，写失败必须让人看见
  - **触发信号**：改了 URL/请求体但 `<接口名>.yaml` 没变；或文件写失败却毫无提示。
  - **根因/约束**：标签是内存里的编辑态，磁盘上的接口文件才是真相；`tab.path` 记住它来自哪个接口文件，没有 path 的标签（新建、复制）不写盘，写盘走 `store.updateRequest`（只改那一个文件，不重写整棵树）。
  - **正确做法**：所有让工作区离开当前标签的动作（切换、新建、复制、关闭、关闭其他、关闭集合）先经 `leaveTab`/`saveTab`；失败记进 `a.saveErr`，下一帧在根视图 `c.Toast` 一次，别静默吞掉。删除集合时用 `forgetCollection`（集合已不存在，不能再写回）。
  - **验证方式**：`TestTabSwitchWritesInterfaceFile`、`TestCloseTabWritesInterfaceFile`、`TestFailedInterfaceSaveIsReported`。
  - **适用范围**：`internal/app/app.go`（`leaveTab`/`saveTab`/`closeCollection`）、`internal/app/view_tabs.go`、`internal/store/service_tree.go`（`updateRequest`）。
- **标题**：一行只留一个入口（行尾 `+` 菜单），行内按钮越多越难命中，也越容易和测试撞名
  - **触发信号**：树行尾并排三个图标把文件夹名挤成省略号；`ui.Tester` 点「新建文件夹」命中了别的行或别的面板。
  - **根因/约束**：每行宽度只有 240-24，多余按钮会挤掉名字；`Find` 只按文案取首个匹配，同屏重复文案会让断言落错行。
  - **正确做法**：同一行的多种新建动作收进一个 `+` 菜单（`folderAddButton`：先「新建文件夹」，分隔线后是类型菜单），菜单项的无障碍名带上是哪个文件夹（`在<文件夹>里新建文件夹`）；跨整棵树的动作（类型筛选）放在「根目录」行尾，只有一行。
  - **验证方式**：`TestRootRowHostsTreeTools`（筛选在 `+` 左侧同一行、菜单能建文件夹）、`TestFolderTreeAddsAndNavigates`。
  - **适用范围**：`internal/app/view_requests.go` 的 `folderRow`/`folderAddButton`/`kindFilterButton`。
  - **补充（标签条的 `+` 同一个道理）**：请求标签条的 `+` 也先弹类型菜单（`newTabButton` 里 `PopoverBase` + `kindMenu(c, p, "新建 ", …)`，**没有「新建文件夹」**——标签条里没有文件夹可建），选完走 `addTabOfKind`（非 REST 清空 method、`bodyType` 默认 json）。因为菜单行与别处的类型名同文案，`kindMenu` 多了一个 `prefix` 参数：标签条传 `"新建 "`，树里的菜单传 `""`，测试点 `新建 WebSocket` 不会撞上别处的 `WebSocket`。
  - **补充（非 REST 只留图标）**：树行与标签页前那一格都由 `kindBadge`/`tabBadge` 画——REST 画「类型图标 + 彩色方法名」，其它类型**只画类型图标**（标签用的是 `tabKindIcon = 12`，树行 13），因为「WebSocket / GraphQL」这种字既长又和名字抢宽度；`requestTabChrome` 的宽度也要跟着换成 `tabBadgeWidth`，否则标题会被算错而挤掉图标或 ×。
- **标题**：应用内菜单用 `PopoverBase` 自绘，右键要用 `RightClicked()` 落在真正被点的元素上
  - **触发信号**：系统菜单（`ContextMenu`）里没有图标、样式与其他菜单不一致；换成 popover 后右键点标签页菜单不出现。
  - **根因/约束**：`ContextMenu` 是平台菜单（纯文本）；`PopoverBase(anchor, open, fn)` 只是按 `*open` 显示面板，不会自己响应右键；`RightClicked()` 报告的是指针下的那个元素，不会向祖先冒泡（`ContextMenu` 才会）。
  - **正确做法**：菜单统一走 `internal/app/view_menu.go` 的 `popoverMenu`（图标 + 文本、`menuRule()` 分组、`menuPanel` 统一样式），展开状态用 `a.menuFlag(key)`（键：`folder/<path>`、`collection/<id>`、`tab/<id>`）；右键把 `RightClicked()` 挂在实际命中的按钮上（集合标签挂在内层 `body`，不是外层 Row），面板锚点仍可用外层元素。
  - **补充（菜单外观）**：面板 `menuPanel` 用 `p.Border`（`Divider` 在白底上几乎看不见），四周 `Padding(4)` + `Radius(4)`；每行经 `menuRow(row, p)` 统一——`FillWidth()` 让高亮铺满菜单宽（`ButtonBase` 不拉伸，行只有文字宽），`Hovered()` 时换 `p.SurfaceHover` 底色；母面板已画边框/底色，嵌套的列（如 `kindMenu`）不要再画一遍，否则出现双层框；`Hovered()` 读的是上一帧状态，慢一帧是正常的。
  - **验证方式**：`TestCollectionTabContextMenu`（`tt.Menu()` 为 nil、popover 项可见并可点）、`TestHomeCollectionMenu`、`internal/app/menu_test.go` 的 `TestMenuPanelDrawsBorder`（行左侧 8px 内应出现 `p.Border` 像素）、`TestMenuRowTintsOnHover`（指针移入后行右端底色为 `p.SurfaceHover`）、`TestMenuStillRunsWhatItLists`（改样式后菜单项仍会执行）。
  - **适用范围**：`internal/app/view_menu.go`、`internal/app/view_topbar.go`、`internal/app/view_home.go`、`internal/app/view_requests.go`、`internal/app/view_tabs.go`（请求标签右键仍是系统菜单）。
- **标题**：调系统外壳的动作走可替换的 `opener`，别在视图里直接调 `mygo.Shell`
  - **触发信号**：离线测试点「打开文件夹」时挂住或 panic（`mygo: Shell.OpenPath called before App.Run`）。
  - **根因/约束**：`mygo.Shell.*` 依赖已启动的事件循环（`needsApp` + `onMainValue`），`ui.Tester` 里没有它。
  - **正确做法**：`app` 上留一个字段（`opener func(string) error`），`internal/app/run.go` 在 `WhenReady` 里填 `mygo.Shell.OpenPath`，动作只调 `a.openFolder(path)`；测试注入自己的函数并断言收到的路径。
  - **验证方式**：`TestHomeCollectionMenu` 注入 opener，断言收到 `a.store.collectionDir("Demo")`。
  - **适用范围**：`internal/app/app.go`（`opener`/`openFolder`）、`internal/app/run.go`、`internal/app/view_home.go`，以及以后任何打开浏览器/文件管理器/回收站的调用。
- **标题**：换数据目录要「先落 pointer 再换 store」，并把窗口状态清干净
  - **触发信号**：选了新目录后列表还是旧的；重启又回到旧目录；或者换完目录左侧还挂着上一个数据目录的集合标签。
  - **根因/约束**：数据目录自身不能记录"我在哪"，指针必须放在**默认目录**（`location.yaml`），否则换完就找不回来；`a.spaces`/`open`/`active`/`foldersClosed`/`menus` 都以旧集合的 id 为键，换库后必须整体丢弃，否则视图会访问不存在的集合。
  - **正确做法**：`store.DirAt(baseDir)`（启动）与 `store.RememberDir(baseDir, dir)`（切换）读写 `location.yaml`；`useDataDir` 先 `store.New(dir).load()` 成功、再写指针、再替换 `a.store`、最后 `forgetEverything()` + `applySettings()`。系统目录选择走可替换的 `chooseDir func() (string, error)`（`internal/app/run.go` 填 `mygo.Dialog.Open(... Directory: true)`），取消返回 `""` 视为不改动。
  - **验证方式**：`TestStoreDirAt`、`TestChooseDataDir`（换完目录、列表是新目录的集合、窗口状态归零、指针落盘）、`TestChooseDataDirCancelled`（取消/失败都不改动）。
  - **适用范围**：`internal/store/service_location.go`、`internal/app/app.go`（`chooseDataDir`/`useDataDir`/`forgetEverything`）、`internal/app/run.go`、`internal/app/view_env.go` 的「数据」分区。
- **标题**：输入框与自绘控件的边框必须共用一个 token，否则同屏出现两种灰
  - **触发信号**：搜索框/URL 是深一点的 `#d1d5db`，请求方法框几乎看不见（`#f3f4f6`），用户一眼看出"border 颜色不一致"。
  - **根因/约束**：`ui.TextInput` 用 `ui.Theme.Border`，自绘的框用 `p.Divider`/`p.DividerStrong` 各自取值；同一屏里三种灰并存。
  - **正确做法**：`Palette.Border`（浅色 `#e5e7eb`、深色 `#2f2f33`）作为控件边框唯一来源：`Theme` 里 `t.Border = p.Border`，自绘控件与弹层（`menuPanel`）都用 `.Border(design.BorderWidth, p.Border)`；分隔线继续用 `Divider`/`DividerStrong`。
  - **验证方式**：离屏截图上采样边框像素（搜索框、方法框、URL 框都应是 `#e5e7eb`）。
  - **适用范围**：`internal/design`（`Palette.Border`/`Theme`）、`internal/app/view_request.go`（方法框）、`internal/app/view_env.go`（语言下拉）、以后任何自绘输入类控件。
- **标题**：改了集合 id 要先写回内存，再让 `saveCollection` 按新 id 找它
  - **触发信号**：重命名集合后列表里出现两条（旧名 + 新名），`collectionIndex` 找不到新 id；或改名"成功"却没生效。
  - **根因/约束**：`saveCollection` 用 `s.Collections[i] = c` 更新，靠 `collectionIndex(c.ID)` 定位；先改 `col.ID` 再调用，内存里还是旧 id，于是被当成新集合 append。
  - **正确做法**：`renameCollection` 里先 rename 目录 → `s.Collections[i] = col`（带新 id）→ `saveCollection` → `sortCollections` → 改 `environments[].Scope` → `saveEnvironments`；app 侧同步 `spaces` 的键、`open`/`closed`、`active` 与各标签的 `collectionID`（`app.renameCollection`）。目录可能不存在（内存里的集合），用 `renameDir` 把 `os.IsNotExist` 当作"没有要搬的"。
  - **验证方式**：`TestRenameCollectionMovesDirectory`（目录、接口、`.env`、`apigo.yaml`、重载后一致）、`TestRenameCollectionThroughMenu`。
  - **适用范围**：`internal/store/service_collection.go`、`internal/app/app.go`，以及以后任何改集合/接口 id 的路径。
- **标题**：`Clicked()` 里的动作会在构建过程中改列表，遍历必须用快照 + 重新取索引
  - **触发信号**：真实窗口关闭一个请求标签就 `panic: index out of range [3] with length 3`（`internal/app/view_tabs.go` 的 `tabItem`）；`ui.Tester` 里同样的点击也会崩。
  - **根因/约束**：`Clicked()`/`RightClicked()` 是构建期的查询——动作在**构建当帧**执行。`for i := range ws.tabs { a.tabItem(c, p, ws, i) }` 的 range 长度在循环开始时固定，第一个标签的 × 调 `closeTab` 把切片缩短，下一次 `ws.tabs[3]` 就越界；同理 `a.open`（集合标签条）、`a.store.Collections`（主窗口列表）、`a.closed`（最近关闭）都可能在遍历中被自己的按钮改掉。
  - **正确做法**：遍历快照（`for _, t := range append([]*tab(nil), ws.tabs...)`），并在真正渲染/动作前用「查在不在」判断（`tabIndex(ws, t) < 0` 跳过、`containsString(a.open, id)` 跳过）；动作闭包捕获**指针/id**而不是下标，执行时再 `tabIndex`/`collectionIndex` 重新定位（`closeTabAt`/`closeTabsExcept`/`duplicateTabAt`）。
  - **验证方式**：`TestCloseTabWhileStripBuilds`（4 个标签关掉第 1 个不再 panic）、`TestDeleteCollectionWhileListBuilds`（两行集合删第一个）。
  - **适用范围**：`internal/app/view_tabs.go`、`internal/app/view_topbar.go`、`internal/app/view_home.go`，以及以后任何「列表行自带关闭/删除按钮」的视图。
- **标题**：测试里不要直接给 `TextInput` 绑定的字段赋值，重建时会被输入框回写
  - **触发信号**：测试里 `a.collectionName = "订单服务"` 后点「保存」，结果名字还是旧的；断言刚赋的值却发现被恢复。
  - **根因/约束**：`ui.TextInput(c, &field)` 每次构建都会把输入框自己的编辑状态写回字段，外部赋值在下一次构建就被覆盖。
  - **正确做法**：像用户那样操作——`tt.Click("<输入框标签>")` → `tt.Key(ui.Cmd, ui.KeyA)` → `tt.Type("新值")`；断言字段值再点保存。
  - **验证方式**：`TestRenameCollectionThroughMenu`、`TestTabRenameThroughContextMenu`。
  - **适用范围**：所有 `ui.Tester` 视图测试里的 `TextInput`/可编辑字段。
- **标题**：自定义字体走 `go:embed` + `ui.RegisterFont` + 主题 `Font`，注册失败要退回系统字体
  - **触发信号**：换了字体后某些文字变成 `PO…`/`Untitl…`；或某平台窗口里字体没变、启动日志报注册失败。
  - **根因/约束**：`ui.RegisterFont(data, family)` 只把字体加进文本引擎；要全局生效必须写进 `ui.Theme.Font`（`design.Theme` 里 `t.Font = design.FontFamily`），否则只有显式 `.Font(family)` 的元素用它。系统没有可注册字体的文本引擎时（stub 引擎、部分 CI）注册会返回错误。
  - **正确做法**：字体文件放 `resources/`，`internal/assets/font.go` 里 `//go:embed` 后 `init()` 注册，失败只 `log.Printf` 不退出（主题仍写 family，绘制时回落系统字体）；测试用 `fontReady` 区分"平台不支持"与"嵌入坏了"（`TestInterfaceFontEmbedded` 在 no-font 平台上 `Skip`）。字体变宽会挤掉原本"刚好够"的文字——固定宽度区域要留余量（标签页给了 `design.TabMinWidth`）。
  - **验证方式**：`TestInterfaceFontEmbedded`（嵌入文件名数字节 + `FontFamilies()` 里为本应用注册）、`TestThemeFollowsPalette`（主题 family）、`APIGO_SHOTS=docs/shots go test -run TestShots ./internal/app/` 出图核对字形与截断。
  - **适用范围**：`internal/assets/font.go`、`internal/design`（`FontFamily`/`Theme`）、所有固定宽度的文字容器；可执行文件随之增大约字体大小。
- **标题**：「字体看着偏瘦」先分清是没生效还是字号，别急着换字体文件
  - **触发信号**：用户说界面字体不是鸿蒙黑体、比样张瘦。样张是 Windows 字体查看器按 12–72 **pt** 绘的，界面是 12 DIP 正文 + 灰字。
  - **根因/约束**：字体其实生效了——主题 `Font` 生效的证据是「同一段文字，走主题（不写 `.Font`）与显式 `.Font(design.FontFamily)` 的墨量完全相等，与系统字体不等」（探针：40 px 下 3222 / 3222 / 3470 个暗像素）。同样的字重，可变字体 `wght` 400→700 墨量 1929→2706，说明 DirectWrite 拿的是真实实例、不是合成粗。真正显瘦的是 ①正文只有 12 DIP（Hoppscotch 采样值），1 px 竖笔落在半像素覆盖上变中灰；②MyGo 走 DirectWrite 的灰度覆盖（`textureClearType3x1` 三通道取平均），字体查看器是 ClearType 次像素，更「厚」；③大量标签用 `TextSecondary/TextMuted` 灰；④同字号下鸿蒙黑体的墨量比系统 UI 字体少约 7%。
  - **正确做法**：字号整体上调一档（`design.FontSizeBody` 12→13、`Tiny` 10→11、`Title` 14→15、`Display` 18→19、`LineHeight` 16→17），`SidebarWidth` 240→264 给变宽的树行留位，日志里的只读 code 用 `design.FontSizeBody - 1` 保持「比正文小一号」。全局字重没有主题开关（`ui.Theme` 只有 `Font`/`FontSize`），想整体加粗只能逐处 `.FontWeight`，代价大、别做。
  - **验证方式**：`TestEnvironmentPickerCentersInTheStrip` 之类尺寸断言全绿；`TestBodyFormatPillsShowTheirLabels` 钉住「胶囊给标签的余量必须严格大于标签的 advance」；截图逐张看窄条是否挤压（页签、日志行尾、树行、格式胶囊）。
  - **适用范围**：`internal/design/tokens.go`（全部字号与 `SidebarWidth`）、`internal/app/socket.go`（日志 code 字号）、一切写死宽度的行。
- **标题**：`c.MeasureText` 只在帧内有效，帧外返回 0——拿它写断言会「静默通过」
  - **触发信号**：测试里 `measureText(ctx, ...)` 得到 0，`rect.W < want` 永远不成立，bug 在场而测试全绿。
  - **根因/约束**：`*ui.Context` 只在 `view(c)` 那一帧里带着排版状态；`tt.Frame()` 之后再调 `c.MeasureText` 没有可量的引擎，返回 (0, 0)。
  - **正确做法**：要在测试里算文字宽度，用 `ui.Shape(s, ui.Font{Family, Size, Weight})` 把 `Advance` 累加（与引擎排版同一套数字，帧外也能用）；确实要测渲染期数值时，在 `tt.Frame()` 里保存所需的值，别保存 ctx。
  - **验证方式**：把被测实现临时改成「没有余量」（`MinWidth(0)`）跑一遍，测试必须变红；`TestBodyFormatPillsShowTheirLabels` 就是这么验的（改前 5 个胶囊报「leaves X for a label that takes X」）。
  - **适用范围**：所有用 `ui.Tester` 量文字的断言（`internal/app/body_test.go`、`tabs_test.go`）。
- **标题**：固定宽度行里「刚好够宽」的标签会被工具包取整成省略号，要显式 MinWidth 给 2 DIP 余量
  - **触发信号**：某个短标签（`XML`）在胶囊行里显示成「X…」，而同样的字在别处正常；量出来的胶囊宽度恰好等于「标签 advance + 两侧内边距」。
  - **根因/约束**：`ButtonBase` 的自然宽度 = 子元素 max-content + padding，工具包排版时按浮点宽度取整分配，子文本拿到的可用宽度正好等于自己的 advance 时判为「超了」，于是缩成「首字 + …」。
  - **正确做法**：胶囊宽度用 `MinWidth(2*padding + measure + tabWiggle)`（`bodyPillWidth`，`tabWiggle = 2`，与标签页标题同一手法），并把整行格式胶囊包进 `noScrollbars(ui.ScrollHorizontal(c)).Grow(1)`：窗口窄时横向滚动，而不是把某个胶囊压成一条缝。
  - **验证方式**：`TestBodyFormatPillsShowTheirLabels`（每个胶囊余量 > 标签 advance）；截图 `docs/shots/body-json.png` 里九个格式全名可见。
  - **适用范围**：`internal/app/view_request.go`（格式胶囊）；任何「子文本 + 显式内边距 + 刚好够宽」的自绘按钮。
- **标题**：可选项（强调色/外观/语言/响应体上限）是调色板与主题的参数，别在常量里写死；改一个就落盘一份 settings.yaml
  - **触发信号**：加了「强调色」设置却发现只有自绘文字变色、按钮还是绿色；设置改完重启就没了。
  - **根因/约束**：`Palette` 里的 `Accent/AccentHover/AccentPressed/AccentText` 原来写死在 `light`/`dark`/`black` 三个常量里；`ui.Theme` 也复制了一份，两个入口都得换。窗口主题按「外观+强调色」缓存（`themeAccent`），只比对外观会漏掉换色。
  - **正确做法**：`design.WithAccent`/`PaletteFor` 由基色派生阴影色与文字色，`Theme(m, accent)` 一并接收；`app.palette()` 与 `app.themeFor()` 都从这里取。设置在 `store.settings`（`settings.yaml`）里持久化，`internal/app/run.go` 启动 `load()` 后调 `applySettings()`；UI 改动处调 `saveSettingsOrToast(c)`，别静默失败。
  - **验证方式**：`TestSettingsModal`（点背景/强调色后 `palette()` 与 `theme.Accent` 同步）、`TestSettingsRoundTrip`（重启后外观/强调色/语言/响应体上限还在）、`TestResponseLimitFollowsSettings`（上限的来源与默认）、`TestSendCutsTheBodyAtTheLimit`（超过上限截断并提示）、`settings-accent` 截图。
  - **适用范围**：`internal/design/accent.go`、`internal/design/theme.go`、`internal/app/app.go`（`palette`/`themeFor`/`applySettings`）、`internal/app/view_env.go` 的设置分区、`internal/store/service_settings.go`。
- **标题**：手型指针要在每个可点击元素上显式 `.Cursor(ui.CursorPointer)`，给祖先容器设无效；`ui.Select` 的选项行设不了光标
  - **触发信号**：按钮/菜单项/标签页/树行悬浮时仍是箭头；给外层 `ui.Row` 设了 `.Cursor(CursorPointer)` 却只有空白处变手型；`ui.Select` 的触发器能改、展开后的选项永远是箭头。
  - **根因/约束**：`ui/runtime.go` 的 `updateCursor` 从指针下最内层元素向外找 cursor，但**停在最内层带 `flagClickable` 的元素**——该元素自己没设 cursor 就显示默认箭头，不再向外继承。`ui.Select` 的选项行由框架内部构建，没有可写的 `ui.Element`，所以它展开后永远是箭头；`ui.TextInput` 由框架负责 I 形光标。
  - **正确做法**：按钮类（`ui.Button`/`PrimaryButton`/`ButtonBase`/`Checkbox`/`PopoverBase` 锚点）逐处链式追加 `.Cursor(ui.CursorPointer)`，不要写 helper 包；需要选项也有手型的下拉用 `ui.SelectBase` 自建（`sel.Trigger` 加 cursor 与 `Hovered()` 换底色、`sel.Popup` 里 `sel.Item(v).Cursor(...)`，变更判断用 `sel.Trigger.Changed()`），弹出面板复用 `internal/app/view_menu.go` 的 `menuPanel`，选项加 `.Label("请求体格式 "+v)` 以免与触发器上的当前值撞名。`ui.Text`/`ui.Icon`/容器/`ui.Scroll`/`ui.TextInput`/`Modal` 背板一律不加。
  - **验证方式**：`internal/app/cursor_test.go` 的 `TestWorkspaceControlsShowPointer`（发送、请求方法、标签页、关闭标签页、新建标签页、美化请求体、添加行、复选框、根目录行、新建接口、筛选类型）与 `TestMenuItemsShowPointer`（方法项、设置里的关闭/更换目录/强调色/跟随系统、`请求体格式 text`）；写法是 `tt.Find(label)` → `tt.Move(r.X+r.W/2, r.Y+r.H/2)` → `tt.Cursor() != ui.CursorPointer` 报错。改完顺手用 `APIGO_SHOTS=<临时目录> go test -run TestShots ./internal/app/` 与改动前的图逐张 `cmp`，除预期那一屏外应逐字节一致。
  - **适用范围**：`internal/app/view_topbar.go`、`internal/app/view_home.go`、`internal/app/view_requests.go`、`internal/app/view_kind.go`、`internal/app/view_menu.go`、`internal/app/view_request.go`、`internal/app/view_response.go`、`internal/app/view_tabs.go`、`internal/app/view_env.go`、`internal/widgets/kvlist.go`；以及以后任何新增按钮、菜单项、列表行的地方。
- **标题**：顶部集合标签被压成「测…」：`Row` 溢出时按 flex 压缩子元素，标签条要独立横向滚动
  - **触发信号**：窗口变窄或打开的集合变多后，集合标签只剩「测…」；同一个集合在宽窗口里显示完整。`ui.Tester` 里量到的标签 x 坐标出现 `238.03` 这种小数，也是同一个压缩。
  - **根因/约束**：`ui.Row` 在内容宽于可用宽度时会收缩子元素（`shrink` 默认 1）；`ui.Text(...).SingleLine()` 拿到收缩后的宽度就省略号。`ButtonBase` 自带的 `Shrink(0)` 只保护按钮自身，包在它外面的 `ui.Row`（标签、行）仍会被压缩；再往外的 `ui.Text` 也会跟着被压。
  - **正确做法**：滚动区用 `ui.ScrollHorizontal`，里面的 `ui.Row` 与每个标签都显式 `.Shrink(0)`，名字就拿到自然宽度，超宽时裁切并可横向滚动；常驻动作（「更多」、齿轮）留在滚动区之外。当前所在的标签（集合或主窗口）图标与文字用 `p.Accent`，其余 `p.TextMuted`，这样「在哪个集合」一眼可见。
  - **验证方式**：`internal/app/topbar_test.go` 的 `TestCollectionTabsShowFullName`（640 / 420 宽下长名标签的 X、W 与 1280 宽一致；`tt.Scroll` 后末位标签左移）、`TestOpenCollectionTabShowsAccent`（当前标签有强调色像素、其他标签没有；点「主窗口」后强调色随之转移）、`TestTopBarScrollsWithoutBar`（指针停在标签条上时，标签下那条像素带仍是纯 `p.Surface`，也就是没有滚动条）。前两条在改回旧写法时实测 FAIL，第三条在去掉 `noScrollbars` 时实测 FAIL（`{184,184,184}` 的条）。
  - **适用范围**：`internal/app/view_topbar.go`（`topBar`/`collectionTab`/`homeButton`）；请求标签条（`internal/app/view_tabs.go` 的 `requestTabs`，同样是 `ScrollHorizontal` + `noScrollbars`，每个标签按 `design.RequestTabMaxWidth` 截断标题）也照此办理。
- **标题**：列表/标签的宽度上限：`MaxWidth` 在内容自适应的 `Row` 上无效、在 `Text` 上会把名字吞成「测…」；用 `Grow` 比例或自己截断
  - **触发信号**：给标签 `ui.Row(...).MaxWidth(240)` 后宽度照旧；改给名字 `ui.Text(...).MaxWidth(186)` 后**所有**名字（包括两个字的）都变成「测…」。`ui.Tester` 里量到的文本宽度正好等于 `c.MeasureText` 的值，渲染却仍是省略号。
  - **根因/约束**：`.MaxWidth` 不参与内容自适应 `Row` 的 flex 收缩；放到 `Text` 上会让它按「最小内容宽度」落位，于是整字被省略号取代。另一头，`c.MeasureText` 与绘制时的行布局有零点几 DIP 的差，`SingleLine()` 按「放不下」处理，会把刚好填满的名字整字吞掉。
  - **正确做法**：要「占窗口几分之几」用比例布局——`Grow(2)` 的内容 + `Grow(1)` 的空白 = 精确 2/3，随窗口变化，不需要测量；要「单个标签最宽」自己截断字符串（`tabName`：用 `c.MeasureText` 逐字裁到上限再加 `…`），并给名字留够 `textWidth+1`，否则刚好填满的名字会被吞成「测…」。
  - **验证方式**：`internal/app/topbar_test.go` 的 `TestCollectionTabCapsItsWidth`（长名标签 ≤ `design.CollectionTabMaxWidth` 且文本以 `…` 结尾；短名标签没被压到上限）、`internal/app/tabs_test.go` 的 `TestRequestTabCapsItsWidth`（请求标签 ≤ `design.RequestTabMaxWidth`、标题以 `…` 结尾）、`internal/app/home_test.go` 的 `TestHomeCollectionListTakesTwoThirds`（1280 / 640 宽下列表右缘 = `32+(宽-64)*2/3`）。把上限调成 10000 或改回 `MaxWidth` 时实测 FAIL。
  - **适用范围**：`internal/app/view_topbar.go`（`tabName`/`textWidth`/`collectionTab`）、`internal/app/view_tabs.go`（`tabContent` 用同一对函数与 `design.RequestTabMaxWidth`）、`internal/app/view_home.go`（`homeView` 的两列比例）、`internal/design`（`CollectionTabMaxWidth`/`RequestTabMaxWidth`）；以后任何「最大宽度」需求先想清楚是「占比例」还是「单元素封顶」。
- **标题**：换成嵌入字体后，按钮里的文字会被「内在宽度」算短：标签会缩到 MinWidth，方法被写成「PO…」；要的是 `MinWidth(测得所需宽度)`，不是 `Width()`
  - **触发信号**：请求标签里 `POST` 显示成「PO-…」而 `GET` 正常；标签宽度正好等于 `design.TabMinWidth`；换 `.Width(所需宽度)` 后文字正常了，但标签右端的 × 点不动（标签关不掉、接口文件也没写回）。
  - **根因/约束**：主题字体换成 `HarmonyOS Sans SC` 后，MyGo 对文本的 max-content 测量比实际绘制偏小，于是 `ButtonBase` 的内在宽度小于内容需要 → 收缩到 `MinWidth`，固定宽/自然宽的子元素先在有限空间里分配，最先让位的正是方法文字。改设 `.Width()` 能让内容排开，但 body 的宽度不再参与父 `Row` 的内在计算，右边的 × 被排到 tab 盒子之外，点击落不到它身上（`Clicked()` 永假）。
  - **正确做法**：`internal/app/view_tabs.go` 里按测量值算 `chrome`（内边距、方法、间隙、未保存点、×），标题用 `tabName(c, title, RequestTabMaxWidth-chrome-2)` 截断，body 用 `.MinWidth(max(TabMinWidth, chrome+textWidth(label)+2))`，标题文本再单独 `.MinWidth(textWidth+2)` 防「刚好填满被吞」；`textWidth` 固定按正文字号测，`measureText(c, s, size, weight)` 用来测方法（10px/700）。
  - **验证方式**：`internal/app/tabs_test.go` 的 `TestRequestTabCapsItsWidth`（长标题标签 ≤ `design.RequestTabMaxWidth` 且文本以 `…` 结尾）、`TestRequestTabStripHidesItsBar`、`TestEnvironmentPickerHasNoEye`；改回 `.Width()` 时 `internal/app/main_test.go` 的 `TestCloseTabWritesInterfaceFile` 立刻失败（× 点不到、文件没写回）。
  - **适用范围**：`internal/app/view_tabs.go`（`tabContent`/`requestTabChrome`/`tabWiggle`）、`internal/app/view_topbar.go`（`textWidth`/`measureText`）；任何「按钮里放文字 + 右侧还有兄弟按钮」的标签类控件。测试里编辑完标签内容后先 `tt.Frame()` 再点标签上的按钮——未保存点会让标签变宽，旧坐标会点偏。
- **标题**：MyGo 0.6.0 仍没有「隐藏滚动条」开关，用 `ScrollbarInsets` 把条挪出窗口就是最省事的做法
  - **触发信号**：标签条/工具条这类高度刚好等于内容高度的滚动区，滚动条会压在内容底边上（40 高的标签条里，条画在 y=31..37，正好压住 32 高的标签下沿）；主窗口集合列表的竖条会贴在窗口最右边。
  - **根因/约束**：滚动条由 `paint.go` 的 `scrollbars` 用 `theme.Scrollbar` 画，主题是整帧全局的，改主题会连侧栏、响应区的条一起改；`ScrollbarInsets` 只改条的几何（轨道与滑块的摆放），不影响布局、裁剪和滚轮逻辑——负值会画到容器外（不裁剪），所以只能往「窗口外」挪，不能靠负值把它压走。
  - **正确做法**：`noScrollbars(e)` = `e.ScrollbarInsets(away, 0, 0, away)`（top 与 left 各给一个远大于任何窗口的常量，横竖条都被移出画面），滚轮、触摸板、Shift+轮子的滚动和 `ScrollIntoView` 全部照旧；只在「条会压住内容」的容器上这么用，正文滚动区保留可见滚动条。代价是滑块拖拽消失（条不可见，本来也拖不到）。
  - **验证方式**：`TestTopBarScrollsWithoutBar`（标签条：指针移上去后，标签下沿往下的 7 像素带逐像素比对 `p.Surface`）与 `TestHomeCollectionListDrawsNoBar`（主窗口列表：指针移上去后，窗口右侧 x∈[宽-44, 宽-34] 的竖带逐像素比对 `p.Background`）；去掉 `noScrollbars` 时两处都会立刻报 `{184,184,184}` 的条。
  - **适用范围**：`internal/app/view.go`（`noScrollbars`）、`internal/app/view_topbar.go`（标签条）、`internal/app/view_tabs.go`（请求标签条）、`internal/app/view_home.go`（集合列表）；以后任何「不要滚动条的横向工具条或列表」。
- **标题**：键值列表不再画「添加行」；env 变量与 Params 各用各的行视图，别硬塞一个 `State.View`
  - **触发信号**：列表尾部有「+ 添加行」按钮占一行（图 3）；环境变量想改成一行一个、只要变量名 + 值 + 删除，却不想带复选框和「添加行」。
  - **根因/约束**：`widgets.State.View` 是「复选框 + key + value + 删除 + 添加行」的固定组合，任何差异都要么改它、要么另画；`State` 的增删是延迟的（`remove`/`add` 在下一帧 `Flush` 里生效），自己画行时仍要复用它，否则边遍历边删会越界。
  - **正确做法**：`internal/widgets/kvlist.go` 的 `View` 去掉「添加行」按钮（末尾那个空行本身就能开始新行，`Flush` 会补上）；需要不同列的环境变量用 `internal/app/view_env.go` 自己的 `envVarRow` 画（`State.Rows` 直接绑定 TextInput），删除走新加的 `State.RemoveAt(i)`，改完调 `State.OnChange()` 立刻落盘；遍历用 `append([]widgets.Row(nil), rows...)` 快照并检查下标。
  - **验证方式**：`internal/app/env_test.go` 的 `TestEnvironmentManagerHasTwoColumns`（左列表在编辑器左侧、变量名/值/删除在同一行、页面里没有「添加行」、删除图标是 `Palette.Danger`）、`TestEnvironmentVariableRowDeletesItself`（点删除后环境里少了 `baseurl`、行数回到「剩下的 + 空行」）、`TestEnvironmentMenuMarksTheActiveOne`（激活的环境有主题色打勾与底色，其它行保持面板底色）；`internal/app/main_test.go` 的 `TestRequestTabs` 改为按「键第 1 行」判断 Params 列表在。
  - **适用范围**：`internal/widgets/kvlist.go`（`View`/`RemoveAt`）、`internal/app/view_env.go`（`envList`/`envRow`/`envEditor`/`envVarRow`）、`internal/app/view_tabs.go`（`environmentPicker`）；以后任何「同一种状态、不同列布局」的列表。
- **标题**：改名要连带改名磁盘文件；树表跟磁盘走——打开时整份重读、之后由目录监听驱动
  - **触发信号**：在标签里重命名请求后，树表和标签页是旧名、磁盘文件名没变；用编辑器改了集合目录里的接口，App 里看不到；接口目录里冒出 `xxx.yaml.tmp` 半截文件。
  - **根因/约束**：标签标题只是内存里的 `t.title`，接口的身份是磁盘上的 `<名字>.yaml`；集合的文件随时可能被别的程序改。监听要用 `mygo.WatchFiles`（0.5.5 起就有，可递归、带去抖与 `Ignore`），但它要求 app 已启动，测试里不能直接调，所以要留一层可替换的接缝。
  - **正确做法**：`commitTabRename`（`a` 上的方法）改名后走 `store.renameRequest`（`uniqueName` 对同级已有名字编号）→ `retargetTabs(old, new)`；集合文件同步用**目录监听**：`internal/app/app_watch.go` 的 `followFiles()` 在每帧比对 `a.active` 与 `a.watched`，换集合就 `stopWatch()` 再 `startWatch()`（`Recursive: true`、`Debounce: 150ms`、`Ignore: reservedInCollection` 跳过 `.env`/`static`/点开头目录），`startWatch` 把 `mygo.WatchFiles` 的 `*mygo.FileWatcher` 收进 `fileEvents` 接口，`drainWatch` 在监听 goroutine 上等 `Next`、再把 `reloadActive()` 交回窗口线程（`a.after`，测试里是通道，没设就就地跑）；`startWatch` 只在集合目录存在时才真去监听，并先写 `a.watched` 表示「已尝试」，平台起不来时只提示一次而不是每帧重试。`openCollection` 仍整份重读一次。临时文件统一写到所属集合根目录的 `.tmp/`，`rename` 到位；打开集合时 `removeTempFiles` 清掉崩溃残留。
  - **验证方式**：`internal/app/tree_test.go` 的 `TestRenameTabRenamesTheFile`（标签改名后 `t.path`、磁盘文件、树表一起变）、`TestTempFilesStayInTmpDir`（接口目录里没有 `*.tmp`、`.tmp/` 存在）、`internal/app/watch_test.go` 的 `TestWatchFindsFilesChangedOutside`（假监听报一个 create，树表立刻出现该接口）、`TestWatchFollowsTheOpenCollection`（换集合时旧监听被关、开新监听；回主窗口停监听）、`TestWatchSkipsCollectionDirs`（`.env`/`.tmp`/`static` 被忽略）；`internal/app/env_test.go` 的 `TestEnvSelectionFollowsTheClick`。真实平台后端可以这样冒烟：写个几行的 mygo 小程序，`App.WhenReady` 里建窗口 + `mygo.WatchFiles`，再从另一个 goroutine 写文件，`Next` 应返回 `create` 事件（本项目在 Windows 上实测得到 `{create 分组 1}` 与 `{create 分组 1/接口.yaml}`）。
  - **适用范围**：`internal/app/app_watch.go`、`internal/app/app.go`（`commitTabRename`/`reloadActive`/`watched`）、`internal/store/service_move.go`（`renameRequest`）、`internal/store/service_reload.go`、`internal/store/service_yaml.go`（`writeYAML`/`tmpRoot`）、`internal/store/service_static.go`、`internal/app/run.go`（`a.watch = watchFiles`）；以后任何「外部可改的持久化目录」都按这个套路接。
- **标题**：带「取消 / 保存」的弹窗要「先改内存、保存才写盘」，取消用快照回滚
  - **触发信号**：环境弹窗里改一个变量名、切一下激活的环境，磁盘立刻多了一次写入；点取消却提示「已经保存」，改不动回去。
  - **根因/约束**：`store` 的 `activateEnvironment`/`renameEnvironment`/`addEnvironment`/`deleteEnvironment` 都是「改内存 + 立即 `saveEnvironments()`」，variables 也是每次 `OnChange` 就写；这些方法被别处（下拉、新建集合）复用，不能直接改成不落盘。
  - **正确做法**：新增一组**只改内存**的 `selectEnvironment`/`nameEnvironment`/`newEnvironment`/`forgetEnvironment`，弹窗全用它们；打开弹窗时 `a.envEdits = store.cloneEnvironments()`（深拷贝 Vars），`取消` 或 `ui.Modal` 自己关掉（Esc/背板，`envModal` 开头检测 `envEdits != nil && !envOpen`）都走 `discardEnvEdits()` 把快照塞回去；`保存` 才 `sync 变量进环境 + saveEnvironments()` 并关窗。变量输入不再挂 `OnChange`（否则一敲键盘就写盘），`State.RemoveAt` 只动延迟删除标记。
  - **验证方式**：`internal/app/env_test.go` 的 `TestEnvironmentManagerCancelDropsEdits`（点 环境 测试 后内存里已切换，点 取消 后回到 本地 且弹窗关闭）、`TestEnvironmentVariableRowDeletesItself`（点删除后行数先减、环境里的变量在点 保存 之后才少），`TestEnvironmentManagerHasTwoColumns` 仍验两栏布局与警示色删除图标；`internal/app/main_test.go` 的 `TestEnvironmentModal` 只断言内存里的增删，不看落盘。
  - **适用范围**：`internal/app/app.go`（`envEdits`/`openEnvManager`/`editEnvironment`/`saveEnvEdits`/`discardEnvEdits`）、`internal/store/service_env.go`（上面四个内存版方法 + `cloneEnvironments`/`restoreEnvironments`）、`internal/app/view_env.go`（`envModal`/`envRow`/`envEditor`/`saveButton`）；以后任何「先编辑后确认」的弹窗照此办理。
- **标题**：树行的菜单挂在「⋯」按钮上，不用右键；集合刚打开时给一块「开始提示」而不是空请求面板
  - **触发信号**：右键点树行才弹菜单，用户（尤其触控板）难发现；打开集合先看到一片空的 Params/响应区，不知道下一步做什么。
  - **根因/约束**：`RightClicked()` 是「指针下那个元素」的事件，行里塞了 ⋯/+ 之后容易落在子按钮上；集合的 `workspace` 一开始很自然地被塞一个空标签，如果没有「空态」判断就会直接画请求面板与标签条。
  - **正确做法**：每行在 `+` 之后加一个 `moreButton`（`icons.MoreVertical`：竖向的「⋮」，只有树里这样，顶栏「更多」与集合行的「…」仍用横向 `icons.More`；`Label("接口操作 <名字>")`/`"文件夹操作 <名字>"`）当作菜单锚点，`RightClicked()` 不再挂；**根目录（`f.Path == ""`）没有 `⋮`**——它没有可重命名/移动/删除的东西，`folderActions` 对根目录返回 nil，画一个空菜单只是噪音；菜单内容按类型分：接口 `重命名 / 复制接口 / 移动接口 / 删除接口`，子文件夹 `重命名 / 移动文件夹 / 删除文件夹`（删除仍走 `confirm` + `Danger`）。`newWorkspace()` 一开始**没有任何标签**（`ws.empty()`），`collectionWorkspace` 里空工作区只画 `startHint`（标题 + 说明 + 四行「动作 + 键帽」）——请求标签条（含环境下拉）随第一个标签出现、随最后一个标签离开，`openRequest`/`addRequest`/`addTab` 统一走 `openTab(ws, t, i)`（`i` 超出当前长度会夹到末尾，空工作区就是第 0 位）。
  - **验证方式**：`internal/app/tree_test.go` 的 `TestRowMenuOpensFromTheMoreButton`（右键不再弹菜单、⋯ 弹出四项）、`TestInterfaceMenuRenamesAndCopies`（重命名连接口文件改名、复制得到「X 副本」）、`TestFolderMenuRenames`（文件夹连带目录改名）、`TestCollectionOpensOnAStartHint`（打开集合只见到提示、没有「新建标签页」/环境下拉；开接口后标签条与请求面板出现；关掉最后一个标签提示回来）；`internal/app/main_test.go` 的 `TestCloseTabLeavesNoTabAtTheEnd`；`internal/app/cursor_test.go` 的指针用例把标签名改成 `Pets`。
  - **适用范围**：`internal/app/view_requests.go`（`folderRow`/`requestLineRow`/`moreButton`/`folderActions`）、`internal/app/view_start.go`、`internal/app/view.go`（`collectionWorkspace`）、`internal/app/app.go`（`workspace.empty`/`newWorkspace`/`openTab`/`openRequest`/`closeTab`）、`internal/app/app_tree.go`（`askRenameRequest`/`askRenameFolder`/`duplicateRequest`/`forgetTabsUnder`）、`internal/store/service_move.go`（`renameFolder`/`duplicateRequest`）。
- **标题**：会丢上下文的动作先弹确认；「删除集合」做成伪删除，绝不 `RemoveAll` 集合目录
  - **触发信号**：点集合标签的 × 或菜单里的「删除集合」直接生效，用户来不及看清是哪个集合；`os.RemoveAll` 一下把整目录的接口与环境删掉，找不回来。
  - **根因/约束**：关闭集合会写回并丢掉标签的布局，删除集合在旧实现里是 `os.RemoveAll(collectionDir)`；两者都不可撤销，而菜单项与 × 就挨着常用按钮。数据目录是用户自己的文件，程序不该替用户做不可逆的事。
  - **正确做法**：动作先落进 `app.question`（`internal/app/view_confirm.go` 的 `confirm`：标题、说明、按钮文案、`danger`、`run func(c *ui.Context)`），由 `confirmModal` 用 `DialogBase` 画；`run` 接收**当时的** `c`（别在 ask 时捕获旧 Context）。删除走 `store.hideCollection`：把 id 记进数据目录根的 `hidden.yaml`、从内存与 `s.environments` 里摘掉（不要调 `saveEnvironments` 的 prune 之外的东西，否则会碰隐藏集合的 `.env`），目录与文件原地保留；`load()` 先读 `hidden.yaml` 再跳过这些目录。确认按钮的无障碍名用「确认+动作」（`确认删除集合`），避免和菜单项同名撞车。
  - **验证方式**：`internal/app/confirm_test.go` 的 `TestCloseCollectionAsksFirst`（弹窗出现、取消不改状态、确认才关闭）与 `TestDeleteCollectionKeepsItsFolder`（`collectionDir`、`apigo.yaml`、`.env` 都还在，`hidden.yaml` 写入，重载后不再列出、`isHidden` 为真）；`internal/app/main_test.go` 里原来直接删除的用例改为点 `确认删除集合`。
  - **适用范围**：`internal/app/view_confirm.go`、`internal/store/service_hidden.go`、`internal/store/service_store.go`（`load` 跳过隐藏）、`internal/app/app.go`（`deleteCollection`）、`internal/app/view_topbar.go`、`internal/app/view_home.go`。
- **标题**：请求体格式做一张表（`bodyFormats`），编辑方式、Content-Type 与发送链路都从它派生
  - **触发信号**：加一种 body 格式要在三处改（下拉列表、编辑器分支、`service_http` 里写死的 `application/json`），漏一处就会「选了 form-data 却按 JSON 发」。
  - **根因/约束**：旧实现只有 `none/json/text` 三个字符串，Content-Type 固定 `application/json`；body 只有一段文本，表单一类格式没有落点。`ui.Select` 的选项行又无法设手型（见光标那条），格式一多更需要整行铺开的胶囊。
  - **正确做法**：`internal/app/app.go` 里用 `bodyFormats`（`Key/Label/MIME/Edits`）当唯一真相源，`edits` 决定面板怎么编辑（`editsNone`/`editsRows`/`editsText`/`editsFile`）、`bodyReader()` 怎么组装（multipart / urlencoded / 文本）；`tab.form widgets.State` 存表单行，`savedRequest.Form` 落盘，`savedFromTab` 按 `edits` 决定存 `body` 还是 `form`。格式胶囊用 `bodyFormatStrip`（选中填强调色、悬浮 `SurfaceHover`），Content-Type 与必填 `*` 右对齐，魔杖只对 JSON/GraphQL 出现。没有实现的格式（Binary/msgpack）在选择上保留、面板给明确提示，发送时既不带 body 也不带 Content-Type。
  - **验证方式**：`internal/app/body_test.go` 的 `TestSendBodyFormats`（七种格式各自的 Content-Type 与 payload，none/Binary/msgpack 不带 body）、`TestSendURLEncodedForm`（`name=Ada+Lovelace`，禁用行不发）、`TestSendMultipartForm`（带 boundary、服务端解析到字段）、`TestBodyPaneOffersTheFormats`（九个胶囊都在、切到 form-data 出现键值表、Binary 出提示）、`TestFormBodyRoundTrips`（`form` 行存进接口文件再读回）。
  - **适用范围**：`internal/app/app.go`（`bodyFormats`/`bodyFormatOf`/`setBodyLanguage`/`savedFromTab`/`send`）、`internal/app/view_request.go`（`bodyPanel`/`bodyFormatStrip`）、`internal/req/http.go`（`requestSpec.bodyReader`）、`internal/store/service_store.go`（`savedRequest.Form`）；以后加格式只改这张表与 `bodyReader` 的一个分支。
- **标题**：form-data 的文件字段要「复制进集合 + 换/清/删都回收旧文件」，槽位用可注入的 `chooseFile`
  - **触发信号**：字段里选了文件，接口文件只记住一个路径、旧文件永远躺在 `static/`；或者没保存过的标签选文件时报「没有集合」；或者测试里点 `选择文件` 直接弹系统对话框。
  - **根因/约束**：文件必须先落到集合目录里才能被 multipart 读到，而标签的 `collectionID` 只有「从接口打开」时才有值（新建标签是空的）；`mygo.Dialog.Open` 依赖已启动的事件循环，测试里不能真弹；`static/` 的文件寿命必须跟着字段走，否则换一次文件留一份垃圾。
  - **正确做法**：`app.chooseFile func() (string, error)` 由 `internal/app/run.go` 填 `mygo.Dialog.Open`（`chooseUpload`），测试注入返回临时文件的函数；文件的落点是 `a.staticScope(t)`（`collectionID`，空则用当前打开的集合 `a.active`），`store.ImportStatic` 复制到 `static/<2006-01-02>/<sha256>`（同名内容只存一份，超 50 MB 拒绝），`removeStatic` 只删 `static/` 开头的相对路径（防目录穿越）。`savedRow`/`widgets.Row` 各带 `File`（相对路径）与 `FileName`（原始名），接口文件因此能原样还原；`kvlist.State.ChooseFile`/`FileDropped` 两个钩子把「选文件」和「回收旧文件」交给 app：换文件、点清除 ×、删行、`Flush` 移除行都会调 `FileDropped`。发送时 `app.formParts` 把相对路径解析成绝对路径，`bodyReader` 用 `CreateFormFile(key, fileName)` 写出文件部分。
  - **验证方式**：`internal/app/upload_test.go` 的 `TestImportStaticKeepsFileByDayAndHash`（落在 `static/<日期>/<sha256>`、同内容去重、`removeStatic` 删除）、`TestRemoveStaticKeepsWhatItDoesNotOwn`（`static/` 之外的路径不动）、`TestFormFileUploadsTheChosenFile`（点「选择文件第 1 行」→ 文件进集合 → 发送后服务端收到 `filename=Vue-Go冲锋Logo.png` 与字节）、`TestFormFileReplacedDeletesTheOldOne`、`TestFormFileClearedDeletesIt`、`TestFormFileDeletedWithItsRow`、`TestFormFileRoundTrips`（`file`/`fileName` 存进接口文件再读回）。
  - **适用范围**：`internal/store/service_static.go`、`internal/app/app.go`（`staticScope`/`pickFormFile`/`dropFormFile`/`formParts`）、`internal/widgets/kvlist.go`（`fileCell`/`ChooseFile`/`FileDropped`）、`internal/store/service_store.go`（`savedRow.File`/`FileName`）、`internal/req/http.go`（`multipartBody`/`writeFilePart`）、`internal/app/run.go`（`chooseUpload`）；以后做 Binary/msgpack 的文件上传可复用这一套。
- **标题**：新建接口是草稿（不落盘），未保存靠「快照比对 + 灰点」，`Ctrl+S` 时才问存到哪个文件夹
  - **触发信号**：点了树里的 `+`，磁盘上立刻多出一个 `Untitled.yaml`，改完还没保存就留下空接口；或者标签上的圆点只在「没选中」时出现、编辑了也不出现；或者 `Ctrl+S` 只在 REST 面板里生效，WebSocket 标签按了没反应。
  - **根因/约束**：旧实现 `addRequest` 先写文件再开标签，用户还没决定要不要这个接口，磁盘已经脏了；旧的圆点是 `dirty || hasContent`，既不代表「和文件不一致」，也只在未选中时画；`OnShortcut` 挂在 `a.request`（REST 专用）里，非 REST 类型走的是 `emptyKind` 分支，快捷键根本不注册。
  - **正确做法**：`addRequest` 只开草稿（`path == ""`，记住 `folder` 作为默认存放位置），`store.addRequest` 到 `Ctrl+S`（`app.saveCurrent`）才调用；`tab.unsaved()` 用**快照**判定——`markSaved()` 在打开、写回、保存后各记一次 `snapshot()`，之后内容一变就不相等；圆点在活动标签上也画（`Label("未保存 "+title)`，方便测试定位）；`Ctrl+S` 注册在 `collectionWorkspace`，任何接口类型都能存；没有 `path` 时弹 `folderQuestion`（`names: true`，名字预填标签标题、可在弹窗里改），有 `path` 时直接 `updateRequest`。
  - **补充（一个接口一个标签）**：`openRequest` 先用 `a.tabOfPath(ws, r.Path)` 找已开着的标签，找到就 `selectTab` 切过去而不是再开一个；草稿标签 `path == ""` 不参与去重，所以 `+` 可以开多个草稿。
  - **验证方式**：`internal/app/main_test.go` 的 `TestNewRequestAddsToCollection`（新建后树里没有它、`unsaved()` 为真、`tt.Key(ui.Cmd, ui.KeyS)` 弹出「保存接口」、改名为「订阅设备」再选文件夹后文件才出现且用新名字、圆点消失）、`TestNewInterfaceWritesFile`（两次保存各得 `Untitled.yaml` / `Untitled 2.yaml`）、`TestFolderTreeAddsAndNavigates`（`+` 所在的文件夹就是默认保存位置）；`internal/app/tree_test.go` 的 `TestRequestHasOneTab`（再次打开同一接口只切标签，不开新标签）。
  - **适用范围**：`internal/app/app.go`（`tab.snapshot`/`markSaved`/`unsaved`/`addRequest`/`openRequest`）、`internal/app/app_tree.go`（`saveCurrent`/`askWhereToSave`/`tabOfPath`）、`internal/app/view_tabs.go`（圆点）、`internal/app/view.go`（`collectionWorkspace` 的 `Ctrl+S`）。
- **标题**：树的「当前接口」高亮看的是**激活的标签**；弹窗里的层级缩进不能拿空路径算出 -1
  - **触发信号**：开了三四个标签后树里好几行一起变绿，分不清现在在改哪个；保存接口的弹窗里「根目录」那一行的图标被挤出选中条外面（左边距算成负数）。
  - **根因/约束**：高亮最初按「有没有标签打开这个接口」判定，于是所有开着的都亮；缩进用 `Padding(0, 8, 0, 8+depth*12)`，而 `depthOf` 之前写成 `len(splitPath(path))-1`，空路径（根目录）拆出 0 段就得到 **-1**，负的左边距把内容推到面板外。
  - **正确做法**：高亮只问 `a.activeShows(req.Path)`（`a.current()` 的 `path` 且非空）；凡是「按层级缩进」的辅助函数，根目录必须落在 0，越界要夹住（`if path == "" { return 0 }`）。弹窗里的文件夹列表按 `depthOf` 缩进、`FillWidth` 撑满，所以层级只体现在内边距上。
  - **验证方式**：`internal/app/tree_test.go` 的 `TestOpenTabHighlightsItsRow`（打开 Pets 后它亮；再打开 列表 后只有 列表 亮、Pets 回到页面底色；草稿标签不点亮任何行）、`TestFolderPickerIndentsByDepth`（弹窗里 根目录 / 分组 1 / 子分组 依次向下，`depthOf` 分别是 0 / 1 / 2）。
  - **适用范围**：`internal/app/app_tree.go`（`activeShows`）、`internal/app/view_requests.go`（`requestLineRow`）、`internal/app/view_picker.go`（`depthOf`/`splitPath`/`folderChoice`）；以后任何「按树层级缩进」的列表。
- **标题**：树里的删除与移动：真实删除才用 `Danger`，移动后要把指针重新找一遍
  - **触发信号**：删除文件夹后界面报「没有文件夹」或树里少了一层；同一个菜单里「删除」和普通项一个颜色，用户看不出危险。
  - **根因/约束**：`findFolder` 返回的是**切片元素指针**，`src.Folders = append(src.Folders[:i], src.Folders[i+1:]...)` 会把后面的元素整体前移，删除前取到的 `dst` 指针会指向错位的元素（移动到它自己或直接 panic）；MyGo 的菜单项颜色只有一个入口（`popoverMenu` 的文本色）。
  - **正确做法**：`internal/store/service_move.go` 里先按路径把源节点摘掉，再 `findFolder(&col.Root, target)` **重新找**目标；`menuAction` 加 `danger bool`，`popoverMenu` 据此把图标与文字都染成 `p.Danger`；删除接口删 `.yaml`、删文件夹 `os.RemoveAll` 整个目录，都走 `a.ask(confirm{...})`（红色确认按钮）并在确认后 `forgetTabsUnder` 关掉相关标签；移动后 `retargetTabs` 让已打开标签的 `path` 跟着改。
  - **验证方式**：`internal/app/tree_test.go` 的 `TestDeleteRequestAsksAndDeletes`（文件消失、树更新、标签关闭）、`TestDeleteFolderAsksAndDeletes`（目录连同 1 个接口消失）、`TestMoveRequestIntoFolder`（文件搬到 `分组 1/Pets.yaml`、标签 `path` 跟着变）、`TestMoveFolder`（嵌套移动后路径重算、不允许移进自己）、`TestDeleteMenuReadsInDangerColor`（菜单行里能采到 `p.Danger` 像素）。
  - **适用范围**：`internal/store/service_move.go`、`internal/app/app_tree.go`、`internal/app/view_menu.go`（`danger`）、`internal/app/view_requests.go`（行右键菜单）、`internal/app/view_picker.go`（文件夹选择弹窗）。
- **标题**：树行的 hover 与「当前接口」高亮：`Hovered()` 自己铺底，高亮用强调色的 12% 叠在页面上；弹窗一律顶部居中
  - **触发信号**：鼠标划过接口树没有反馈，看不出会点中哪一行；开了好几个标签后，树里分不清哪个接口正开着；确认弹窗挡在窗口正中，正好盖住它要说明的那份列表。
  - **根因/约束**：`ui.Row` 默认不跟踪悬浮，要调一次 `Hovered()` 它才会被塞进 hover 链（副作用是设置 `flagHover`），所以底色得自己按 `Hovered()` 铺；强调色不透明，直接铺会盖住文字，要先 `.Alpha(0.12)` 变成浅底；草稿标签的 `path` 是空的，拿 `path` 去比对自然会落空——这正好是「草稿不点亮任何行」想要的效果。`DialogBase` 默认居中，顶部居中要自己在 backdrop 上 `Justify(ui.Start)` + 顶部 padding。
  - **正确做法**：树行统一 `row := ui.Row(...).Cursor(CursorPointer)` 后，把图标、方法/名称一起放进一个 `Grow(1)` 的 `ButtonBase`——**整行就是按钮**（行尾的筛选与 `+` 仍是兄弟按钮，自己处理点击），行再用 `if row.Hovered() { row.Background(p.SurfaceHover) }` 铺底；`requestLineRow` 用 `a.activeShows(req.Path)`（激活标签的 `path` 且与之相同）判定「当前接口」，命中时名字取 `p.Accent`、底色 `p.Accent.Alpha(accentTint=0.12)`，并**不**再叠加 hover；所有确认/消息类弹窗用 `internal/app/view.go` 的 `topBackdrop(backdrop)`（`Justify(Start)` + `Padding(48, 0, 0, 0)`），新建集合弹窗、确认弹窗、文件夹选择/命名弹窗共用它。
  - **验证方式**：`internal/app/tree_test.go` 的 `TestTreeRowsTintUnderThePointer`（指针离开/悬浮时行的底色分别是 `p.Background`/`p.SurfaceHover`，接口行与文件夹行都验）、`TestTreeRowOpensWhereverItIsClicked`（在行的最左端点击也打开接口/折叠文件夹，且那里是手型指针）、`TestOpenTabHighlightsItsRow`（打开后行底不再是页面色、行内能采到 `p.Accent` 像素；换成另一个标签后只有它亮；草稿标签时行底仍是页面色）、`TestDialogsSitAtTopCenter`（删除确认的按钮 y 在上方、水平居中；保存弹窗同样在顶部）。
  - **适用范围**：`internal/app/view_requests.go`（`folderRow`/`requestLineRow`）、`internal/app/app_tree.go`（`activeShows`/`accentTint`）、`internal/app/view.go`（`topBackdrop`）、`internal/app/view_confirm.go`、`internal/app/view_picker.go`、`internal/app/view_home.go`（新建集合弹窗）；以后加行式列表或确认弹窗照此办理。
- **标题**：视图测试的 store 目录留空会写进仓库；演示数据应当在内存里播种
  - **触发信号**：`task check` 之后 `git status` 里多出一个 `collections/` 目录；或者环境相关的用例突然报 `no environment in the list`（明明是内存里造的环境）。
  - **根因/约束**：`store.New("")` 的 `dir` 是空串，`collectionsDir()` 拼出的是相对当前工作目录的 `collections`，于是测试往仓库里写；而 `openCollection` 会 `reloadActive()`，只要 `collections/<id>` 真在磁盘上，就按磁盘内容覆盖内存里的集合、并先把该集合的环境清掉再重读；`saveTab` 的条件只是「标签有 `path` 与 `collectionID`」，`openTab`/`addTab` 前的 `leaveTab` 正好会调它。
  - **正确做法**：测试里的 store 目录一律给 `t.TempDir()`（`envApp` 就是这么做的）；`TestShots` 的 `demoApp` 用 `shotStore` 变量指向临时目录，并且**每个镜头一个子目录**（共用一份会让先跑的镜头写盘、后跑的镜头在 `openCollection` 时把内存里的集合读成磁盘上的那份，截图里集合名会退回目录名）；演示用的请求标签直接在内存里播种（`newTab()` + 设 `kind/collectionID/title/method/url/path` + `markSaved()` + `openTab`），不要为了「打开一个请求」去调 `openRequest`——那会把标签写回磁盘。
  - **验证方式**：`task check` 与 `APIGO_SHOTS=... go test -run TestShots ./internal/app/` 之后 `git status --short` 里没有 `collections/`；`internal/app/env_test.go` 的用例不再依赖「当前目录里没有 `collections`」；截图顶部集合标签显示的是集合名（`示例接口集`），不是目录 id。
  - **适用范围**：所有 `*_test.go` 里构造 store/app 的地方，尤其 `internal/app/shot_test.go` 的 `demoApp`。

- **标题**：升级 MyGo 小版本（0.5.5 → 0.6.0）的核对清单
  - **触发信号**：`go.mod` 里 mygo 的版本变了；升级后 `task check` 报某元素「outside the window」，或截图逐字节全变。
  - **根因/约束**：小版本也会改渲染与测量实现——0.6.0 只重绘变化、文本每元素只量一次、flexbox 按 Chrome 的方式布局、鼠标滚轮滚动带缓动；`ui.Tester.Click/ClickWith/RightClick` 从 0.6.0 起对**窗口外**元素返回 error（之前是静默点空）。所以「截图逐像素变」不等于坏了，而 `Tester` 的新报错几乎总是真 bug（元素被 flex 挤出盒子）。
  - **正确做法**：`go get github.com/egoist/mygo@<版本>` + `go mod tidy` → `task check` → 把越界报错当真实 bug 修（别改用 `ClickAt` 绕过）→ `APIGO_SHOTS=docs/shots go test -run TestShots ./internal/app/` 出图，与上一版逐张比；用逐像素差异框定位，把「测试数据改名/临时路径」这类噪声与真正的布局位移分开 → 最后核对 `mygo.json` 的新配置项（如 0.6.0 的 `"linux": {"webview": false}`）。
  - **验证方式**：`task check` 全绿；截图差异只落在预期区域（本次是 setting 弹窗的示例路径与文字抗锯齿）；`go tool mygo build -platform linux/amd64` 后 `ar x *.deb` 看 control 的依赖。
  - **适用范围**：每次 mygo 版本变更；`internal/app/shot_test.go`、`go.mod`、`mygo.json`、`.github/workflows/*`。

- **标题**：响应面板四个页签（预览/原始内容/响应头/请求头）：位图与播放器都是「有寿命的资源」，不能每帧重建
  - **触发信号**：图片响应显示成一片空白或乱码；音频响应只有一行 `RIFF...` 文本；换一个请求后旧声音还在响；播放器进度条不动。
  - **根因/约束**：`ui.DecodeBitmap` 的位图在 GPU 上常驻，「每一帧都解码一次」会不断建纹理（文档明确要求 make bitmaps once）；`audio.OpenStream` 拿到的 `*audio.Player` 持有解码器与输出，不 `Stop`/`Close` 就一直占着；播放器自己走时间，视图不主动要帧就画不出来；WAV 是内置的，MP3/Ogg 要 `_ "github.com/egoist/mygo/audio/mp3"` / `.../vorbis"` 空导入注册解码器（会引入 `go-mp3` 等间接依赖，要 `go get` 补 go.sum）。
  - **正确做法**：`req.Response` 多带一份 `Raw []byte`（正文的原始字节，`Body` 仍是文本）；`app.apply` 里一次性 `t.dropResponse()` → `adoptResponse(res)`：分类（`responseBodyOf`：JSON/文本、`image/*`、`audio/*`、`video/*`、其它）→ 图片当场解码并记 `pictureNote`（`Bitmap.Size()`），音频当场 `OpenStream` 并 `SetVolume`；换响应时先关掉旧播放器；播放中在视图里 `c.After(200ms)` 让进度条与时钟跟上；视频没有解码器，如实给「文件卡 + 保存」而不是假装能播。
  - **补充（页签与表头）**：页签名固定为 `预览`/`原始内容`/`响应头`/`请求头`（`responseTabNames`），页签与 `响应` 同一行（`responseMeta` 里画，宽度不够时靠右的状态块先让位）；`tab.respTab` 选页签：`respPreview`（`responseBodyView` 按 `t.view` 分派）、`respRaw`（`t.rawEdit` 只读缓冲，二进制用 `rawBodyText` 给一行说明）、`respHeaders`/`respRequestHeaders`（`headerList` 两列表格：名称小写、值右列，`sortedHeaderNames` 排序，`Selectable()` 可复制，页签名后 `countBadge` 给条数）。请求头来自 `send` 里 `t.sentHeaders = built.Header.Clone()`——发送那一刻的头，比事后从输入框重建更准（`Content-Type` 是 `BuildRequest` 加的）。
  - **补充（下载）**：下载入口只有一个——meta 栏右端（状态码/耗时/大小之后）的 `icons.Download` 图标（`Label("下载响应")`），三种视图里不再各放一个按钮；`saveResponse` 通过可替换的 `a.pickDownloadDir`（生产是 `chooseDownloadDir` → `chooseFolder`：`mygo.App.Path(mygo.PathDownloads)` 作 `DefaultPath` 打开 `Directory: true` 的对话框，取消返回 `""`；测试注入临时目录或 `""` 模拟取消）拿目录，取消时**什么都不写也不提示**；文件名优先 `Content-Disposition`（`mime.ParseMediaType` 直接给解码后的 `filename`，含 `filename*`），其次 URL 末段，最后 `mime.ExtensionsByType` 补扩展名，并过一遍 `safeDownloadName`（去掉路径与 Windows 非法字符）；同名用 `名字 (2).ext` 编号；`pickDownloadDir` 为 nil 或响应没有字节时提示而不是 panic。
  - **验证方式**：`response_body_test.go` 的 `TestResponseBodyOf`（九种 Content-Type/嗅探分类）、`TestDownloadName`（建议名/编码名/URL 名/类型补扩展名/去掉 `../`）、`TestResponseMetaHoldsTheDownloadIcon`（图标在状态码/耗时之后）、`TestResponsePictureIsShown`（图片视图出现、尺寸行、下载图标可点）、`TestResponsePlayerPlaysAndPauses`（点播放后 `Playing()` 为真、点暂停为假）、`TestResponseFileSavesToDownloads`（落盘字节一致、第二次编号 `(2)`、取消后目录不变）、`TestResponseVideoOffersSave`；`response_tabs_test.go` 的 `TestResponseTabsShowTheBodyItsHeadersAndTheRequest`（四个页签各自显示预览缩进体/原样体/响应头/请求头）、`TestResponseLanguageFollowsTheContentType`（九种内容类型 → 语法，未知回落纯文本）、`TestResponseTabsAreFixed`（页签名固定，首个为「预览」）、`TestHeaderTabCountsWhatItLists`、`TestSendButtonCancelsTheRequest`（等待中按钮变「取消」、点后 `请求已取消` 且可重试）；截图 `docs/shots/response-{json,image,audio,file,headers,request-headers}`。
  - **适用范围**：`internal/app/view_response.go`、`internal/app/view_response_tabs.go`、`internal/app/view_response_media.go`、`internal/app/response_body.go`、`internal/app/response_media.go`、`internal/app/app.go`（`tab.view/picture/audio/respTab/sentHeaders/cancel`、`send`/`cancelSend`/`apply`）、`internal/req/http.go`（`Response.Raw`）、`internal/icons`（Play/Pause/Music/Download/Picture/Video）、`internal/app/run.go`（`pickDownloadDir`/`chooseFolder`）。
- **标题**：查找「集合根」的向上遍历不能一路爬到盘符根，否则第一次写盘会落到 `C:\.tmp`
  - **触发信号**：写集合的 `apigo.yaml`（一个集合的第一个文件）时，临时文件出现在 `C:\.tmp`；某些机器上直接报 `The process cannot access the file because it is being used by another process`；`C:\` 下莫名多出 `.tmp`。
  - **根因/约束**：`tmpRoot(dir)` 从目标目录向上找含 `apigo.yaml` 的目录，找不到就继续往上，直到 `filepath.Dir(dir) == dir` 才停——第一次写时整个集合都还没有 `apigo.yaml`，于是走到了卷根。
  - **正确做法**：记下起点，遇到 `parent == dir || parent == "." || parent == 卷分隔符` 就返回**起点**（把临时文件留在目标所在目录），而不是返回卷根；这样「集合第一个文件」的临时文件落在集合目录的 `.tmp/`。
  - **验证方式**：`internal/store` 的 `TestFirstWriteStaysInTheCollection`（新建集合后集合目录有 `.tmp`、store 上一层没有）。
  - **适用范围**：`internal/store/service_yaml.go`（`tmpRoot`/`WriteYAML`）、任何「向上找项目根」的遍历。

- **标题**：响应体上限既要有默认值又要在设置里可改：给 `Execute` 显式传字节数，别在 `req` 里写死常量
  - **触发信号**：默认 10 MB 的 `maxResponseBytes` 常量把上千 MB 的下载/图片响应截断，用户还要能自己调这个上限。
  - **根因/约束**：上限原本是 `internal/req` 里的包级常量，设置只能改应用状态；`Execute` 签名不带上限的话，`req` 就只能读全局。
  - **正确做法**：`req.DefaultMaxResponse`（100 MB）当默认，`req.Execute(ctx, req, limit int64)` 显式收字节数、`limit <= 0` 退回默认；应用侧 `store.Settings.MaxResponseMB`（MB，0 缺省）→ `applySettings` 读成 `a.maxResponseMB float64`（给 `ui.NumberInput` 用）→ `a.responseLimit()` 换算成字节，`send` 时传进去；`rememberSettings` 用 `math.Round` 写回整数 MB；越界（<1 或 >10240）一律夹到默认或上限，别让设置文件写坏一个 0 出来。截断提示用 `limitText`（<1 MB 显示 KB），`Size` 也改成实际留下的字节数。
  - **验证方式**：`internal/req` 的 `TestExecuteCapsTheBody`（超限截断、错误里写 2 KB）、`TestExecuteDefaultsItsLimit`、`TestLimitTextWritesSizes`；`internal/app` 的 `TestResponseLimitFollowsSettings`（默认/保存值/越界）、`TestResponseLimitIsRemembered`（落盘 32）、`TestSendCutsTheBodyAtTheLimit`（1 MB 上限截断且面板提示）、`TestDefaultLimitMatchesTheRequestPackage`；`TestSettingsModal` 断言「响应体上限（MB）」在。
  - **适用范围**：`internal/req/http.go`、`internal/store/service_settings.go`、`internal/app/app.go`（`maxResponseMB`/`responseLimit`/`rememberSettings`）、`internal/app/view_env.go`（`responseLimitField`）。

- **标题**：`Window.Update(fn)` 是「把 fn 放到窗口线程上跑」，不是「在后台跑完再回来」——把 `Execute` 塞进 fn 会冻住整个窗口
  - **触发信号**：一个慢接口（或服务端一直不回）在等的时候，整个界面点不动、别的标签页发不出请求；所有请求要排队一个个来。
  - **根因/约束**：`a.after = window.Update`；把「发送 + 应用结果」写成 `a.after(run)`，`run` 里的 `req.Execute` 就在主线程执行，30 s 超时期间窗口一帧都画不出来。异步不是「套个 goroutine 就完事」——关键在于**哪一段在哪个线程**：网络与重活在自己的 goroutine，只有落在应用状态上的赋值回窗口线程。
  - **正确做法**：`send` 里主线程先算好要发送的一切（`responseLimit()` 之类读应用状态的也要在主线程取值），然后 `go func(){ sh := prepareResponse(req.Execute(ctx, built, limit)); a.after(func(){ a.apply(t, sh) }) }()`；`prepareResponse` 把「解码位图 / 打开播放器 / JSON 美化」这些重活也放在 goroutine 里，只把结果（`shown`）带回来；`apply` 只做赋值与 `AddHistory`。同一标签页用 `t.loading` 挡住重复发送，不同标签页各自一条 goroutine；`a.after == nil`（测试）时全部就地跑，结果同步可用。
  - **验证方式**：`internal/app/send_async_test.go` 的 `TestSendDoesNotBlockOtherRequests`（一个 handler 挂住不返回：另一标签页照常收到 200、第一个仍在 `loading`、界面还能点标签页切换）；`go test -race ./...` 覆盖 `after` 与 tab 状态的跨线程交接。
  - **适用范围**：`internal/app/app.go`（`send`/`apply`/`showResponse`）、`internal/app/response_media.go`（`shown`/`prepareResponse`）、`internal/app/app_watch.go`（同一模式：`Next` 在监听 goroutine、`reloadActive` 回窗口线程）。

- **标题**：WebSocket 面板：连接与读循环各有归宿，日志只存窗口线程上的数据
  - **触发信号**：连上后收不到消息；关标签后 goroutine 还在跑；断线后日志里再也没有新行；切换标签时旧连接的消息混进新标签。
  - **根因/约束**：gorilla 的读是阻塞的，必须有一条 goroutine 专门读；写不能并发（多个 goroutine 同时 `WriteMessage` 会 panic），要串行化；`ui` 的元素与状态只能碰窗口线程；`a.after` 是「把函数放到窗口线程跑」，测试里为 nil 时是就地跑。
  - **正确做法**：`internal/ws` 只做协议层——`Dial` 成功后起一条读 goroutine，把 `Received`/`Disconnected` 投进 `Events()` 通道，通道随连接结束关闭；`Send` 用互斥锁串行化并设写超时；`Close` 先发关闭帧（1s 超时）再关底层连接，`sync.Once` 保证只关一次。应用侧 `connectSocket` 在 goroutine 里 `Dial`，结果经 `a.after` 落回 `t.conn` 与日志；`readSocket` 把每条事件也经 `a.after` 追加，并在回调里比对 `t.conn != conn` 丢弃已被替换/断开的连接尾巴；标签关闭走 `dropTabMenu` → `closeSocket`，日志上限 500 条；用户断开连接（`disconnectSocket`）直接清空日志，服务端关闭则留一行「已断开」；日志行 `摘要 Grow(1)`（内容在 `appendSocket` 里用 `collapseText` 折叠成一行、上限 `summaryLine=1024` 字符，**不再按 72 字符硬截断**——宽度交给行布局，窗口宽窄决定省略位置）+ 行尾定宽的大小与时间（`logSizeWidth`/`logTimeWidth`，`TextAlign(End)`，时间排在最后）；`过滤日志` 输入框写 `t.logFilter`，`logLines()` 按消息内容不区分大小写包含匹配给出要画的索引，头部计数在筛选时变成 `M / N 条`、无命中给「没有匹配「…」的日志」；消息行点开才建 `editor.State`（`toggleLogLine`，缓冲区内容先过 `prettyResponse`——JSON 消息按两空格缩进排版，和响应预览同一套缩进，否则原样；只读、`FitContent`、字号比编辑器默认小一号），关掉即置 nil，日志被裁剪时随之释放。
  - **补充（按钮）**：请求行的 `发送`/`保存`/`连接` 都是「图标 + 文字」，用 `iconButtonSpec{label, name, icon, disabled, plain, run}` + 自绘 `ButtonBase`（主按钮 `Background(p.Accent)` + `AccentText` 前景；次要按钮 `plain: true` 不铺底色、**保留 1px 边框**、用 `TextSecondary` 着色、悬浮时 `SurfaceHover`；`small: true` 再降一级（高 28、字号 Tiny、图标 12）；两者都 `MinWidth(测量宽度)`）；**不要用 `PrimaryButton(c, label)` 再挂 `Children`**——按钮自带的文字会和 Children 里的文字一起画，出现「发送 ⇄ 发送」。`name` 要与可见文字不同（`保存接口`/`发送请求`/`确认保存`/`保存环境`），否则 `ui.Tester.Find` 会命中同屏同名的另一个按钮（请求行、文件夹弹窗、环境弹窗都有「保存」）。
  - **验证方式**：`internal/ws/ws_test.go`（文本/二进制回声、子协议、关闭后事件通道关闭、握手被拒报状态码）；`internal/app/ws_test.go` 的 `TestSocketConnectsAndEchoes`（连接→日志→发送→回声→发送后清空→断开）、`TestSocketLogClears`、`TestSocketMessageFormatFollowsThePick`（格式落盘 + Binary 禁用提示）、`TestSocketSubprotocolReachesTheHandshake`（服务端真的收到 `chat`）、`TestSocketRequestKeepsItsMessage`（子协议/格式/消息写进接口文件再读回）。
  - **适用范围**：`internal/ws`、`internal/app/{view_websocket,socket}.go`、`internal/app/app.go`（`tab.conn/socketLog/protocol`、`savedFromTab`）、`internal/store/service_store.go`（`Request.Protocol`）、`internal/app/view_request.go`（`iconButton`）；以后接 SSE/Socket.IO 复用同一套「协议层通道 + 窗口线程落地」。

- **标题**：同一种控件出现在多处时抽成一个函数，别各画各的——`selectTrigger`/`selectChoiceRow`
  - **触发信号**：同一个应用里出现两种「带边框 + 右侧箭头」的下拉（环境下拉窄而土灰、消息格式下拉的箭头贴着文字、选中项一处带勾一处不带），用户会问「这两个为什么不一样」。
  - **根因/约束**：每个下拉各自 `ButtonBase` + 子元素拼装，写着写着就分叉：内边距 8/10、箭头 11/12、`Grow(1)` 撑开 vs `SpaceBetween`、选中项有无勾、文字 `p.Text`/`p.TextSecondary`。视觉一致性靠「记住上次怎么写的」不可能稳住。
  - **正确做法**：`internal/app/view_menu.go` 里 `selectSpec{name,label,width}` + `selectTrigger`（`TabStripHeight-12` 高、`Padding(0,10)`、`MinWidth(0)`、`Background(p.Surface)`、1px `p.Border`、`Justify(SpaceBetween)`、悬浮 `SurfaceHover`、label 用 `TextSecondary`、`icons.Chevron` 12 号 `TextMuted`）与 `selectChoice{name,label,chosen,run}` + `selectChoiceRow`（选中项 `icons.Check` 12 号 `Accent` + 文字 `Accent` + `Accent.Alpha(accentTint)` 底色，未选中的用 `ui.Row(c).Width(12)` 占住勾位）。环境下拉与 WS 消息格式下拉都调这两个；`name` 保持旧值（`环境：<名字>`、`消息格式 <格式>`、`切换到 <环境>`），测试不用改。
  - **验证方式**：`internal/app/env_test.go`、`ws_test.go` 的格式切换用例仍绿；`APIGO_SHOTS=docs/shots go test -run TestShots ./internal/app/` 后肉眼比对 `websocket-format-menu.png` 与顶栏环境下拉。
  - **补充（垂直居中）**：标签条 `requestTabs` 是 `AlignItems(Start)`（标签的 2px 强调线要贴顶），直接把 28 高的下拉放进去会**贴着上沿**（上 0 / 下 12）。给下拉套一层 `ui.Row(c).Height(design.TabStripHeight).AlignItems(ui.Center)` 再放进去，上下各 6；WS 工具栏那一行本来就是 `AlignItems(ui.Center)`，不用套。同理：任何「矮控件落在高高度的 Start 对齐行里」都要自己套一层居中行或 `AlignSelf(ui.Center)`。
  - **适用范围**：`internal/app/view_menu.go`、`view_tabs.go`（`environmentPicker`）、`view_websocket.go`（`messageFormatMenu`）；以后任何「从列表里挑一个」的控件。

- **标题**：一行的「剩余宽度」要用 `Grow(1)` 给内容，而不是在末尾塞一个 `Grow(1)` 空格——否则文字会被算短
  - **触发信号**：首页集合列表里名字只剩「sta…」「测…」，右边却有一大片空白；接口树里也偶发同类现象。
  - **根因/约束**：`ui.Column` 不写宽度时按内容自适应，而工具包对嵌入字体的文本 max-content 测量比实际绘制小（见上一条）；行尾再放一个 `ui.Row(c).Grow(1)` 当弹簧，弹簧先把剩余宽度吃掉，名字只能拿到被算小的那点宽度 → `SingleLine` 判「放不下」，三两个字就省略。
  - **正确做法**：把「谁该占剩余宽度」写成那个元素自己的 `Grow(1)`：首页行改成 `ui.Column(c).Grow(1)`（名称 `Grow(1)` 在列里铺满），删掉行尾的弹簧；右侧 `打开`/`…` 保持自然宽度。这样窗口宽就长、窄就在按钮前省略。顺手把名称旁的 `HTTP` 类型标记删了——集合名本身不必再标协议。
  - **验证方式**：`TestHomeCollectionNameFillsTheRow`（1280 与 640 两个宽度下名称右缘距「打开」都在 20 px 内、宽窗口的名称更宽、文本里没有 `HTTP`）；截图 `docs/shots/home-list.png`。去掉 `Column` 的 `Grow(1)` 时该测试的宽度比较会退化成「三字省略」。
  - **适用范围**：`internal/app/view_home.go`（`homeCollectionBody`）；任何「图标 + 标题 + 行尾按钮」的列表行。

- **标题**：列表要「跟着磁盘变」就多开一条 watch 盯它自己的目录，别指望能靠帧去 stat
  - **触发信号**：用户在文件管理器里删掉一个集合目录，首页列表还当它存在；点「打开」后是空的或者报错。
  - **根因/约束**：`followFiles()` 原来只 watch **当前打开的集合**目录（为了树跟着变），首页在 `active == ""` 时一条 watch 都没有；每帧对每个集合 `os.Stat` 又太贵（10 个集合 × 60 fps = 600 次/秒）。
  - **正确做法**：多开一条 `followList()`，watch `store.CollectionsDir()`（非递归），事件回到窗口线程后跑 `refreshMissing()`：它对每个集合 `os.Stat` 一次，得到 `map[string]bool` 缓存。顺序上**先起列表 watch 再起集合 watch**，这样 `watchLog.last()` 仍是集合 watch，原有的 watch 测试不用改。`listWatched` 保证只试一次（平台不支持也不每帧重试），新建集合时置回 `false`（这时目录才第一次存在）；换数据目录（`useDataDir`）要停掉旧 watch 再重置。列表行按 `a.collectionMissing(id)` 灰显 + `Strikethrough()`，点击/双击走 `openOrForgetCollection` → `askToForgetMissingCollection`（确认框复用 `a.ask`，动作是 `deleteCollection` = `HideCollection`，只从列表移除，目录本来就没了）。
  - **验证方式**：`TestWatchMarksDeletedCollections`（首页首选 watch 的路径是 `CollectionsDir()`；`os.RemoveAll` 后 report 一次，`applyWatch` 后 `collectionMissing` 为真且列表仍显示该集合）、`TestHomeMarksMissingCollections`（名称最长连续墨迹 ≥ 半个 advance，即真的有中划线；点「打开」不开集合、弹出「集合不存在」、确认后 `Collections` 清空）。把 `Strikethrough()` 去掉时前者立刻报「longest ink run is 15 px of 75」。
  - **适用范围**：`internal/app/app_watch.go`（`followFiles`/`followList`/`drainList`/`refreshMissing`）、`internal/app/view_home.go`、`internal/app/view_confirm.go`；以后任何「列表要跟着磁盘状态变」的地方。

- **标题**：多行「标签 + 说明」要做成两列对齐：标签用 `MinWidth(最长标签)`，别给内层 Column 加 `AlignItems(Start)`
  - **触发信号**：引导页/快捷键表里每行按自身宽度居中，标签左边缘参差、说明/按键起始位置各不相同；想「左对齐」时给 `Column` 加 `AlignItems(ui.Start)`，结果按键方块被压成「…」。
  - **根因/约束**：`ui.Column(c).Center()` 会把每个子元素按自身宽度水平居中，所以四行各自漂移；而内层 `Column.AlignItems(Start)` 让它的宽度按子元素的最小内容算，比最宽的一行还窄 → 行里的按键在有限宽度里被挤掉（同一类「内在宽度被算小」的坑）。
  - **正确做法**：内层 Column **不要**加 `AlignItems`（交给默认拉伸，行宽 = 最宽一行，四行共享同一左边缘），标签文字用 `MinWidth(hintLabelWidth)`（`max(measureText(各标签)) + tabWiggle`）撑出统一的标签列，按键/说明自然从同一条左边缘开始；行距继续用 `Gap`。方块（`keycap`）自己也要 `MinWidth(2*7 + measureText + tabWiggle)`，否则刚好够宽时文字又被省略号吃掉。
  - **验证方式**：`TestStartHintLinesUpItsActions`（四行标签 X 相同、行距 ≥ 20、第一行与第二行的说明以及第三行的按键 X 都相同——把标签的 `MinWidth` 去掉时立刻报「tips and keys start at 777.0 and 751.0」）。
  - **适用范围**：`internal/app/view_start.go`（`startHint`/`hintList`/`hintRow`/`keycap`）；任何「标签 + 值/按键」的两列表格式排版。

- **标题**：联网检查新版本要「默认关、可注入、结果如实说」——别用 mygo 自带 Updater（需要签名 feed）
  - **触发信号**：想做自动更新，发现 `mygo.Updater` 只在 `mygo.config.ts` 配了签名 feed 的构建里可用（`ErrUpdatesDisabled`），而本项目的 Release 只是 GitHub 上的普通产物。
  - **根因/约束**：签名 update feed 需要额外密钥与 `mygo build` 配置；`Updater.Check` 在开发构建/未配置构建里直接报错。要「够用的自动更新」只需要查 GitHub 最新 Release 并让用户去下载页。
  - **正确做法**：`internal/app/update.go` 用 GitHub API（`/releases/latest`，带 `User-Agent`，15s 超时）读 `tag_name/html_url/body`，跳过 `draft`/`prerelease`；版本比较自己写 `newerVersion`（按点分数字逐段比，支持前缀 `v`，非数字后缀截断，空版本当 `0.0.0`，读不出来的版本永不算新）。`update_check.go` 里 `checkUpdates(ask)`：`a.checkRelease` 可注入（测试用假函数），`a.after == nil`（测试）时同步跑，否则 goroutine 里查、`a.after` 回窗口线程写 `a.updateNote`；发现新版本用 `a.ask(confirm{... c.OpenURL(url)})` 问一句再开浏览器。**`autoUpdate` 默认 false**（`settings.AutoUpdate` + 设置「通用 → 更新」复选框），只在 `run.go` 的 `checkUpdatesAtStart` 里按开关决定要不要联网。
  - **验证方式**：`TestNewerVersion`（0.2.0>0.1.0、1.10.0>1.9.9、`nightly` 不算新）、`TestReleaseAtReadsGitHub`（httptest 验 User-Agent、草稿/预发布/无 tag 视为无更新、403 报错）、`TestAutoUpdateIsOffUntilTurnedOn`（默认关→勾选落盘→点「检查更新」出现「有新版本 0.2.0」与确认框→最新/失败两种结果的措辞）、`TestStartupCheckRunsOnlyWhenEnabled`（开关关时一次网络都不发）。
  - **适用范围**：`internal/app/update.go`、`update_check.go`、`internal/store/service_settings.go`、`internal/app/view_env.go`（`autoUpdateField`）、`internal/app/run.go`。

- **标题**：显示名与磁盘名分开：界面叫 `ApiGo`，落盘仍是 `apigo`
  - **触发信号**：用户要把项目名从 apigo 改成 ApiGo，但一改 `mygo.json` 的 `name` 就会连带换掉可执行文件名**和**数据目录（`App.Name()` 决定 `%APPDATA%/<name>`），已有集合会像凭空消失。
  - **根因/约束**：`mygo.App.Path(PathUserData)` = `os.UserConfigDir()/App.Name()`；`App.Name()` 来自 `mygo.json` 的 `name`。`apigo.yaml` 是集合目录里的元文件名，改了旧目录读不出来；Go 模块路径 `apigo` 与 GitHub 仓库名也绑在一起。这些是「磁盘/构建标识」，不是显示名。
  - **正确做法**：显示名集中到 `internal/app/view_footer.go` 的 `const appName = "ApiGo"`，窗口标题（`run.go` 的 `Title: appName`）与状态栏/设置里的版本行都用它；**不动** `mygo.json` 的 `name`/`identifier`、数据目录、`store.MetaFile = "apigo.yaml"`、模块路径与仓库名。文案（README/设计文档）里作为产品名的「apigo」改成 ApiGo，作为文件名/路径/包名的保持原样。
  - **验证方式**：`TestVersionLineNamesTheApp`（`appVersion()` 以 `ApiGo` 开头、状态栏文本里有 `ApiGo`）；`task check` + 截图里状态栏右下角显示 `ApiGo`。
  - **适用范围**：`internal/app/view_footer.go`（`appName`/`appVersion`）、`internal/app/run.go`、README/设计文档文案；以后任何「改产品名但不想动磁盘格式」的需求。

- **标题**：`ui.Modal` 里的面板高度要自己算：给「滚动区」显式 `Height`，不是给面板 `Height`
  - **触发信号**：设置弹窗在矮窗口里把底部（版本/数据）顶出窗口，圆角都被切掉；给面板加 `.Height(...)` 或 `.MaxHeight(...)` 都没用。
  - **根因/约束**：`ui.Modal` 自身会在窗口内留约 40 DIP 的内边距，但**不会裁剪**超出的面板；面板的高度仍是内容撑出来的，里面 `ui.Scroll` 用 `Grow(1)` 时按内容高度参与计算，于是「限高」完全失效（`Height`/`MaxHeight` 都被内容顶破）。此外 `tt.Find` 返回的是布局位置而非可见性，滚动区里的内容即使被裁剪也照样能 Find 到，别拿它判断「是否显示」。
  - **正确做法**：先算窗口能给的高度 `tall := min(settingsPanelHeight, height - 2*settingsModalInset)`，再把**滚动区**写成 `ui.Scroll(c).Height(tall - 2*settingsPadding - settingsTitleHeight - settingsGap)`（显式高度），面板本身不加 `Height`；面板高度自然等于「内边距 + 标题 + 间距 + 滚动区」。滚动区不要套 `noScrollbars`，滚动条是「下面还有内容」的提示。横排的标题、分栏等用常量把这个式子写清楚（`settingsPanelHeight/settingsPadding/settingsGap/settingsTitleHeight/settingsModalInset`）。
  - **验证方式**：`TestSettingsFitAShortWindow`（1280×560 下窗口底部的像素是遮罩色而不是面板白，`tt.Scroll` 之后「版本」上移）；截图 `docs/shots/settings-modal.png`。改回「给面板 `Height`」时底部像素立刻变白（255）。
  - **适用范围**：`internal/app/view_env.go`（`settingsModal`）；任何放进 `ui.Modal` 且内容可能超过窗口的面板。

- **标题**：折叠开关放在状态栏最左、用无边框图标，两态各一个图标；浮层按钮（`Overlay`+`AttachTo`）是可行的备选
  - **触发信号**：接口树的折叠按钮一开始做成请求页左下角的浮层按钮（带边框的方块），被要求挪到「底部左下角」、去掉边框、并且收起/展开用不同图标。
  - **根因/约束**：`ui.Row`/`ui.Column` 只是顺序排布，没有叠加层；浮层要用 `ui.Overlay(c, fn)` 把元素画在整窗之上，再用 `Element.AttachTo(target, at, self)`（`ui.AnchorBottomLeft` 等）贴到某块区域的角上，之后 `Left/Top/Right/Bottom` 偏移，且 target 必须同帧先构建。状态栏则本来就在窗口最底部，把按钮放进 `footer` 的第一位就是「底部左下角」，不需要浮层，也不会盖住响应内容。
  - **正确做法**：折叠按钮放进 `internal/app/view_footer.go` 的 `footer()` 最左侧（`a.active != ""` 时才画），`ui.ButtonBase` 20×20、无 `Background`/`Border`，只挂 `ui.Icon`（13 号 `TextMuted`），悬浮才铺 `SurfaceHover`；两态用两个图标：展开时 `icons.SidebarClose`（面板 + 箭头朝左，点了收起），收起时 `icons.SidebarOpen`（箭头朝右）；无障碍名保持「收起接口面板」/「展开接口面板」。收起时 `collectionWorkspace` 里整块 `requestsPanel` 连分隔线一起不画。
  - **验证方式**：`TestSidebarFoldsAway`（点「收起接口面板」后树消失、请求页还在、按钮变「展开接口面板」，再点树回来）；截图 `docs/shots/request-filled.png` 与 `docs/shots/sidebar-collapsed.png` 的状态栏最左侧两个不同图标。
  - **适用范围**：`internal/app/view_footer.go`（`footer`/`sidebarToggle`）、`internal/app/view.go`（`collectionWorkspace`）、`internal/icons`（`SidebarClose`/`SidebarOpen`）；以后「放在窗口边缘的开关」优先进状态栏或标签条，真的需要浮在内容上时再用 `Overlay` + `AttachTo`。

- **标题**：导入 OpenAPI/Swagger 分两层——`internal/openapi` 只把文档读成接口，`store.Import` 管落盘与同名覆盖
  - **触发信号**：要做「导入集合」，直接照着 `AddRequest`/`AddFolder` 一个个写文件会立刻撞上三件事：名字里带 `/`、`:` 等字符，同一份文档里两个接口重名，以及重复导入时不知道该覆盖谁。
  - **根因/约束**：文档是外来的动态结构（`map[string]any`，YAML 解码器把 JSON 也一起吃下），而集合的落盘规则是「一个接口一个文件、文件名即接口名、同层唯一」。把两者混在一个包里，解析测试就得搭磁盘；而直接 `AddRequest` 一条条写，每次都要整集 `saveCollection`，还会在「同名覆盖」上丢掉文件既有的 `Path/Name`。
  - **正确做法**：①`internal/openapi` 只做纯函数 `Parse([]byte) (*Document, error)`，返回 `Title/Spec/APIVersion/BaseURL` + `Requests`（无 tag）+ `Folders`（按 tag 首次出现的顺序），`Row{Key,Value}` 自成一类；`$ref` 在文档内解析并**用 `refs` 集合 + `exampleDepth` 双重封顶**，否则自引用的 schema（`Pet.owner → Pet`）会递归到爆栈。②`server` 缺失或是相对路径时，接口地址保留 `/path` 形式（应用按「以 `/` 开头就接 `baseurl`」的规则拼接，见后一条经验）。③落盘统一走 `store.Import(ImportOptions)`：按 `Name` 找现有集合（有则更新、无则新建），`mergeFolder` **按 `safeFileName` 后的名字**匹配文件夹与接口，命中就整条覆盖但**保留既有的 `Path/Name`**，最后 `normalizeTree` 一次分配路径、净化名字、给重名编号；新集合再补三个默认环境，并把文档的 server 写进第一个环境的 `baseurl`。
  - **验证方式**：`go test ./internal/openapi/`（v3/v2/fetch/自环 schema/重名编号）、`internal/store/import_test.go`（新建成集 + 落盘文件、同名覆盖且不动无关接口、同名 tag 合并且重名编号）、`internal/app/import_test.go`（弹窗打开、选文件、拖拽 `tt.DropFiles`、URL 读取、失败提示、同名集合提示 + 覆盖）。
  - **适用范围**：`internal/openapi/**`、`internal/store/service_import.go`、`internal/app/{view_import,import}.go`；以后加 Postman/Insomnia 导入时复用 `store.Import`，只需要再写一个「文档 → `store.Folder`」的转换。

- **标题**：文件拖拽用 `Element.DroppedFiles()`/`FileDragOver()`，测试用 `Tester.DropFiles` 才能覆盖
  - **触发信号**：导入弹窗要「点击或拖拽文件到本区域导入」，但不确定 native UI 是否支持从资源管理器拖入，或写了拖拽后测试点不到。
  - **根因/约束**：MyGo 的文件拖放不是系统菜单或剪贴板，而是 `flagDropTarget`：**每帧都要调用** `DroppedFiles()`（返回值是「上一帧以来」落下的路径）或 `FileDragOver()` 才会登记为投放目标；`transfer.FileList` 那套 `ui.DropData` 是另一条通用拖放通道，文件用前者更直接。`Tester` 有 `DropFiles(x, y, paths...)`，返回 false 说明坐标下没有元素认领。
  - **正确做法**：拖放区域做成一个带 `Label()` 的可点元素（`ui.Column` 即可，纯文本子元素不抢点击），每帧先读 `FileDragOver()` 改边框/底色做提示，再读 `DroppedFiles()` 取 `[0]` 去解析；点击走 `chooseDoc`（`mygo.Dialog.Open` + `Filters` 限定 json/yaml/yml）。
  - **验证方式**：`TestImportTakesADroppedFile`（`tt.Find("选择文档文件")` 中心点 `tt.DropFiles(...)` 后摘要出现）；真实窗口里从资源管理器拖一个 json 进去。
  - **适用范围**：`internal/app/view_import.go`（`importDropZone`）、`internal/app/run.go`（`chooseDocument`）；任何「拖文件进来」的区域。

- **标题**：地址以 `/` 开头就接 `baseurl`：拼接放在 `app.targetURL`，前缀画进地址框内部，`baseurl` 在环境弹窗里是独立且删不掉的一项
  - **触发信号**：希望「接口只写 `/ws`，主机由环境给」，且地址框里要直接看到当前环境的前缀（例如 `192.168.110.53:6976` `/ws`），而不是让用户自己敲 `{{baseurl}}/ws`。
  - **根因/约束**：`req.ResolveVars` 只做 `{{name}}` 替换，`req.BuildRequest` 在缺 scheme 时会补 `https://`——对一个 `ip:port` 的 baseurl 补 https 往往连不上，WebSocket 更是必须 `ws://`。所以「补什么 scheme」取决于接口类型，只有 app 层知道（`tab.kind`），不能塞进 `req`。另外 `ui.TextInput` 自带边框/背景，单独画一个胶囊当前缀会读成两个控件，用户要的是「一个输入框」。
  - **正确做法**：①拼接一处收口：`internal/app/target.go` 的 `a.targetURL(t)`——`strings.HasPrefix(raw, "/")` 时取 `VarsFor(active)[store.BaseURLVariable]`（去尾部 `/`），空的返回 `missing=[baseurl]`；没有 `://` 的按 `schemeOf(kind)` 补（WebSocket/Socket.IO 补 `ws`，其余 `http`），拼好再走 `req.ResolveVars`；`send` 与 `connectSocket` 都必须用它，缺前缀的提示由 `missingText` 统一成「URL 以 / 开头，但当前环境没有设置前置URL」。②前缀画进框里：自己搭外框（`Surface` 底 + 1px 边框 + 圆角 + 高 40 + 左右内边距 10），里面先 `ui.Text` 画前缀（mono、`TextMuted`、只读），再 `ui.TextInputBase`（**Base 版没有边框/背景**）`Grow(1)`，两者无间距；前缀为空就只画输入框；焦点环画在外框上（`input.Focused()` 时给外框换 `Accent` 边框），因为内层 Base 输入没有焦点边框。③`baseurl` 在环境弹窗里从变量列表里拆出来单独一项：`editEnvironment` 用 `baseURLOf` 把 `baseurl` 抽进 `a.envBaseURL`（行里不再有它，所以没有删除按钮、删不掉），`saveEnvEdits` 用 `withBaseURL` 把它作为第一行写回；变量名统一 `store.BaseURLVariable = "baseurl"`，旧版本的 `BaseURL` 在 `readEnvironments`/`loadGlobalEnvironments` 里由 `migrateBaseURL` 改写。
  - **验证方式**：`internal/app/target_test.go` 的 `TestSendReadsPathUnderBaseURL`（`/pets?limit=10` → path `/pets`，界面出现前缀）、`TestSendReadsAddressWithoutScheme`（`host:port` 按 http 连）、`TestSendRefusesAPathWithoutBaseURL`（空前缀时不显示前缀、发送给提示）、`TestRequestLinePrefixFollowsTheAddress`（前缀只在 `/` 开头时出现，且落在外框内）、`TestSocketConnectsUnderBaseURL`（`/ws` + `host:port` → `ws://` 连上 echo socket）；`env_test.go` 的 `TestEnvironmentPrefixURLEditsAndKeeps`（改完保存后 `baseurl` 是第一行、变量值可读）与 `TestEnvironmentVariableRowDeletesItself`（删变量不会删掉前缀，保存后只剩 `baseurl`）。注意：弹窗里「前置URL（baseurl）」标题文字与输入框 label「前置URL」必须不同，否则 `Tester.Find` 会命中标题、点击落空。
  - **适用范围**：`internal/app/target.go`、`view_request.go`（`addressField`）、`view_websocket.go`（`socketLine`）、`view_prefix.go`（`prefixURLField`/`baseURLOf`/`withBaseURL`）、`socket.go`、`internal/store/service_env.go`、`internal/openapi/servers.go`（相对 server 输出 `/path`）；以后任何「地址拼接」都走 `targetURL`，别在视图里自己拼。

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
