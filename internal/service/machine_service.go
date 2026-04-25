package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"traffic-monitor/internal/config"
	"traffic-monitor/internal/dto"
	"traffic-monitor/internal/model"
	"traffic-monitor/internal/repo"
)

var (
	ErrMachineNotFound       = errors.New("machine not found")
	ErrSSHKeyNotFound        = errors.New("ssh key not found")
	ErrInvalidMachineConfig  = errors.New("invalid machine config")
	ErrVNStatUnavailable     = errors.New("vnstat unavailable")
	ErrSSHKeyDecryptFailed   = errors.New("ssh key decrypt failed")
)

type MachineStore interface {
	Create(ctx context.Context, machine *model.Machine) error
	List(ctx context.Context) ([]model.Machine, error)
	GetByID(ctx context.Context, machineID uint) (*model.Machine, error)
	Update(ctx context.Context, machine *model.Machine) error
	DeleteByID(ctx context.Context, machineID uint) error
}

type SSHKeyLookup interface {
	GetByID(ctx context.Context, sshKeyID uint) (*model.SSHKey, error)
}

type SSHCommandRunner interface {
	Run(ctx context.Context, host string, port int, sshUser string, privateKeyPEM []byte, command string) (SSHExecutionResult, error)
}

type SSHExecutionResult struct {
	Stdout string
	Stderr string
}

type MachineService struct {
	machineStore  MachineStore
	sshKeyStore   SSHKeyLookup
	dataProtector SSHKeyProtector
	sshRunner     SSHCommandRunner
	sshConfig     config.SSHConfig
}

func NewMachineService(machineStore *repo.MachineRepo, sshKeyStore *repo.SSHKeyRepo, dataProtector SSHKeyProtector, sshRunner SSHCommandRunner, sshConfig config.SSHConfig) *MachineService {
	return &MachineService{
		machineStore:  machineStore,
		sshKeyStore:   sshKeyStore,
		dataProtector: dataProtector,
		sshRunner:     sshRunner,
		sshConfig:     sshConfig,
	}
}

func (service *MachineService) Create(ctx context.Context, req dto.CreateMachineReq) (dto.MachineResp, error) {
	machine := &model.Machine{
		Name:             strings.TrimSpace(req.Name),
		Host:             strings.TrimSpace(req.Host),
		Port:             req.Port,
		SSHUser:          strings.TrimSpace(req.SSHUser),
		NetworkInterface: strings.TrimSpace(req.NetworkInterface),
		SSHKeyID:         req.SSHKeyID,
		CollectEnabled:   req.CollectEnabled,
		Remark:           strings.TrimSpace(req.Remark),
	}

	if err := service.validateMachine(ctx, machine); err != nil {
		return dto.MachineResp{}, err
	}

	if err := service.machineStore.Create(ctx, machine); err != nil {
		return dto.MachineResp{}, fmt.Errorf("create machine: %w", err)
	}

	return toMachineResp(machine), nil
}

func (service *MachineService) List(ctx context.Context) ([]dto.MachineResp, error) {
	machines, err := service.machineStore.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list machines: %w", err)
	}

	result := make([]dto.MachineResp, 0, len(machines))
	for _, machine := range machines {
		result = append(result, toMachineResp(&machine))
	}

	return result, nil
}

func (service *MachineService) Get(ctx context.Context, machineID uint) (dto.MachineResp, error) {
	machine, err := service.machineStore.GetByID(ctx, machineID)
	if err != nil {
		if repo.IsRecordNotFound(err) {
			return dto.MachineResp{}, ErrMachineNotFound
		}

		return dto.MachineResp{}, fmt.Errorf("get machine: %w", err)
	}

	return toMachineResp(machine), nil
}

func (service *MachineService) Update(ctx context.Context, machineID uint, req dto.UpdateMachineReq) (dto.MachineResp, error) {
	machine, err := service.machineStore.GetByID(ctx, machineID)
	if err != nil {
		if repo.IsRecordNotFound(err) {
			return dto.MachineResp{}, ErrMachineNotFound
		}

		return dto.MachineResp{}, fmt.Errorf("get machine for update: %w", err)
	}

	applyMachineUpdate(machine, req)
	if err := service.validateMachine(ctx, machine); err != nil {
		return dto.MachineResp{}, err
	}

	if err := service.machineStore.Update(ctx, machine); err != nil {
		return dto.MachineResp{}, fmt.Errorf("update machine: %w", err)
	}

	return toMachineResp(machine), nil
}

func (service *MachineService) Delete(ctx context.Context, machineID uint) error {
	if _, err := service.machineStore.GetByID(ctx, machineID); err != nil {
		if repo.IsRecordNotFound(err) {
			return ErrMachineNotFound
		}

		return fmt.Errorf("get machine for delete: %w", err)
	}

	if err := service.machineStore.DeleteByID(ctx, machineID); err != nil {
		return fmt.Errorf("delete machine: %w", err)
	}

	return nil
}

