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

package git

import (
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadPackedRefs(t *testing.T) {
	t.Parallel()

	allBytes, err := os.ReadFile(path.Join("testdata", "packed-refs"))
	require.NoError(t, err)

	branches, tags, err := parsePackedRefs(allBytes)

	require.NoError(t, err)
	hexBranches := map[string]string{}
	for branch, hash := range branches {
		hexBranches[branch] = hash.Hex()
	}
	hexTags := map[string]string{}
	for tag, hash := range tags {
		hexTags[tag] = hash.Hex()
	}
	assert.Equal(t, hexBranches, map[string]string{
		"amckinney/template-plugin-DO-NOT-DELETE":                      "e1ce94c9dd3a187478e672d73e5d8a31fbf4a21b",
		"automated-changelog":                                          "102fd69fd306f2bc6e875ffa7d58a1fb4337d9ba",
		"buf-beta-price-organization":                                  "837b8eb6a1ec4799a95350a726774c2e55b73cc4",
		"bufgenv2":                                                     "31dc47f8d9625b229f72fd5a983880d3aaf6c3aa",
		"croche/bsr-1614-dogfood-remote-packages-in-cli":               "4d3dedec75a6efcb60827c9054056d88c768b4b0",
		"cyinma/deprecated-curated-plugin":                             "ba5ca857e8522eb8be435cc8a926014754139c43",
		"do-not-delete-this-branch-comments-exclude-request-responses": "a3265eb607323e8ca2771b52f2fc5486ecaeadfc",
		"do-not-delete-this-branch-format-stdin":                       "ed8a55932ff3f77304e9acadf1670415c40c77d6",
		"do-not-delete-this-branch-plugin-extensions":                  "f7c6405aff1684b25a78740e87508bd8a6a2eba2",
		"ejackson/lint-poc":                                            "ea2e60b5a7a06de033219e695fc3a6ca33e0b45e",
		"example-validate-rule":                                        "2cfc2bb386bde9cae3830b0e99ae448256338b6e",
		"fix-breaking-workspace":                                       "a59b57ae4a01d6171afc29458293324d2833f748",
		"fsnotify":                                                     "55e5b99a957c38c993c18c29047bdd3b6be5292d",
		"jfigueroa/module-pin-string-parser":                           "62bc3be78d1bf38e57aa2aae1aa0077a0b962eb5",
		"main":                                                         "45c2edc61040013349e094663e492996e0c044e3",
		"paralleltest":                                                 "1fddd89116e24df213d43b7d837f5dd29ee9cbf0",
		"proto3-optional-oneof-breaking":                               "dfe0ff6db8bc524976b2f9b65dfe9105a665e0ce",
		"release-improvement":                                          "eabe913e1653b05ed0fee809c9c5a6f52948b3c3",
		"robbert/wasm-plugins-in-path":                                 "f9500f579132152e7129a499629a24ec1616e684",
		"saquib/bsr-1760-scaffold-workspace-push-command":              "13832b664214c7746c140d23773e6f9640b08857",
		"twilly/git-storage-reader-v2":                                 "9bd7f631281cd67685458e49e54cc85f9a7e527e",
	})
	assert.Equal(t, hexTags, map[string]string{
		"v0.1.0":      "157c7ae554844ff7ae178536ec10787b5b74b5db",
		"v0.10.0":     "84a94a34350133e615aa642ea872e76dac2c4db0",
		"v0.11.0":     "6741ced6ba6b2eabc23d87bdf154672ac90920d4",
		"v0.12.0":     "a88f7f6c09ebc93f52d1af2cbe7c84e3e2ee48da",
		"v0.12.1":     "0767b6189c52b377d7f4657130cc75e08cd53f2a",
		"v0.13.0":     "6b204c2ab54eec69d897666f4ff74d2e8e6c046c",
		"v0.14.0":     "19658a000ffd50b4fa3cb2f5ea6ac9a342663c99",
		"v0.15.0":     "de4c1d0a5d6cd397bac90de8afb49c4f0e677e6f",
		"v0.16.0":     "a9e3bc0b2c0b051cc0fe7fd5889a8feab0690a94",
		"v0.17.0":     "fa74aa9c4161304dfa83db4abc4a0effe886d253",
		"v0.18.0":     "e1ac5100d5ebb31f716aac9b4d466b0284fad440",
		"v0.18.1":     "c1f8a7ebff899fc5debd3ac7b5d6deb438a9cc95",
		"v0.19.0":     "6c364549e6c2e8d8d07b5d128339061be73b0117",
		"v0.19.1":     "8de28927de4b6125db9f200b59fba31d3426219c",
		"v0.2.0":      "15c4afca023acf2c5584d3ba76a254b7a5a16be2",
		"v0.20.0":     "8bbf2eb5425c889c8c7d38d3f5cfd6957ff3907e",
		"v0.20.1":     "5e8bf4c800de911764ffdf8d2188b7f6f54476e4",
		"v0.20.2":     "490d7cbb9609b2e94afd028655e08652d1739756",
		"v0.20.3":     "97842c1b2c7d9572679ef2e2e49d2cb56927f840",
		"v0.20.4":     "ab95eaa8c272da7674308af62646f031469437c5",
		"v0.20.5":     "3750c611dd6399b65c56575fb66b87770a48328a",
		"v0.21.0":     "e17910be783788ae24bc2b391c39bc5af7543f30",
		"v0.22.0":     "ff32115343fcc114f5011142418c4374f6b91062",
		"v0.23.0":     "174433e7ea4f2a9bdcd4f49f88aed17d0264ea2b",
		"v0.24.0":     "90e8ae66f47da723cb9f9e6f1d15be150602a540",
		"v0.25.0":     "f8551098343fced3c5f44d24485f590c1ad16150",
		"v0.26.0":     "0bfcfefd428f91935fcea3bb20614530e529e758",
		"v0.27.0":     "dae5d874d20a82a1be76f109e05087c5c4d6155b",
		"v0.27.1":     "0483ecae653cd98b0367146bd72fcf6d199620d8",
		"v0.28.0":     "514ee8fae6346ee6f054103c72c39cfd281cf707",
		"v0.29.0":     "01296b2bccd890b665d016a2f6f3751acb49a84f",
		"v0.3.0":      "75c833fa648895831afeca0a319a682688000eeb",
		"v0.30.0":     "41140c671ff89a3edba35bd0f1fbe4dc55e5ed18",
		"v0.30.1":     "1ff6ad3bf4fccae698b9c7fc535ae0630c5c996b",
		"v0.31.0":     "efb320747adaea7fbe583704d0e7295c090af87c",
		"v0.31.1":     "70582b6bbd88ba1f126bbba400590bc78bb599c7",
		"v0.32.0":     "a1d8e6d4a615d2345648cd54fa4124768c02609e",
		"v0.32.1":     "78d698a5942a8097264167c1ad111be8847a71ae",
		"v0.33.0":     "d4cb4baff4db5061c3cc6e767da7a1089db1157e",
		"v0.34.0":     "fa62d8b03d036c021144f102316d020e8748656b",
		"v0.35.0":     "457c3c661804a5dced987d92e80d19b1ad9c06ba",
		"v0.35.1":     "67f4a0ce414f588e25f21375adba12a0b49ad264",
		"v0.36.0":     "fb925f0ae4cedbcb15e1b9ede897fd81bc092bb2",
		"v0.37.0":     "e871670ca95cc453233ab7362bbe9d7ccab2dcf8",
		"v0.37.1":     "4404b02c756932d76df4d81331d15f293685d206",
		"v0.38.0":     "0d4edaf2772cff0a02a14ecb74f0cba3ca576ff3",
		"v0.39.0":     "d55f33d896f0340525396c277138b148741c7db3",
		"v0.39.1":     "0e3780c7f18ad6ad93f6ed01e60bcc9065b50580",
		"v0.4.0":      "76d9224c708e03475187613ad253565d30b4449b",
		"v0.4.1":      "abed61f0c6edd512530a12c00a80d70ae9d922de",
		"v0.40.0":     "20e5b09c61c04b07e758b455b81e796f2d61c05a",
		"v0.41.0":     "4eb511f804ccd7efe8ebf0eeba46c36eb8bd93fe",
		"v0.42.0":     "3c4b36d71f7953be15aa47787c319693e0a2999a",
		"v0.42.1":     "3ccc883da1b06744be112e24f062921780fae203",
		"v0.43.0":     "04c44161c740fcde4637cc97619217b4f019a1c5",
		"v0.43.1":     "dc5a59594998fa5d0a4944e3d7a2cf60643ec20c",
		"v0.43.2":     "50263c34d287d175f0da634d7ce8d4a43e704e8d",
		"v0.44.0":     "aaca4e43354d2eafff4efab76391fcd8fe7dcd9c",
		"v0.45.0":     "05263d288607fe4872ca5da92514dd76d748ee57",
		"v0.46.0":     "8bbd1eda67268fe9b7d8da9b9e10670aa1a25fc6",
		"v0.47.0":     "447e55a758e2b440b7562daecced16d4aa94bfa4",
		"v0.48.0":     "4969d10415a42105ef91234410cc6774aa900efc",
		"v0.48.1":     "71bdee50c214451fc1fb33d701aa8d03f8019101",
		"v0.48.2":     "a7fbfbe4eb5ba0727a4c8ea294d1fd3155dcd0d7",
		"v0.49.0":     "8f9fafb9dcfd8fa355374fde15c2ba40cc52a70a",
		"v0.5.0":      "5725ae7dc08cb3ac72b9c9ff0a851739eeaed42f",
		"v0.50.0":     "70bacd18d876eb75715b198026ed4a9a290f834b",
		"v0.51.0":     "81dd60f405624a0e349ace93eea33069caaa06b3",
		"v0.51.1":     "e077b275cf597b661c9a4638137cb44d3ad7a520",
		"v0.52.0":     "6ff3b25d29c2a5a50eddde6175a705414b412dfb",
		"v0.53.0":     "847d45959ff3d442e99bcc25f02042cdf6a52de2",
		"v0.54.0":     "bd6afd7e419aa7dfffeb49d17896a57ddf421cbd",
		"v0.54.1":     "474eedb6b5cef9a0a782d59705407bc6a1499138",
		"v0.55.0":     "f9ac53fbdafc42a108b2b8d4b5685930c9544d80",
		"v0.56.0":     "66d1594abcdd9ff9014af789f23f3f3a6b81ae1d",
		"v0.6.0":      "5a96bbf85540c8bcf16c4077a135474430b4df9f",
		"v0.7.0":      "0afe7d55870fd1b322b01cacc6951eefdb24e9f2",
		"v0.7.1":      "939e29f7f353d4d9b76ce5af0d305377327cf2aa",
		"v0.8.0":      "0c4701fc4e2b9415c693bb6dd2bf8910e1a84c14",
		"v0.9.0":      "de34b69ac71cec073b472d4b7c850a0fe7eb6aa1",
		"v1.0.0":      "2db2bfff3f7605071182f7b7b8ca28cc26e62a89",
		"v1.0.0-rc1":  "490455648d46bda293ce84efb91f8810d2424b96",
		"v1.0.0-rc10": "e753dad1050641e57f5a3fc1fe46f1936aa0db07",
		"v1.0.0-rc11": "dead8bb4283e550012939f42308c71038e79be48",
		"v1.0.0-rc12": "748ff0c93fabc390fcdec3d9124a7ddc833b3e7f",
		"v1.0.0-rc2":  "6bb7239d78f54cb24bb4f046da35c2ffa275b955",
		"v1.0.0-rc3":  "36764d00e61cb52c864b303f588ff7a793c7697f",
		"v1.0.0-rc4":  "94e38f49794d00d01be8ed1ee0177ae9d4370d57",
		"v1.0.0-rc5":  "4e1ed57752d3e385c58790c548b1b4e7720f494b",
		"v1.0.0-rc6":  "c198b8a493dd0ac4f52ab687792d50bdc4fb09bf",
		"v1.0.0-rc7":  "5706eaa830d1f0537697145128514a4befb0cf24",
		"v1.0.0-rc8":  "3ca7f923c888458bd58ece400be555ea5c0a9a4a",
		"v1.0.0-rc9":  "2c216fdd07eb54af17f5ec9bcbd6360d03fd5792",
		"v1.1.0":      "51a7ec794e001c68ba20b9050c0f8984be8dabe5",
		"v1.1.1":      "dacba58916c206288fdf0ff72c17e9b3443712db",
		"v1.10.0":     "ebb191e8268db7cee389e3abb0d1edc1852337a3",
		"v1.11.0":     "b6d1820662c585b372b948056e2b7c47fec9b72e",
		"v1.12.0":     "58ba9bbc8640dddb263b1d6ce7dc8f777f6f8e19",
		"v1.13.0":     "c0c42cdfe0fb929efc386f2688b2bcb08e517570",
		"v1.13.1":     "e63c47ba49e4b4be4cd919764bde4e8db1110bdf",
		"v1.14.0":     "954dc0dcd5a0831b6c617f6af6a0d8729a410986",
		"v1.15.0":     "a0ffc318d04fe8de1e2e04b73ff7a135d67c5442",
		"v1.15.1":     "5d924a674cf977ab6e1994f127046ef6880aaaa2",
		"v1.16.0":     "37f8c8b8de8c0656e43aefe3ba5405551130d0c6",
		"v1.17.0":     "95ad89070db495ba63e5a7faaea45e8a9daccdab",
		"v1.18.0":     "95f33895a322647f9c9ee362637d58a19dcd58fb",
		"v1.19.0":     "24160b7b12356bf7306f8b4fe4713f5e7981b8b7",
		"v1.2.0":      "17dfdb3d5e0c13531be85339682a7321df103601",
		"v1.2.1":      "611e4b6586531c4367bb1d8c1b7b336d75cb2928",
		"v1.3.0":      "8b26061eeebc26fa9b68583d1a4f14315cdfa69b",
		"v1.3.1":      "df26a6c3f5626a752ccefc5e549db9dba6b13233",
		"v1.4.0":      "bd759fc726f4f9ef07841457fefa3314e1c9f0a3",
		"v1.5.0":      "4c5463f963863508c1425e1767f8e88b7f4a296b",
		"v1.6.0":      "25520a3fb15c8fab140e76e7238e6ace8dac13d9",
		"v1.7.0":      "028fdd557df6aebe93ae4682fc4be62b490a4e60",
		"v1.8.0":      "dbd4a13a086870e6953ce06d843fc73eae0988e9",
		"v1.9.0":      "0d39fac763025d6e86f7145d9836c9214a628c12",
	})
}
