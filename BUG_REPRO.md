# 离线私聊消息重连丢失复现

## Bug 是什么

当接收方在重连期间的 WebSocket 写缓冲已满时，离线私聊消息会被当作已经交付而从待发送队列删除。再次重连后，该消息不再出现；在存在后续待发送消息时，重新入队的位置还可能改变消息顺序。

## 如何触发

在 Go 1.21 环境进入项目目录后执行：

```bash
cd /workplace/we-chat__002
go test -race -count=20 ./internal/websocket -run 'TestOfflineDelivery'
```

## 错误信息

基线会稳定报出如下断言失败：

```text
ready queue after backpressure = []*websocket.offlineMessageClaim{}, want only second
```