func (service *MachineService) TestConnection(ctx context.Context, machineID uint) (dto.MachineConnectionTestResp, error) {
	machine, err := service.machineStore.GetByID(ctx, machineID)
	if err != nil {
		if repo.IsRecordNotFound(err) {
			return dto.MachineConnectionTestResp{}, ErrMachineNotFound
		}

		return dto.MachineConnectionTestResp{}, fmt.Errorf("get machine for connection test: %w", err)
	}

	sshKey, err := service.sshKeyStore.GetByID(ctx, machine.SSHKeyID)
	if err != nil {
		if repo.IsRecordNotFound(err) {
			return dto.MachineConnectionTestResp{}, ErrSSHKeyNotFound
		}

		return dto.MachineConnectionTestResp{}, fmt.Errorf("get ssh key for connection test: %w", err)
	}

	privateKeyPEM, err := service.dataProtector.Decrypt(sshKey.PrivateKeyCiphertext)
	if err != nil {
		return dto.MachineConnectionTestResp{}, fmt.Errorf("%w: current APP_MASTER_KEY may not match the key used when this SSH key was imported", ErrSSHKeyDecryptFailed)
	}

	commandContext, cancel := context.WithTimeout(ctx, service.sshConfig.CommandTimeout)
	defer cancel()

	result, err := service.sshRunner.Run(commandContext, machine.Host, machine.Port, machine.SSHUser, privateKeyPEM, machineVNStatCheckCommand())
	if err != nil {
		response := dto.MachineConnectionTestResp{
			MachineID:     machine.ID,
			SSHReachable:  false,
			VNStatReady:   false,
			VNStatVersion: "",
			Status:        "ssh_failed",
		}

		if strings.TrimSpace(result.Stdout) != "" {
			response.SSHReachable = true
		}

		if strings.Contains(strings.ToLower(result.Stdout), "vnstat") {
			response.VNStatReady = true
			response.Status = "ok"
			response.VNStatVersion = strings.TrimSpace(result.Stdout)
			return response, nil
		}

		if strings.Contains(err.Error(), "run ssh command") {
			response.SSHReachable = true
			response.Status = "vnstat_unavailable"
			return response, ErrVNStatUnavailable
		}

		return response, err
	}

	return dto.MachineConnectionTestResp{
		MachineID:     machine.ID,
		SSHReachable:  true,
		VNStatReady:   true,
		VNStatVersion: strings.TrimSpace(result.Stdout),
		Status:        "ok",
	}, nil
}

func (service *MachineService) validateMachine(ctx context.Context, machine *model.Machine) error {
	if machine.Name == "" || machine.Host == "" || machine.SSHUser == "" || machine.NetworkInterface == "" {
		return ErrInvalidMachineConfig
	}

	if machine.Port <= 0 || machine.Port > 65535 {
		return ErrInvalidMachineConfig
	}

	if machine.SSHKeyID == 0 {
		return ErrInvalidMachineConfig
	}

	if _, err := service.sshKeyStore.GetByID(ctx, machine.SSHKeyID); err != nil {
		if repo.IsRecordNotFound(err) {
			return ErrSSHKeyNotFound
		}

		return fmt.Errorf("validate ssh key: %w", err)
	}

	return nil
}

func applyMachineUpdate(machine *model.Machine, req dto.UpdateMachineReq) {
	if req.Name != nil {
		machine.Name = strings.TrimSpace(*req.Name)
	}

	if req.Host != nil {
		machine.Host = strings.TrimSpace(*req.Host)
	}

	if req.Port != nil {
		machine.Port = *req.Port
	}

	if req.SSHUser != nil {
		machine.SSHUser = strings.TrimSpace(*req.SSHUser)
	}

	if req.NetworkInterface != nil {
		machine.NetworkInterface = strings.TrimSpace(*req.NetworkInterface)
	}

	if req.SSHKeyID != nil {
		machine.SSHKeyID = *req.SSHKeyID
	}

	if req.CollectEnabled != nil {
		machine.CollectEnabled = *req.CollectEnabled
	}

	if req.Remark != nil {
		machine.Remark = strings.TrimSpace(*req.Remark)
	}
}

func toMachineResp(machine *model.Machine) dto.MachineResp {
	return dto.MachineResp{
		ID:               machine.ID,
		Name:             machine.Name,
		Host:             machine.Host,
		Port:             machine.Port,
		SSHUser:          machine.SSHUser,
		NetworkInterface: machine.NetworkInterface,
		SSHKeyID:         machine.SSHKeyID,
		CollectEnabled:   machine.CollectEnabled,
		Remark:           machine.Remark,
		CreatedAt:        machine.CreatedAt,
		UpdatedAt:        machine.UpdatedAt,
	}
}

func machineVNStatCheckCommand() string {
	return "command -v vnstat >/dev/null 2>&1 && vnstat --version"
}
