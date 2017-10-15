package main

import (
	"fmt"
	"os"
	"os/exec"
)

func checkErr(err error) {
	if err != nil {
		panic(err)
	}
}

func checkWarning(err error) {
	if err != nil {
		fmt.Println(err)
	}
}

func runFindbugs(projGroup group) {
	for _, project := range projGroup.artifacts {
		home := os.Getenv("HOME")
		fileName := project.name + "-" + project.latest + ".jar"
		filePath := home + "/.m2/repository/" + projGroup.groupId + "/" + project.latest + "/" + fileName
		cmd := exec.Command("findbugs", "-textui", filePath, "-xml")
		cmd.Run()
	}
}

func main() {
	projectList := GetProjects(20000)

	for i := range projectList {
		runFindbugs(projectList[i])
		fmt.Println("finished analysing " + projectList[i].groupId)
	}
}
