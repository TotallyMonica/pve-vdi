package main

import (
	"crypto/tls"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

func main() {
	if _, keyExists := os.LookupEnv("SSLKEYLOGFILE"); keyExists {
		keyLogFile, err := os.OpenFile("tls_key_log.log", os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
		if err != nil {
			log.Fatalf("Failed to open key log file: %v", err)
		}

		defer func(keyLogFile *os.File) {
			err := keyLogFile.Close()
			if err != nil {
				log.Fatalf("Error while closing key log file: %+v\n", err)
			}
		}(keyLogFile)

		client = &http.Client{
			Timeout: 10 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true, KeyLogWriter: keyLogFile},
			},
		}
	}

	creds, err := login()
	if err != nil {
		log.Fatalf("Error while getting Proxmox credentials: %+v\n", err)
	}
	token, err := connectToProxmox(creds)
	if err != nil {
		log.Fatalf("Error while logging into Proxmox: %+v\n", err)
	}
	vms, err := getAvailableVMList(creds, token)
	if err != nil {
		log.Fatalf("Error while getting available VMs: %+v\n", err)
	}

	fmt.Printf("Enter the number of the VM you'd like to connect to:\n")
	for _, vm := range vms.Data {
		if strings.Contains(vm.Type, "qemu") {
			fmt.Printf("%s: %s\n", strings.Split(vm.Id, "/")[1], vm.Name)
		}
	}

	var id int
	_, err = fmt.Scanf("%04d", &id)
	if err != nil {
		log.Fatalf("Error while parsing user input: %+v\n", err)
	}

	for _, vm := range vms.Data {
		if strings.Contains(vm.Id, strconv.Itoa(id)) {
			err = connectToSpice(creds, token, vm, id)
			if err != nil {
				log.Fatalf("Could not connect to spice client: %+v\n", err)
			}
			break
		}
	}

	// Create USB redirect rules
	//redirectRules := make([]string, 0)

	// Block USB HID devices
	//redirectRules = append(redirectRules, "0x03,-1,-1,-1,0")

	// Block USB Hubs
	//redirectRules = append(redirectRules, "0x09,-1,-1,-1,0")

	// Allow all USB devices
	//redirectRules = append(redirectRules, "-1,-1,-1,-1,1")

	// Kiosk Mode
	vdiArgs := make([]string, 0)

	// Redirect USB rules: Block any HID device from being redirected, allow everything else
	//vdiArgs = append(vdiArgs, fmt.Sprintf("--spice-usbredir-auto-redirect-filter=%s", strings.Join(redirectRules, "|")))

	// Kiosk mode - Don't allow user to configure anything
	vdiArgs = append(vdiArgs, "-k", "--kiosk-quit", "on-disconnect")

	// Full screen, but allow user to configure
	//vdiArgs = append(vdiArgs, "-f")

	vdiArgs = append(vdiArgs, os.Getenv("VDI_TEMPFILE_FILENAME"))
	cmd := exec.Command("remote-viewer", vdiArgs...)

	if errors.Is(cmd.Err, exec.ErrDot) {
		cmd.Err = nil
	}
	if err := cmd.Run(); err != nil {
		log.Fatalf("Error while executing thin client profile: %+v\n", err)
	}
}
