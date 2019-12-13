package bufbuild

import (
	"sort"
	"strings"

	"github.com/bufbuild/buf/internal/buf/buferrs"
	"github.com/bufbuild/buf/internal/pkg/storage/storagepath"
	"github.com/bufbuild/buf/internal/pkg/stringutil"
)

type config struct {
	Roots    []string
	Excludes []string
}

func newConfig(inputRoots []string, inputExcludes []string) (*config, error) {
	if len(inputRoots) == 0 {
		inputRoots = []string{"."}
	}
	roots, err := transformFileListForConfig(inputRoots, "root")
	if err != nil {
		return nil, err
	}
	var excludes []string
	if len(inputExcludes) > 0 {
		excludes, err = transformFileListForConfig(inputExcludes, "exclude")
		if err != nil {
			return nil, err
		}

		rootMap := stringutil.SliceToMap(roots)
		excludeMap := stringutil.SliceToMap(excludes)

		// verify that no exclude equals a root directly
		for exclude := range excludeMap {
			if _, ok := rootMap[exclude]; ok {
				return nil, buferrs.NewUserErrorf("%s is both a root and exclude, which means the entire root is excluded, which is not valid", exclude)
			}
		}
		// verify that all excludes are within a root
		for exclude := range excludeMap {
			if !storagepath.MapContainsMatch(rootMap, exclude) {
				return nil, buferrs.NewUserErrorf("exclude %s is not contained in any root, which is not valid", exclude)
			}
		}
	}

	return &config{
		Roots:    roots,
		Excludes: excludes,
	}, nil
}

func transformFileListForConfig(inputs []string, name string) ([]string, error) {
	if len(inputs) == 0 {
		return inputs, nil
	}

	var outputs []string
	for _, input := range inputs {
		if input == "" {
			return nil, buferrs.NewUserErrorf("%s value is empty", name)
		}
		output, err := storagepath.NormalizeAndValidate(input)
		if err != nil {
			// user error
			return nil, err
		}
		outputs = append(outputs, output)
	}
	sort.Strings(outputs)

	for i := 0; i < len(outputs); i++ {
		for j := i + 1; j < len(outputs); j++ {
			output1 := outputs[i]
			output2 := outputs[j]

			if output1 == output2 {
				return nil, buferrs.NewUserErrorf("duplicate %s %s", name, output1)
			}
			if strings.HasPrefix(output1, output2) {
				return nil, buferrs.NewUserErrorf("%s %s is within %s %s which is not allowed", name, output1, name, output2)
			}
			if strings.HasPrefix(output2, output1) {
				return nil, buferrs.NewUserErrorf("%s %s is within %s %s which is not allowed", name, output2, name, output1)
			}
		}
	}

	// already checked duplicates, but if there are multiple directories and we have ".", then the other
	// directories are within the output directory "."
	var notDotDir []string
	hasDotDir := false
	for _, output := range outputs {
		if output != "." {
			notDotDir = append(notDotDir, output)
		} else {
			hasDotDir = true
		}
	}
	if hasDotDir {
		if len(notDotDir) == 1 {
			return nil, buferrs.NewUserErrorf("%s %s is within %s . which is not allowed", name, notDotDir[0], name)
		}
		if len(notDotDir) > 1 {
			return nil, buferrs.NewUserErrorf("%ss %v are within %s . which is not allowed", name, notDotDir, name)
		}
	}

	return outputs, nil
}
