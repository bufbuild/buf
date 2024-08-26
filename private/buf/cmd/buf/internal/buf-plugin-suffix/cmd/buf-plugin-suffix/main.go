package main

import (
	suffixesplugin "github.com/bufbuild/buf/private/buf/cmd/buf/internal/buf-plugin-suffix"
	"github.com/bufbuild/bufplugin-go/check"
)

func main() {
	check.Main(suffixesplugin.Spec)
}
