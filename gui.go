package main

import (
	"errors"
	"fmt"
	"net/netip"
	"os/exec"

	"log"
	"strconv"
	"strings"

	//"fmt"
	"github.com/mappu/miqt/qt6"

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
		// Ensure it's actually a VM we should present to the user
		if strings.Contains(vm.Type, "qemu") {
			var err error

			// Create the button with the text as the name of the VM
			vmButton := qt6.NewQPushButton3(vm.Name)
			vmid, err := strconv.ParseInt(strings.Split(vm.Id, "/")[1], 10, 32)
			if err != nil {
				log.Fatalf("Error while parsing VM ID %s: %+v\n", vm.Id, err)
			}
			vm.VmNumber = int32(vmid)

			// Start the VM (if necessary) and connect to vm.VmNumber the VM via SPICE.
			vmButton.OnClicked(func() {
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

				clonedVm, job, err := cloneTemplate(creds, token, vm)
				if err != nil {
					statusLabel.SetText(fmt.Sprintf("Error: %v\n", err))
					log.Fatalf("Error while cloning VM: %v\n", err)
				}

				for status, err := getJobStatus(creds, token, job); err != nil && strings.Compare(status.Status, "stopped") == 0; status, err = getJobStatus(creds, token, job) {
					fmt.Printf("Status: %s\n", status)
					statusLabel.SetText("Status: Cloning")
					statusLabel.Show()
					connectingLayout.AddWidget(statusLabel.QWidget)
					connectingLayout.AddSpacing(statusLabel.Height())
				}

				err = startVM(creds, token, clonedVm)
				if err != nil {
					return
				}

				// Create the status label for the child window
				for status, err := getVmHealth(creds, token, clonedVm); err != nil && strings.Contains(status, "200 OK"); status, err = getVmHealth(creds, token, clonedVm) {
					fmt.Printf("Status: %s\n", status)
					statusLabel.SetText("Status: Starting")
					statusLabel.Show()
					connectingLayout.AddWidget(statusLabel.QWidget)
					connectingLayout.AddSpacing(statusLabel.Height())
				}

				if err != nil {
					statusLabel.SetText(fmt.Sprintf("Error: %s\n", err))
				}

				specifiedNode, err := login()
				var specifiedToken ProxmoxAuth

				if err != nil {
					statusLabel.SetText(fmt.Sprintf("Error: %s\n", err))
				}

				// Check if the node the VM is on is the same one as we're logging into
				if strings.Compare(specifiedNode.Server, clonedVm.Node) != 0 {
					// Get the network of the first node
					originalNodeAddrs, err := getNodeAddresses(creds, token)
					if err != nil {
						statusLabel.SetText(fmt.Sprintf("Error: %s\n", err))
						log.Fatalf("Error while getting the IP addresses for Node %s: %+v\n", creds.Server, err)
					}

					// Try to find the original IP that we were given for the original node
					var network netip.Prefix
					for _, addr := range originalNodeAddrs {
						if strings.Compare(addr.Address, creds.Address) == 0 {
							network = netip.MustParsePrefix(addr.Cidr)
						}
					}

					// Now get the new node's addresses
					specifiedNode.Server = clonedVm.Node
					newNodeAddrs, err := getNodeAddresses(specifiedNode, token)
					if err != nil {
						statusLabel.SetText(fmt.Sprintf("Error: %s\n", err))
						log.Fatalf("Error while getting the IP addresses for Node %s: %+v\n", specifiedNode.Server, err)
					}

					// Compare each of the new node's addresses and see if they are in the original node's network
					for _, addr := range newNodeAddrs {
						if strings.Compare(addr.Address, "") != 0 && network.Contains(netip.MustParseAddr(addr.Address)) {
							specifiedNode.Address = addr.Address
						}
					}

					specifiedToken, err = connectToProxmox(specifiedNode)
					if err != nil {
						statusLabel.SetText(fmt.Sprintf("Error: %s\n", err))
						log.Fatalf("Error while connecting to the node %s: %+v\n", specifiedNode.Server, err)
					}
				} else {
					specifiedToken = token
				}

				statusLabel.SetText("Started!")

				// Log in with the required node's credentials
				err = connectToSpice(specifiedNode, specifiedToken, clonedVm)

				if err != nil {
					statusLabel.SetText(fmt.Sprintf("Couldn't connect to VM: %s\n", err))
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
				vdiArgs = append(vdiArgs, "-f")

				vdiArgs = append(vdiArgs, os.Getenv("VDI_TEMPFILE_FILENAME"))
				cmd := exec.Command("remote-viewer", vdiArgs...)

				if errors.Is(cmd.Err, exec.ErrDot) {
					cmd.Err = nil
				}

				if err := cmd.Run(); err != nil {
					statusLabel.SetText(fmt.Sprintf("Couldn't connect to VM: %s\n", err))
					log.Fatalf("Error while executing thin client profile: %+v\n", err)
				}

				qt6.QCoreApplication_Exit()
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
