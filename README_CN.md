<p align="center">
  <img src="https://img.shields.io/badge/🔗-safechain-00ADD8?style=for-the-badge&logo=go&logoColor=white" alt="safechain logo" />
</p>

<p align="center">
  <a href="https://pkg.go.dev/github.com/mredencom/safechain"><img src="https://pkg.go.dev/badge/github.com/mredencom/safechain.svg" alt="Go Reference"></a>
  <a href="https://goreportcard.com/report/github.com/mredencom/safechain"><img src="https://goreportcard.com/badge/github.com/mredencom/safechain" alt="Go Report Card"></a>
  <a href="https://github.com/mredencom/safechain/actions"><img src="https://github.com/mredencom/safechain/workflows/CI/badge.svg" alt="CI"></a>
  <a href="https://codecov.io/gh/mredencom/safechain"><img src="https://codecov.io/gh/mredencom/safechain/branch/main/graph/badge.svg" alt="Coverage"></a>
  <a href="https://github.com/mredencom/safechain/releases"><img src="https://img.shields.io/github/v/release/mredencom/safechain?color=blue" alt="Release"></a>
  <a href="https://github.com/mredencom/safechain/blob/main/LICENSE"><img src="https://img.shields.io/badge/license-MIT-blue.svg" alt="License"></a>
  <a href="https://github.com/mredencom/safechain"><img src="https://img.shields.io/github/stars/mredencom/safechain?style=social" alt="Stars"></a>
</p>

<p align="center">
  <b>Go 语言深层嵌套结构体指针链的 nil 安全访问工具。</b><br/>
  告别 <code>if a != nil && a.B != nil && a.B.C != nil</code> 的样板代码。
</p>

<p align="center">
  <a href="#安装">安装</a> •
  <a href="#快速上手">快速上手</a> •
  <a href="#api-说明">API 说明</a> •
  <a href="#性能测试">性能测试</a> •
  <a href="./README.md">English</a>
</p>

---

## 痛点

Go 没有可选链（optional chaining）。访问 `req.A.B.C.D` 需要逐层判空：

```go
var token string
if req != nil && req.Auth != nil && req.Auth.Key != nil &&
    req.Auth.Key.Session != nil && req.Auth.Key.Session.Token != nil {
    token = *req.Auth.Key.Session.Token
}
```

**safechain** 提供两种方式消除这些样板代码：

| 方式 | 适用场景 | 开销 |
|------|---------|------|
| `Safe` / `Must` / `OrVal` | 不需要知道哪个字段为 nil | ~4 ns，0 分配 |
| `Dig` + `S()` | 需要精确知道哪个字段为 nil | ~190 ns/100层，0 分配 |
| `Ensure` + `Set` | 无需判空构建/赋值深层嵌套结构体 | ~4 ns，0 分配 |

## 安装

```bash
go get github.com/mredencom/safechain
```

## 快速上手

```go
import "github.com/mredencom/safechain"

// 读取 — 任意指针为 nil 就返回零值
token := safechain.Must(func() string {
    return *req.Auth.Key.Session.Token
})

// 读取 — 带默认值
token := safechain.OrVal(func() string {
    return *req.Auth.Key.Session.Token
}, "N/A")

// 读取 — comma-ok 风格
token, ok := safechain.Safe(func() string {
    return *req.Auth.Key.Session.Token
})

// 写入 — 构建嵌套结构体并赋值，无需判空
var req Request
safechain.E(&safechain.E(&safechain.E(&req.Auth).Key).Session).Token = ptr("my_token")
```

## API 说明

### 核心 — 基于 recover（最常用）

```go
// 返回 (值, 是否成功) — 指针字段（如 *string）需要 * 解引用
val, ok := Safe(func() string { return *req.Auth.Key.Session.Token })

// 返回值，失败返回零值
val := Must(func() string { return *req.Auth.Key.Session.Token })

// 返回值，失败返回默认值
val := OrVal(func() string { return *req.Auth.Key.Session.Token }, "N/A")

// 值类型字段（如 int）不需要 *
count, ok := Safe(func() int { return req.Auth.Key.Session.RetryCount })
```

