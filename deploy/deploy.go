package main

// TODO: Problem with compression - too large file
//

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"
)

type BearerTokenResponse struct {
	Type         string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	ExtExpiresIn int    `json:"ext_expires_in"`
	BearerToken  string `json:"access_token"`
}

func exploreDir(dirs []os.DirEntry, filepath string, files []string) []string {
	for _, dir := range dirs {
		if dir.IsDir() {
			path := filepath + "/" + dir.Name()
			d, err := os.ReadDir(path)
			if err != nil {
				fmt.Printf("Error during reading: %v\n", err)
			}
			exploreDir(d, path, files)
		} else {
			files = append(files, filepath+"/"+dir.Name())
		}
	}
	return files
}

func authenticate(tenantid string, username string, password string) (string, error) {
	contentType := "application/x-www-form-urlencoded"
	reqBody := "client_id=" + username + "&scope=https%3A%2F%2Fgraph.microsoft.com%2F.default&client_secret=" + password + "&grant_type=client_credentials"
	endpoint := "https://login.microsoftonline.com/" + tenantid + "/oauth2/v2.0/token"
	payload := bytes.NewReader([]byte(reqBody))
	resp, err := http.Post(endpoint, contentType, payload)
	if err != nil {
		return "", err
	}

	body, err := io.ReadAll(resp.Body)
	defer resp.Body.Close()
	if err != nil {
		return "", err
	}
	var token = BearerTokenResponse{}
	json.Unmarshal(body, &token)

	return token.BearerToken, nil
}

// "https://management.azure.com"
func queryAzure(token string, endpoint string, method string, payload io.Reader) ([]byte, error) {
	contentType := "application/zip"
	req, err := http.NewRequest(method, endpoint, payload)
	if err != nil {
		return []byte{}, err
	}
	req.Header.Add("Content-Type", contentType)
	req.Header.Add("Authorization", "Bearer "+token)

	client := http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return []byte{}, err
	}
	body, err := io.ReadAll(resp.Body)
	defer resp.Body.Close()
	if err != nil {
		return []byte{}, err
	}

	return body, nil
}

func main() {
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	sep := strings.SplitAfter(cwd, "/")
	newPath := strings.Join(sep[:len(sep)-1], "")

	cmd := exec.Command("go", "build")
	var stderr, stdout bytes.Buffer
	cmd.Stderr = &stderr
	cmd.Stdout = &stdout

	cmd.Env = append(os.Environ(), "GOOS=windows", "GOARCH=amd64", "CGO_ENABLED=0")
	cmd.Args = append(cmd.Args, "-C", newPath+"webserver", newPath+"webserver/server.go")
	err = cmd.Run()

	if err != nil {
		fmt.Println("STDERR:", stderr.String())
		fmt.Printf("Error during compilation: %s", err)
	} else if stdout.String() != "" {
		fmt.Println("STDOUT:", stdout.String())
	}

	zipFile, err := os.Create("webserver.zip")
	if err != nil {
		log.Fatalln("Failed during creation of archive.")
	}
	defer zipFile.Close()
	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	dirs, err := os.ReadDir(newPath + "webserver")
	if err != nil {
		log.Fatalln("Failed during exploriation of directory")
	}
	files := []string{}
	files = exploreDir(dirs, newPath+"webserver", files)
	for _, file := range files {
		fileToZip, err := os.Open(file)
		if err != nil {
			panic(err)
		}
		defer fileToZip.Close()

		fileInfo, err := fileToZip.Stat()
		if err != nil {
			panic(err)
		}
		header, err := zip.FileInfoHeader(fileInfo)
		if err != nil {
			panic(err)
		}

		header.Name = file
		writer, err := zipWriter.CreateHeader(header)
		if err != nil {
			panic(err)
		}

		_, err = io.Copy(writer, fileToZip)
		if err != nil {
			panic(err)
		}
	}

	fmt.Println("Compression done !!!")

	functionAppName := "funapp407202jb" // TODO: Edit
	url := fmt.Sprintf("https://%s.scm.azurewebsites.net/api/zipdeploy", functionAppName)
	tenantId := "84f1...."
	username := "837....."
	password := "WKM8Q~......"

	file, err := os.Open(newPath + "/webserver/server.go")
	if err != nil {
		log.Fatalln("Failed to read ZIP file:", err)
		return
	}
	token, err := authenticate(tenantId, username, password)
	fmt.Println(token)
	if err != nil {
		log.Fatalln("Fail to authenticate:", err)
	}
	resp, err := queryAzure(token, url, http.MethodPost, file)
	if err != nil {
		log.Fatalln("Fail deploy:", err)
	}

	fmt.Printf("Resp: %s\n", resp)
	fmt.Println("Deployment successfull !!!")
}
