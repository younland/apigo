---
name: readme
description: 本仓库 README.md 的写作规范——结构模板、排版约定与内容同步要求，风格提炼自 RcloneDrive 参考范本
whenToUse: 当用户让我新写、重写或大幅更新 README.md、项目主页文档时
---

# README 写作规范

风格基准：[RcloneDrive README](https://github.com/younland/RcloneDrive/blob/main/README.md)。核心原则：**信息密度高、面向两类读者（使用者与贡献者）、表格优先于列表、具体细节优于形容词**。

## 受众与目标

README 同时服务两类读者，段落顺序按"使用者在前、贡献者在后"排布：

1. **使用者**：这个项目是什么、有什么功能、怎么装、怎么跑
2. **贡献者**：技术栈、目录结构、开发命令、构建与发布流程

## 结构模板

```markdown
<div align="center">            # ① Hero 区
  <img logo /> <h1>项目名</h1>
  <p><strong>一句话定位（ bold，含核心技术栈）</strong></p>
  <p>关键卖点 · 用间隔号分隔的一行</p>
  <p>徽章组 + 锚点导航</p>
</div>

### 🗂️ 核心特性小节 # ② 3-5 个，### + emoji + 短段落

## 📷 应用截图 # ③ 有截图才放，居中 + 图注

## 📦 安装 # ④ 平台/方式表格

## 🚀 快速开始 / 本地运行 # ⑤ 前置依赖 + 命令序列

## ⚙️ 配置说明 # ⑥ 环境变量表格

## 🛠️ 构建 / 开发指南 # ⑦ 命令 + 目录结构

## 🧩 功能细节（可选多个） # ⑧ 表格化文档

## 🗺️ Roadmap（可选） # ⑨ GitHub task list

## ⚠️ 免责声明（可选）

## 📜 开源协议 # ⑩ 最后一节，链接 LICENSE
```

章节用 `<a id="xxx"></a>` 锚点 + 二级标题（`## 📷 应用截图`），Hero 导航链接到锚点。

## 排版约定

- **语言**：中文；一句话定位用 `<strong>` 加粗
- **标题**：二级章节 `## emoji + 名词`，特性小节 `### emoji + 短语`
- **表格**：功能说明、平台差异、环境变量、技术栈一律用表格，不用 bullet 列表
- **提示块**：用 GitHub admonition——`> [!NOTE]`（补充说明）、`> [!TIP]`（使用技巧）、`> [!IMPORTANT]`（行为约束/限制）
- **徽章**：`img.shields.io` for-the-badge 风格，logo 官方色 + 自定义 labelColor，每张徽章都带链接；放 Releases、License、核心语言/框架、平台
- **代码块**：标语言（`bash`、`text`），命令可直接复制执行
- **目录结构**：`text` 代码块树，每行带 `# 注释` 说明用途
- emoji 只出现在章节标题，正文不用

## 内容规则

- **事实必须来自仓库本身**：写之前先查 `configs/config.yaml`（端口、默认配置）、`Taskfile.yml`（命令）、`go.mod`/`package.json`（版本）、`Dockerfile`、workflow 文件，禁止凭记忆写版本号与端口号
- **不写没有的功能**：README 描述的每个命令、端口、环境变量都要真实存在；删掉模板遗留与实际不符的内容
- **具体优于宏大**：写"JWT + 图形验证码登录、RBAC 到按钮级"而不是"安全可靠的企业级方案"
- **密钥只出现变量名**：配置示例里写 `KRATOS_AUTH_JWT_KEY=<random>`，绝不给真实默认值
- **同步义务**：改动了 README 描述的命令、端口、目录结构、环境变量时，必须同步更新 README

## 工作流程

1. 收集事实：版本（go.mod / package.json / Dockerfile）、端口、env 变量清单、task 命令、CI workflow、目录结构
2. 按模板列大纲，特性小节从用户视角写（解决什么问题），技术细节放表格
3. 写完后逐条核对：每个命令真实可执行、每个版本号与锁文件一致、每个链接可达