### 逻辑判断 — And / Or / Any / Not / None / Count / AtLeast / NotNil

```go
// 所有条件都为 true
ok := And(
    Check(func() { _ = *req.Auth.Key.Session.Token }),
    HasPrefix(func() string { return *req.Auth.Key.Session.Token }, "Bearer"),
    Gt(func() int { return *req.Auth.RetryCount }, 0),
)

// 任一条件为 true（Or 是 Any 的别名）
ok := Or(
    Check(func() { _ = *req.Auth.Key.Session.Token }),
    Check(func() { _ = *req.Fallback }),
)

// 取反
ok := And(
    Check(func() { _ = *req.Auth.Key.Session.Token }),
    Not(HasPrefix(func() string { return *req.Auth.Key.Session.Token }, "Bearer")),
)

// 所有条件都为 false
ok := None(
    Check(func() { _ = *req.BannedToken }),
    Check(func() { _ = *req.ExpiredToken }),
)

// 统计 true 的个数
n := Count(
    Check(func() { _ = *req.Auth.Key.Session.Token }),
    Check(func() { _ = *req.Fallback }),
    Check(func() { _ = *req.Meta.TraceID }),
)

// 至少 N 个条件为 true
ok := AtLeast(2,
    Check(func() { _ = *req.Auth.Key.Session.Token }),
    Check(func() { _ = *req.Fallback }),
    Check(func() { _ = *req.Meta.TraceID }),
)

// NotNil — 简化的 nil 检查，不需要 _ = 和 *
ok := NotNil(func() any { return req.Auth.Key.Session })
```

### 取值 — First / MustFirst

```go
// 返回第一个成功的值（类似 SQL COALESCE）
token := MustFirst(
    func() string { return *req.Auth.Key.Session.Token },
    func() string { return *req.Fallback },
    func() string { return "anonymous" },
)
```

### 比较 — Eq / Ne / Gt / Gte / Lt / Lte / Between

指针字段（如 `*string`、`*int`）需要 `*` 解引用，值类型字段（如 `int`、`string`）不需要。

```go
// *r.A.Name — Name 是 *string，需要 *
Eq(func() string { return *r.A.Name }, "admin")
Ne(func() string { return *r.A.Name }, "guest")

// *r.A.Score — Score 是 *int，需要 *
Gt(func() int { return *r.A.Score }, 10)
Gte(func() int { return *r.A.Score }, 10)
Lt(func() float64 { return *r.A.Rate }, 3.14)
Lte(func() float64 { return *r.A.Rate }, 3.14)
Between(func() int { return *r.A.Score }, 1, 100)

// r.A.Count — Count 是 int（值类型），不需要 *
Gt(func() int { return r.A.Count }, 0)

// 区间变体
BetweenExcl(func() int { return *r.A.Score }, 0, 100)   // (0, 100)  开区间
BetweenLExcl(func() int { return *r.A.Score }, 0, 100)  // (0, 100]  左开右闭
BetweenRExcl(func() int { return *r.A.Score }, 0, 100)  // [0, 100)  右开左闭

// 自定义谓词
Match(func() string { return *r.A.Name }, func(v string) bool { return len(v) > 3 })
```

### 字符串匹配

```go
HasPrefix(func() string { return *r.A.Name }, "hello")
HasSuffix(func() string { return *r.A.Name }, "world")
Contains(func() string { return *r.A.Name }, "llo_wor")
EqFold(func() string { return *r.A.Name }, "HELLO")
MatchRegexp(func() string { return *r.A.Name }, `^\d+$`)
MatchRegexpCompiled(func() string { return *r.A.Name }, re)  // 预编译，热路径推荐
```

### []byte 匹配

