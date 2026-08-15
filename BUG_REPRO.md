# 多标签页 WebSocket 消息投递异常

## Bug 是什么

同一个用户在两个浏览器标签页建立 WebSocket 连接后，连接注册表只保留最后建立的标签页。消息不能同时投递给两个标签页；关闭标签页时也无法正确判断该用户是否仍有活跃连接。

## 如何触发

在 bug 分支根目录执行：

```bash
go test ./internal/websocket -run TestConnectionRegistryKeepsEveryTabRoutableUntilItCloses
```

## 错误信息

```text
--- FAIL: TestConnectionRegistryKeepsEveryTabRoutableUntilItCloses
    connection_registry_test.go:17: active tabs = 1, want 2
FAIL
```
