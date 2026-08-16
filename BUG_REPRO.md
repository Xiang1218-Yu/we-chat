# JWT 算法策略未生效复现

## Bug 是什么

服务默认使用 HS256 签发 JWT，但验证阶段没有限制令牌头中的签名算法。只要攻击者持有同一个 HMAC 密钥，使用另一种 HMAC 算法签发的令牌也会被解析为有效令牌。

## 如何触发

```bash
cd /workplace/we-chat__003
go test -count=20 ./pkg/jwt -run TestParseTokenRejectsUnexpectedHMACAlgorithm
```

## 错误信息

基线会稳定报出：

```text
HS384 token was accepted although this service is configured for HS256
```
