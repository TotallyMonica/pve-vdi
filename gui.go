package main

import (
	"fmt"
	"github.com/mappu/miqt/qt6"
	"log"
	"os"
	"strconv"
	"strings"
)

func connectingWindow(vmName string) *qt6.QWidget {
	// Create the child window
	connectingWidget := qt6.NewQWidget2()
	defer connectingWidget.Delete()

	// Build the layout for the child window
	layout := qt6.NewQVBoxLayout(connectingWidget)
	statusLabel := qt6.NewQLabel2()

	// Create the label for the child window
	statusLabel.SetText("Status: Broken")

	// Add the widgets with appropriate spacing
	layout.AddChildWidget(statusLabel.QWidget)

	// Set window presentation settings
	connectingWidget.SetWindowTitle(fmt.Sprintf("Connecting to %s", vmName))

	return connectingWidget
}

func buildWindow(vms ProxmoxVmList, creds ProxmoxCreds, token ProxmoxAuth) {
	qt6.NewQApplication(os.Args)
	var err error

	// Create the home widget
	homeWidget := qt6.NewQWidget2()
	defer homeWidget.Delete()
	homeWidget.SetWindowTitle("Proxmox VDI Client")

	// Build the layout
	vbox := qt6.NewQVBoxLayout(homeWidget)

	// Create header and add to layout
	header := qt6.NewQLabel3("Choose the VM that you would like to connect to")
	header.Show()
	vbox.AddChildWidget(header.QWidget)
	vbox.AddSpacing(header.Height())

	// Create a button for every VM
	for _, vm := range vms.Data {
		// Ensure it's actually a VM
		if strings.Contains(vm.Type, "qemu") {
			// Create the button with the text as the name of the VM
			vmButton := qt6.NewQPushButton3(vm.Name)
			vm.VmNumber, err = strconv.Atoi(strings.Split(vm.Id, "/")[1])
			if err != nil {
				log.Fatalf("Error while parsing VM ID %s: %+v\n", vm.Id, err)
			}

			// Start the VM (if necessary) and connect to the VM via SPICE.
			vmButton.OnPressed(func() {
				// TODO: Spawn a child window giving the status
				connecting := connectingWindow(vmButton.Text())
				connecting.Show()
			})

			// Add the button to the layout
			vmButton.SetFixedWidth(320)
			vbox.AddWidget(vmButton.QWidget)
		}
	}

	// Show the window
	homeWidget.Show()
	qt6.QApplication_Exec()
}
