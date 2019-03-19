package main

import (
	"io"
	"path"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
)

type job struct {
	AllowFailure   bool   `json:"allowFailure"`
	ArtifactsCount int    `json:"artifactsCount"`
	JobID          string `json:"jobId"`
	Name           string `json:"name"`
	// OsType         string `json:"osType"`
	Status string `json:"status"`
}

type build struct {
	// AuthorName     string `json:"authorName"`
	// AuthorUsername string `json:"authorUsername"`
	CommitID string `json:"commitId"`
	IsTag    bool   `json:"isTag"`
	Message  string `json:"message"`
	Status   string `json:"status"`
	Tag      string `json:"tag"`
	Branch   string `json:"branch"`
	// BuildID        string `json:"buildId"`
	// BuildNumber    string `json:"buildNumber"`
	Jobs []job `json:"jobs"`
}

// type project struct {
// 	AccountID      string `json:"accountId"`
// 	AccountName    string `json:"accountName"`
// 	Name           string `json:"name"`
// 	ID             string `json:"projectId"`
// 	RepositoryName string `json:"repositoryName"`
// 	Slug           string `json:"slug"`
// }

type buildInfoResponse struct {
	Build build `json:"build"`
}

type artifact struct {
	FileName string `json:"fileName"`
	Size     int    `json:"size"`
}

func api(method, url string, params ...interface{}) (*http.Request, error) {
	uri := fmt.Sprintf("https://ci.appveyor.com/api/"+(strings.TrimLeft(url, "/")), params...)
	req, err := http.NewRequest(method, uri, nil)

	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/json")
	return req, nil
}

func main() {
	project := os.Args[1]
	build := os.Args[2]

	buildInfoReq, err := api("GET", "/projects/%s/build/%s", project, build)

	fmt.Printf("GET %s\n", buildInfoReq.URL.String())

	if err != nil {
		panic(err)
	}

	client := &http.Client{}

	res, err := client.Do(buildInfoReq)

	if err != nil {
		panic(err)
	}

	body, err := ioutil.ReadAll(res.Body)

	if err != nil {
		panic(err)
	}

	var buildInfo buildInfoResponse
	err = json.Unmarshal(body, &buildInfo)

	if err != nil {
		panic(err)
	}

	fmt.Printf(
		"Branch    %s\n"+
			"Commit ID %s %s\n"+
			"Tag:      %s\n",
		buildInfo.Build.Branch,
		buildInfo.Build.CommitID[0:6],
		buildInfo.Build.Message,
		buildInfo.Build.Tag)

	if buildInfo.Build.Status != "success" {
		fmt.Println("WARN: Downloading artifacts of failed build")
	}

	for _, j := range buildInfo.Build.Jobs {
		fmt.Printf("JOB: %s %s\n", j.JobID, j.Name)
		retrieveJob(j.JobID)
	}
}

func retrieveJob(jobID string) {
	req, err := api("GET", "/buildjobs/%s/artifacts", jobID)

	if err != nil {
		panic(err)
	}

	fmt.Printf("GET %s\n", req.URL.String())

	client := &http.Client{}

	res, err := client.Do(req)

	if err != nil {
		panic(err)
	}

	body, err := ioutil.ReadAll(res.Body)

	if err != nil {
		panic(err)
	}

	var artifacts []artifact
	err = json.Unmarshal(body, &artifacts)
	if err != nil {
		panic(err)
	}

	for _, a := range artifacts {
		err := downloadArtifact(jobID, a)
		if err != nil {
			fmt.Printf("%s: Error downloading: %s\n", path.Base(a.FileName), err)
		}
	}
}

func downloadArtifact(jobID string, a artifact) error {
	req, err := api("GET", "/buildjobs/%s/artifacts/%s", jobID, a.FileName)
	req.Header.Set("Accept", "*/*")

	if err != nil {
		panic(err)
	}

	fmt.Printf("GET %s\n", req.URL.String())

	client := &http.Client{}

	res, err := client.Do(req)

	if err != nil {
		panic(err)
	}

	fileName := path.Base(a.FileName)
	fmt.Printf("Downloading %s %d bytes\n", fileName, a.Size)

	file, err := os.Create(fileName)
	if err != nil {
		return err
	}

	downloaded, err := io.Copy(file, res.Body)
	defer file.Close()
	defer res.Body.Close()

	if err != nil {
		return err
	}

	fmt.Printf("Downloaded %d bytes\n", downloaded)
	return nil

}
