package main

import (
	"crypto/tls"
	"log"
	"net/http"
	"os"
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

	buildWindow(vms, creds, token)
}
