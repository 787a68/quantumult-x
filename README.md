# Quantumult X 规则片段生成器

一个用 Go 编写的自动化工具，从多个上游规则源抓取、清洗、转换、去重并生成 Quantumult X 可用的 `*.snippet` 规则片段文件。通过 GitHub Actions 自动构建并发布到 `release` 分支。

---

## 工作流程

```
上游规则源 (HTTP/HTTPS)
      │
      ▼
  并发抓取 (worker pool)
      │
      ▼
  清洗 (去注释/空行/行尾注释)
      │
      ▼
  格式转换 (rule / set / qx → 统一 QX 格式)
      │
      ▼
  文本去重 (完全匹配)
      │
      ▼
  分组排序 + 插入头部规则
      │
      ▼
  语义去重 (先出现优先，keyword 覆盖 suffix/host)
      │
      ▼
  应用排除列表
      │
      ▼
  追加策略后缀 + 原子写入 *.snippet
```

**rewrite 文件**额外处理：
- 解析 QX rewrite 格式（`^URL_PATTERN url ACTION [args]`）
- 按 URL pattern 语义去重
- 替换 GitHub/raw 链接为加速域名
- 提取 `hostname =` 行，去重排序后追加到文件底部

---

## 输出文件

| 文件 | 策略后缀 | 说明 |
|------|----------|------|
| `direct.snippet` | `,direct` | 直连规则 |
| `proxy.snippet` | `,proxy` | 代理规则 |
| `reject.snippet` | `,reject` | 拒绝/广告拦截规则 |
| `rewrite.snippet` | 无 | URL 重写规则 |

---

## 配置文件

### `.conf` 文件语法

四个配置文件（`direct.conf`、`proxy.conf`、`reject.conf`、`rewrite.conf`）使用统一语法：

```bash
# 这是注释（# 或 // 或 ; 开头）

# 上游规则源（URL + 格式，格式可选：rule/set/qx，缺省 qx）
https://example.com/rules.txt,rule
https://example.com/domains.txt,set
https://example.com/filter.list,qx

# 头部规则（直接插入到输出最前面，优先级最高）
host-keyword,adserve,reject
host,local.example.com

# 排除规则（以 - 开头，前缀匹配删除最终结果中对应的行）
-host-suffix,apple.com
-host,stats.example.com
```

### 支持的上游格式

| 格式标识 | 说明 | 示例输入 | 转换结果 |
|----------|------|----------|----------|
| `rule` | Surge 格式 | `DOMAIN-SUFFIX,google.com` | `host-suffix,google.com` |
| `set` | 域名后缀列表 | `.google.com` 或 `google.com` | `host-suffix,google.com` |
| `qx` | Quantumult X 原生 | `host-suffix,google.com,proxy` | `host-suffix,google.com` |

转换时会自动尝试多种格式，混合格式的上游源也能正确处理。

---

## 本地使用

### 环境要求

- Go 1.26+

### 构建

```bash
go build -o quantumult-x ./src
```

### 运行

```bash
# 在线模式：从 .conf 中的 URL 抓取上游规则
./quantumult-x --conf-dir . --out-dir out

# 离线模式：使用 examples/ 目录下的本地文件作为上游源
./quantumult-x --conf-dir . --out-dir out --examples

# 指定配置文件
./quantumult-x --conf-dir . --out-dir out --config src/config/config.yaml

# 设置加速域名（替换 rewrite 中的 GitHub/raw 链接）
ACCEL_DOMAIN=your-proxy.example.com ./quantumult-x --conf-dir . --out-dir out
```

### CLI 参数

| 参数 | 默认值 | 说明 |
|------|--------|------|
| `--conf-dir` | `.` | `.conf` 文件所在目录 |
| `--out-dir` | `out` | 输出目录 |
| `--config` | `src/config/config.yaml` | 配置文件路径 |
| `--dry-run` | `false` | 只生成不发布 |
| `--examples` | `false` | 使用 `examples/` 本地文件代替网络抓取 |

### 配置文件 `config.yaml`

```yaml
concurrency: 16          # 并发抓取数
http_timeout_sec: 30     # HTTP 超时（秒）
http_retries: 2          # HTTP 重试次数
log_level: info          # 日志级别：debug/info/warn/error
releases_branch: release # 发布分支名
```

### 加速域名

通过环境变量 `ACCEL_DOMAIN` 注入。设置后，rewrite 规则中的 GitHub 链接会被替换：

```
# 原始
^https://example.com/api url script-response-body https://raw.githubusercontent.com/user/repo/main/script.js

# 替换后（ACCEL_DOMAIN=myproxy.com）
^https://example.com/api url script-response-body https://myproxy.com/raw.githubusercontent.com/user/repo/main/script.js
```

