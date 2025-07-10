package core

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand/v2"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/data/aztables"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/appservice/armappservice/v5"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/cosmos/armcosmos/v2"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/storage/armstorage"
)

var (
	globalRowId           = 0
	globalPartition       = 0
	subscriptionId        = ""
	resourceGroupName     = ""
	resourceGroupLocation = ""
	tenantId              = ""
	username              = ""
	password              = ""
	accountName           = "myaccount1235jb"
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
		case "AZURE_TENANT_ID":
			tenantId = value
		case "AZURE_ACCOUNT":
			username = value
		case "AZURE_SECRET":
			password = value
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

func assignCosmosRole(ctx context.Context, dbAccount, assignmentName, principalID string) error {
	credential, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		log.Fatalln(err)
	}
	client, err := armcosmos.NewSQLResourcesClient(subscriptionId, credential, nil)
	if err != nil {
		log.Fatalln(err)
	}

	scope := "/subscriptions/" + subscriptionId + "/resourceGroups/" + resourceGroupName + "/providers/Microsoft.DocumentDB/databaseAccounts/" + dbAccount
	roleId := "/subscriptions/" + subscriptionId + "/resourceGroups/" + resourceGroupName + "/providers/Microsoft.DocumentDB/databaseAccounts/" + dbAccount + "/sqlRoleDefinitions/00000000-0000-0000-0000-000000000002"
	params := armcosmos.SQLRoleAssignmentCreateUpdateParameters{
		Properties: &armcosmos.SQLRoleAssignmentResource{
			PrincipalID:      to.Ptr(principalID),
			RoleDefinitionID: to.Ptr(roleId),
			Scope:            to.Ptr(scope),
		},
	}

	_, err = client.BeginCreateUpdateSQLRoleAssignment(ctx, assignmentName, resourceGroupName, dbAccount, params, nil)

	return err
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

type BearerTokenResponse struct {
	Type         string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	ExtExpiresIn int    `json:"ext_expires_in"`
	BearerToken  string `json:"access_token"`
}

func randRange(min, max int) int {
	return rand.IntN(max-min) + min
}

func getStorageConnectionString(ctx context.Context, client *armstorage.AccountsClient, accountName string) (string, error) {
	keysResp, err := client.ListKeys(ctx, resourceGroupName, accountName, nil)
	if err != nil {
		return "", err
	}

	if keysResp.Keys == nil || len(keysResp.Keys) == 0 || keysResp.Keys[0].Value == nil {
		return "", fmt.Errorf("no keys found")
	}

	key := *keysResp.Keys[0].Value
	connectionString := fmt.Sprintf(
		"DefaultEndpointsProtocol=https;AccountName=%s;AccountKey=%s;EndpointSuffix=core.windows.net",
		accountName, key,
	)

	return connectionString, nil
}

func createStorageAccount(ctx context.Context, location string) (string, string) {
	storageAccoutName := "storageaccount" + strconv.Itoa(randRange(100000, 999999)) + "jb"

	credential, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		log.Fatalln(err)
	}
	clientFactory, err := armstorage.NewClientFactory(subscriptionId, credential, nil)
	if err != nil {
		log.Fatalf("failed to create client: %v", err)
	}
	client := clientFactory.NewAccountsClient()
	poller, err := client.BeginCreate(ctx, resourceGroupName, storageAccoutName, armstorage.AccountCreateParameters{
		Kind:     to.Ptr(armstorage.KindStorageV2),
		Location: to.Ptr(location),
		Properties: &armstorage.AccountPropertiesCreateParameters{
			EnableExtendedGroups: to.Ptr(true),
			IsHnsEnabled:         to.Ptr(true),
			EnableNfsV3:          to.Ptr(true),
			NetworkRuleSet: &armstorage.NetworkRuleSet{
				Bypass:        to.Ptr(armstorage.BypassAzureServices),
				DefaultAction: to.Ptr(armstorage.DefaultActionDeny),
				IPRules:       []*armstorage.IPRule{},
			},
			EnableHTTPSTrafficOnly: to.Ptr(false),
		},
		SKU: &armstorage.SKU{
			Name: to.Ptr(armstorage.SKUNameStandardLRS),
		},
	}, nil)
	if err != nil {
		log.Fatalf("failed to finish the request: %v", err)
	}
	res, err := poller.PollUntilDone(ctx, nil)
	if err != nil {
		log.Fatalf("failed to pull the result: %v", err)
	}
	connString, err := getStorageConnectionString(ctx, client, *res.Name)
	if err != nil {
		log.Fatalf("Connection string error: %v", err)
	}

	return *res.Name, connString
}

func createAppPlan(ctx context.Context, location string) string {
	planName := "fplan" + strconv.Itoa(randRange(100000, 999999)) + "jb"

	credential, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		log.Fatalln(err)
	}
	client, _ := armappservice.NewPlansClient(subscriptionId, credential, nil)

	poller, err := client.BeginCreateOrUpdate(
		ctx, resourceGroupName, planName,
		armappservice.Plan{
			Location: to.Ptr(location),
			Kind:     to.Ptr("functionapp"),
			SKU: &armappservice.SKUDescription{
				Name: to.Ptr("B1"),
				Tier: to.Ptr("Dynamic"),
				Size: to.Ptr("B1"),
			},
		}, nil,
	)
	if err != nil {
		log.Fatalln(err)
	}
	res, err := poller.PollUntilDone(ctx, nil)
	if err != nil {
		log.Fatalf("failed to pull the result: %v", err)
	}

	return *res.ID
}

