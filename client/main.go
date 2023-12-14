package main

import (
	"context"
	"fmt"
	"github.com/davecgh/go-spew/spew"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"log"
	"time"

	pb "atn.lie/grpc/price-aggregator/modules/user"
)

func getUserData(client pb.UserDataClient, userId string) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req := pb.GetUserDataRequest{UserId: userId}
	user, err := client.GetUserData(ctx, &req)
	if err != nil {
		log.Fatalf("error while get User %s", userId)
	}

	spew.Dump(user)
}

func main() {
	var opts []grpc.DialOption
	opts = append(opts, grpc.WithBlock())
	opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))

	conn, err := grpc.Dial(":8880", opts...)
	if err != nil {
		log.Fatalf("Error in Dial %s", err.Error())
	}

	defer conn.Close()

	fmt.Println("Ready to call")

	client := pb.NewUserDataClient(conn)
	getUserData(client, "abcdef-123456")
	getUserData(client, "abcdeh-123458")

}
