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
AZURE_TENANT_ID="..."
AZURE_ACCOUNT="..."
AZURE_SECRET="..."
```

## Login
Login with Azure CLI before executing `main.go`.
Example:
`az login --service-principal  --tenant 84f..... --username ca9....` and then enter password.

## TODO
- After provision : just enter variables tab in function app and change on storage accout access from all networks. It will help with permision problem
- Account and password from key vault insead of `.env` file
- Deployment not working - post is not properly structured. Use bash script instead.
- Login with Go not with `az login ...`
- Clearing function

## Example request
```
{
"url": "https://dorzeczy.pl/feed",
"account": "myaccount1234jb",
"table": "mytable123"
}
```


## Note
Successfull post with curl:
`-u <deployment username>:<deployment password>` - can be obtained from function app profile file.
```
curl -X POST https://<func app>.scm.azurewebsites.net/api/zipdeploy -T "webserver.zip" -u \$fun....:vz.......
```
