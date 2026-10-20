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

apigo 是一个用 Go 编写、由 MyGo 原生 UI 直接绘制的接口调试工具，形态对标 Hoppscotch：在一个窗口里填写请求（方法、URL、Query/Header/Body）、发送、查看响应，并把请求整理成集合、按环境切换变量。它没有前端构建链，也没有浏览器运行时——界面和逻辑都是 Go，单文件分发、启动即用、完全离线。

当前处于脚手架阶段：窗口、视图与视图测试已经跑通，请求执行、集合与环境等能力按 [Roadmap](#roadmap) 逐步实现；开发与协作规则见 `AGENTS.md`。

<a id="status"></a>
## 📌 项目现状

> [!IMPORTANT]
> 本仓库目前是 MyGo native 模板起步的**脚手架**：只包含一个示例视图（问候语 + 计数器）与对应的视图测试。HTTP 请求、集合、环境变量等接口调试能力尚未实现，清单见 [Roadmap](#roadmap)。

| 能力 | 现状 |
|---|---|
| 原生 UI 单窗口（自绘界面，无 WebView） | 已实现：`main.go` 的 `app.view` |
| 窗口尺寸与位置记忆 | 已实现：`mygo.WindowOptions.StateKey` |
| 主题跟随系统、布局基础组件 | 已实现：示例视图用 `ui.Column`/`ui.Row`/`ui.TextInput` |
| 视图离屏测试 | 已实现：`main_test.go` 用 `ui.NewTester` 点击、输入并断言文案 |
| 请求工作区、集合、环境、历史 | 未实现 |

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
├── main.go                      # 入口：app 状态、view 视图、窗口创建
├── main_test.go                 # ui.NewTester 视图测试（点击 / 输入 / 文本断言）
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

<a id="tech"></a>
## 🧱 技术栈

| 层面 | 选型 |
|---|---|
| 语言 | Go 1.27.2（`go.mod`） |
| UI 框架 | MyGo v0.5.5 的 `ui` 包，native UI 模式（无 HTML/JS，无 WebView） |
| 渲染后端 | 平台 GPU 后端：Windows Direct3D 11 / macOS Metal / Linux OpenGL |
| 打包 | `go tool mygo build`：exe + NSIS 安装包 / `.app`+`.dmg` / deb+tar.gz / WASM |
| 测试 | 标准库 `testing` + `ui.NewTester` 离屏视图测试 |
| cgo | 不需要（`CGO_ENABLED=0` 可测可构建） |

<a id="roadmap"></a>
## 🗺️ Roadmap

- [ ] 请求工作区：方法 / URL / Query / Header / Body（raw JSON、form、none）
- [ ] 响应面板：状态码、耗时、响应体大小、JSON 高亮与格式化
- [ ] 请求在后台 goroutine 执行，结果经 `Window.Update` 回主线程，支持取消
- [ ] 集合与请求历史：本地 JSON 存储，服务层（`*service.go`）不依赖 UI
- [ ] 环境与变量：`{{var}}` 替换、按环境切换、密钥不落盘明文/不进日志
- [ ] 多标签请求编辑与快捷键
- [ ] 代码片段导出（curl / fetch / Go）
- [ ] 视图测试覆盖关键交互，布局在亮暗主题与 100%–150% 缩放下核验

<a id="license"></a>
## 📜 开源协议

本项目采用 [MIT 协议](LICENSE)，Copyright (c) 2026 younland。
