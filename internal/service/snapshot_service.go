package service

import (
	"context"
	"fmt"
	"strconv"

	"github.com/google/uuid"

	"github.com/Echo-Note/portunus/ent/generated"
	"github.com/Echo-Note/portunus/ent/generated/configsnapshot"
)

// SnapshotService 处理配置快照和回滚相关的业务逻辑。
// 阶段 2 将实现完整的快照创建、列表和回滚功能。
type SnapshotService struct {
	client *generated.Client
}

// NewSnapshotService 创建快照服务实例。
func NewSnapshotService(client *generated.Client) *SnapshotService {
	return &SnapshotService{client: client}
}

// ListSnapshots 列出项目的配置快照。
func (s *SnapshotService) ListSnapshots(ctx context.Context, projectID uuid.UUID, limit, offset int) ([]*generated.ConfigSnapshot, error) {
	if limit <= 0 {
		limit = 50
	}
	snapshots, err := s.client.ConfigSnapshot.Query().
		Where(configsnapshot.ProjectIDEQ(projectID)).
		Order(generated.Desc(configsnapshot.FieldCreatedAt)).
		Limit(limit).
		Offset(offset).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("查询快照列表: %w", err)
	}
	return snapshots, nil
}

// RollbackSnapshot 回滚到指定版本的快照。
// 阶段 2 将实现完整的回滚逻辑（通过 POST /load 恢复 Caddy 配置）。
func (s *SnapshotService) RollbackSnapshot(ctx context.Context, projectID uuid.UUID, version string) error {
	versionInt, err := strconv.Atoi(version)
	if err != nil {
		return fmt.Errorf("%w: 无效的版本号 %s", ErrValidation, version)
	}

	snapshot, err := s.client.ConfigSnapshot.Query().
		Where(configsnapshot.ProjectIDEQ(projectID), configsnapshot.VersionEQ(versionInt)).
		Only(ctx)
	if err != nil {
		if generated.IsNotFound(err) {
			return fmt.Errorf("%w: 快照版本 %s 不存在", ErrNotFound, version)
		}
		return fmt.Errorf("查询快照: %w", err)
	}

	_ = snapshot
	// 阶段 2 实现：通过 POST /load 将 snapshot.CaddyJSON 恢复到 Caddy
	return fmt.Errorf("%w: 快照回滚功能将在阶段 2 实现", ErrValidation)
}
