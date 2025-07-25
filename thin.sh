#!/bin/bash
set -e

# Set auth options
SECRET='7ac9ecd5-a664-4114-bc7f-fb0955250805'
TOKEN_ID='_pve_service_vdi@pve!vdiclient'
PASSWORD='Lhfd24XSNdsTtx9Rr85WM3QBxF5DkAX6'
USERNAME='_pve_service_vdi@pve'

# Set VM IDs
WIN11_ID="1001"
WINSRV2025_ID="1000"
VMID=$WIN11_ID

# Set Node
# This must either be a DNS address or name of the node in the cluster
NODE="PVE-Node01"

# Proxy equals node if node is a DNS address
# Otherwise, you need to set the IP address of the node here
PROXY="172.16.200.152"

#The rest of the script from Proxmox
NODE="${NODE%%\.*}"

echo "Authenticating"
DATA="$(curl -f -s -S -k --data-urlencode "username=$USERNAME" --data-urlencode "password=$PASSWORD" "https://$PROXY:8006/api2/json/access/ticket")"

echo "AUTH OK"

TICKET="${DATA//\"/}"
TICKET="${TICKET##*ticket:}"
TICKET="${TICKET%%,*}"
TICKET="${TICKET%%\}*}"

CSRF="${DATA//\"/}"
CSRF="${CSRF##*CSRFPreventionToken:}"
CSRF="${CSRF%%,*}"
CSRF="${CSRF%%\}*}"

curl -f -s -S -k -b "PVEAuthCookie=$TICKET" -H "CSRFPreventionToken: $CSRF" "https://$PROXY:8006/api2/spiceconfig/nodes/$NODE/qemu/$VMID/spiceproxy" -d "proxy=$PROXY" > spiceproxy

#Launch remote-viewer with spiceproxy file, in kiosk mode, quit on disconnect
#The run loop will get a new ticket and launch us again if we disconnect
exec remote-viewer -k --kiosk-quit on-disconnect spiceproxy
