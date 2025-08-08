package main

import (
	"fmt"
	//"fmt"
	"github.com/mappu/miqt/qt6"
	"log"
	"strconv"
	"strings"

	//"log"
	"os"
	//"strconv"
	//"strings"
)

func buildWindow(vms ProxmoxVmList, creds ProxmoxCreds, token ProxmoxAuth) {
	qt6.NewQApplication(os.Args)

	// Create the home widget
	homeWidget := qt6.NewQMainWindow2()
	defer homeWidget.Delete()
	homeWidget.SetWindowTitle("Proxmox VDI Client")

	// Build the layout
	mainWindowLayout := qt6.NewQVBoxLayout2()

	// Create header and add to layout
	header := qt6.NewQLabel3("Choose the VM that you would like to connect to")
	header.Show()
	mainWindowLayout.AddWidget(header.QWidget)
	mainWindowLayout.AddSpacing(header.Height() * 2)

	// Create container widget
	testWidget := qt6.NewQWidget(homeWidget.QWidget)
	testWidget.SetLayout(mainWindowLayout.Layout())

	// Create a button for every VM
	for _, vm := range vms.Data {
		// Ensure it's actually a VM
		if strings.Contains(vm.Type, "qemu") {
			var err error

			// Create the button with the text as the name of the VM
			vmButton := qt6.NewQPushButton3(vm.Name)
			vm.VmNumber, err = strconv.Atoi(strings.Split(vm.Id, "/")[1])
			if err != nil {
				log.Fatalf("Error while parsing VM ID %s: %+v\n", vm.Id, err)
			}

			// Start the VM (if necessary) and connect to the VM via SPICE.
			vmButton.OnClicked(func() {
				err := startVM(creds, token, vm, vm.VmNumber)
				if err != nil {
					return
				}
				// Create the child window
				fmt.Printf("Connecting to %s\n", vmButton.Text())
				connectingWindow := qt6.NewQWidget2()
				defer connectingWindow.Delete()
				connectingWindow.SetWindowTitle(fmt.Sprintf("Connecting to %s", vmButton.Text()))

				connectingLayout := qt6.NewQVBoxLayout2()

				// Set connecting container widget settings
				connectingWidget := qt6.NewQWidget(connectingWindow)
				connectingWidget.SetLayout(connectingLayout.QLayout)

				// Set window presentation settings
				homeWidget.SetCentralWidget(connectingWidget)

				// Build the layout for the child window
				statusLabel := qt6.NewQLabel2()
				vmNameLabel := qt6.NewQLabel2()

				// Create the VM Name label
				vmNameLabel.SetText(fmt.Sprintf("Virtual desktop chosen: %s\n", vmButton.Text()))
				vmNameLabel.Show()
				connectingLayout.AddWidget(vmNameLabel.QWidget)
				connectingLayout.AddSpacing(vmNameLabel.Height())

				statusLabel.SetText("Status: Starting")
				statusLabel.Show()
				connectingLayout.AddWidget(statusLabel.QWidget)
				connectingLayout.AddSpacing(statusLabel.Height())

				// Create the status label for the child window
				for status, err := getVmHealth(creds, token, vm); err != nil && !strings.Contains(status, "200 OK"); status, err = getVmHealth(creds, token, vm) {
					statusLabel.SetText("Status: Starting")
					statusLabel.Show()
					connectingLayout.AddWidget(statusLabel.QWidget)
					connectingLayout.AddSpacing(statusLabel.Height())
				}

				if err != nil {
					statusLabel.SetText(fmt.Sprintf("Error: %s\n", err))
				}

				statusLabel.SetText("Started!")

				err = connectToSpice(creds, token, vm, vm.VmNumber)

				if err != nil {
					statusLabel.SetText(fmt.Sprintf("Couldn't connect to VM: %s\n", err))
				}
			})

			// Add the button to the layout
			vmButton.SetFixedWidth(320)
			mainWindowLayout.AddWidget(vmButton.QWidget)
		}
	}

	// Show the window
	homeWidget.SetCentralWidget(testWidget)
	homeWidget.Show()
	qt6.QApplication_Exec()
}
