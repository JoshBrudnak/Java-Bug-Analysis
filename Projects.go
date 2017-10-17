package main

import (
	"bufio"
	"encoding/xml"
	"fmt"
	"golang.org/x/net/html"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"strings"
)

type projectRelease struct {
	latest   string   `xml:"versioning>latest"`
	release  string   `xml:"versioning>release"`
	versions []string `xml:"versioning>versions>version"`
}

type artifact struct {
	name    string
	latest  string
	release string
}

type group struct {
	groupId   string
	artifacts []artifact
}

func getPageLinks(url string) []string {
	var list []string
	page, err := http.Get(url)

	if err == nil {
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
	}

	return list
}

func saveGroups(groups []group) {
	file, _ := os.Create("groupIds.txt")

	for i := range groups {
		groupCsv := groups[i].groupId

		for _, artifact := range groups[i].artifacts {
			groupCsv = groupCsv + "," + artifact.name + "." + artifact.latest + "." + artifact.release
		}
		newLine := []byte(groupCsv + "\n")

		file.Write(newLine)
	}
	file.Close()
}

func readGroups(file *os.File) []group {
	var groups []group
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := string(scanner.Text())
		lineParts := strings.Split(line, ",")
        var projects []artifact

        for i := 1; i < len(lineParts); i++ {
		  artifactParts := strings.Split(lineParts[i], ".")

          if len(artifactParts) == 2 {
            projects = append(projects, artifact{artifactParts[0], artifactParts[1], artifactParts[2]})
          } else {
            fmt.Println("Incorrect csv formatting")
          }
        }

		groups = append(groups, group{lineParts[0], projects})
	}
	file.Close()

	return groups
}

func getProjectList(baseUrl string) []group {
	groupIds := getPageLinks(baseUrl)
	file, err := os.Open("groupIds.txt")

	if err != nil {
		groupList := make([]group, len(groupIds))

		for i := range groupIds {
			artifactNames := getPageLinks(baseUrl + groupIds[i])
			artifacts := make([]artifact, len(artifactNames))
			for j := range artifacts {
				metaData := getMetaData(groupIds[i], artifactNames[j], baseUrl)
                fmt.Println(groupList[i])
				artifacts[j] = artifact{artifactNames[j], metaData.latest, metaData.release}
			}
			groupList[i] = group{groupIds[i], artifacts}
		}
		saveGroups(groupList)
		file.Close()

		return groupList
	} else {
		return readGroups(file)
	}

}

func getMetaData(group string, artifact string, url string) projectRelease {
	var metaData projectRelease

	cmdUrl := url + string(group) + "/" + artifact + "/maven-metadata.xml"
	page, err := http.Get(cmdUrl)

	if err == nil && page.StatusCode == 200 {
		data, _ := ioutil.ReadAll(page.Body)
		xml.Unmarshal(data, &metaData)
	} else {
		versions := getPageLinks(cmdUrl)
		if len(versions) > 0 {
			latest := versions[len(versions)-1]
			metaData = projectRelease{latest, latest, versions}
		} else {
			metaData = projectRelease{"", "", versions}
		}
	}

	return metaData
}

func saveFile(url string, filePath string, fileName string) {
	_, err := os.Open(filePath + "/" + fileName)
	if err != nil {
      fmt.Println(err)
		page, pageErr := http.Get(url + "/" + fileName)
		file, _ := os.Create(filePath + "/" + fileName)

		if pageErr == nil {
			if page.StatusCode == 200 {
				io.Copy(file, page.Body)
			}
		}
    }
}

func downloadArtifact(group string, project artifact, repoUrl string, finished chan bool) {
	artifactPath := group + "/" + project.name + "/" + project.latest
	url := repoUrl + artifactPath
	fileName := project.name + "-" + project.latest
	home := os.Getenv("HOME")
	filePath := home + "/.m2/repository/" + artifactPath
	os.MkdirAll(filePath, os.ModePerm)

	saveFile(url, filePath, fileName+".jar")
	saveFile(url, filePath, fileName+".pom")
	saveFile(url, filePath, fileName+".pom.sha1")
	saveFile(url, filePath, fileName+".pom.md5")

	finished <- true
}

func downloadProject(repoUrl string, project group, complete chan bool) {
	finished := make([]chan bool, len(project.artifacts))
	for i := range project.artifacts {
		finished[i] = make(chan bool)
		go downloadArtifact(project.groupId, project.artifacts[i], repoUrl, finished[i])
	}

	for i := range finished {
		<-finished[i]
	}

	complete <- true
}

func GetProjects(numberOfProjects int) []group {
	repository := "http://repo1.maven.org/maven2/"
	projects := getProjectList(repository)
	complete := make([]chan bool, len(projects))
	count := 0
	projectsUsed := len(projects)
	runtime.GOMAXPROCS(runtime.NumCPU())

	for i := range projects {
		if count >= numberOfProjects {
			projectsUsed = i
			break
		}

		complete[i] = make(chan bool, 1)
		count += len(projects[i].artifacts)
		go downloadProject(repository, projects[i], complete[i])
	}

	for i := 0; i < projectsUsed; i++ {
		<-complete[i]
		fmt.Println("Downloaded " + projects[i].groupId)
	}

	fmt.Println("Download completed")
	fmt.Println("Downloaded " + strconv.Itoa(count) + " projects")

	return projects[0 : projectsUsed-1]
}

func main() {
  GetProjects(20000)
}
