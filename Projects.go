package main

import (
  "fmt"
  "strings"
  "os/exec"
  "net/http"
  "golang.org/x/net/html"
)

func getProjectList(url string) []string {
  var list []string
  page, _ := http.Get(url)
  tokenizer := html.NewTokenizer(page.Body)

  for {
    token := tokenizer.Next()

    if token == html.ErrorToken {
      break
    }
    t := tokenizer.Token()

    if t.Data == "a" {
      for _, a := range t.Attr {
        xml := strings.Contains(string(a.Key), "xml")
        txt := strings.Contains(string(a.Key), "txt")
        if a.Key == "href" && !xml && !txt {
          list = append(list, strings.Trim(a.Val, "/"))
        }
      }
    }
  }

  page.Body.Close()

  return list
}

func downloadProject(project string, finished chan bool) {
  cmdUrl := "-DrepoUrl=\"http://repo1.maven.org/maven2\""
  artifact := "-Dartifact=" + project + ":" + project + ":LATEST"

  cmd := exec.Command("mvn", "dependency:get", cmdUrl, artifact)
  cmd.Run()
  fmt.Println("Downloaded " + project)

  finished <- true
}

func main() {
  projects := getProjectList("http://repo1.maven.org/maven2")

  complete := make([]chan bool, len(projects))
  for i := 0; i < 10; i++ {
    go downloadProject(projects[i], complete[i])
  }

  for i := range complete {
    <-complete[i]
  }
  fmt.Println("Download completed")
}
