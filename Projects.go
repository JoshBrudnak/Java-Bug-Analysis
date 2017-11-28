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

func getPageLinks(url string) ([]string, error) {
	var list []string
	page, err := http.Get(url)

	if err != nil {
		fmt.Println("page link error")
		fmt.Println(err)

		return nil, err
	}
	defer page.Body.Close()

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

	return list, err
}

func saveGroups(groups []group) {
	file, err := os.Create("groupIds.txt")
	fmt.Println(err)
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

var finIds []string

func getGroupIds(baseUrl string) {
	fmt.Println(baseUrl)
	page, err := http.Get(baseUrl + "maven-metadata.xml")

	if page != nil {
		page.Body.Close()
	}

	if err != nil {
		finIds = append(finIds, baseUrl)
	} else {
		newGParts, linkErr := getPageLinks(baseUrl)
		fmt.Println(linkErr)
		if linkErr == nil {
			for i := range newGParts {
				newUrl := baseUrl + newGParts[i] + "/"
				getGroupIds(newUrl)
			}
		}
	}
}

func getMetaData(projects []string, url string) []projectRelease {
	var metaData []projectRelease

	for _, artifact := range projects {
		var data projectRelease
		cmdUrl := url + "/" + artifact + "/maven-metadata.xml"
		page, err := http.Get(cmdUrl)

		if err != nil {
			if page != nil && page.StatusCode == 200 {
				pageData, _ := ioutil.ReadAll(page.Body)
				xml.Unmarshal(pageData, &data)
				page.Body.Close()
			}
		} else {
			versions, _ := getPageLinks(cmdUrl)
			if len(versions) > 0 {
				latest := versions[len(versions)-1]
				data = projectRelease{latest, latest, versions}
			} else {
				data = projectRelease{"", "", versions}
			}
		}
		metaData = append(metaData, data)
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

		defer page.Body.Close()
		defer file.Close()
	}
}

func downloadProject(projName string, repoUrl string) {
	latest := "1.0.0"
	artifactPath := projName + "/" + latest
	url := repoUrl + artifactPath
	fileName := projName + "-" + latest
	home := os.Getenv("HOME")
	filePath := home + "/.m2/repository/" + artifactPath
	os.MkdirAll(filePath, os.ModePerm)

	saveFile(url, filePath, fileName+".jar")
	saveFile(url, filePath, fileName+".pom")
	saveFile(url, filePath, fileName+".pom.sha1")
	saveFile(url, filePath, fileName+".pom.md5")

	//finished <- true
}

func downloadBatchs(repoUrl string) {
    newGParts, linkErr := getPageLinks(repoUrl)
	finished := make([]chan bool, len(project.artifacts))
	for i := range project.artifacts {
		finished[i] = make(chan bool)
		go downloadArtifact(project[i], repoUrl, finished[i])
	}

	for i := range finished {
		<-finished[i]
	}
}

func GetProjects(numberOfProjects int) []string {
	repository := "http://repo1.maven.org/maven2/"
	getGroupIds(repository)
	projects := getMetaData(groups, repository)

	for i := range groups {
		fmt.Println("main!!!!!!!!!!!!!!!!!")
		fmt.Println(groups[i])
	}

	fmt.Println(len(groups))
	//complete := make([]chan bool, len(projects))
	count := 0
	projectsUsed := len(projects)
	runtime.GOMAXPROCS(runtime.NumCPU())

	for i := range projects {
		if count >= numberOfProjects {
			projectsUsed = i
			break
		}

		//complete[i] = make(chan bool, 1)
		count++
		fmt.Println("Downloading " + groups[i] + " ...")
		downloadProject(repository, groups[i])
		fmt.Println("Downloaded " + groups[i])
	}

	for i := 0; i < projectsUsed; i++ {
		//<-complete[i]
		fmt.Println("Downloaded " + groups[i])
	}

	fmt.Println("Download completed")
	fmt.Println("Downloaded " + strconv.Itoa(count) + " projects")

	//return projects[0 : projectsUsed-1]
	return nil
}
