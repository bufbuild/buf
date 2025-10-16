package p

func foo() {
	//fileName := ""
	//_ = fileName
	//filename := "" // `Use "fileName" instead of "filename"`
	//_ = filename
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
	//FileName := ""
	//_ = FileName
	//Filename := "" // `Use "FileName" instead of "Filename"`
	//_ = Filename
	FilePath := ""
	_ = FilePath
	Filepath := "" // want `Use "FilePath" instead of "Filepath"`
	_ = Filepath
	DirName := ""
	_ = DirName
	Dirname := "" // want `Use "DirName" instead of "Dirname"`
	_ = Dirname
	DirPath := ""
	_ = DirPath
	Dirpath := "" // want `Use "DirPath" instead of "Dirpath"`
	_ = Dirpath
}
