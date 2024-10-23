package utils

import (
	"log"
	"os"
)

func GetCurrentWorkingDirectory() string {
	cwd, err := os.Getwd()
	if err != nil {
		log.Println(err)
	}
	return cwd
}