func createFunctionApp(ctx context.Context, location, planID, storageConn string) string {
	appName := "funapp" + strconv.Itoa(randRange(100000, 999999)) + "jb"

	credential, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		log.Fatalln(err)
	}
	client, _ := armappservice.NewWebAppsClient(subscriptionId, credential, nil)

	poller, err := client.BeginCreateOrUpdate(
		ctx, resourceGroupName, appName,
		armappservice.Site{
			Location: to.Ptr(location),
			Kind:     to.Ptr("functionapp"),
			Properties: &armappservice.SiteProperties{
				ServerFarmID: to.Ptr(planID),
				SiteConfig: &armappservice.SiteConfig{AppSettings: []*armappservice.NameValuePair{
					{Name: to.Ptr("AzureWebJobsStorage"), Value: to.Ptr(storageConn)},
					{Name: to.Ptr("FUNCTIONS_EXTENSION_VERSION"), Value: to.Ptr("~4")},
					{Name: to.Ptr("FUNCTIONS_WORKER_RUNTIME"), Value: to.Ptr("custom")},
				}},
			},
		}, nil,
	)
	if err != nil {
		log.Fatalf("Failed to create functionapp: %v", err)
	}
	res, err := poller.PollUntilDone(ctx, nil)
	if err != nil {
		log.Fatalf("failed to pull the result: %v", err)
	}

	allowedWebpages := [...]string{"http://localhost", "https://portal.azure.com"}
	allowedPtr := []*string{}
	for _, webpage := range allowedWebpages {
		allowedPtr = append(allowedPtr, &webpage)
	}
	err = updateCors(ctx, *res.Name, allowedPtr)
	if err != nil {
		log.Fatalf("Error during add of cors: %v\n", err)
	}
	return *res.Name
}

func updateCors(ctx context.Context, appName string, origins []*string) error {
	credential, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		log.Fatalln(err)
	}
	client, _ := armappservice.NewWebAppsClient(subscriptionId, credential, nil)

	config := armappservice.SiteConfigResource{
		Properties: &armappservice.SiteConfig{
			Cors: &armappservice.CorsSettings{
				AllowedOrigins:     origins,
				SupportCredentials: to.Ptr(true),
			},
		},
	}
	_, err = client.UpdateConfiguration(ctx, resourceGroupName, appName, config, nil)
	return err
}

func CreateResources(ctx context.Context, location string) error {
	storageAccountName, connString := createStorageAccount(ctx, location)
	fmt.Printf("Storage account: %v\n", storageAccountName)
	functionPlanId := createAppPlan(ctx, location)
	fmt.Printf("Function app plan id: %v\n", functionPlanId)
	functionAppName := createFunctionApp(ctx, location, functionPlanId, connString)
	fmt.Printf("Function app name: %v\n", functionAppName)
	objectId := queryObjectId(ctx, username)
	if objectId == "" {
		log.Fatalf("Empty object id!!")
	}
	fmt.Printf("Object Id: %v\n", objectId)
	err := assignCosmosRole(ctx, accountName, "00000000-0000-0000-0000-000000000003", objectId)
	if err != nil {
		fmt.Printf("Failed to create role assigment!!: %v\n", err)
	}

	return nil
}

func queryObjectId(ctx context.Context, clientId string) string {
	token, err := authenticate(tenantId, username, password)
	if err != nil {
		log.Fatalln("Authentication error!")
	}
	payload := bytes.NewReader([]byte(""))
	data, err := queryAzure(token, "https://graph.microsoft.com/v1.0/servicePrincipals?$filter=appId%20eq%20'"+clientId+"'", http.MethodGet, payload)
	if err != nil {
		log.Fatalln("Query error!")
	}
	query := responseObjectId{}
	json.Unmarshal(data, &query)

	return query.Value[0].Id
}

func authenticate(tenantid string, username string, password string) (string, error) {
	contentType := "application/x-www-form-urlencoded"
	reqBody := "client_id=" + username + "&scope=https%3A%2F%2Fgraph.microsoft.com%2F.default&client_secret=" + password + "&grant_type=client_credentials"
	endpoint := "https://login.microsoftonline.com/" + tenantid + "/oauth2/v2.0/token"
	payload := bytes.NewReader([]byte(reqBody))
	resp, err := http.Post(endpoint, contentType, payload)
	if err != nil {
		return "", err
	}

	body, err := io.ReadAll(resp.Body)
	defer resp.Body.Close()
	if err != nil {
		return "", err
	}
	var token = BearerTokenResponse{}
	json.Unmarshal(body, &token)

	return token.BearerToken, nil
}

// "https://management.azure.com"
func queryAzure(token string, endpoint string, method string, payload io.Reader) ([]byte, error) {
	contentType := "application/json"
	req, err := http.NewRequest(method, endpoint, payload)
	if err != nil {
		return []byte{}, err
	}
	req.Header.Add("Content-Type", contentType)
	req.Header.Add("Authorization", "Bearer "+token)

	client := http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return []byte{}, err
	}
	body, err := io.ReadAll(resp.Body)
	defer resp.Body.Close()
	if err != nil {
		return []byte{}, err
	}

	return body, nil
}
