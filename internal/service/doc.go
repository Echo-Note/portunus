// Package service 提供业务逻辑层
//
// 此层是框架无关的，不依赖任何 Web 框架类型。
// 所有方法接收 context.Context 和普通结构体参数，
// 可被 REST Handler、MCP Tool、Eino AI Agent 三方调用。
//
// 设计原则：
//   - 依赖接口而非具体实现（通过构造函数注入 *ent.Client）
//   - 不 import "github.com/gin-gonic/gin"
//   - 错误使用 fmt.Errorf 包装上下文，不吞掉底层错误
//   - 所有 IO 操作通过传入的 context.Context 控制超时和取消
package service
