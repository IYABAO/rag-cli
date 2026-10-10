# Go context 笔记

context.Context 是 Go 并发控制的核心。它用来在 API 边界之间传递截止时间、取消信号和请求范围的值。

## 基本用法

最常见的是 context.WithCancel。当你需要取消一个 goroutine 时，调用 cancel 函数，所有监听该 context 的 goroutine 都会收到信号。

```go
ctx, cancel := context.WithCancel(context.Background())
defer cancel()
go worker(ctx)
```

## 超时控制

context.WithTimeout 用于设置操作超时。比如 HTTP 请求、数据库查询，超过时间自动取消。

```go
ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
defer cancel()
```

## 值传递

context.WithValue 可以传递请求范围的数据，比如 request ID、认证 token。但不要用它传业务参数。

## 常见坑

第一，不要把 context 存在 struct 里，应该显式作为函数第一个参数传递。

第二，cancel 函数必须调用，否则会泄漏 goroutine。即使 context 超时了，也要 defer cancel()。

第三，context.Background() 只在 main 或测试里用，其他地方用 WithCancel 或 WithTimeout 派生。
