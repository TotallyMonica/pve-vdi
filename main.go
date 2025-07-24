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
	"strings"
	"time"
)

type ProxmoxCreds struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Server   string `json:"host"`
	Address  string `json:"server"`
}

type ProxmoxAuth struct {
	Data struct {
		CSRF   string `json:"CSRFPreventionToken"`
		Ticket string `json:"ticket"`
	} `json:"data"`
}

type ProxmoxVmList struct {
	Data []struct {
		Id     string `json:"id"`
		Status string `json:"status"`
		Name   string `json:"name"`
		Node   string `json:"node"`
	}
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

func connectToProxmox(creds ProxmoxCreds) ProxmoxAuth {
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

	var parsedResponse ProxmoxAuth

	err = json.Unmarshal(token, &parsedResponse)
	if err != nil {
		log.Fatalf("Error while unmarshalling response: %+v\n", err)
	}

	return parsedResponse
}

func getAvailableVMList(creds ProxmoxCreds, token ProxmoxAuth) ProxmoxVmList {
	client := &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	authCookie := &http.Cookie{
		Name:  "PVEAuthCookie",
		Value: token.Data.Ticket,
	}

	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("https://%s:8006/api2/json/cluster/resources/?type=vm", creds.Address), nil)
	if err != nil {
		fmt.Errorf("error while creating request: %+v\n", err)
	}

	req.AddCookie(authCookie)
	req.Header.Add("CSRFPreventionToken", token.Data.CSRF)

	resp, _ := client.Do(req)
	response, _ := io.ReadAll(resp.Body)

	var availableVMs ProxmoxVmList
	_ = json.Unmarshal(response, &availableVMs)

	return availableVMs
}

func main() {
	creds := login()
	token := connectToProxmox(creds)
	vms := getAvailableVMList(creds, token)

	fmt.Printf("Enter the number of the VM you'd like to connect to:\n")
	for _, vm := range vms.Data {
		fmt.Printf("%s: %s\n", strings.Split(vm.Id, "/")[1], vm.Name)
	}

	var id int
	_, _ = fmt.Scanf("%04d", &id)
	fmt.Printf("Read: %d\n", id)
}
