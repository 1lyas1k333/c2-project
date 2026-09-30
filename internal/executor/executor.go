// Package executor предоставляет выполнение shell-команд с тайм-аутом и отменой.
package executor

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/transform"
)

// Ошибки пакета.
var (
	// ErrEmptyCommand - пустая команда.
	ErrEmptyCommand = errors.New("empty command")

	// ErrTimeout - команда превысила тайм-аут.
	ErrTimeout = errors.New("command timeout")

	// ErrCommandFailed - команда завершилась с ненулевым кодом.
	ErrCommandFailed = errors.New("command failed")
)

// DefaultTimeout - тайм-аут по умолчанию для одной команды.
const DefaultTimeout = 30 * time.Second

// MaxCompressSize - если вывод больше этого размера, его нужно сжать.
// (Сжатие применяется в вызывающем коде, здесь только ограничение.)
const MaxCompressSize = 5000

// Executor - выполняет команды с тайм-аутом.
type Executor struct {
	timeout time.Duration
}

// New - создаёт Executor с указанным тайм-аутом.
// Если timeout <= 0 — используется DefaultTimeout.
func New(timeout time.Duration) *Executor {
	if timeout <= 0 {
		timeout = DefaultTimeout
	}
	return &Executor{timeout: timeout}
}

// Execute - выполняет команду в shell с тайм-аутом.
// Возвращает объединённый stdout+stderr.
func (e *Executor) Execute(cmd string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), e.timeout)
	defer cancel()
	return e.ExecuteContext(ctx, cmd)
}

// ExecuteContext - выполняет команду с переданным контекстом.
// Если ctx отменён — команда прерывается.
func (e *Executor) ExecuteContext(ctx context.Context, cmd string) (string, error) {
	if len(strings.TrimSpace(cmd)) == 0 {
		return "", ErrEmptyCommand
	}

	var (
		output []byte
		err    error
	)

	if runtime.GOOS == "windows" {
		output, err = e.executeWindows(ctx, cmd)
	} else {
		output, err = e.executeUnix(ctx, cmd)
	}

	if err != nil {
		// Проверяем тайм-аут или отмену контекста.
		if ctx.Err() == context.DeadlineExceeded {
			return string(output), fmt.Errorf("%w: %v", ErrTimeout, ctx.Err())
		}
		if ctx.Err() == context.Canceled {
			return string(output), fmt.Errorf("%w: %v", ErrTimeout, ctx.Err())
		}
		// Если вывод пустой — возвращаем ошибку.
		if len(output) == 0 {
			return "", fmt.Errorf("%w: %v", ErrCommandFailed, err)
		}
	}

	return string(output), nil
}

// executeWindows - выполняет команду на Windows.
// Использует Start/Wait + select, чтобы при отмене контекста
// убить ДЕРЕВО процессов через taskkill /T.
func (e *Executor) executeWindows(ctx context.Context, cmd string) ([]byte, error) {
	hasPipe := hasPipeOperator(cmd)

	var execCmd *exec.Cmd
	if hasPipe {
		// Через PowerShell для поддержки пайпов.
		psCmd := strings.ReplaceAll(cmd, " && ", " ; ")
		psCmd = strings.ReplaceAll(psCmd, " || ", " ; ")
		execCmd = exec.CommandContext(ctx, "powershell", "-Command", psCmd)
	} else {
		execCmd = exec.CommandContext(ctx, "cmd", "/c", cmd)
	}

	// Перенаправляем stdout+stderr в один буфер.
	var buf bytes.Buffer
	execCmd.Stdout = &buf
	execCmd.Stderr = &buf

	if err := execCmd.Start(); err != nil {
		return nil, err
	}

	// Ждём завершения в отдельной горутине.
	done := make(chan error, 1)
	go func() {
		done <- execCmd.Wait()
	}()

	select {
	case <-ctx.Done():
		// Контекст отменён — пытаемся убить дерево процессов.
		if execCmd.Process != nil {
			_ = exec.Command("taskkill", "/F", "/T", "/PID",
				fmt.Sprintf("%d", execCmd.Process.Pid)).Run()
		}
		// ВАЖНО: НЕ ждём <-done. Возвращаемся сразу.
		return e.decodeWindowsOutput(buf.Bytes()), ctx.Err()

	case err := <-done:
		return e.decodeWindowsOutput(buf.Bytes()), err
	}
}

// executeUnix - выполняет команду на Unix (Linux, macOS).
// На Unix exec.CommandContext с SIGKILL убивает сам процесс;
// дочерние процессы могут остаться, но stdout-пайп закрывается.
func (e *Executor) executeUnix(ctx context.Context, cmd string) ([]byte, error) {
	execCmd := exec.CommandContext(ctx, "sh", "-c", cmd)

	var buf bytes.Buffer
	execCmd.Stdout = &buf
	execCmd.Stderr = &buf

	if err := execCmd.Start(); err != nil {
		return nil, err
	}

	done := make(chan error, 1)
	go func() {
		done <- execCmd.Wait()
	}()

	select {
	case <-ctx.Done():
		return buf.Bytes(), ctx.Err() // ← тоже без <-done
	case err := <-done:
		return buf.Bytes(), err
	}
}

// decodeWindowsOutput - конвертирует CP866 → UTF-8 и заменяет мусорные символы.
func (e *Executor) decodeWindowsOutput(output []byte) []byte {
	if len(output) == 0 {
		return output
	}
	decoder := charmap.CodePage866.NewDecoder()
	if utf8Output, _, convErr := transform.Bytes(decoder, output); convErr == nil {
		output = utf8Output
	}
	return bytes.ReplaceAll(output, []byte("?"), []byte(" "))
}

// hasPipeOperator - проверяет, содержит ли команда пайп (|), но не ||.
func hasPipeOperator(cmd string) bool {
	for i := 0; i < len(cmd); i++ {
		if cmd[i] != '|' {
			continue
		}
		if i+1 < len(cmd) && cmd[i+1] == '|' {
			continue
		}
		if i > 0 && cmd[i-1] == '|' {
			continue
		}
		return true
	}
	return false
}
