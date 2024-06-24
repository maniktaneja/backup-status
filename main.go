package main

import (
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
)

type AccessTokenResponse struct {
	AccessToken string   `json:"accessToken"`
	TokenType   string   `json:"tokenType"`
	Privileges  []string `json:"privileges"`
}

type ProtectionGroup struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	LastRun *Run   `json:"lastRun"`
}

type Run struct {
	ID                        string      `json:"id"`
	ProtectionGroupInstanceID int         `json:"protectionGroupInstanceId"`
	IsReplicationRun          bool        `json:"isReplicationRun"`
	LocalBackupInfo           *BackupInfo `json:"localBackupInfo"`
}

type BackupInfo struct {
	RunType                   string        `json:"runType"`
	IsSlaViolated             bool          `json:"isSlaViolated"`
	StartTimeUsecs            int64         `json:"startTimeUsecs"`
	EndTimeUsecs              int64         `json:"endTimeUsecs"`
	Status                    string        `json:"status"`
	Messages                  []string      `json:"messages"`
	SuccessfulObjectsCount    int           `json:"successfulObjectsCount"`
	FailedObjectsCount        int           `json:"failedObjectsCount"`
	CancelledObjectsCount     int           `json:"cancelledObjectsCount"`
	SkippedObjectsCount       int           `json:"skippedObjectsCount"`
	SuccessfulAppObjectsCount int           `json:"successfulAppObjectsCount"`
	FailedAppObjectsCount     int           `json:"failedAppObjectsCount"`
	CancelledAppObjectsCount  int           `json:"cancelledAppObjectsCount"`
	LocalSnapshotStats        SnapshotStats `json:"localSnapshotStats"`
	IndexingTaskID            string        `json:"indexingTaskId"`
	LocalTaskID               string        `json:"localTaskId"`
}

type SnapshotStats struct {
	LogicalSizeBytes int64 `json:"logicalSizeBytes"`
	BytesWritten     int64 `json:"bytesWritten"`
	BytesRead        int64 `json:"bytesRead"`
}

type Response struct {
	ProtectionGroups []ProtectionGroup `json:"protectionGroups"`
}

func main() {

	// Define the IP address flag
	ipPtr := flag.String("ip", "0.0.0.0", "IP address of the server")
	username := flag.String("username", "admin", "Username")
	password := flag.String("password", "admin", "Password")
	domain := flag.String("domain", "LOCAL", "Default domain")
	flag.Parse()

	// First request to get the access token
	url := fmt.Sprintf("https://%s/irisservices/api/v1/public/accessTokens", *ipPtr)
	method := "POST"

	payload := strings.NewReader(fmt.Sprintf(`{
		"password": "%s",
		"username": "%s",
		"domain": "%s"
	}`, *password, *username, *domain))

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	req, err := http.NewRequest(method, url, payload)
	if err != nil {
		fmt.Println(err)
		return
	}
	req.Header.Add("Content-Type", "application/json")

	res, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		fmt.Println(err)
		return
	}

	var tokenResponse AccessTokenResponse
	err = json.Unmarshal(body, &tokenResponse)
	if err != nil {
		fmt.Println(err)
		return
	}
	accessToken := tokenResponse.AccessToken

	// Second request using the access token
	url = fmt.Sprintf("https://%s/v2/data-protect/protection-groups?useCachedData=false&pruneSourceIds=true&pruneExcludedSourceIds=true&isDeleted=false&includeTenants=true&includeLastRunInfo=true", *ipPtr)

	method = "GET"

	req, err = http.NewRequest(method, url, nil)
	if err != nil {
		fmt.Println(err)
		return
	}
	req.Header.Add("Authorization", "Bearer "+accessToken)

	res, err = client.Do(req)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer res.Body.Close()

	body, err = ioutil.ReadAll(res.Body)
	if err != nil {
		fmt.Println(err)
		return
	}

	var response Response
	err = json.Unmarshal(body, &response)
	if err != nil {
		fmt.Println(err)
		return
	}

	// Check each protection group for the "SucceededWithWarning" status
	for _, group := range response.ProtectionGroups {
		if group.LastRun != nil && group.LastRun.LocalBackupInfo != nil && group.LastRun.LocalBackupInfo.Status == "SucceededWithWarning" {
			fmt.Printf("Protection Group ID: %s, Name: %s\n", group.ID, group.Name)
			for _, message := range group.LastRun.LocalBackupInfo.Messages {
				fmt.Printf("  Message: %s\n", message)
			}
		}
	}

}
