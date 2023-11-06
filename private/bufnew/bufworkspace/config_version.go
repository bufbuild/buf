package bufworkspace

import (
	"fmt"
	"strconv"
)

const (
	ConfigVersionV1Beta1 ConfigVersion = iota + 1
	ConfigVersionV1
)

var (
	configVersionToString = map[ConfigVersion]string{
		ConfigVersionV1Beta1: "v1beta1",
		ConfigVersionV1:      "v1",
	}
	stringToConfigVersion = map[string]ConfigVersion{
		"v1beta1": ConfigVersionV1Beta1,
		"v1":      ConfigVersionV1,
	}
)

type ConfigVersion int

func (c ConfigVersion) String() string {
	s, ok := configVersionToString[c]
	if !ok {
		return strconv.Itoa(int(c))
	}
	return s
}

func ParseConfigVersion(s string) (ConfigVersion, error) {
	c, ok := stringToConfigVersion[s]
	if !ok {
		return 0, fmt.Errorf("unknown ConfigVersion: %q", s)
	}
	return c, nil
}
