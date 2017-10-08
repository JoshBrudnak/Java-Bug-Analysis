package main

import (
  "fmt"
  "os/exec"
  "sync"
)

func downloadProject(project string, finished chan bool) {
  cmdUrl := "-DrepoUrl=\"http://repo1.maven.org/maven2\""
  artifact := "-Dartifact=" + project + ":" + project + ":LATEST"

  cmd := exec.Command("mvn", "dependency:get", cmdUrl, artifact)
  cmd.Run()
  fmt.Println("Downloaded " + project)

  finished <- true
}

func main() {
  projects := []string{"abbot", "acegisecurity", "activation", "activecluster", "activeio", "activemq"}

  complete := make([]chan bool, len(projects))
  for i := range projects {
    wg.Add(1)
    go downloadProject(projects[i], complete[i])
  }

  for i := range complete {
    <-complete[i]
  }
  fmt.Println("Download completed")
}
