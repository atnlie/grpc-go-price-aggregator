package main

import (
	"context"
	"encoding/json"
	"fmt"
	"google.golang.org/grpc"
	"log"
	"net"
	"os"
	"sync"

	pb "atn.lie/grpc/price-aggregator/modules/user"
)

type UserDataServer struct {
	pb.UnimplementedUserDataServer
	mu    sync.Mutex
	users []*pb.GetUserDataResponse
}

func (d *UserDataServer) GetUserData(_ context.Context, user *pb.GetUserDataRequest) (*pb.GetUserDataResponse, error) {
	for _, v := range d.users {
		if v.UserId == user.UserId {
			return v, nil
		}
	}

	return nil, nil
}

func (d *UserDataServer) loadDataFromFile() {
	data, err := os.ReadFile("./server/fixtures/users.json")
	if err != nil {
		log.Fatalf("Error while read file %s", err.Error())
	}

	if err := json.Unmarshal(data, &d.users); err != nil {
		log.Fatalf("Error while unmarshal json %s", err.Error())
	}
}

func dataServer() *UserDataServer {
	s := UserDataServer{}
	s.loadDataFromFile()
	return &s
}

func main() {
	listener, err := net.Listen("tcp", ":8880")
	fmt.Println("Starting Serve on port 8080")
	if err != nil {
		log.Fatalf("Error while listen %s", err.Error())
	}

	grpcServer := grpc.NewServer()
	pb.RegisterUserDataServer(grpcServer, dataServer())

	if err := grpcServer.Serve(listener); err != nil {
		log.Fatalf("Error while serve %s", err.Error())
	}
}
