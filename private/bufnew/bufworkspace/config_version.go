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

package bufworkspace

//const (
//ConfigVersionV1Beta1 ConfigVersion = iota + 1
//ConfigVersionV1
//)

//var (
//configVersionToString = map[ConfigVersion]string{
//ConfigVersionV1Beta1: "v1beta1",
//ConfigVersionV1:      "v1",
//}
//stringToConfigVersion = map[string]ConfigVersion{
//"v1beta1": ConfigVersionV1Beta1,
//"v1":      ConfigVersionV1,
//}
//)

//type ConfigVersion int

//func (c ConfigVersion) String() string {
//s, ok := configVersionToString[c]
//if !ok {
//return strconv.Itoa(int(c))
//}
//return s
//}

//func ParseConfigVersion(s string) (ConfigVersion, error) {
//c, ok := stringToConfigVersion[s]
//if !ok {
//return 0, fmt.Errorf("unknown ConfigVersion: %q", s)
//}
//return c, nil
//}
