package p

func foo() {
	fileName := ""
	_ = fileName
	filename := "" // want `Use "fileName" instead of "filename"`
	_ = filename
	filePath := ""
	_ = filePath
	filepath := "" // want `Use "filePath" instead of "filepath"`
	_ = filepath
	dirName := ""
	_ = dirName
	dirname := "" // want `Use "dirName" instead of "dirname"`
	_ = dirname
	dirPath := ""
	_ = dirPath
	dirpath := "" // want `Use "dirPath" instead of "dirpath"`
	_ = dirpath
}
