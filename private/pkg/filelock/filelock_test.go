// Copyright 2020-2023 Buf Technologies, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package filelock

import (
	"context"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestGlobalBasic(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	tempDirPath := t.TempDir()
	filePath := filepath.Join(tempDirPath, "path/to/lock")
	unlocker, err := Lock(ctx, filePath)
	require.NoError(t, err)
	_, err = Lock(ctx, filePath, LockWithTimeout(100*time.Millisecond), LockWithRetryDelay(10*time.Millisecond))
	require.Error(t, err)
	require.NoError(t, unlocker.Unlock())
	unlocker, err = Lock(ctx, filePath, LockWithTimeout(100*time.Millisecond), LockWithRetryDelay(10*time.Millisecond))
	require.NoError(t, err)
	require.NoError(t, unlocker.Unlock())
	unlocker, err = RLock(ctx, filePath)
	require.NoError(t, err)
	unlocker2, err := RLock(ctx, filePath)
	require.NoError(t, err)
	_, err = Lock(ctx, filePath, LockWithTimeout(100*time.Millisecond), LockWithRetryDelay(10*time.Millisecond))
	require.Error(t, err)
	require.NoError(t, unlocker.Unlock())
	require.NoError(t, unlocker2.Unlock())
}

func TestLockerBasic(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	tempDirPath := t.TempDir()
	filePath := "path/to/lock"
	locker, err := NewLocker(tempDirPath)
	require.NoError(t, err)
	unlocker, err := locker.Lock(ctx, filePath)
	require.NoError(t, err)
	_, err = locker.Lock(ctx, filePath, LockWithTimeout(100*time.Millisecond), LockWithRetryDelay(10*time.Millisecond))
	require.Error(t, err)
	require.NoError(t, unlocker.Unlock())
	unlocker, err = locker.Lock(ctx, filePath, LockWithTimeout(100*time.Millisecond), LockWithRetryDelay(10*time.Millisecond))
	require.NoError(t, err)
	require.NoError(t, unlocker.Unlock())
	unlocker, err = locker.RLock(ctx, filePath)
	require.NoError(t, err)
	unlocker2, err := locker.RLock(ctx, filePath)
	require.NoError(t, err)
	_, err = locker.Lock(ctx, filePath, LockWithTimeout(100*time.Millisecond), LockWithRetryDelay(10*time.Millisecond))
	require.Error(t, err)
	require.NoError(t, unlocker.Unlock())
	require.NoError(t, unlocker2.Unlock())
	absolutePath := "/not/normalized/and/validated"
	if runtime.GOOS == "windows" {
		absolutePath = "C:\\not\\normalized\\and\\validated"
	}
	_, err = locker.Lock(ctx, absolutePath)
	require.Error(t, err)
}
