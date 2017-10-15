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
	"os/exec"
	"strings"
)

type Project struct {
	GroupId    string   `xml:"groupId"`
	ArtifactId string   `xml:"artifactId"`
	Latest     string   `xml:"versioning>latest"`
	Release    string   `xml:"versioning>release"`
	Versions   []string `xml:"versioning>versions>version"`
}

type Dependencies struct {
	GroupIds    []string `xml:"dependencies>dependency>groupId"`
	ArtifactIds []string `xml:"dependencies>dependency>artifactId"`
	Versions    []string `xml:"dependencies>dependency>version"`
}

type group struct {
	groupId   string
	artifacts []string
}

func resolveDependencies(projectUrl string, repoUrl string) {
	var deps Dependencies
	page, _ := http.Get(projectUrl)
	if page.StatusCode == 200 {
		data, _ := ioutil.ReadAll(page.Body)
		xml.Unmarshal(data, &deps)
		/*
		   for i := range deps.GroupIds {
		     depUrl := repoUrl + deps.GroupIds[i] + "/" + deps.ArtifactIds[i] + "/" + deps.Versions[i]
		     //downloadProject(depUrl)
		   }
		*/
	}
}

func getGroupIds(url string) []string {
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
			groupCsv = groupCsv + "," + artifact
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
		groups = append(groups, group{lineParts[0], lineParts[1:]})
	}
	file.Close()

	return groups
}

func getProjectList(baseUrl string) []group {
	groupIds := getGroupIds(baseUrl)
	file, err := os.Open("groupIds.txt")

	if err != nil {
		groupList := make([]group, len(groupIds))

		for i := range groupIds {
			artifacts := getGroupIds(baseUrl + groupIds[i])
			groupList[i] = group{groupIds[i], artifacts}
		}
		saveGroups(groupList)
		file.Close()

		return groupList
	} else {
		return readGroups(file)
	}

}

func getMetaData(group string, artifact string, url string) Project {
	var metaData Project

	cmdUrl := url + string(group) + "/" + artifact + "/maven-metadata.xml"
	page, err := http.Get(cmdUrl)

	if err == nil && page.StatusCode == 200 {
		data, _ := ioutil.ReadAll(page.Body)
		xml.Unmarshal(data, &metaData)
	} else {
		versions := getGroupIds(cmdUrl)
		if len(versions) > 0 {
			latest := versions[len(versions)-1]
			metaData = Project{group, artifact, latest, latest, versions}
		} else {
			metaData = Project{group, artifact, "", "", versions}
		}
	}

	return metaData
}

func saveFile(url string, filePath string, fileName string) {
	page, err := http.Get(url + "/" + fileName)
	file, _ := os.Create(filePath + "/" + fileName)

	if err == nil {
		if page.StatusCode == 200 {
			io.Copy(file, page.Body)
		}
	}
}

func downloadProject(repoUrl string, project group, complete chan bool) {
	for _, artifact := range project.artifacts {
		metaData := getMetaData(project.groupId, artifact, repoUrl)

		artifactPath := project.groupId + "/" + artifact + "/" + metaData.Latest
		url := repoUrl + artifactPath
		fileName := artifact + "-" + metaData.Latest
		home := os.Getenv("HOME")
		filePath := home + "/.m2/repository/" + artifactPath
		os.MkdirAll(filePath, os.ModePerm)

		saveFile(url, filePath, fileName+".jar")
		saveFile(url, filePath, fileName+".pom")
		saveFile(url, filePath, fileName+".pom.sha1")
		saveFile(url, filePath, fileName+".pom.md5")

	}

	complete <- true
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
	batchLength := 1000
	repository := "http://repo1.maven.org/maven2/"
	projects := getProjectList(repository)
	if batchLength >= len(projects) {
		batchLength = len(projects)
	}
	complete := make([]chan bool, batchLength)
	count := 0

	for i := range complete {
		complete[i] = make(chan bool, 1)
	}

	for i := 0; i < batchLength; i++ {
		count += len(projects[i].artifacts)
		go downloadProject(repository, projects[i], complete[i])
	}

	for i := 0; i < batchLength; i++ {
		<-complete[i]
		fmt.Println("Downloaded " + projects[i].groupId)
	}
	fmt.Println("Download completed")
	fmt.Println("Downloaded " + count + "projects")
}