```go
BytesHasPrefix(func() []byte { return *r.A.Data }, []byte("hello"))
BytesHasSuffix(func() []byte { return *r.A.Data }, []byte("world"))
BytesContains(func() []byte { return *r.A.Data }, []byte("llo"))
BytesEq(func() []byte { return *r.A.Data }, []byte("exact"))
BytesMatchRegexp(func() []byte { return *r.A.Data }, `\d+`)
BytesMatchRegexpCompiled(func() []byte { return *r.A.Data }, re)
```

### SafeDig — 命名链 + 精确错误定位

不需要 `unsafe.Pointer`，用 `F()` 定义每一步：

```go
// MustSafeDig — 只检查路径是否存在，返回 bool
ok := MustSafeDig(req,
    F("Auth", func(r *Request) any { return r.Auth }),
    F("Key", func(a *Auth) any { return a.Key }),
    F("Session", func(k *Key) any { return k.Session }),
)

// SafeDig — 取值 + bool
token, ok := SafeDig[string](req,
    F("Auth", func(r *Request) any { return r.Auth }),
    F("Key", func(a *Auth) any { return a.Key }),
    F("Session", func(k *Key) any { return k.Session }),
    F("Token", func(s *Session) any { return s.Token }),
)

// SafeDigErr — 取值 + *NilError 精确定位 nil 字段
token, err := SafeDigErr[string](req,
    F("Auth", func(r *Request) any { return r.Auth }),
    F("Key", func(a *Auth) any { return a.Key }),
)
// err: nil pointer at field "Key" in path "Auth.Key"
```

### Dig — 零分配高级版（unsafe.Pointer）

热路径场景下使用：

```go
import "unsafe"

val, err := safechain.Dig[string](req,
    safechain.S("Auth", func(r *Request) unsafe.Pointer { return unsafe.Pointer(r.Auth) }),
    safechain.S("Key", func(a *Auth) unsafe.Pointer { return unsafe.Pointer(a.Key) }),
    safechain.S("Session", func(k *Key) unsafe.Pointer { return unsafe.Pointer(k.Session) }),
    safechain.S("Token", func(s *Session) unsafe.Pointer { return unsafe.Pointer(s.Token) }),
)
```

### 在 And / Or 中组合使用

所有比较和匹配函数都返回 `bool`，可以直接传入 `And`/`Or`：

```go
ok := And(
    HasPrefix(func() string { return *req.Auth.Key.Session.Token }, "Bearer"),
    Gt(func() int { return *req.Auth.RetryCount }, 0),
    Check(func() { _ = *req.Meta.TraceID }),
)
```

### 转换 — Map / MustMap

```go
// 安全取值后做转换
upper, ok := Map(func() string { return *r.A.Name }, strings.ToUpper)
upper := MustMap(func() string { return *r.A.Name }, strings.ToUpper)
length := MustMap(func() string { return *r.A.Name }, func(s string) int { return len(s) })
```

### 集合 — In / NotIn

```go
In(func() string { return *r.A.Role }, "admin", "superadmin")
NotIn(func() string { return *r.A.Role }, "banned", "suspended")
```

### 零值 — IsZero / NotZero

```go
IsZero(func() string { return *r.A.Name })    // *string: "" → true, "abc" → false
NotZero(func() int { return r.A.Count })       // int（值类型）: 0 → false, 5 → true
```

### 长度 — Len / MustLen

```go
n, ok := Len(func() string { return *r.A.Name })
n := MustLen(func() []byte { return *r.A.Data })
```

### 副作用 — IfOk

```go
IfOk(func() string { return *r.A.Token }, func(token string) {
    fmt.Println("got token:", token)
})
```

### 错误 — SafeErr

```go
// 类似 Safe 但返回 error 而不是 bool
val, err := SafeErr(func() string { return *r.A.Name })
// err: "nil pointer dereference: runtime error: ..."
```

### 写入 — E / Ensure / Set / SetErr

无需逐层判空即可构建深层嵌套结构体。`E()`（`Ensure` 的简写）自动分配 nil 指针字段并返回值，支持一行链式赋值。

