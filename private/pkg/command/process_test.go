package command

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockCmdCalls struct {
	startErr error
	exit     chan error
	killed   bool
}

func newMockCmdCalls() *mockCmdCalls {
	return &mockCmdCalls{
		exit: make(chan error),
	}
}

func (m *mockCmdCalls) Start() error {
	return m.startErr
}

func (m *mockCmdCalls) Wait() error {
	return <-m.exit
}

func (m *mockCmdCalls) Kill() error {
	m.killed = true
	return nil
}

func TestProcessWait(t *testing.T) {
	t.Parallel()
	cbCalled := make(chan struct{})
	done := func() {
		cbCalled <- struct{}{}
		return
	}
	calls := newMockCmdCalls()
	proc := newProcess(calls, done)
	require.NoError(t, proc.Start())
	// Exit the call and wait on the callback.
	calls.exit <- nil
	<-cbCalled
	// Pick up with a Wait.
	err := proc.Wait(context.Background())
	assert.NoError(t, err)
}

func TestProcessExitBeforeWait(t *testing.T) {
	t.Parallel()
	cbCalled := make(chan struct{})
	done := func() {
		cbCalled <- struct{}{}
		return
	}
	calls := newMockCmdCalls()
	// Throw away the process so we cannot Wait on it.
	proc := newProcess(calls, done)
	require.NoError(t, proc.Start())
	proc = nil
	// Exit and wait for the callback to be called.
	calls.exit <- nil
	timer := time.NewTimer(5 * time.Second)
	select {
	case <-cbCalled:
	case <-timer.C:
		t.Fatal("timed out waiting for the process exit callback")
	}
}

func TestProcessDoubleWaitWithError(t *testing.T) {
	t.Parallel()
	cbCalled := make(chan struct{})
	done := func() {
		cbCalled <- struct{}{}
		return
	}
	calls := newMockCmdCalls()
	proc := newProcess(calls, done)
	require.NoError(t, proc.Start())
	// Exit the call and wait on the callback.
	expectedErr := errors.New("its the end of the world")
	calls.exit <- expectedErr
	<-cbCalled
	// Pick up with a Wait.
	err := proc.Wait(context.Background())
	assert.ErrorIs(t, err, expectedErr)
	// The next error should be a permanent error.
	err = proc.Wait(context.Background())
	assert.ErrorIs(t, err, errWaitAlreadyCalled)
}

func TestProcessWaitTimeout(t *testing.T) {
	t.Parallel()
	cbCalled := make(chan struct{})
	done := func() {
		cbCalled <- struct{}{}
		return
	}
	calls := newMockCmdCalls()
	proc := newProcess(calls, done)
	require.NoError(t, proc.Start())
	// Exit the call without error but don't unblock the callback.
	calls.exit <- nil
	// Pick up with a Wait that immediately times out.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := proc.Wait(ctx)
	assert.Error(t, err)
	assert.True(t, calls.killed)
}

func TestProcessFailedStart(t *testing.T) {
	t.Parallel()
	calls := newMockCmdCalls()
	calls.startErr = errors.New("not an executable")
	proc := newProcess(calls, func() {})
	assert.Error(t, proc.Start())
}
