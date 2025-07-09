package main

import (
	"azure/core"
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
)

var (
	tenantid string
	account  string
	secret   string
)

// Example request:
// {
// "url": "https://dorzeczy.pl/feed",
// "account": "myaccount1234jb",
// "table": "mytable123"
// }
type POSTRequest struct {
	Url     string `json:"url"`
	Account string `json:"account"`
	Table   string `json:"table"`
}

func ImportEnv(filename string) {
	filePtr, err := os.Open(filename)
	if err != nil {
		panic(err)
	}
	defer filePtr.Close()

	scanner := bufio.NewScanner(filePtr)
	for scanner.Scan() {
		envVar := strings.Split(scanner.Text(), "=")
		key, value := envVar[0], envVar[1]
		value = strings.TrimPrefix(value, "\"")
		value = strings.TrimSuffix(value, "\"")

		switch key {
		case "AZURE_TENANT_ID":
			tenantid = value
		case "AZURE_ACCOUNT":
			account = value
		case "AZURE_SECRET":
			secret = value
		}
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}
}

func handleRequest(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method == "GET" {
		w.Write([]byte("hello world"))
	} else if r.Method == "POST" {
		var buf bytes.Buffer
		var postRequest POSTRequest
		buflog := log.New(&buf, "[buf:]", log.LstdFlags)

		if r.Body == nil {
			buflog.Printf("Body is nil")
			w.Write([]byte("Body is nil"))
		}
		data, err := io.ReadAll(r.Body)
		if err != nil {
			buflog.Printf("Error during reading of request body: %v\n", err)
			w.Write(buf.Bytes())
		}
		err = json.Unmarshal(data, &postRequest)
		if err != nil {
			buflog.Printf("Wrong request format: %v\n", err)
			w.Write(buf.Bytes())
		}
		w.Write([]byte(postRequest.Url + " " + postRequest.Table + " " + postRequest.Account))

		feedURL := postRequest.Url
		resp, err := http.Get(feedURL)
		if err != nil {
			buflog.Printf("Fail to fetch data: %v\n", err)
			w.Write(buf.Bytes())
		}
		defer resp.Body.Close()

		feed := core.AtomFormat{}
		decoder := xml.NewDecoder(resp.Body)
		err = decoder.Decode(&feed)
		if err != nil {
			buflog.Printf("Error during paring: %v\n", err)
			w.Write(buf.Bytes())
		}

		ImportEnv("./.env")
		context := context.Background()
		credentials, err := azidentity.NewClientSecretCredential(tenantid, account, secret, nil)
		if err != nil {
			buflog.Println(err)
			w.Write(buf.Bytes())
		}

		items := feed.AtomChannel.AtomEntries
		tableClient := core.GetTable(credentials, "https://"+postRequest.Account+".table.cosmos.azure.com", postRequest.Table)
		for _, item := range items {
			time, err := time.Parse(time.RFC1123Z, item.Date)
			if err != nil {
				buflog.Printf("Problem with parsing: %v\n", err)
				w.Write(buf.Bytes())
			}
			err = core.InsertData(context, tableClient, core.News{
				Title:       item.Title,
				Date:        time,
				Description: item.Description.Data,
			})
			if err != nil {
				buflog.Printf("Failed to insert: %v\n", err)
				w.Write(buf.Bytes())
			}
		}
		w.Write([]byte("Succeed!!!"))
	}
}

func main() {
	customHandlerPort, exists := os.LookupEnv("FUNCTIONS_CUSTOMHANDLER_PORT")
	if !exists {
		fmt.Println("Variable: FUNCTIONS_CUSTOMHANDLER_PORT not exists!")
		customHandlerPort = "8080"
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/", handleRequest)
	fmt.Println("Go server Listening on: ", customHandlerPort)
	err := http.ListenAndServe(":"+customHandlerPort, mux)
	if err != nil {
		log.Fatal(err)
	}
}
