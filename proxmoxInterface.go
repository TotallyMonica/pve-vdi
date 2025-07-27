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
	Server   string `json:"node"`
	Address  string `json:"proxy"`
}

type ProxmoxHost struct {
	Name    string
	Address string
}

type ProxmoxAuth struct {
	Data struct {
		CSRF   string `json:"CSRFPreventionToken"`
		Ticket string `json:"ticket"`
	} `json:"data"`
}

type ProxmoxVm struct {
	Id       string `json:"id"`
	Status   string `json:"status"`
	Name     string `json:"name"`
	Node     string `json:"node"`
	Type     string `json:"type"`
	VmNumber int
}

type ProxmoxVmList struct {
	Data []ProxmoxVm
}

var client = &http.Client{
	Timeout: 10 * time.Second,
	Transport: &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	},
}

func login() (ProxmoxCreds, error) {
	var creds ProxmoxCreds

	// Open credentials file
	credsHandler, err := os.Open("creds.json")
	if err != nil {
		return ProxmoxCreds{}, fmt.Errorf("error while openings creds file: %+v\n", err)
	}
	defer func(credsHandler *os.File) {
		err := credsHandler.Close()
		if err != nil {
			log.Fatalf("error while closing login credentials handler: %+v\n", err)
		}
	}(credsHandler)

	// Cread credentials data
	credsData, err := io.ReadAll(credsHandler)
	if err != nil {
		return ProxmoxCreds{}, fmt.Errorf("error while reading credentials: %+v\n", err)
	}

	// Parse credentials
	err = json.Unmarshal(credsData, &creds)
	if err != nil {
		return ProxmoxCreds{}, fmt.Errorf("error while unmarshalling json: %+v\n", err)
	}

	return creds, nil
}

func connectToProxmox(creds ProxmoxCreds) (ProxmoxAuth, error) {
	data := url.Values{}
	data.Set("username", creds.Username)
	data.Set("password", creds.Password)
	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("https://%s:8006/api2/json/access/ticket", creds.Address), bytes.NewBufferString(data.Encode()))
	if err != nil {
		log.Fatalf("error while creating request: %+v\n", err)
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

	return parsedResponse, nil
}

func getAvailableVMList(creds ProxmoxCreds, token ProxmoxAuth) (ProxmoxVmList, error) {
	authCookie := &http.Cookie{
		Name:  "PVEAuthCookie",
		Value: token.Data.Ticket,
	}

	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("https://%s:8006/api2/json/cluster/resources/", creds.Address), nil)
	if err != nil {
		return ProxmoxVmList{}, fmt.Errorf("error while creating request: %+v\n", err)
	}

	req.AddCookie(authCookie)
	req.Header.Add("CSRFPreventionToken", token.Data.CSRF)

	resp, err := client.Do(req)
	if err != nil {
		return ProxmoxVmList{}, fmt.Errorf("error while performing request: %+v\n", err)
	}

	response, err := io.ReadAll(resp.Body)
	if err != nil {
		return ProxmoxVmList{}, fmt.Errorf("error while reading response: %+v\n", err)
	}

	var availableVMs ProxmoxVmList
	err = json.Unmarshal(response, &availableVMs)
	if err != nil {
		return ProxmoxVmList{}, fmt.Errorf("error while unmarshalling json: %+v\n", err)
	}

	return availableVMs, nil
}

func getVmHealth(creds ProxmoxCreds, token ProxmoxAuth, vm ProxmoxVm) (string, error) {
	authCookie := &http.Cookie{
		Name:  "PVEAuthCookie",
		Value: token.Data.Ticket,
	}

	apiUrl := fmt.Sprintf("https://%s:8006/api2/json/nodes/%s/%s/agent/ping", creds.Address, vm.Node, vm.Id)

	req, err := http.NewRequest(http.MethodPost, apiUrl, nil)
	if err != nil {
		return "", fmt.Errorf("error while creating request: %+v\nurl: %s\n", err, apiUrl)
	}

	req.AddCookie(authCookie)
	req.Header.Add("CSRFPreventionToken", token.Data.CSRF)
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("error while performing request: %+v\n", err)
	}
	fmt.Printf("Status from VM %s: %s\n", vm.Id, resp.Status)

	if resp.StatusCode == 500 && strings.Contains(resp.Status, "is not running") {
		return getVmHealth(creds, token, vm)
	} else if resp.StatusCode != 200 {
		return "", fmt.Errorf("unexpected status code %d from response: %s\n", resp.StatusCode, resp.Status)
	}

	return "", nil
}

func startVM(creds ProxmoxCreds, token ProxmoxAuth, vm ProxmoxVm, id int) error {
	authCookie := &http.Cookie{
		Name:  "PVEAuthCookie",
		Value: token.Data.Ticket,
	}

	apiUrl := fmt.Sprintf("https://%s:8006/api2/json/nodes/%s/%s/status/start", creds.Address, vm.Node, vm.Id)

	req, err := http.NewRequest(http.MethodPost, apiUrl, nil)
	if err != nil {
		return fmt.Errorf("error while creating request: %+v\nurl: %s\n", err, apiUrl)
	}

	req.AddCookie(authCookie)
	req.Header.Add("CSRFPreventionToken", token.Data.CSRF)
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error while performing request: %+v\n", err)
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("unexpected status code %d received: %s\nurl: %s\n", resp.StatusCode, resp.Status, apiUrl)
	}

	return nil
}

func connectToSpice(creds ProxmoxCreds, token ProxmoxAuth, vm ProxmoxVm, id int) error {
	authCookie := &http.Cookie{
		Name:  "PVEAuthCookie",
		Value: token.Data.Ticket,
	}

	data := url.Values{}
	data.Add("proxy", creds.Address)

	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("https://%s:8006/api2/spiceconfig/nodes/%s/qemu/%d/spiceproxy", creds.Address, vm.Node, id), bytes.NewBufferString(data.Encode()))
	if err != nil {
		return fmt.Errorf("error while creating request: %+v\n", err)
	}

	req.AddCookie(authCookie)
	req.Header.Add("CSRFPreventionToken", token.Data.CSRF)
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error while performing request: %+v\n", err)
	}

	if resp.StatusCode == 500 && strings.Contains(resp.Status, "not running") {
		err = startVM(creds, token, vm, id)
		if err != nil {
			return fmt.Errorf("error while starting VM: %+v\n", err)
		}

		_, err = getVmHealth(creds, token, vm)
		if err != nil {
			return fmt.Errorf("error while getting health for VM: %+v\n", err)
		}

		return connectToSpice(creds, token, vm, id)
	} else if resp.StatusCode != 200 {
		return fmt.Errorf("unexpected status %d received: %s\n", resp.StatusCode, resp.Status)
	}

	response, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("error while reading request: %+v\n", err)
	}

	filename := os.Getenv("VDI_TEMPFILE_FILENAME")

	spiceHandler, err := os.OpenFile(filename, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return fmt.Errorf("error while creating file %s: %+v\n", filename, err)
	}

	_, err = spiceHandler.Write(response)
	if err != nil {
		return fmt.Errorf("error while writing connection info to %s: %+v\n", filename, err)
	}

	return nil
}