```go
// E()：一行链式 — 自动创建所有中间指针并赋值
var req Request
E(&E(&E(&req.Auth).Key).Session).Token = ptr("my_token")

// 值类型字段同样适用 — 链到父级直接赋值
E(&E(&req.Auth).Key).Name = "admin"     // string 字段
E(&E(&req.Auth).Key).Score = 100        // int 字段

// 拿到引用避免重复链路
key := E(&E(&req.Auth).Key)
key.Name = "admin"
key.Score = 100
key.Token = ptr("abc")

// Set：链式 + 赋值，带 panic 恢复，返回 bool
ok := Set(func() **string {
    return &E(&E(&E(&req.Auth).Key).Session).Token
}, ptr("my_token"))

// SetErr：类似 Set 但失败时返回 error
_, err := SetErr(func() **string {
    return &E(&E(&E(&req.Auth).Key).Session).Token
}, ptr("my_token"))
// err: "set failed: runtime error: invalid memory address..."
```

## 性能测试

测试环境：Apple M4（10 核），Go 1.22+。

### 单协程 — 各函数（4 层结构体）

| 函数 | ns/op | 分配 |
|------|-------|------|
| `Safe` | 7.0 | 0 |
| `Must` | 8.0 | 0 |
| `OrVal` | 8.3 | 0 |
| `Check` | 8.9 | 0 |
| `NotNil` | 8.2 | 0 |
| `Eq` | 9.1 | 0 |
| `Gt` | 7.8 | 0 |
| `Between` | 8.2 | 0 |
| `In`（3 个值） | 13 | 0 |
| `Match` | 9.4 | 0 |
| `HasPrefix` | 12 | 0 |
| `Contains` | 13 | 0 |
| `Map` | 9.5 | 0 |
| `First` | 9.7 | 0 |
| `SafeErr` | 7.0 | 0 |
| `And`（2 个检查） | 18 | 0 |
| `SafeDig`（4 层） | 141 | 5 |
| `MustSafeDig`（3 层） | 110 | 4 |
| `Dig`（4 层） | 45 | 1 |

### 并发 — 多协程吞吐量（10 goroutine）

| 函数 | ns/op | 分配 |
|------|-------|------|
| `Safe` | 2.2 | 0 |
| `Must` | 2.2 | 0 |
| `Check` | 2.5 | 0 |
| `Eq` | 2.3 | 0 |
| `HasPrefix` | 2.8 | 0 |
| `In` | 3.2 | 0 |
| `Dig`（4 层） | 43 | 1 |
| `SafeDig`（4 层） | 102 | 5 |

### 深度嵌套 — 100 层

| 方式 | 单协程 | 并发 | 分配 |
|------|--------|------|------|
| 手写 `if != nil` | 93 ns | 16 ns | 0 |
| `Safe`（recover） | 88 ns | 18 ns | 0 |
| `Dig`（unsafe.Pointer） | 589 ns | 718 ns | 1 |

- `Safe` 在任意深度下与手写 if 判断性能一致，零分配。
- 所有 recover 系列函数随协程数线性扩展，无竞争。
- `Dig`/`SafeDig` 因 `names` 切片有分配开销，但仍在亚微秒级。

## 并发安全

所有函数都是 **goroutine 安全** 的 — 没有共享状态，所有操作都在栈上完成。已通过 1000 goroutine 压测 + `-race` 检测器验证所有公开函数：

`Safe`, `Must`, `OrVal`, `Check`, `NotNil`, `And`, `Or`, `Eq`, `Gt`, `Between`, `In`, `Match`, `HasPrefix`, `Contains`, `Map`, `First`, `SafeErr`, `Dig`, `SafeDig`, `MustSafeDig`

唯一的要求是被访问的结构体不能被其他 goroutine 同时修改（这和手写 `if != nil` 是一样的）。

## 环境要求

- Go 1.22+

## 开源协议

[MIT](LICENSE)
