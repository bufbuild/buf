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

// Code generated by buf-modulecycle-go-data. DO NOT EDIT.

package datamodulecycle

import (
	"encoding/hex"
	"strings"

	"github.com/bufbuild/buf/private/pkg/shake256"
)

const (
	// TestModuleFullNameStringA is a test ModuleFullName string that will always result in Exists returning true.
	TestModuleFullNameStringA = "bufbuild.internal/foo/bar-a"
	// TestModuleFullNameStringB is a test ModuleFullName string that will always result in Exists returning true.
	TestModuleFullNameStringB = "bufbuild.internal/foo/bar-b"
	// TestModuleFullNameStringC is a test ModuleFullName string that will always result in Exists returning true.
	TestModuleFullNameStringC = "bufbuild.internal/foo/bar-c"
)

var (
	// moduleFullNameStringHexEncodedDigests are the shake256 digests of the module names that are allowed to have cycles for legacy reasons.
	//
	// This list always includes TestModuleFullNameString.* for testing.
	moduleFullNameStringHexEncodedDigests = map[string]struct{}{
		"09130371eeba542cfa55f6fcb852063e2e167a3d3045fd3ff8214d825ae5567f0008ff22ae1c343bfc6bf5b8576084219ee30b26afd36ffee6642d6a64c0605d": {},
		"179cc5f7503893b6eda40952fc3e7532d38074f2ec6af70e4ed205df81ba17e03a8deee5b665e28cc362b715368e86f59a3f931e753e618e7e97a11d5e3751d6": {},
		"188746438e9c8c83bf48700963a3a04ad86add657e6d5139fe3479bb7dec8e63aa0a3589da1c3c7c2ec1a459040c77c5963661c32df4ae1201eeb6f08386a832": {},
		"1d16ef3c94ee1dd38c1ec3e4e9508d6aaacf9d029c695a80cbdeb6ddf234dcbf0a1f2b68e7cd61654f1b11b916e4505a1967d6807368a965ad52344d3d5a1d8f": {},
		"344b3178de70d7f64dd602182e59256529f574194ed5f2fad6accefbe223e132d32d46b6008a460ef6b4cfe4fd1d6402c8a6a7463e56b1b378e811212cde38b3": {},
		"3bf0bafe9957243b2605d91c739bdaeb719fc825841f46314e488338c3c6cb3acc99d0afbf8d9359e99fcf3d6e7611f4b94fab9e32d5cb730863902a0f31a4f3": {},
		"64d7892dbe0ce4b5f8a3bb04e0a0568aeeb8b82ba0db289c5396c119777ddad81de3fa24b2ab18e642313db8e76daa4302ca57b7e2b471c730f72cf17d4f0efe": {},
		"65b99c59d12b7b2ed6668146b6387eadec6c8c0f5277f862ba77c74704147d0f812881cead2d9a3421d35420b0eb935f35d151d7a909428dce27bdd39633b6ef": {},
		"78965c68ca9a463d082333229d7b9dae2561aa7cc98411982d4926655fbf8f02937348ba4ffcc2b3d43d00d7e8dd34e46c3d360384919768e2505698b8a41d40": {},
		"8dc88b52b37b1a96674878c402dff52261880ef107d45270b50ecefe017ccbd1a563e713659eced14f29129b1ac6d0b2a982062f78b5ac842d18089c0e207e57": {},
		"9bf5f1c2708271b06741c55d5fc84c2ec16be24b94436f6b0ef546c22684b4fed939060f026cf254dd3d83b12cb52dd465e7efa4414e1c3a5d71f36bfddae6ea": {},
		"9c18c39e1937e442b091f0ba236323b32b828eecdddf126a2e083147a764674f38702f11763e20d72ff2f672ed44a39983ce52897e60a7e1dd64b2cce9dd58ea": {},
		"9ecc0e8d05fea51a11111f13680cae6e42305c91954392d11b104d76e5b00f55e7474997039fb7e61abf7cab1efe98904a96733af6238690bf0bbcb62b390402": {},
		"a3d44a3716daa5a757da9bb8e741c1a0777ab63968369fa37d5d62dbfd95058c6d0d49938de04dc4cb6eaf181e241485b0ed04b8d8183daf6a42d4ac6fdd7acd": {},
		"ac7c5aa10572900ac6f1a9611a45e38cd3dbacc91daa8903dfad4f293bb8899edd9b8fd43dc180197e4fb07b6b58daa9b0a693bc6238ea8833c88c71ddfcdfba": {},
		"b2aa97399d244353a76eea3ca410915c79bd59f17a41153e7339cea7e5ea9a4023e037fab3cbca2719a54f810a9adf8fe36bda910b5ef7c45db2b2ace77d582a": {},
		"beb91b8da0ac9843330810baf064f3bbd30f5b7d8ded50456e377e3c660ac490fc31cef3ee1a4f9626845310e2d24cabd1fc4f11a5c91d27daf0cdc77d5b44e5": {},
		"bfc761edfcfe0def215887f00f7e73329f1871a75792bfc7cb14de6e37444c4019e324b66c92f5015a36c28bccfeb7a716b4740423fcfc8577dd50a73cff662e": {},
		"d1691090846735247275e3fd397521419d7f75d432ec74c1995b09f1407adac6bd2162f6cb2eff5361ebd3acea232df892b0234472c97b38e659dfcea6f92891": {},
		"de858f9ffeeb88fde5ec08cec8d18f5ae4686c01b369aa329a6e189a43387a2a8c36ea80c2bf152f4a104d2e2e8ad24d2f05a2c3d0b8a49092159cf6b67728db": {},
		"e02c8d6206745b82ec251148807d1a50e9f66442e1c7fe61c016544aa9021f3f8280ec61d53806f75246a86eb65ed7b55b1d1549a81eedc27a64ec25c39b4201": {},
		"edd0cab1de5159f7570dda99acde00307b44ebcfd9b1002e304bd419ad075f429302c3116935c2b2190130c204e7186c42600f02e9866a6d62ea7404896c29bf": {},
		"f78cdd5d6879ba93e3cf8083caa967353d6905037b4abd7853a8141eec417af48f209dffe21566cc866b3bbb1a05beec72dc6f21f88867cb729830e8ee86884d": {},
	}
)

// Exists returns true if the ModuleFullName string is allowed to have a cycle.
func Exists(moduleFullNameString string) (bool, error) {
	if moduleFullNameString == "" {
		return false, nil
	}
	digest, err := shake256.NewDigestForContent(strings.NewReader(moduleFullNameString))
	if err != nil {
		return false, err
	}
	hexEncodedDigest := hex.EncodeToString(digest.Value())
	_, ok := moduleFullNameStringHexEncodedDigests[hexEncodedDigest]
	return ok, nil
}