匹配的域名包括：`github.com`、`raw.githubusercontent.com`、`raw.github.com`。

---

## 语义去重

除了完全文本去重外，工具还实现了向后语义去重（先出现的规则优先保留）：

| 已有规则 | 后续规则 | 是否覆盖 | 原因 |
|----------|----------|----------|------|
| `host-keyword,google` | `host-suffix,google.com` | ✅ 删除 | keyword 匹配 suffix |
| `host-keyword,google` | `host,mail.google.com` | ✅ 删除 | keyword 匹配 host |
| `host-suffix,example.com` | `host,sub.example.com` | ✅ 删除 | suffix 包含 host |
| `host-suffix,example.com` | `host-suffix,sub.example.com` | ✅ 删除 | 父域名已覆盖 |

头部规则（HeadRules）插入到列表最前面，因此优先级最高。

---

## rewrite 规则处理

### 支持的 rewrite 动作

| 动作 | 说明 |
|------|------|
| `reject` / `reject-200` / `reject-img` / `reject-dict` / `reject-array` | 拦截请求 |
| `302` / `307` | URL 重定向 |
| `request-header` / `request-body` | 修改请求 |
| `response-header` / `response-body` | 修改响应 |
| `echo-response` | 返回本地文件 |
| `jsonjq-response-body` | JSON 处理 |
| `script-request-header` / `script-request-body` | 请求脚本 |
| `script-response-header` / `script-response-body` | 响应脚本 |
| `script-echo-response` / `script-analyze-echo-response` | 回声脚本 |
| `url-and-header` | URL+Header 联合匹配 |

### hostname 处理

rewrite 源文件中的 `hostname =` 行（MITM 主机名声明）会被提取、去重、排序，合并为单行追加到 `rewrite.snippet` 末尾，前面隔一个空行。

---

## GitHub Actions

### generate.yml — 自动生成与发布

**触发条件**：
- 推送到 `main` 分支
- 每天定时（UTC 06:06）
- 手动触发

**流程**：
1. 构建 Go 程序
2. 运行生成器（注入 `ACCEL_DOMAIN` 仓库变量）
3. 切换到 `release` 分支（只含 snippet 文件）
4. 提交变更，commit message 包含每个文件的行数变化：

```
update snippets
direct.snippet: 112605 lines (+12)
proxy.snippet: 4126 lines (-3)
reject.snippet: 218605 lines (+269)
rewrite.snippet: 1621 lines (+1)
```

### cleanup.yml — 历史清理

**触发条件**：
- 每月 1 号定时（UTC 00:00）
- 手动触发

**功能**：
- 将 `release` 分支中超过 6 个月的提交压缩为单条 commit
- 保留最近 6 个月的完整提交历史
- 防止长期运行后提交历史过大

---

## 在 Quantumult X 中使用

在 Quantumult X 配置中引用 `release` 分支的 snippet 文件：

```ini
[filter_remote]
https://raw.githubusercontent.com/<your-username>/<repo>/release/direct.snippet, tag=Direct, force-policy=direct, enabled=true
https://raw.githubusercontent.com/<your-username>/<repo>/release/proxy.snippet, tag=Proxy, force-policy=proxy, enabled=true
https://raw.githubusercontent.com/<your-username>/<repo>/release/reject.snippet, tag=Reject, force-policy=reject, enabled=true

[rewrite_remote]
https://raw.githubusercontent.com/<your-username>/<repo>/release/rewrite.snippet, tag=Rewrite, enabled=true
```

---

## 项目结构

```
quantumult-x/
├── direct.conf              # 直连规则配置
├── proxy.conf               # 代理规则配置
├── reject.conf              # 拒绝规则配置
├── rewrite.conf             # 重写规则配置
├── examples/                # 本地测试用的示例上游文件
├── .github/workflows/
│   ├── generate.yml         # 自动生成与发布
│   └── cleanup.yml          # 月度历史清理
├── doc/                     # 参考文档
└── src/
    ├── main.go              # CLI 入口与处理管线
    ├── config/              # 配置加载
    ├── fetch/               # 并发 HTTP 抓取
    ├── clean/               # conf 解析与行清洗
    ├── transform/           # 格式转换 (rule/set/qx)
    ├── dedup/               # 文本去重与语义去重
    ├── rewrite/             # rewrite 合并/去重/加速替换
    ├── io/                  # 文件写入与发布
    ├── log/                 # 日志
    └── util/                # 共享类型
```

---

## 许可证

MIT
