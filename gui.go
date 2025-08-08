package main

import (
	"fmt"
	"github.com/mappu/miqt/qt6"
	"log"
	"os"
	"strconv"
	"strings"
)

func buildWindow(vms ProxmoxVmList, creds ProxmoxCreds, token ProxmoxAuth) {
	qt6.NewQApplication(os.Args)

	// Create the home widget
	homeWidget := qt6.NewQMainWindow2()
	defer homeWidget.Delete()
	homeWidget.SetWindowTitle("Proxmox VDI Client")

	// Build the layout
	vbox := qt6.NewQVBoxLayout2()

	// Create header and add to layout
	header := qt6.NewQLabel3("Choose the VM that you would like to connect to")
	header.Show()
	vbox.AddWidget(header.QWidget)
	vbox.AddSpacing(header.Height() * 2)

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
				// TODO: Spawn a child window giving the status
				// Create the child window
				fmt.Printf("Connecting to %s\n", vmButton.Text())
				connectingWindow := qt6.NewQMainWindow(homeWidget.QWidget)
				connectingWidget := qt6.NewQWidget(connectingWindow.QWidget)
				defer connectingWidget.Delete()

				// Build the layout for the child window
				layout := qt6.NewQVBoxLayout(connectingWidget)
				statusLabel := qt6.NewQLabel2()
				vmNameLabel := qt6.NewQLabel2()

				// Create the VM Name label
				vmNameLabel.SetText(fmt.Sprintf("Virtual desktop chosen: %s\n", vmButton.Text()))
				vmNameLabel.Show()
				layout.AddChildWidget(vmNameLabel.QWidget)
				layout.AddSpacing(vmNameLabel.Height() * 2)

				// Create the status label for the child window
				statusLabel.SetText("Status: Broken")
				statusLabel.Show()
				layout.AddChildWidget(statusLabel.QWidget)
				layout.AddSpacing(statusLabel.Height() * 2)

				// Set window presentation settings
				connectingWindow.SetWindowTitle(fmt.Sprintf("Connecting to %s", vmButton.Text()))
				connectingWindow.SetCentralWidget(connectingWidget)
				connectingWindow.Show()
			})

			// Add the button to the layout
			vmButton.SetFixedWidth(320)
			vbox.AddWidget(vmButton.QWidget)
		}
	}

	// Create container widget
	testWidget := qt6.NewQWidget(homeWidget.QWidget)
	testWidget.SetLayout(vbox.Layout())

	// Show the window
	homeWidget.SetCentralWidget(testWidget)
	homeWidget.Show()
	qt6.QApplication_Exec()
}
