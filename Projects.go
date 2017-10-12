package main

import (
  "fmt"
  "strings"
  "os/exec"
  "net/http"
  "golang.org/x/net/html"
  "io/ioutil"
  "encoding/xml"
)

type Query struct {
	GroupId string `xml:"groupId"`
	ArtifactId string `xml:"artifactId"`
	Latest string `xml:"versioning>latest"`
	Release string `xml:"versioning>release"`
	Versions []string `xml:"versioning>versions>version"`
}

type group struct {
  groupId string
  artifacts []string
}

func parsePomFiles() {

}

func getGroupIds(url string) []string {
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
        if a.Key == "href" && !strings.Contains(a.Val, ".") {
          list = append(list, strings.Trim(a.Val, "/"))
        }
      }
    }
  }

  page.Body.Close()

  return list
}

func getProjectList(baseUrl string) []group {
  groupIds := getGroupIds(baseUrl)
  groupList := make([]group, len(groupIds))

  for i := range groupIds {
    artifacts := getGroupIds(baseUrl + groupIds[i])
    groupList[i] = group{groupIds[i], artifacts}
  }

  return groupList
}

func getMetaData(group string, artifact string, url string) {
    var metaData Query

    cmdUrl := url + string(group) + "/" + artifact + "/maven-metadata.xml"
    fmt.Println(cmdUrl)
    page,_ := http.Get(cmdUrl)
    if page.StatusCode == 200 {
      data,_ := ioutil.ReadAll(page.Body)
      xml.Unmarshal(data, &metaData)
    } else {
      versions := getGroupIds(cmdUrl)
      latest := versions[len(versions) - 1]
      metaData = metaData{group, artifact, latest, latest, versions}
    }

    return metaData
}

func downloadProject(repoUrl string, project group) {
  for _,artifact := range project.artifacts {
    metaData := getMetaData(project.groupId, artifact, repoUrl)
    url := repoUrl + string(project.groupId) + "/" + artifact + "/" + mataData.Latest
}

func mvnDownloadProject(repoUrl string, project group, finished chan bool) {
  cmdUrl := "-DrepoUrl=\"" + repoUrl + "\""
  for i := range project.artifacts {
    artifact := "-Dartifact=" + project.groupId + ":" + project.artifacts[i] + ":LATEST"
    cmd := exec.Command("mvn", "dependency:get", cmdUrl, artifact)
    cmd.Run()
  }

  finished <- true
}

func main() {
  //batchLength := 10
  //batchNum := 10
  repository := "http://repo1.maven.org/maven2/"
  projects := getProjectList(repository)

  downloadProject(repository, projects[1])
  /*
  for i := 0; i < batchNum; i++ {
    complete := make([]chan bool, batchLength)
    for j := 0; j < batchLength; j++ {
      complete[j] = make(chan bool, 1)
      go mvnDownloadProject(repository, projects[(i + 1) * j], complete[(i + 1) * j])
    }

    for i := 0; i < batchLength; i++ {
      <-complete[i]
      fmt.Println("Downloaded " + projects[i].groupId)
    }
  }
  fmt.Println("Download completed")
  */
}
