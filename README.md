<div align="center">
  <img src="resources/icon.png" width="96" alt="apigo" />
  <h1>apigo</h1>
  <p><strong>基于 MyGo 原生 UI（Go 直接自绘界面，无 HTML/JS/WebView）的跨平台接口调试工具</strong></p>
  <p>原生 GPU 渲染 · 单二进制分发 · 秒开低占用 · 主题跟随系统 · 视图可离屏测试</p>
  <p>
    <a href="#tech"><img src="https://img.shields.io/badge/Go-1.27.2-00ADD8?style=for-the-badge&logo=go&logoColor=white" alt="Go 1.27.2" /></a>
    <a href="#tech"><img src="https://img.shields.io/badge/MyGo-v0.5.5-7B61FF?style=for-the-badge&logo=go&logoColor=white" alt="MyGo v0.5.5" /></a>
    <a href="#features"><img src="https://img.shields.io/badge/UI-Native_UI-2D9CDB?style=for-the-badge" alt="Native UI" /></a>
    <a href="#install"><img src="https://img.shields.io/badge/Platform-Windows_%7C_macOS_%7C_Linux-4B5563?style=for-the-badge" alt="Windows | macOS | Linux" /></a>
    <a href="#status"><img src="https://img.shields.io/badge/Status-early_development-F2994A?style=for-the-badge" alt="early development" /></a>
    <a href="LICENSE"><img src="https://img.shields.io/badge/License-MIT-3DA639?style=for-the-badge&logo=opensourceinitiative&logoColor=white" alt="MIT License" /></a>
  </p>
  <p>
    <a href="#about">项目说明</a> ·
    <a href="#features">核心特性</a> ·
    <a href="#status">项目现状</a> ·
    <a href="#install">安装</a> ·
    <a href="#start">快速开始</a> ·
    <a href="#dev">开发与构建</a> ·
    <a href="#config">配置</a> ·
    <a href="#structure">目录结构</a> ·
    <a href="#roadmap">Roadmap</a> ·
    <a href="#license">开源协议</a>
  </p>
</div>

<a id="about"></a>
## 📖 项目说明

apigo 是一个用 Go 编写、由 MyGo 原生 UI 直接绘制的接口调试工具，形态对标 Hoppscotch：先建集合，再在集合里建接口（REST / GraphQL / WebSocket / SSE / Socket.IO / MQTT），接口标签归属于创建它的集合；集合自带「本地 / 测试 / 生产」三个环境和 BaseURL 变量。界面与逻辑都是 Go，没有前端构建链和浏览器运行时——单文件分发、启动即用、完全离线。

使用流程：没有集合时停在「主窗口」，列出全部集合并引导新建；新建后集合出现在顶部集合标签上，点开进入该集合的接口管理——左侧搜索/筛选/新建接口，右侧请求与响应，右上角切换环境。集合、接口、环境与历史都以本地 YAML 保存。开发与协作规则见 `AGENTS.md`。

<a id="status"></a>
## 📌 项目现状

