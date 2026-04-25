package repo

import (
	"context"
	"fmt"

	"traffic-monitor/internal/model"

	"gorm.io/gorm"
)

type SSHKeyRepo struct {
	db *gorm.DB
}

func NewSSHKeyRepo(db *gorm.DB) *SSHKeyRepo {
	return &SSHKeyRepo{db: db}
}

func (repo *SSHKeyRepo) Create(ctx context.Context, sshKey *model.SSHKey) error {
	if err := repo.db.WithContext(ctx).Create(sshKey).Error; err != nil {
		return fmt.Errorf("create ssh key: %w", err)
	}

	return nil
}

func (repo *SSHKeyRepo) List(ctx context.Context) ([]model.SSHKey, error) {
	var sshKeys []model.SSHKey
	if err := repo.db.WithContext(ctx).Order("id desc").Find(&sshKeys).Error; err != nil {
		return nil, fmt.Errorf("list ssh keys: %w", err)
	}

	return sshKeys, nil
}

func (repo *SSHKeyRepo) GetByID(ctx context.Context, sshKeyID uint) (*model.SSHKey, error) {
	var sshKey model.SSHKey
	if err := repo.db.WithContext(ctx).Where("id = ?", sshKeyID).First(&sshKey).Error; err != nil {
		return nil, fmt.Errorf("get ssh key by id: %w", err)
	}

	return &sshKey, nil
}

func (repo *SSHKeyRepo) UpdateName(ctx context.Context, sshKeyID uint, name string) (*model.SSHKey, error) {
	if err := repo.db.WithContext(ctx).
		Model(&model.SSHKey{}).
		Where("id = ?", sshKeyID).
		Update("name", name).Error; err != nil {
		return nil, fmt.Errorf("update ssh key name: %w", err)
	}

	sshKey, err := repo.GetByID(ctx, sshKeyID)
	if err != nil {
		return nil, fmt.Errorf("get ssh key after rename: %w", err)
	}

	return sshKey, nil
}

func (repo *SSHKeyRepo) DeleteByID(ctx context.Context, sshKeyID uint) error {
	if err := repo.db.WithContext(ctx).Delete(&model.SSHKey{}, sshKeyID).Error; err != nil {
		return fmt.Errorf("delete ssh key: %w", err)
	}

	return nil
}
