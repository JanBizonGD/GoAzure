# Fetch and store data with Azure

## Init package 
```
go get github.com/Azure/azure-sdk-for-go/sdk/azcore/to
go get "github.com/Azure/azure-sdk-for-go/sdk/azidentity"
go get github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources
go get "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/cosmos/armcosmos/v2"

```

## Export credentials
Export Azure credentails inside `.env` file.
Credentials required: 
```
AZURE_SUBSCRIPTION_ID="..."
AZURE_RES_GROUP_NAME="..."
AZURE_RES_GROUP_LOC="..."
```

## Login
Login with Azure CLI before executing `main.go`.
Example:
`az login --service-principal  --tenant 84f..... --username ca9....` and then enter password.