> [!IMPORTANT]
> 界面按「集合 → 接口」的工作流分阶段复刻，还原口径与已知差异见 [`docs/design-baseline.md`](docs/design-baseline.md)。REST 链路完整可用；GraphQL/WebSocket/SSE/Socket.IO/MQTT 目前只建接口与开标签，执行按 [Roadmap](#roadmap) 推进。

| 能力 | 现状 |
|---|---|
| 原生 UI 单窗口（自绘界面，无 WebView） | 已实现：`main.go` 的 `app.view` |
| 窗口尺寸与位置记忆 | 已实现：`mygo.WindowOptions.StateKey` |
| 布局：集合标签条 40 / 左侧接口树 240（铺满高）/ 右侧请求面板（顶部请求页签 40 + 请求-响应分栏）/ 状态栏 32 | 已实现：`view.go`、`view_topbar.go`、`view_tabs.go`、`view_requests.go` |
| 集合：顶部标签切换、关闭、右键关闭/关闭其他、`…` 里重开最近关闭的集合 | 已实现：`view_topbar.go` + `app.open/closed` |
| 接口树：「根目录」（集合自己的目录）+ 子文件夹 + 叶子接口，可折叠；文件夹内新建接口；搜索与类型筛选 | 已实现：`view_requests.go` + `service_tree.go`（一个接口一个 YAML 文件） |
| 主窗口：有集合时头部「导入集合」（占位）+「新建集合」（弹窗输入名称）与集合列表；无集合时只显示居中引导与「新建集合 / 从集合文件导入」 | 已实现：`view_home.go` |
| 新建接口：REST / GraphQL / WebSocket / SSE / Socket.IO / MQTT 六类，写成 `<接口名>.yaml` 并开标签 | 已实现：`view_requests.go` + `app.addRequest` |
| 请求标签：每个集合独立的一组标签，新建/切换/关闭/复制/重命名（右键菜单） | 已实现：`view_tabs.go` + `workspace` |
| 环境：集合自带「本地 / 测试 / 生产」，各含 `BaseURL`；支持新建、删除、改名、激活 | 已实现：`service_env.go` + `view_env.go` |
| 变量替换：`{{name}}` 用于 URL/参数/请求头/请求体；未解析则拒绝发送 | 已实现：`service_vars.go` + `app.send` |
| 持久化：一个集合一个目录（`apigo.yaml` + `.env/` + `static/` + 每个接口一个 YAML），原子写入 | 已实现：`service_store.go`、`service_tree.go`、`service_env.go`（`gopkg.in/yaml.v3`） |
| 接口写回：切换 / 新建 / 复制 / 关闭标签与关闭集合时，把当前标签写回它的接口文件，写失败会提示 | 已实现：`app.saveTab` + `store.updateRequest` |
| 历史：发送即记录（方法/URL/状态），`接口管理` 面板可切换查看并重开 | 已实现：`app.apply` + `view_requests.go` |
| 请求行：方法下拉（彩色）、URL、发送、`Ctrl+Enter` | 已实现：`view_request.go` |
| Params / Headers 键值编辑（增删、启用开关、自动空行、窄栏紧凑模式） | 已实现：`widgets/kvlist.go` |
| 请求体：none / json / text，JSON 走代码编辑器（高亮、行号、校验、格式化） | 已实现：`ui/editor` + `app.formatBody` |
| 响应：状态码/耗时/大小、JSON 只读高亮渲染、错误态与重试 | 已实现：`view_response.go` + `service_http.go` |
| 网络执行：10 MB 上限、30 s 超时、结果经 `Window.Update` 回主线程 | 已实现：`service_http.go`（取消待补） |
| 外观：跟随系统 / 浅色 / 深色 / 纯黑（设置弹窗） | 已实现：`internal/design.Theme` |
| 非 REST 接口的执行（GraphQL/WebSocket/SSE/Socket.IO/MQTT） | 未实现（打开显示占位） |
| 响应头表格、JSON 折叠树、集合导入导出 | 未实现 |
| 视图离屏测试与截图 | 已实现：`main_test.go`、`collections_test.go` 断言行为，`shot_test.go` 出 PNG |

<a id="features"></a>
## ✨ 核心特性

### ⚡ 原生渲染，没有前端

界面由 Go 直接构建、由 GPU 绘制（Windows Direct3D 11 / macOS Metal / Linux OpenGL），窗口不启动 WebView：没有 HTML、没有 JavaScript、没有前端构建链，启动即出画面，内存占用远低于 Electron 类工具。

### 📦 单二进制，处处可跑

`go tool mygo build` 产出各平台可执行文件，Windows 附带 NSIS 安装包，macOS 附带 `.app`/`.dmg`，Linux 附带 deb 与 tar.gz；同一份代码也能编译成浏览器上运行的 WebAssembly 版本。

### 🎨 与系统一致的控件

请求方法下拉、多行 Body 编辑器、集合树、标签页、对话框等全部使用 MyGo 组件，键盘导航、焦点、禁用态与无障碍语义由框架负责；配色跟随桌面亮暗主题。

### 🧪 界面可当单元测试跑

视图是 `func(c *ui.Context)`，`ui.NewTester` 能在没有窗口的情况下点击、输入、断言文本，按 L1 回归成本维护布局与交互。请求执行放在 goroutine，结果经 `Window.Update` 回主线程。

<a id="install"></a>
## 📦 安装

> [!NOTE]
> 目前没有发布任何安装包（Releases 为空），开发期请从源码运行。

| 方式 | 命令 | 说明 |
|---|---|---|
| 源码运行（推荐） | `go run .` | 需要 Go 1.27.2 或更高版本 |
| 本地产出可执行文件 | `go tool mygo build` | 产物写入 `mygo.json` 的 `out`（`build/`） |

<a id="start"></a>
## 🚀 快速开始

前置依赖：

| 依赖 | 版本 / 说明 |
|---|---|
| Go | 1.27.2+（`go.mod` 中声明） |
| MyGo CLI | 无需单独安装，`go tool mygo` 直接运行 `go.mod` 里 pin 的 v0.5.5 |
| Task | 3.54.0+，任务入口见 `Taskfile.yml`（`task --list` 查看全部） |

```bash
# 1. 拉取依赖
go mod download

# 2. 启动窗口（native UI，无需前端构建）
task run

# 3. 全量检查：go build/vet + mygo vet + 测试 + 竞态
task check
```

<a id="dev"></a>
## 🛠️ 开发与构建

日常入口是 Task，等价命令一并列出：

| 任务 | 等价命令 | 作用 |
|---|---|---|
| `task dev` | `go tool mygo dev` | 开发模式：改 Go 代码自动重建并重启窗口 |
| `task run` | `go run .` | 直接启动窗口 |
| `task build` | `go build -trimpath -ldflags "-s -w -H=windowsgui" -o bin/apigo.exe .` | 裸可执行文件到 `bin/`（版本号先写入 `mygo.json`，无安装包） |
| `task package` / `task package:windows` | `go tool mygo build -platform windows/amd64` | Windows：exe + NSIS 安装包，输出 `build/windows-amd64/` |
| `task package:darwin` | `go tool mygo build -platform darwin/universal` | macOS 通用包 `.app`（dmg 只能在 macOS 上生成） |
| `task package:linux ARCH=arm64` | `go tool mygo build -platform linux/arm64` | Linux：deb + tar.gz（默认 amd64） |
| `task package:all` | — | 三平台依次打包 |
| `task check` | `go build ./...`、`go vet ./...`、`go tool mygo vet .`、`CGO_ENABLED=0 go test ./...`、`go test -race ./...` | 静态检查 + 测试 + 竞态 |
| `task sync:version APP_VERSION=1.2.3` | `go run ./tools/syncversion -version 1.2.3` | 把版本号写进 `mygo.json`（唯一版本来源） |
| `task doctor` | `go tool mygo doctor` | 检查 MyGo 开发环境 |
| `task clean` | — | 清理 `bin/`、`build/`、`dist/`、`.mygo/` |
| — | `APIGO_SHOTS=docs/shots/m1 go test -run TestShots .` | 离屏渲染当前界面的三套外观 PNG，用于布局核验 |

> [!TIP]
> `go tool mygo build` 默认只构建 `windows/amd64`；其他形态直接加参数：`-platform web`（浏览器版 WASM + WebGL 2，输出 `<out>/web`）、`-debug`（保留 Web Inspector）。

Agent 协作规则见 `AGENTS.md`，MyGo 框架实践见 `.agents/skills/mygo-maintenance/SKILL.md`。

### GitHub 工作流

CI 不重复写命令，直接调用 Taskfile（`TASK_VERSION` 固定 3.54.0）：

| 工作流 | 触发 | 内容 |
|---|---|---|
| `Check` | push 到 `main`、所有 PR | `task check`；Windows 作业额外 `task package:windows` 打包冒烟（校验 exe 与 NSIS 安装包） |
| `Package` | 打 `v*` tag、手动触发 | 解析 tag 得到版本号，用 `task package:windows/darwin/linux APP_VERSION=<版本>` 构建三平台；Linux 包在收集阶段剥掉 mygo 硬编码的 GTK/WebKit 依赖；附 `.sha256` 校验和；tag 触发时创建/更新 Release |

发布命令序列：

```bash
git tag v0.1.0
git push origin v0.1.0     # 触发 Package，版本号取自 tag（去掉 v 前缀）
```

> [!NOTE]
> 版本号来自 tag 而非 `mygo.json` 当前值：`task sync:version` 会先改写 `mygo.json` 再构建（`mygo build` 没有 `-version` 参数）。macOS 产物未签名（未配置 `macos.signingIdentity`），首次打开需在系统中放行。

<a id="config"></a>
## ⚙️ 配置说明

应用元信息由 `mygo.json` 描述（native UI 项目无需 `mygo.config.ts`）：

| 字段 | 当前值 | 说明 |
|---|---|---|
| `name` | `apigo` | 用户可见的应用名、可执行文件名、安装包名 |
| `identifier` | `com.example.apigo` | 反向域名，系统据此记录偏好与注册信息；发布前应改为自有域名 |
| `version` | `0.1.0` | `App.Version()` 返回值，更新比对依据 |
| `out` | `build` | `mygo build` 输出目录 |
| 图标 | `resources/icon.png` | 默认取该路径；建议 1024×1024 方形 PNG |

> [!TIP]
> 主题无需配置：MyGo 跟随桌面亮暗设置，自定义颜色时用 `c.Theme()` 的语义色而非硬编码色值。

<a id="structure"></a>
## 🗂️ 目录结构

```text
apigo/
├── main.go                      # 入口：窗口创建与 App.Run
├── app.go                       # 应用状态：集合工作区、环境、外观、请求/响应动作
├── view.go                      # 根布局：集合标签条 / 主窗口或集合工作区 / 状态栏
├── view_topbar.go               # 集合标签条：主窗口、集合切换与关闭、最近关闭、设置
├── view_home.go                 # 主窗口：集合列表、新建/打开/删除、空状态引导
├── view_requests.go             # 接口管理面板：搜索、类型筛选、新建接口、历史
├── view_tabs.go                 # 请求标签页：新建、切换、关闭、右键菜单、环境选择器
├── view_kind.go                 # 接口类型图标与类型菜单
├── view_env.go                  # 环境管理弹窗与设置弹窗
├── view_request.go              # 请求行、请求子标签与面板
├── view_response.go             # 响应 meta 栏与空状态
├── view_footer.go               # 状态栏（集合名、接口类型、版本）
├── service_http.go              # 请求执行（goroutine + Window.Update，超时与 10 MB 上限）
├── service_store.go             # 集合目录布局、apigo.yaml、YAML 读写（原子写入）
├── service_tree.go              # 接口树与磁盘映射：目录=文件夹、<接口名>.yaml=接口
├── service_env.go               # 环境：每个环境一个 .env/<环境>.yaml
├── service_history.go           # 请求历史（一个文件，最多 200 条）
├── service_yaml.go              # YAML 读写、文件名净化、id 编号
├── service_kind.go              # 六种接口类型与它们的显示名
├── service_vars.go              # {{variable}} 解析与作用域合并
├── main_test.go                 # ui.NewTester 视图测试（区域尺寸、外观循环、面板与标签）
├── collections_test.go          # 磁盘布局、集合/环境/历史存储与变量替换
├── shot_test.go                 # APIGO_SHOTS=<dir> 时离屏渲染 PNG（布局核验/CI 产物）
├── internal/design/             # 设计令牌：色值、尺寸、字号、方法/状态配色、主题
├── internal/icons/              # 自绘 SVG 图标（currentColor，24×24）
├── docs/
│   ├── design-baseline.md       # 还原基线：尺寸/色值真相源与已知差异
│   ├── baseline/                # 目标截图基线（待补充）
│   └── shots/                   # 里程碑截图归档
├── LICENSE                      # MIT 协议
├── mygo.json                    # 应用元信息与构建输出目录
├── go.mod                       # 模块 apigo；Go 1.27.2；pin mygo v0.5.5 作为 tool
├── Taskfile.yml                 # 任务入口：dev / build / package:* / check / sync:version
├── tools/syncversion/           # 把 APP_VERSION 写进 mygo.json（task sync:version 调用）
├── resources/
│   └── icon.png                 # 应用图标（默认读取该路径）
├── .github/workflows/
│   ├── check.yml                # CI：静态检查 + 测试 + 竞态 + Windows 打包冒烟
│   └── release.yml              # tag 触发的三平台打包与 Release
└── .agents/
    ├── skills/mygo-maintenance/ # MyGo 官方维护技能（生命周期/身份/线程/测试）
    ├── readme/SKILL.md          # README 写作规范
    └── commit/SKILL.md          # 提交信息规范
```

### 集合在磁盘上的样子

一个集合就是一个目录，界面里的「根目录」就是它本身（没有同名文件夹），子文件夹与接口各对应真实的目录与文件：

```text
collections/智慧康养机器人/
├── apigo.yaml            # 集合自己的说明与设置：name / description
├── .env/                 # 环境：一个环境一个文件
│   ├── 本地.yaml          # id / name / active / variables
│   ├── 测试.yaml
│   └── 生产.yaml
├── static/               # 上传等接口要用的临时文件（接口里只留相对路径）
├── 列出宠物.yaml          # 根目录下的接口 = 集合目录下的文件
├── 新建宠物.yaml
└── 设备/                 # 子文件夹 = 真实目录，可以继续嵌套
    ├── 订阅设备事件.yaml
    └── 设备列表.yaml
```

- 接口文件名就是接口名：改名即改文件名，内容只存方法、URL、参数、请求头与请求体。
- 环境文件名就是环境名，`active: true` 的是当前环境；改名会跟着改文件名。
- `.env`、`static` 与以 `.` 开头的目录不是接口文件夹，读取时跳过。

<a id="tech"></a>
## 🧱 技术栈

| 层面 | 选型 |
|---|---|
| 语言 | Go 1.27.2（`go.mod`） |
| UI 框架 | MyGo v0.5.5 的 `ui` 包，native UI 模式（无 HTML/JS，无 WebView） |
| 渲染后端 | 平台 GPU 后端：Windows Direct3D 11 / macOS Metal / Linux OpenGL |
| 打包 | `go tool mygo build`：exe + NSIS 安装包 / `.app`+`.dmg` / deb+tar.gz / WASM |
| 测试 | 标准库 `testing` + `ui.NewTester` 离屏视图测试 |
| 存储 | 本地 YAML（`gopkg.in/yaml.v3`）：一个集合一个目录（`apigo.yaml` + `.env/<环境>.yaml` + `static/` + `<接口名>.yaml`），历史一文件；原子写入 `*.tmp` + rename |
| 数据目录 | MyGo 用户数据目录（Windows 为 `%APPDATA%\apigo`）下的 `collections/<集合目录>`，设置面板可查看 |
| cgo | 不需要（`CGO_ENABLED=0` 可测可构建） |

<a id="roadmap"></a>
## 🗺️ Roadmap

- [x] M1 应用骨架：顶部集合标签条 / 请求标签条 / 接口面板 / 状态栏
- [ ] M0 设计基线：等待 `docs/baseline/` 截图后逐屏标定尺寸与色值
- [x] M2 请求编辑器：方法下拉、URL、发送、参数/请求头键值表格、Body 编辑器
- [ ] M3 响应面板：JSON 折叠树、Headers 表格（状态码/耗时/大小与 Raw 正文已完成）
- [x] M4 网络执行层：goroutine + `Window.Update`、超时、10 MB 上限（取消与代理待补）
- [x] M5a 集合工作流：主窗口列表、新建集合（默认三环境 + BaseURL）、六类接口、保存与会话恢复
- [x] M5b 环境与变量：环境增删改激活、`{{name}}` 替换与未解析拒发
- [x] M5c 集合目录化存储：`apigo.yaml` / `.env` / `static` / 每个接口一个文件，树与磁盘一一对应
- [x] M6a 请求多标签：每个集合一组标签，新建/切换/关闭/复制/重命名
- [ ] M6b Ctrl K 命令面板与全局快捷键
- [ ] M7 设置页、缩放回归、无障碍抽查
- [ ] M8 三平台打包与发布、CI 截图产物
- [ ] M9 非 REST 接口执行：GraphQL、WebSocket、SSE、Socket.IO、MQTT
- [ ] M10 集合导入：Postman、在线 Swagger、OpenAPI v2/v3；集合导出；环境变量的全局作用域

<a id="license"></a>
## 📜 开源协议

本项目采用 [MIT 协议](LICENSE)，Copyright (c) 2026 younland。
