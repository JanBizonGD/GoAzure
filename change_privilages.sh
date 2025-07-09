RESOURCE_GROUP=""
ACCOUNT_NAME=""
PRINCIPAL_ID=""
SUB_ID=""

az cosmosdb sql role assignment create \
  --resource-group $RESOURCE_GROUP \
  --account-name $ACCOUNT_NAME \
  --role-definition-id 00000000-0000-0000-0000-000000000002 \
  --principal-id $PRINCIPAL_ID \
  --scope /subscriptions/$SUB_ID/resourceGroups/$RESOURCE_GROUP/providers/Microsoft.DocumentDB/databaseAccounts/$ACCOUNT_NAME
