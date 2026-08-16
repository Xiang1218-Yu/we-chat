# 图片上传会接受伪造内容类型

## 问题
`/api/upload/image` 只检查 multipart 请求头中的 `Content-Type`。客户端可以把任意文本伪装为 `image/png`，服务端会返回成功并写入上传目录。

## 触发方式
发送一个名为 `avatar.png` 的 multipart 文件，声明其内容类型为 `image/png`，但内容为普通文本或脚本。

## 错误现象
接口返回 HTTP 200，响应包含图片 URL，并且伪造文件被保存到 `uploads/images`。
