package util

import (
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/analysis"
)

// ForEachComment iterates over every Comment and calls f.
func ForEachComment(pass *analysis.Pass, f func(*ast.Comment) error) error {
	for _, file := range pass.Files {
		for _, commentGroup := range file.Comments {
			for _, comment := range commentGroup.List {
				if err := f(comment); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// ForEachObject iterates over every Object and calls f.
func ForEachObject(pass *analysis.Pass, f func(types.Object) error) error {
	if typesInfo := pass.TypesInfo; typesInfo != nil {
		for _, object := range pass.TypesInfo.Defs {
			if object != nil {
				if err := f(object); err != nil {
					return err
				}
			}
		}
	}
	return nil
}
