package main

import (
	"azure/core"
	"context"
	"fmt"
	"log"
)

var (
	shouldCleanUp = false
)

func main() {
	context := context.Background()
	core.ImportEnv("./.env")
	client := core.GetClient(context)
	resourceGroupId := core.GetResourceGroupID(context, client)
	fmt.Printf("Resource group ID: %s\n", resourceGroupId)
	core.CreateDB(context)

	// TODO: cleanup function app
	core.CreateResources(context, *core.RGLocation)

	// rgName := *core.RGName
	// rgLocation := *core.RGLocation //"westus" //   // south central region in cloudguru doesnt allow to create
	// cmd := exec.Command("/bin/bash", "sh/create_upload_azfunc.sh", "-r", rgName, "-l", rgLocation)
	// var stderr, stdout bytes.Buffer
	// cmd.Stderr = &stderr
	// cmd.Stdout = &stdout
	// err := cmd.Run()

	// if err != nil {
	// 	fmt.Println("STDERR:", stderr.String())
	// 	fmt.Printf("Error during executing bash script (Function app): %s", err)
	// }
	// fmt.Println("STDOUT:", stdout.String())

	if shouldCleanUp {
		err := core.Cleanup(context, client)
		if err != nil {
			log.Fatal(err)
		}
		log.Println("Cleaned up successfully.")
	}
}
