---
name: commit
description: 本仓库的 Git 提交信息规范（Conventional Commits 风格，含类型与固定图标约定）
whenToUse: 当用户让我生成提交信息、提交代码、总结变更或整理 git 历史时
---

# Git 提交规范

本仓库采用 **Conventional Commits** 风格的提交信息，并为每种类型指定了固定图标。

## 提交前必须经用户交互式确认（强制）

**任何 `git commit` 之前都必须由用户确认，且必须是按钮式提问**：先把准备提交的内容展示出来，再用提问工具发起带选项的确认，得到明确同意后才能执行提交。

- 确认必须采用**按钮式提问交互**（如 AskUserQuestion），选项至少包含：
  - **确认提交**（推荐项）：按展示的文件清单与提交信息执行 `git add` + `git commit`。
  - **暂不提交**：保持工作区现状。
  - **调整提交信息/范围**：用户回复修改意见，改完后重新展示并再次发起按钮式确认。
  - 环境不支持按钮式提问时，才退化为文字展示 + 等待用户明确回复；用户没有回答不等于同意。
- 禁止在用户未回复确认时自行提交；也不要拿"之前提过要提交"当作授权。
- **逐次确认**：上一条提交被批准，不构成对下一条提交的授权；每次提交都要重新展示并确认。
- 展示内容至少包含：
  - 将要提交的文件清单（`git status --porcelain` 的输出），并说明哪些改动**不会**进入本次提交（未跟踪、被忽略或故意留下的文件）。
  - 完整的提交信息（标题行 + 正文要点），可直接复制。
  - 涉及多种类型改动时，给出拆分提交的建议。
- 顺序：`git add` / `git status` 准备好 → 展示内容 + 发起按钮式提问 → 用户选择「确认提交」后 `git commit`。
- 未经确认前不要执行 `git commit`、`git commit --amend`、`git push` 等任何写入历史的操作。

## 提交格式

```
<icon> <type>(<scope>): <subject>
```

- `icon`: 提交类型图标（见下表）
- `type`：提交类型（见下表）
- `scope`（可选）：影响范围，如 `core`、`web`、`auth`、`user` 等
- `subject`：简短描述，使用中文或英文均可，但同一仓库建议统一

## 提交正文（Body）

非 trivial 的提交必须带正文：标题行后空一行，用 `- ` 无序列表逐条说明改动要点（每条一句话，说清"做了什么"而非复述标题）。琐碎改动（一两个文件的小修复）可只写标题。

模板：

```
<icon> <type>(<scope>): <subject>

- 改动要点 1
- 改动要点 2
- 改动要点 3
```

示例：

```
✨ feat(user): 新增用户导出功能与列表搜索优化

- 新增用户列表 Excel 导出接口，支持按搜索条件过滤
- 列表查询支持按部门与状态组合筛选
- 修复导出大数据量时的内存溢出问题
- 同步生成 gfa-web 前端 API 客户端代码
- 补充导出功能的单元测试
```

## 提交类型与图标

| 类型       | 图标 | 描述                                             |
| ---------- | ---- | ------------------------------------------------ |
| `init`     | 🎉   | 初始化项目                                       |
| `feat`     | ✨   | 新功能                                           |
| `fix`      | 🐞   | 错误修复                                         |
| `docs`     | 📃   | 修改文档                                         |
| `style`    | 🌈   | 格式修改（不影响功能，例如空格、分号等格式修正） |
| `refactor` | 🦄   | 代码重构                                         |
| `perf`     | 🎈   | 代码优化 / 性能优化                              |
| `test`     | 🧪   | 新增测试 / 修改测试                              |
| `build`    | 🔧   | 构建依赖更改                                     |
| `ci`       | 🐎   | 更改 CI 配置文件或者脚本                         |
| `chore`    | 🐳   | 更改构建流程、或者增加依赖库、工具等             |
| `revert`   | ↩    | 代码回退                                         |

## 示例

```
🎉 init: initialize gfa-core project
✨ feat(auth): add JWT login
🐞 fix(user): correct pagination total count
📃 docs(readme): update local development guide
🌈 style(web): format tailwind class order
🦄 refactor(core): extract common middleware
🎈 perf(dao): add index for deleted_at filter
🧪 test(auth): add login unit tests
🔧 build(core): bump gf to v2.10.2
🐎 ci: add github actions workflow
🐳 chore: add commitlint config
↩ revert: revert "feat(auth): add JWT login"
```

## 注意事项

- **提交前必须经用户交互式确认**（见上文「提交前必须经用户交互式确认」），未经同意不得自行提交。
- 优先选择能准确表达改动意图的最小类型。
- 当一次改动同时包含多个类型时，建议拆分为多个提交；若必须合并，使用最主要的类型。
- 图标与类型之间保留一个空格，例如 `✨ feat(auth): ...`。
