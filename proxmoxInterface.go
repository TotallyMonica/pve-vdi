package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	rand2 "math/rand/v2"
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
	VmNumber int32
}

type rawProxmoxInterfaces struct {
	Data []ProxmoxInterfaces `json:"data"`
}

type ProxmoxInterfaces struct {
	Address   string `json:"address"`
	Active    int    `json:"active"`
	Interface string `json:"iface"`
	Cidr      string `json:"cidr"`
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

	return resp.Status, nil
}

func startVM(creds ProxmoxCreds, token ProxmoxAuth, vm ProxmoxVm) error {
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

func connectToSpice(creds ProxmoxCreds, token ProxmoxAuth, vm ProxmoxVm) error {
	authCookie := &http.Cookie{
		Name:  "PVEAuthCookie",
		Value: token.Data.Ticket,
	}

	data := url.Values{}
	data.Add("proxy", creds.Address)

	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("https://%s:8006/api2/spiceconfig/nodes/%s/qemu/%d/spiceproxy", creds.Address, vm.Node, vm.VmNumber), bytes.NewBufferString(data.Encode()))
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
		err = startVM(creds, token, vm)
		if err != nil {
			return fmt.Errorf("error while starting VM: %+v\n", err)
		}

		return connectToSpice(creds, token, vm)
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

func getNodeAddresses(creds ProxmoxCreds, token ProxmoxAuth) ([]ProxmoxInterfaces, error) {
	authCookie := &http.Cookie{
		Name:  "PVEAuthCookie",
		Value: token.Data.Ticket,
	}

	apiUrl := fmt.Sprintf("https://%s:8006/api2/json/nodes/%s/network", creds.Address, creds.Server)

	req, err := http.NewRequest(http.MethodGet, apiUrl, nil)
	if err != nil {
		return nil, fmt.Errorf("error while creating request: %+v\nurl: %s\n", err, apiUrl)
	}

	req.AddCookie(authCookie)
	req.Header.Add("CSRFPreventionToken", token.Data.CSRF)

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error while performing request: %+v\n", err)
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("unexpected status code %d received: %s\nurl: %s\n", resp.StatusCode, resp.Status, apiUrl)
	}

	interfaces, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("Error while parsing response: %+v\n", err)
	}

	var parsedResponse rawProxmoxInterfaces

	err = json.Unmarshal(interfaces, &parsedResponse)
	if err != nil {
		return nil, fmt.Errorf("Error while unmarshalling response: %+v\n", err)
	}

	return parsedResponse.Data, nil
}

func cloneTemplate(creds ProxmoxCreds, token ProxmoxAuth, vm ProxmoxVm) (ProxmoxVm, error) {
	// Boilerplate create cookie
	authCookie := &http.Cookie{
		Name:  "PVEAuthCookie",
		Value: token.Data.Ticket,
	}

	var newVm ProxmoxVm
	var err error

	// This is completely stupid, but I guess the least race condition prone? Oh dear god
	for newVm.VmNumber = rand2.Int32(); newVm.VmNumber < 100000; newVm.VmNumber = rand2.Int32() {
	}

	// Create data to clone the new VM to
	data := url.Values{}
	data.Set("newid", fmt.Sprint(newVm.VmNumber))
	data.Set("storage", os.Getenv("PVE_VDI_STORAGE"))
	data.Set("pool", os.Getenv("PVE_VDI_POOL"))

	// Create POST request
	apiUrl := fmt.Sprintf("https://%s:8006/api2/json/nodes/%s/qemu/%d/clone", creds.Address, vm.Node, vm.VmNumber)
	cloneVmReq, err := http.NewRequest(http.MethodPost, apiUrl, bytes.NewBufferString(data.Encode()))
	if err != nil {
		return ProxmoxVm{}, fmt.Errorf("error while creating request: %+v\nurl: %s\n", err, apiUrl)
	}

	// Add the auth cookies to the request
	cloneVmReq.AddCookie(authCookie)
	cloneVmReq.Header.Add("CSRFPreventionToken", token.Data.CSRF)

	// Perform the request
	cloneVmResp, err := client.Do(cloneVmReq)
	if err != nil {
		return ProxmoxVm{}, fmt.Errorf("error while performing request: %+v\n", err)
	}

	if cloneVmResp.StatusCode != 200 {
		response, err := io.ReadAll(cloneVmResp.Body)
		if err != nil {
			return ProxmoxVm{}, fmt.Errorf("got unexpected status code %d: %s\n", cloneVmResp.StatusCode, cloneVmResp.Status)
		}

		// Bodge solution for generating an invalid VM ID
		if cloneVmResp.StatusCode == 400 && strings.Contains(fmt.Sprintf("%s", response), "invalid format - value does not look like a valid VM ID\\n") {
			generatedVm, err := cloneTemplate(creds, token, vm)
			return generatedVm, err
		} else {
			return ProxmoxVm{}, fmt.Errorf("got unexpected status code %d: %s: %s\n", cloneVmResp.StatusCode, cloneVmResp.Status, response)
		}
	}

	fmt.Printf("Status Code: %d\n", cloneVmResp.StatusCode)
	fmt.Printf("Status: %s\n", cloneVmResp.Status)

	return newVm, nil
}
