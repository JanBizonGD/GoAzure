#!/bin/bash -e

RESOURCE_GROUP=""
FUNCTION_APP=""

while [[ $# -gt 0 ]]; do
    case $1 in
    -r|--resource-group )
        RESOURCE_GROUP="$2"
        shift
        shift
        ;;
    -n|--fname )
        FUNCTION_APP="$2"
        shift
        shift
        ;;
    esac
done

if [ -z "$RESOURCE_GROUP" ]; then
    echo "Resource group name is required !!"
    echo "... -r [rg-name] ..."
    exit 1
fi
if [ -z "$FUNCTION_APP" ]; then
    echo "Function app name is required !!"
    echo "... -n [func-app-name] ..."
    exit 1
fi

az functionapp deployment source config-zip \
  --resource-group $RESOURCE_GROUP \
  --name $FUNCTION_APP \
  --src ../webserver.zip
