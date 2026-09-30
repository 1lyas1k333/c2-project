package executor

import (
	"context"
	"errors"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestExecute_SimpleCommand(t *testing.T) {
	e := New(5 * time.Second)

	out, err := e.Execute("echo hello")
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !strings.Contains(out, "hello") {
		t.Errorf("Execute() = %q, want contains 'hello'", out)
	}
}

func TestExecute_EmptyCommand(t *testing.T) {
	e := New(5 * time.Second)

	_, err := e.Execute("")
	if !errors.Is(err, ErrEmptyCommand) {
		t.Errorf("Execute(\"\") error = %v, want ErrEmptyCommand", err)
	}
}

func TestExecute_WhitespaceCommand(t *testing.T) {
	e := New(5 * time.Second)

	_, err := e.Execute("   \t\n  ")
	if !errors.Is(err, ErrEmptyCommand) {
		t.Errorf("Execute(whitespace) error = %v, want ErrEmptyCommand", err)
	}
}

func TestExecute_Timeout(t *testing.T) {
	e := New(500 * time.Millisecond)

	var cmd string
	if runtime.GOOS == "windows" {
		cmd = "ping -n 10 127.0.0.1 > nul"
	} else {
		cmd = "sleep 10"
	}

	start := time.Now()
	_, err := e.Execute(cmd)
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("Execute() expected timeout error")
	}
	if !errors.Is(err, ErrTimeout) {
		t.Errorf("Execute() error = %v, want ErrTimeout", err)
	}
	if elapsed > 3*time.Second {
		t.Errorf("Execute() took %v, want ~500ms (timeout didn't work)", elapsed)
	}
	t.Logf("timeout worked: elapsed = %v", elapsed)
}

func TestExecuteContext_Cancel(t *testing.T) {
	e := New(10 * time.Second)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(200 * time.Millisecond)
		cancel()
	}()

	var cmd string
	if runtime.GOOS == "windows" {
		cmd = "ping -n 10 127.0.0.1 > nul"
	} else {
		cmd = "sleep 10"
	}

	start := time.Now()
	_, err := e.ExecuteContext(ctx, cmd)
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("ExecuteContext() expected error on cancel")
	}
	if elapsed > 3*time.Second {
		t.Errorf("ExecuteContext() took %v, want ~200ms", elapsed)
	}
	t.Logf("cancel worked: elapsed = %v, err = %v", elapsed, err)
}

func TestExecute_CommandFails(t *testing.T) {
	e := New(5 * time.Second)

	var cmd string
	if runtime.GOOS == "windows" {
		cmd = "exit /b 1"
	} else {
		cmd = "exit 1"
	}

	_, err := e.Execute(cmd)
	if err == nil {
		t.Error("Execute() expected error on failing command")
	}
	if !errors.Is(err, ErrCommandFailed) {
		t.Errorf("Execute() error = %v, want ErrCommandFailed", err)
	}
}

func TestNew_DefaultTimeout(t *testing.T) {
	e := New(0)
	if e.timeout != DefaultTimeout {
		t.Errorf("New(0).timeout = %v, want %v", e.timeout, DefaultTimeout)
	}

	e = New(-5 * time.Second)
	if e.timeout != DefaultTimeout {
		t.Errorf("New(-5s).timeout = %v, want %v", e.timeout, DefaultTimeout)
	}
}

func TestHasPipeOperator(t *testing.T) {
	tests := []struct {
		cmd  string
		want bool
	}{
		{"ls -la", false},
		{"ls | grep go", true},
		{"ls || echo fail", false},
		{"a || b", false},
		{"a | b | c", true},
		{"", false},
	}

	for _, tc := range tests {
		t.Run(tc.cmd, func(t *testing.T) {
			got := hasPipeOperator(tc.cmd)
			if got != tc.want {
				t.Errorf("hasPipeOperator(%q) = %v, want %v", tc.cmd, got, tc.want)
			}
		})
	}
}
