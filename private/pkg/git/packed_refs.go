// Copyright 2020-2024 Buf Technologies, Inc.
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

package git

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"strings"
)

const (
	// packedRefsHeader is the head for the `packed-refs` file
	// based on https://github.com/git/git/blob/master/refs/packed-backend.c#LL1084C41-L1084C41
	packedRefsHeader      = "# pack-refs with: peeled fully-peeled sorted "
	tagRefPrefix          = "refs/tags/"
	localBranchRefPrefix  = "refs/heads/"
	remoteBranchRefPrefix = "refs/remotes/"
	unpeeledRefPrefix     = '^'
)

// parsePackedRefs reads a `packed-refs` file, returning the packed branches and tags
func parsePackedRefs(data []byte) (
	map[string]map[string]Hash, // branches
	map[string]Hash, // tags
	error,
) {
	var (
		packedBranches = make(map[string]map[string]Hash)
		packedTags     = make(map[string]Hash)
	)
	/*
		data is in the format
			<packedRefsHeader>\n
			repeated:
				<hash><space><ref>\n
				(optional for tags if unpeeled)^<hash>\n

		for branches, the hash is the commit object
		for tags, the hash is the tag object; the following line is the commit hash
	*/
	var lines []string
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if scanner.Err() != nil {
		return nil, nil, scanner.Err()
	}
	for i := 0; i < len(lines); i++ {
		line := lines[i]
		if strings.HasPrefix(line, "#") {
			// Git tells us the way that these refs are packed. In theory the refs
			// may be packed in different ways, but as of today's writing, they are
			// always packed fully-peeled.
			//
			// The comment should match `packedRefsHeader`. We can safely skip this comment if so.
			if line != packedRefsHeader {
				return nil, nil, fmt.Errorf("unknown packed-refs header: %q", line)
			}
			continue
		}
		hashHex, ref, found := strings.Cut(line, " ")
		if !found {
			return nil, nil, errors.New("invalid packed-refs file")
		}
		hash, err := parseHashFromHex(hashHex)
		if err != nil {
			return nil, nil, fmt.Errorf("invalid hash for ref %s: %w", ref, err)
		}
		if strings.HasPrefix(ref, localBranchRefPrefix) {
			branchName := strings.TrimPrefix(ref, localBranchRefPrefix)
			// empty remote means local
			if packedBranches[""] == nil {
				packedBranches[""] = make(map[string]Hash)
			}
			packedBranches[""][branchName] = hash
		} else if strings.HasPrefix(ref, remoteBranchRefPrefix) {
			remoteAndBranch := strings.TrimPrefix(ref, remoteBranchRefPrefix)
			remoteName, branchName, found := strings.Cut(remoteAndBranch, "/")
			if !found {
				return nil, nil, fmt.Errorf("invalid packed-refs file: invalid ref %s", ref)
			}
			if remoteName == "" {
				return nil, nil, fmt.Errorf("invalid packed-refs file: empty remote for ref %s", ref)
			}
			if branchName == "" {
				return nil, nil, fmt.Errorf("invalid packed-refs file: empty branch for ref %s", ref)
			}
			if packedBranches[remoteName] == nil {
				packedBranches[remoteName] = make(map[string]Hash)
			}
			packedBranches[remoteName][branchName] = hash
		} else if strings.HasPrefix(ref, tagRefPrefix) {
			tagName := strings.TrimPrefix(ref, tagRefPrefix)
			// We're looking at a tag. If the tag is annotated, the next line is our actual
			// commit hash, prefixed with '^'. If not, the already read hash is our commit hash.
			// We need to look ahead to see the next line.
			if len(lines) > i+1 && lines[i+1][0] == unpeeledRefPrefix {
				// We have an annotated tag that's been peeled. Let's read it.
				i++
				nextLine := lines[i]
				nextLine = strings.TrimPrefix(nextLine, string(unpeeledRefPrefix))
				hash, err = parseHashFromHex(nextLine)
				if err != nil {
					return nil, nil, err
				}
			}
			packedTags[tagName] = hash
		}
		// We ignore all kinds of refs.
	}
	return packedBranches, packedTags, nil
}
