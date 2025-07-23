package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"
)

type ProxmoxCreds struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Server   string `json:"host"`
	Address  string `json:"server"`
}

func login() ProxmoxCreds {
	// Open credentials file
	credsHandler, err := os.Open("creds.json")
	if err != nil {
		log.Fatalf("error while openings creds file: %+v\n", err)
	}
	defer credsHandler.Close()

	// Cread credentials data
	credsData, err := io.ReadAll(credsHandler)
	if err != nil {
		log.Fatalf("error while reading credentials: %+v\n", err)
	}

	// Parse credentials
	var creds ProxmoxCreds
	err = json.Unmarshal(credsData, &creds)
	if err != nil {
		log.Fatalf("error while unmarshalling json: %+v\n", err)
	}

	return creds
}

func connectToProxmox(creds ProxmoxCreds) {
	client := &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
	data := url.Values{}
	data.Set("username", creds.Username)
	data.Set("password", creds.Password)
	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("https://%s:8006/api2/json/access/ticket", creds.Address), bytes.NewBufferString(data.Encode()))
	if err != nil {
		fmt.Errorf("error while creating request: %+v\n", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("error while performing request: %+v\n", err)
	}

	token, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("Error while parsing response: %+v\n", err)
	}

	fmt.Printf("Response: %s\n", token)
}

func main() {
	creds := login()
	connectToProxmox(creds)
}
