package constants

import (
	"strings"
	"sync"

	"github.com/avantifellows/nex-gen-cms/utils"
)

type SortOrder string

const (
	SortOrderAsc  SortOrder = "asc"
	SortOrderDesc SortOrder = "desc"

	StatusArchived = 1
)

// runtime constant
var (
	htmlFolder string
	once       sync.Once
)

// Initialize the variable once
func InitRuntimeConstant() {
	once.Do(initialize)
}

func initialize() {
	// get current working directory
	cwd := utils.GetCurrentWorkingDirectory()

	htmlFolder = "web/html"
	/**
	  main_test.go executes from cmd directory, which requires to go back by one level to find web directory;
	  otherwise actual project executes from project root directory, hence doesn't need any change in HtmlFolder path
	*/
	if strings.HasSuffix(cwd, "\\cmd") {
		htmlFolder = "../" + htmlFolder
	}
}

func GetHtmlFolderPath() string {
	return htmlFolder
}
