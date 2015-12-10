package main

import (
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"
	//"bytes"
	"encoding/hex"
)

type Account struct {
	AccountId      string
	ApplicationKey string
}

type Bucket struct {
	BucketId string
}

type Error struct {
	Code    string
	Message string
	Status  int
}

type Session struct {
	AccountId          string
	ApiUrl             string
	AuthorizationToken string
	DownloadUrl        string
}

type Upload struct {
	BucketId           string
	UploadUrl          string
	AuthorizationToken string
}

type UploadFile struct {
	fileName string
	filePath string
	fileType string
	fileSha1 string
	fileSize string
}

type UploadResponse struct {
	FileId        string
	FileName      string
	AccountId     string
	BucketId      string
	ContentLength int
	ContentSha1   string
	ContentType   string
	FileInfo      interface{}
}

func readJSON(r interface{}, v interface{}) error {
	var parsee interface{}
	var err error

	switch r.(type) {
	case string:
		file, err := os.Open(r.(string))
		if err != nil {
			return err
		}
		defer file.Close()
		parsee = file
	case io.Reader:
		parsee = r
	}

	err = json.NewDecoder(parsee.(io.Reader)).Decode(&v)
	if err != nil {
		return err
	}

	return nil
}

func apiRequest(method string, headers map[string]string, url string, v interface{}, body string) error {
	// setup the request
	req, err := http.NewRequest(method, url, strings.NewReader(body))
	if err != nil {
		return err
	}

	// set headers from map
	for k, m := range headers {
		fmt.Println("Adding Header: ", k, " => ", m)
		req.Header.Set(k, m)
	}

	fmt.Println("Requesting: ", url)

	// send the request
	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	// return parsed information on success, or output error message on failure
	switch res.StatusCode {
	case 200:
		fmt.Println("Status: ", res.Status)
		// parse the response
		err = readJSON(res.Body, &v)
		if err != nil {
			return err
		}
	default:
		fmt.Println("Status: ", res.Status)
		var e Error
		err = readJSON(res.Body, &e)
		if err != nil {
			return err
		}
		fmt.Printf("%+v", e)
	}

	return nil
}

func main() {
	authURL := "https://api.backblaze.com"
	apiPath := "/b2api/v1"
	headers := make(map[string]string)

	///////////////////////////////////
	// GET Initial Authorization Token
	///////////////////////////////////

	// read in the account information
	var account Account
	err := readJSON("account.json", &account)
	if err != nil {
		fmt.Println("error: ", err)
		return
	}

	//build credentials
	credentials := base64.StdEncoding.EncodeToString([]byte(account.AccountId + ":" + account.ApplicationKey))
	headers["Authorization"] = "Basic " + credentials

	// get authorized
	var session Session
	err = apiRequest("GET", headers, authURL+apiPath+"/b2_authorize_account", &session, "")
	if err != nil {
		fmt.Println("error: ", err)
	}

	fmt.Printf("%+v", session)
	fmt.Println()
	fmt.Println("---")

	///////////////////
	// POST Upload URL
	///////////////////

	// put the authorization token in the header
	headers["Authorization"] = session.AuthorizationToken

	// read the bucket ID from file
	bucketJSON, err := ioutil.ReadFile("bucket.json")

	// get upload information
	var upload Upload
	err = apiRequest("POST", headers, session.ApiUrl+apiPath+"/b2_get_upload_url", &upload, string(bucketJSON))
	if err != nil {
		fmt.Println("error: ", err)
	}

	fmt.Printf("%+v", upload)
	fmt.Println()
	fmt.Println("---")

	////////////////////
	// POST Upload File
	////////////////////

	uploadFile := UploadFile{fileName: "background.png", filePath: "background.png", fileType: "image/png", fileSha1: "", fileSize: ""}

	// open upload file
	file, err := os.Open(uploadFile.filePath)
	if err != nil {
		fmt.Println("error: ", err)
	}
	defer file.Close()

	//get file size
	fileInfo, _ := file.Stat()
	uploadFile.fileSize = strconv.FormatInt(fileInfo.Size(), 10)

	//get sha1 of file contents
	fileContent, err := ioutil.ReadAll(file)
	if err != nil {
		fmt.Println("error: ", err)
	}
	hash := sha1.New()
	hash.Write(fileContent)
	uploadFile.fileSha1 = hex.EncodeToString(hash.Sum(nil))

	// build the headers
	headers["Authorization"] = upload.AuthorizationToken
	headers["X-Bz-File-Name"] = uploadFile.filePath
	headers["Content-Type"] = uploadFile.fileType
	headers["Content-Length"] = uploadFile.fileSize
	headers["X-Bz-Content-Sha1"] = uploadFile.fileSha1

	fmt.Printf("%+v", headers)
	fmt.Println()

	var uploadResponse UploadResponse
	err = apiRequest("POST", headers, upload.UploadUrl, &uploadResponse, string(fileContent))
	if err != nil {
		fmt.Println("error: ", err)
	}

	fmt.Printf("%+v", uploadResponse)
	fmt.Println()
}
