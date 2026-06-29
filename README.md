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

**rewrite 文件**处理流程：
1. 分离 `hostname =` 行
2. 按 action 类别排序，同类内按字母序
3. 插入 HeadRules 到最前面
4. 文本去重（完全匹配）
5. 语义去重（pattern 归一化 + 子串覆盖 + reject 类覆盖任意类型 + 分支展开）+ 加速域名替换
6. 冗余转义归一化输出（`\/`→`/` 等冗余反斜杠转义剥离，保留 `\.`/`\d` 等语义转义）
7. 应用 Excludes 前缀排除
8. 追加空行 + `hostname =` 去重排序后的域名

### rewrite pattern 归一化

rewrite 去重比较前会对 pattern 做归一化，使不同写法的等价 pattern 能互相识别：

| 归一化 | 示例 | 说明 |
|--------|------|------|
| `\/` → `/` | `^https:\/\/x.com` ≡ `^https://x.com` | `/` 在正则中非元字符，`\/` 冗余 |
| `\:` `\;` `\!` `\#` `\&` `\=` `\@` `\%` `\,` `\~` → 去反斜杠 | 同上 | 这些字符非元字符，转义冗余 |
| 剥离 `(?i)` 并记录大小写不敏感标志 | `(?i)\b/ad/` → body=`/ad/`, ci=true | 用于判定覆盖方向 |
| 剥离 `\b` `\B` | `\b/ad/` → `/ad/` | 词边界在 `/` 定界场景下由 `/` 天然替代 |

**不动的转义**（有真实语义）：`\.` `\*` `\+` `\?` `\^` `\$` `\(` `\)` `\[` `\]` `\{` `\}` `\|` `\\` `\d` `\w` `\s` `\b`（比较时剥离，输出时保留）等。

输出时会剥离冗余转义（第 6 步），使最终 `rewrite.snippet` 中不再出现 `\/` 等冗余反斜杠。

### `(?i)` 大小写不敏感方向判定

当 pattern 含 `(?i)` 标志时，覆盖判定遵循方向规则：

| 已保留 (kept) | 新进 (new) | 判定 |
|---------------|------------|------|
| 有 `(?i)` | 无 `(?i)` | kept 更宽 → 删除 new ✅ |
| 无 `(?i)` | 有 `(?i)` | new 更宽 → **不删除** ❌ |
| 都有或都无 | — | 正常归一化比较 |

### 安全守卫

为防止短 pattern 误删无关规则，仅当 kept pattern 满足以下条件之一时才触发子串删除：
- 长度 ≥ 6 字符
- 含 `/`
- 以 `^` 开头

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

# 自定义策略名称（覆盖 snippet 中的策略后缀）
POLICY_DIRECT=DIRECT POLICY_PROXY=MyProxy POLICY_REJECT=AdBlock ./quantumult-x --conf-dir . --out-dir out
```

### CLI 参数

| 参数 | 默认值 | 说明 |
|------|--------|------|
| `--conf-dir` | `.` | `.conf` 文件所在目录 |
| `--out-dir` | `out` | 输出目录 |
| `--config` | `src/config/config.yaml` | 配置文件路径 |
| `--examples` | `false` | 使用 `examples/` 本地文件代替网络抓取 |

> 生成器只负责在本地生成 `*.snippet` 文件；发布到 `release` 分支由 GitHub Actions 完成，因此不再需要 `--dry-run`。

### 配置文件 `config.yaml`

```yaml
concurrency: 16          # 并发抓取数
http_timeout_sec: 30     # HTTP 超时（秒）
http_retries: 2          # HTTP 重试次数
log_level: info          # 日志级别：debug/info/warn/error
releases_branch: release # 发布分支名
policies:                # 策略名称（可通过环境变量 POLICY_DIRECT/POLICY_PROXY/POLICY_REJECT 覆盖）
  direct: direct
  proxy: proxy
  reject: reject
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

### 策略变量

通过环境变量 `POLICY_DIRECT`、`POLICY_PROXY`、`POLICY_REJECT` 自定义 snippet 文件中的策略后缀名称，也可在 `config.yaml` 的 `policies` 段配置。

```
# 默认输出
host-keyword,adserv,reject
host-suffix,google.com,proxy

# 设置 POLICY_REJECT=AdBlock POLICY_PROXY=MyProxy 后
host-keyword,adserv,AdBlock
host-suffix,google.com,MyProxy
```

已有默认策略后缀（`,direct`/`,proxy`/`,reject`）的行会被**替换**为新策略名，而非追加。`ip-cidr,...,no-resolve` 等非策略后缀不受影响。

---

## 语义去重

除了完全文本去重外，工具还实现了向后语义去重（先出现的规则优先保留）：

| 已有规则 | 后续规则 | 是否覆盖 | 原因 |
|----------|----------|----------|------|
| `host-keyword,google` | `host-suffix,google.com` | ✅ 删除 | keyword 匹配 suffix |
| `host-keyword,google` | `host,mail.google.com` | ✅ 删除 | keyword 匹配 host |
| `host-suffix,example.com` | `host,sub.example.com` | ✅ 删除 | suffix 包含 host |
| `host-suffix,example.com` | `host-suffix,sub.example.com` | ✅ 删除 | 父域名已覆盖 |

头部规则（HeadRules）插入到列表最前面，因此优先级最高。可通过在 `.conf` 文件中书写 `host-keyword,xxx,reject` 形式的 headrule 来批量覆盖上游规则。

### reject headrules 示例

`reject.conf` 中通过 14 条 `host-keyword` headrules 覆盖广告/追踪域名：

```
host-keyword,adserv,reject
host-keyword,advert,reject
host-keyword,adsdk,reject
host-keyword,adx,reject
host-keyword,adtarget,reject
host-keyword,adsys,reject
host-keyword,beacon,reject
host-keyword,tracker,reject
host-keyword,umeng,reject
host-keyword,dsp,reject
host-keyword,rtb,reject
host-keyword,adnet,reject
host-keyword,admob,reject
host-keyword,pangolin,reject
```

### rewrite headrules 示例

`rewrite.conf` 中通过正则 pattern headrules 覆盖上游 URL 重写规则：

```
/ad/ url reject
(?i)\b/ad/ url reject
(?i)\b/ads/ url reject
(?i)\b/adv/ url reject
(?i)\b/advert/ url reject
(?i)\b/adx/ url reject
(?i)\b/ad\? url reject
(?i)\b/ads\? url reject
(?i)\badvertisement url reject
(?i)\badvertising url reject
(?i)\bsplash_screen url reject
```

这些通用 pattern 会覆盖所有含 `/ad/`、`/ads/`、`/advert/` 等路径段的上游具体 URL 规则。

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
    ├── main.go              # CLI 入口与处理管线编排
    ├── config/              # 配置加载与校验
    ├── fetch/               # 并发 HTTP 抓取
    ├── clean/               # conf 解析、行清洗、HeadRules/Excludes 应用
    ├── transform/           # 格式转换 (rule/set/qx) 与混合格式回退
    ├── dedup/               # 文本去重与语义去重
    ├── rewrite/             # rewrite 合并/去重/加速替换/处理流水线
    ├── rules/               # 规则分组排序
    ├── pipeline/            # 上游源抓取+清洗+转换流水线
    ├── io/                  # snippet 原子写入
    ├── log/                 # 日志
    └── util/                # 共享类型
```

---

## 许可证

MIT
