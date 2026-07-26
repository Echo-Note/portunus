// Package api 提供 HTTP 接入层
//
// 包含路由注册、Handler 实现、中间件和请求/响应 DTO。
// 此层严格遵守"薄 Handler"原则：
//   - 只负责参数绑定、上下文提取、错误映射和响应序列化
//   - 绝不将 gin.Context 传入 Service 层
//   - 绝不在 Handler 中编写核心业务逻辑
package api
