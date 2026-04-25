package bootstrap

import (
	"context"
	"fmt"
	"net"

	"traffic-monitor/internal/config"
	"traffic-monitor/internal/service"

	"golang.org/x/crypto/ssh"
)

type SSHRunner struct {
	cfg config.SSHConfig
}

func NewSSHRunner(cfg config.SSHConfig) *SSHRunner {
	return &SSHRunner{cfg: cfg}
}

func (runner *SSHRunner) Run(ctx context.Context, host string, port int, sshUser string, privateKeyPEM []byte, command string) (service.SSHExecutionResult, error) {
	signer, err := ssh.ParsePrivateKey(privateKeyPEM)
	if err != nil {
		return service.SSHExecutionResult{}, fmt.Errorf("parse ssh private key: %w", err)
	}

	clientConfig := &ssh.ClientConfig{
		User:            sshUser,
		Auth:            []ssh.AuthMethod{ssh.PublicKeys(signer)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         runner.cfg.DialTimeout,
	}

	address := net.JoinHostPort(host, fmt.Sprintf("%d", port))
	dialer := &net.Dialer{Timeout: runner.cfg.DialTimeout}
	conn, err := dialer.DialContext(ctx, "tcp", address)
	if err != nil {
		return service.SSHExecutionResult{}, fmt.Errorf("dial ssh: %w", err)
	}

	sshConn, channels, requests, err := ssh.NewClientConn(conn, address, clientConfig)
	if err != nil {
		_ = conn.Close()
		return service.SSHExecutionResult{}, fmt.Errorf("create ssh client connection: %w", err)
	}

	client := ssh.NewClient(sshConn, channels, requests)
	defer func() {
		_ = client.Close()
	}()

	session, err := client.NewSession()
	if err != nil {
		return service.SSHExecutionResult{}, fmt.Errorf("create ssh session: %w", err)
	}
	defer func() {
		_ = session.Close()
	}()

	type result struct {
		output []byte
		err    error
	}

	done := make(chan result, 1)
	go func() {
		output, runErr := session.CombinedOutput(command)
		done <- result{output: output, err: runErr}
	}()

	select {
	case <-ctx.Done():
		_ = client.Close()
		return service.SSHExecutionResult{}, fmt.Errorf("ssh command context done: %w", ctx.Err())
	case runResult := <-done:
		if runResult.err != nil {
			return service.SSHExecutionResult{
				Stdout: string(runResult.output),
				Stderr: string(runResult.output),
			}, fmt.Errorf("run ssh command: %w", runResult.err)
		}

		output := string(runResult.output)
		return service.SSHExecutionResult{
			Stdout: output,
			Stderr: "",
		}, nil
	}
}
