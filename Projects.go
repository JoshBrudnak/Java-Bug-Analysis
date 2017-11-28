package main

import (
	"bufio"
	"encoding/xml"
	"fmt"
	"golang.org/x/net/html"
	"io"
	"io/ioutil"
	"net/http"
	"runtime"
    "os"
	//"strconv"
	"strings"
)

var finIds []string
var count int

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

func getGroupIds(baseUrl string) {
	fmt.Println(baseUrl)
	page, err := http.Get(baseUrl + "maven-metadata.xml")

	if page != nil {
		page.Body.Close()
	}

	if err == nil && page.StatusCode != 404 {
		finIds = append(finIds, baseUrl)
	} else {
		newGParts, linkErr := getPageLinks(baseUrl)
		if linkErr == nil {
			for i := range newGParts {
				newUrl := baseUrl + newGParts[i] + "/"
				getGroupIds(newUrl)
			}
		}
	}
}

func getMetaData() []projectRelease {
	var metaData []projectRelease

	for _, artifact := range finIds {
		var data projectRelease
		cmdUrl := artifact + "maven-metadata.xml"
		page, err := http.Get(cmdUrl)
        if page != nil {
          defer page.Body.Close()
        }

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

func saveGroups() {
	file, _ := os.Create("groupIds.txt")

	for i := range finIds {
		newLine := []byte(finIds[i] + "\n")

		file.Write(newLine)
	}
	file.Close()
}


func downloadProject(artifact string, version string) {
	url := artifact + version
    artifactPath := strings.Trim(url, "http://repo1.maven.org/")
	fileName := projName + "-" + version
	home := os.Getenv("HOME")
	filePath := home + "/.m2/repository/" + artifactPath
	os.MkdirAll(filePath, os.ModePerm)

	saveFile(url, filePath, fileName+".jar")
	saveFile(url, filePath, fileName+".pom")
	saveFile(url, filePath, fileName+".pom.sha1")
	saveFile(url, filePath, fileName+".pom.md5")
}

func getArtifactUrls(repoUrl string) {
    file,fileErr := os.Open("groups.txt")

    if file != nil {
      defer file.Close()
    }

    if fileErr != nil {
		rootProjects, err := getPageLinks(repoUrl)

		if err != nil {
           panic(err)
        }

	    for i := range rootProjects {
            proj := repoUrl + rootProjects[i] + "/"
		    getGroupIds(proj)
	    }

        fmt.Println("Final Length")
        fmt.Println(len(finIds))
        saveGroups()
    } else {
	   scanner := bufio.NewScanner(file)

	   for scanner.Scan() {
          finIds = append(finIds, string(scanner.Text()))
       }
    }
}

func GetProjects(numberOfProjects int) []string {
    count = 0
	runtime.GOMAXPROCS(runtime.NumCPU())
	repository := "http://repo1.maven.org/maven2/"
	getArtifactUrls(repository)
	projects := getMetaData()

	projectsUsed := len(projects)

	for i := range projects {
		if count >= numberOfProjects {
			projectsUsed = i
			break
		}

		count++
		fmt.Println("Downloading " + finIds[i] + " ...")
		downloadProject(finIds[i], projects[i].latest)
		fmt.Println("Downloaded " + finIds[i])
	}

	fmt.Println("Download completed")
	fmt.Println("Downloaded " + strconv.Itoa(count) + " projects")

	return projects[0 : projectsUsed-1]
}
