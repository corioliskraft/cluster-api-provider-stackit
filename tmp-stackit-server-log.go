package main

import (
	"context"
	"fmt"
	"os"

	"github.com/stackitcloud/stackit-sdk-go/core/config"
	iaas "github.com/stackitcloud/stackit-sdk-go/services/iaas/v1api"
)

func main() {
	key, err := os.ReadFile("./sa/serviceaccount.json")
	if err != nil {
		panic(err)
	}
	client, err := iaas.NewAPIClient(config.WithServiceAccountKey(string(key)))
	if err != nil {
		panic(err)
	}
	resp, err := client.DefaultAPI.GetServerLog(context.Background(), os.Getenv("STACKIT_PROJECT_ID"), os.Args[1]).Length(800).Execute()
	if err != nil {
		panic(err)
	}
	fmt.Print(resp.GetOutput())
}
