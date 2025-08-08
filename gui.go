package main

import (
	"errors"
	"fmt"
	"github.com/mappu/miqt/qt6"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

func buildWindow(vms ProxmoxVmList, creds ProxmoxCreds, token ProxmoxAuth) {
	qt6.NewQApplication(os.Args)
	var err error

	// Create the home widget
	homeWidget := qt6.NewQWidget2()
	defer homeWidget.Delete()

	homeWidget.SetWindowTitle("Proxmox VDI Client")

	vbox := qt6.NewQVBoxLayout(homeWidget)

	header := qt6.NewQLabel3("Choose the VM that you would like to connect to")
	header.Show()
	vbox.AddChildWidget(header.QWidget)
	vbox.AddSpacing(header.Height())

	buttonList := make([]*qt6.QPushButton, 0)

	for _, vm := range vms.Data {
		if strings.Contains(vm.Type, "qemu") {
			vmButton := qt6.NewQPushButton3(vm.Name)
			vm.VmNumber, err = strconv.Atoi(strings.Split(vm.Id, "/")[1])
			if err != nil {
				log.Fatalf("Error while parsing VM ID %s: %+v\n", vm.Id, err)
			}

			vmButton.OnPressed(func() {

				err := connectToSpice(creds, token, vm, vm.VmNumber)
				if err != nil {
					log.Fatalf("Could not connect to spice client: %+v\n", err)
				}

				status, err := getVmHealth(creds, token, vm)
				fmt.Printf("Status: %s\n", status)

				for !strings.Contains(status, "200 OK") {
					//healthCheckStatus.SetText(status)
					status, err = getVmHealth(creds, token, vm)
					fmt.Printf("Status: %s\n", status)
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
				//vdiArgs = append(vdiArgs, "-k", "--kiosk-quit", "on-disconnect")

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

				qt6.QApplication_CloseAllWindows()
			})

			vmButton.SetFixedWidth(320)
			buttonList = append(buttonList, vmButton)
		}
	}

	for _, btn := range buttonList {
		vbox.AddWidget(btn.QWidget)
	}

	homeWidget.Show()
	qt6.QApplication_Exec()
}
