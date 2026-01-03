package main

import (
	"context"
	crawlergrpc "distributed-crawler/pkg/v1"
	"fmt"
	"log"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/encoding/protojson"
)

func main() {
	conn, err := grpc.NewClient(
		":8082",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)

	if err != nil {
		log.Fatal(err)
	}

	client := crawlergrpc.NewCrawlerServiceClient(conn)

	res, err := client.ListJobs(context.Background(), &crawlergrpc.ListJobsRequest{})
	if err != nil {
		log.Fatal(err)
	}

	// fmt.Printf("%#v", res.GetJobs())

	bytes, _ := protojson.Marshal(res)

	fmt.Println(string(bytes))
}
