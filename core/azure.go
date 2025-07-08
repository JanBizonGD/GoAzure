package core

import (
	"bufio"
	"context"
	"encoding/json"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/data/aztables"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/cosmos/armcosmos/v2"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources"
)

var (
	globalRowId           = 0
	globalPartition       = 0
	subscriptionId        = ""
	resourceGroupName     = ""
	resourceGroupLocation = ""
	accountName           = "myaccount1234jb"
	tableName             = "mytable123"
	RGName                = &resourceGroupName
	RGLocation            = &resourceGroupLocation
)

func ImportEnv(filename string) {
	filePtr, err := os.Open(filename)
	if err != nil {
		panic(err)
	}
	defer filePtr.Close()

	scanner := bufio.NewScanner(filePtr)
	for scanner.Scan() {
		envVar := strings.Split(scanner.Text(), "=")
		key, value := envVar[0], envVar[1]
		value = strings.TrimPrefix(value, "\"")
		value = strings.TrimSuffix(value, "\"")

		switch key {
		case "AZURE_SUBSCRIPTION_ID":
			subscriptionId = value
		case "AZURE_RES_GROUP_NAME":
			resourceGroupName = value
		case "AZURE_RES_GROUP_LOC":
			resourceGroupLocation = value
		}
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}
}

func GetClient(context context.Context) *armresources.ResourceGroupsClient {
	credential, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		log.Fatalln(err)
	}

	clientFactory, err := armresources.NewClientFactory(subscriptionId, credential, nil)
	if err != nil {
		panic(err)
	}
	client := clientFactory.NewResourceGroupsClient()

	return client
}

func GetResourceGroupID(context context.Context, client *armresources.ResourceGroupsClient) string {
	response, err := client.CreateOrUpdate(context, resourceGroupName,
		armresources.ResourceGroup{
			Location: to.Ptr(resourceGroupLocation),
		}, nil)
	if err != nil {
		panic(err)
	}

	return *response.ResourceGroup.ID
}

func CreateDB(context context.Context) {
	credential, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		log.Fatalln(err)
	}
	cosmosClientFactory, err := armcosmos.NewClientFactory(subscriptionId, credential, nil)
	if err != nil {
		log.Fatal(err)
	}
	databaseAccountsClient := cosmosClientFactory.NewDatabaseAccountsClient()
	tableResourcesClient := cosmosClientFactory.NewTableResourcesClient()

	log.Println("Creating database account ...")
	databaseAccount, err := createDatabaseAccount(context, databaseAccountsClient)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Cosmos database account:", *databaseAccount.ID)

	log.Println("Creating new table ...")
	table, err := createTable(context, tableResourcesClient)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Cosmos table:", *table.ID)
}

func createDatabaseAccount(context context.Context, client *armcosmos.DatabaseAccountsClient) (*armcosmos.DatabaseAccountGetResults, error) {
	pollerResp, err := client.BeginCreateOrUpdate(
		context,
		resourceGroupName,
		accountName,
		armcosmos.DatabaseAccountCreateUpdateParameters{
			Location: to.Ptr(resourceGroupLocation),
			Kind:     to.Ptr(armcosmos.DatabaseAccountKindGlobalDocumentDB),
			Properties: &armcosmos.DatabaseAccountCreateUpdateProperties{
				DatabaseAccountOfferType: to.Ptr("Standard"),
				Locations: []*armcosmos.Location{
					{
						FailoverPriority: to.Ptr[int32](0),
						IsZoneRedundant:  to.Ptr(false),
						LocationName:     to.Ptr(resourceGroupLocation),
					},
				},
				Capabilities: []*armcosmos.Capability{
					{
						Name: to.Ptr("EnableTable"),
					},
				},
				APIProperties: &armcosmos.APIProperties{},
			},
		},
		nil,
	)
	if err != nil {
		return nil, err
	}

	resp, err := pollerResp.PollUntilDone(context, nil)
	if err != nil {
		return nil, err
	}
	return &resp.DatabaseAccountGetResults, nil
}

func createTable(context context.Context, client *armcosmos.TableResourcesClient) (*armcosmos.TableGetResults, error) {
	pollerResp, err := client.BeginCreateUpdateTable(
		context,
		resourceGroupName,
		accountName,
		tableName,
		armcosmos.TableCreateUpdateParameters{
			Location: to.Ptr(resourceGroupLocation),
			Properties: &armcosmos.TableCreateUpdateProperties{
				Resource: &armcosmos.TableResource{
					ID: to.Ptr(tableName),
				},
				Options: &armcosmos.CreateUpdateOptions{},
			},
		},
		nil,
	)
	if err != nil {
		return nil, err
	}

	resp, err := pollerResp.PollUntilDone(context, nil)
	if err != nil {
		return nil, err
	}
	return &resp.TableGetResults, nil
}

func Cleanup(ctx context.Context, resourceGroupClient *armresources.ResourceGroupsClient) error {
	pollerResp, err := resourceGroupClient.BeginDelete(ctx, resourceGroupName, nil)
	if err != nil {
		return err
	}

	_, err = pollerResp.PollUntilDone(ctx, nil)
	if err != nil {
		return err
	}
	return nil
}

func GetTable(credential azcore.TokenCredential, tableAccountEndpoint string, tableName string) *aztables.Client {
	client, err := aztables.NewServiceClient(tableAccountEndpoint, credential, nil)
	if err != nil {
		log.Fatal(err)
	}

	table := client.NewClient(tableName)

	return table
}

func InsertData(context context.Context, table *aztables.Client, item News) error {
	entity := aztables.EDMEntity{
		Entity: aztables.Entity{
			RowKey:       strconv.Itoa(globalRowId), // TODO: need to be changed: after first run it will override existing data
			PartitionKey: "id",                      // globalPartition
		},
		Properties: map[string]any{
			"Title":       item.Title,
			"Description": item.Description,
			"Date":        aztables.EDMDateTime(item.Date),
		},
	}
	globalRowId += 1
	globalPartition = (globalPartition + 1) % 5

	bytes, err := json.Marshal(entity)
	if err != nil {
		return err
	}

	_, err = table.UpsertEntity(context, bytes, nil)
	if err != nil {
		return err
	}
	return nil
}
