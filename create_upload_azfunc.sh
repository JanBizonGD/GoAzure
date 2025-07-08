#!/bin/sh -e

# NOTE: Before executing this script login to azure with cli
# This script was intedend to be executed with go script
#
# Currently only for Windows, (to change to linux - change windows to linux 
# inside this file and change configuriaotn of a host.json to correct extention)

# NOTE : JOT only on mac, use shuf on linux (coreutils on mac)

# Default values
RESOURCE_GROUP=""
STORAGE_ACCOUNT="storageaccount$(jot -r 1 100000 999999)jb"
FUN_PLAN_NAME="fplan$(jot -r 1 100000 999999)jb"
FUN_NAME="funapp$(jot -r 1 100000 999999)jb"
LOCATION="westus"
FUN_ARCH="amd64"

# Parse input
while [[ $# -gt 0 ]]; do
    case $1 in
    -r|--resource-group )
        RESOURCE_GROUP="$2"
        shift
        shift
        ;;
    -s|--storage-account )
        STORAGE_ACCOUNT="$2"
        shift
        shift
        ;;
    -l|--location )
        LOCATION="$2"
        shift
        shift
        ;;
    -a|--arch )
        FUN_ARCH="$2"
        shift
        shift
        ;;
    -n|--fname )
        FUN_NAME="$2"
        shift
        shift
        ;;
    esac
done

# Check input
if [ -z "$RESOURCE_GROUP" ]; then
    echo "Resource group name is required !!"
    echo "... -r [rg-name] ..."
    exit 1
fi
echo "Using:"
echo "* STORAGE_ACCOUNT: $STORAGE_ACCOUNT"
echo "* FUN_PLAN_NAME: $FUN_PLAN_NAME"
echo "* FUN_NAME: $FUN_NAME"
echo "* LOCATION: $LOCATION"
echo "* FUN_ARCH: $FUN_ARCH"


# Execute deployment
echo "Creating azure function ...."
az storage account create \
  --name $STORAGE_ACCOUNT \
  --location $LOCATION \
  --resource-group $RESOURCE_GROUP \
  --sku Standard_LRS

az functionapp plan create \
  --name $FUN_PLAN_NAME \
  --resource-group $RESOURCE_GROUP \
  --location $LOCATION \
  --number-of-workers 1 \
  --sku B1

az functionapp create \
  --name $FUN_NAME \
  --plan $FUN_PLAN_NAME \
  --storage-account $STORAGE_ACCOUNT \
  --resource-group $RESOURCE_GROUP \
  --runtime custom \
  --os-type Windows

echo "Adding CORS ...."
az functionapp cors add \
  --resource-group $RESOURCE_GROUP \
  --name $FUN_NAME \
  --allowed-origins "http://localhost,https://portal.azure.com"
echo "Infrastructure created !!!"

echo "Compiling GO ...."
GOOS=windows GOARCH=$FUN_ARCH CGO_ENABLED=0 go build -C webserver server.go

zip -r webserver.zip ./webserver/.

echo "Deploying GO to function app ...."
az functionapp deployment source config-zip \
  --resource-group $RESOURCE_GROUP \
  --name $FUN_NAME \
  --src webserver.zip

echo "Deployment successfull !!!!"

rm webserver.zip webserver/server.exe
