package service

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	"github.com/Echo-Note/portunus/ent/generated"
	"github.com/Echo-Note/portunus/ent/generated/projectauditlog"
)

// AuditService 处理审计日志相关的业务逻辑。
// 审计日志为只追加（append-only），不提供修改和删除操作。
type AuditService struct {
	client *generated.Client
}

// NewAuditService 创建审计服务实例。
func NewAuditService(client *generated.Client) *AuditService {
	return &AuditService{client: client}
}

// CreateAuditLogInput 创建审计日志输入参数。
type CreateAuditLogInput struct {
	ProjectID      uuid.UUID                 `json:"project_id"`      // 必填，所属项目 ID
	ActorType      projectauditlog.ActorType `json:"actor_type"`      // 必填，操作者类型
	ActorID        uuid.UUID                 `json:"actor_id"`        // 可选，操作者 ID
	ActorName      string                    `json:"actor_name"`      // 可选，操作者名称
	ActorIP        string                    `json:"actor_ip"`        // 可选，操作者 IP
	Action         string                    `json:"action"`          // 必填，操作类型
	ResourceType   string                    `json:"resource_type"`   // 必填，资源类型
	ResourceID     string                    `json:"resource_id"`     // 可选，资源 ID
	ChangesBefore  map[string]any            `json:"changes_before"`  // 可选，变更前数据
	ChangesAfter   map[string]any            `json:"changes_after"`   // 可选，变更后数据
	Result         projectauditlog.Result    `json:"result"`          // 必填，操作结果
	ErrorMessage   string                    `json:"error_message"`   // 可选，错误信息
	Via            projectauditlog.Via       `json:"via"`             // 必填，调用来源
	McpToolName    string                    `json:"mcp_tool_name"`   // 可选，MCP 工具名称
	UserAgent      string                    `json:"user_agent"`      // 可选，User-Agent
	RequestBody    map[string]any            `json:"request_body"`    // 可选，请求体
	ResponseStatus int                       `json:"response_status"` // 可选，HTTP 响应状态码
	RequestID      string                    `json:"request_id"`      // 可选，请求 ID
	CorrelationID  string                    `json:"correlation_id"`  // 可选，关联 ID
}

// Log 写入一条审计日志。
func (s *AuditService) Log(ctx context.Context, input CreateAuditLogInput) (*generated.ProjectAuditLog, error) {
	if input.ProjectID == uuid.Nil {
		return nil, fmt.Errorf("%w: 项目 ID 不能为空", ErrValidation)
	}
	if input.Action == "" || input.ResourceType == "" {
		return nil, fmt.Errorf("%w: action、resource_type 不能为空", ErrValidation)
	}

	builder := s.client.ProjectAuditLog.Create().
		SetProjectID(input.ProjectID).
		SetActorType(input.ActorType).
		SetAction(input.Action).
		SetResourceType(input.ResourceType).
		SetResult(input.Result).
		SetVia(input.Via)

	if input.ActorID != uuid.Nil {
		builder.SetActorID(input.ActorID)
	}
	if input.ActorName != "" {
		builder.SetActorName(input.ActorName)
	}
	if input.ActorIP != "" {
		builder.SetActorIP(input.ActorIP)
	}
	if input.ResourceID != "" {
		builder.SetResourceID(input.ResourceID)
	}
	if input.ChangesBefore != nil {
		builder.SetChangesBefore(input.ChangesBefore)
	}
	if input.ChangesAfter != nil {
		builder.SetChangesAfter(input.ChangesAfter)
	}
	if input.ErrorMessage != "" {
		builder.SetErrorMessage(input.ErrorMessage)
	}
	if input.McpToolName != "" {
		builder.SetMcpToolName(input.McpToolName)
	}
	if input.UserAgent != "" {
		builder.SetUserAgent(input.UserAgent)
	}
	if input.RequestBody != nil {
		builder.SetRequestBody(input.RequestBody)
	}
	if input.ResponseStatus != 0 {
		builder.SetResponseStatus(input.ResponseStatus)
	}
	if input.RequestID != "" {
		builder.SetRequestID(input.RequestID)
	}
	if input.CorrelationID != "" {
		builder.SetCorrelationID(input.CorrelationID)
	}

	log, err := builder.Save(ctx)
	if err != nil {
		// 审计日志写入失败不应阻塞主流程，仅记录警告
		slog.WarnContext(ctx, "审计日志写入失败",
			"action", input.Action,
			"resource_type", input.ResourceType,
			"err", err,
		)
		return nil, fmt.Errorf("写入审计日志: %w", err)
	}

	return log, nil
}

// QueryInput 审计日志查询参数。
type QueryInput struct {
	ProjectID    uuid.UUID `json:"project_id"`    // 必填，项目 ID
	Action       string    `json:"action"`        // 可选，按操作类型筛选
	ResourceType string    `json:"resource_type"` // 可选，按资源类型筛选
	ResourceID   string    `json:"resource_id"`   // 可选，按资源 ID 筛选
	ActorID      uuid.UUID `json:"actor_id"`      // 可选，按操作者筛选
	Limit        int       `json:"limit"`         // 可选，每页条数，默认 50
	Offset       int       `json:"offset"`        // 可选，分页偏移
}

// Query 查询审计日志。
func (s *AuditService) Query(ctx context.Context, input QueryInput) ([]*generated.ProjectAuditLog, error) {
	if input.ProjectID == uuid.Nil {
		return nil, fmt.Errorf("%w: 项目 ID 不能为空", ErrValidation)
	}

	query := s.client.ProjectAuditLog.Query().
		Where(projectauditlog.ProjectIDEQ(input.ProjectID))

	if input.Action != "" {
		query = query.Where(projectauditlog.ActionEQ(input.Action))
	}
	if input.ResourceType != "" {
		query = query.Where(projectauditlog.ResourceTypeEQ(input.ResourceType))
	}
	if input.ResourceID != "" {
		query = query.Where(projectauditlog.ResourceIDEQ(input.ResourceID))
	}
	if input.ActorID != uuid.Nil {
		query = query.Where(projectauditlog.ActorIDEQ(input.ActorID))
	}

	limit := input.Limit
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	offset := input.Offset
	if offset < 0 {
		offset = 0
	}

	logs, err := query.
		Order(generated.Desc(projectauditlog.FieldCreatedAt)).
		Limit(limit).
		Offset(offset).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("查询审计日志: %w", err)
	}

	return logs, nil
}
