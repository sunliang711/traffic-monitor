package repo

import (
	"context"
	"fmt"

	"traffic-monitor/internal/model"

	"gorm.io/gorm"
)

type MachineRepo struct {
	db *gorm.DB
}

func NewMachineRepo(db *gorm.DB) *MachineRepo {
	return &MachineRepo{db: db}
}

func (repo *MachineRepo) Create(ctx context.Context, machine *model.Machine) error {
	if err := repo.db.WithContext(ctx).Create(machine).Error; err != nil {
		return fmt.Errorf("create machine: %w", err)
	}

	return nil
}

func (repo *MachineRepo) List(ctx context.Context) ([]model.Machine, error) {
	var machines []model.Machine
	if err := repo.db.WithContext(ctx).Order("id desc").Find(&machines).Error; err != nil {
		return nil, fmt.Errorf("list machines: %w", err)
	}

	return machines, nil
}

func (repo *MachineRepo) GetByID(ctx context.Context, machineID uint) (*model.Machine, error) {
	var machine model.Machine
	if err := repo.db.WithContext(ctx).Where("id = ?", machineID).First(&machine).Error; err != nil {
		return nil, fmt.Errorf("get machine by id: %w", err)
	}

	return &machine, nil
}

func (repo *MachineRepo) Update(ctx context.Context, machine *model.Machine) error {
	if err := repo.db.WithContext(ctx).Save(machine).Error; err != nil {
		return fmt.Errorf("update machine: %w", err)
	}

	return nil
}

func (repo *MachineRepo) DeleteByID(ctx context.Context, machineID uint) error {
	if err := repo.db.WithContext(ctx).Delete(&model.Machine{}, machineID).Error; err != nil {
		return fmt.Errorf("delete machine: %w", err)
	}

	return nil
}
