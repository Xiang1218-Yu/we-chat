# 带凭据的跨域请求被浏览器拦截

## 问题
CORS 中间件同时返回 `Access-Control-Allow-Credentials: true` 和通配符 `Access-Control-Allow-Origin: *`。浏览器不接受这种组合，因此带凭据的跨域请求不能读取响应。

## 触发方式
使用 Origin 为 `https://chat.example.test` 的请求访问需要凭据的接口。

## 错误现象
响应的 `Access-Control-Allow-Origin` 为 `*`，与带凭据请求不兼容，浏览器会阻止前端读取响应。
