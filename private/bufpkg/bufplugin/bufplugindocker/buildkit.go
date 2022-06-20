// Copyright 2020-2022 Buf Technologies, Inc.
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

package bufplugindocker

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/moby/buildkit/session"
	"go.uber.org/zap"
)

const pathBuildkitNodeID = ".buildkit_node_id"
const nodeIDLength = 32

func createSession(contextDir, configDirPath string, logger *zap.Logger) (*session.Session, error) {
	sharedKey := getBuildSharedKey(contextDir, configDirPath, logger)
	s, err := session.NewSession(context.Background(), filepath.Base(contextDir), sharedKey)
	if err != nil {
		return nil, err
	}
	return s, nil
}

func getBuildSharedKey(dir, configDirPath string, logger *zap.Logger) string {
	// build session is hash of build dir with node based randomness
	s := sha256.Sum256([]byte(fmt.Sprintf("%s:%s", getBuildNodeID(configDirPath, logger), dir)))
	return hex.EncodeToString(s[:])
}

func getBuildNodeID(configDirPath string, logger *zap.Logger) string {
	var nodeIDPath string
	if len(configDirPath) > 0 {
		nodeIDPath = filepath.Join(configDirPath, pathBuildkitNodeID)
		if nodeID := loadNodeID(nodeIDPath, logger); len(nodeID) > 0 {
			return nodeID
		}
	}
	b := make([]byte, nodeIDLength)
	if _, err := rand.Read(b); err != nil {
		return configDirPath
	}
	nodeID := hex.EncodeToString(b)
	if len(configDirPath) > 0 {
		if err := os.MkdirAll(configDirPath, 0755); err == nil {
			if err := os.WriteFile(nodeIDPath, []byte(nodeID), 0600); err != nil {
				logger.Warn("failed to store buildkit node id", zap.String("path", nodeIDPath), zap.Error(err))
			}
		}
	}
	return nodeID
}

func loadNodeID(nodeIDPath string, logger *zap.Logger) string {
	nodeID := ""
	if nodeIDBytes, err := os.ReadFile(nodeIDPath); err == nil {
		nodeID = strings.TrimSpace(string(nodeIDBytes))
		decoded, err := hex.DecodeString(nodeID)
		if err != nil || len(decoded) != nodeIDLength {
			// Ignore node id - not in expected format
			logger.Debug("invalid buildkit node id - ignoring", zap.String("path", nodeIDPath))
			nodeID = ""
		}
	}
	return nodeID
}
