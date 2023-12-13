package main

import (
	"google.golang.org/grpc"
	"sync"
	"context"
	"io/ioutil"
	"encoding/json"
	"log"
	"net"

	pb "atn.lie/grpc/price-aggregator/modules/user"
)

type UserDataServer struct {
	pb.UnimplementedUserDataServer
	mu sync.Mutex
	users []*pb.GetUserDataResponse
}

func (d *UserDataServer) GetUserData(ctx context.Context, user *pb.GetUserDataRequest) (*pb.GetUserDataResponse, error) {
	for _, v := range d.users {
		if v.UserId == user.UserId {
			return v, nil
		}
	}

	return nil, nil
}

func (d *UserDataServer) loadDataFromFile() {
	data, error := ioutil.ReadFile("fixture/users.json")
	if error != nil {
		log.Fatalf("Error while read file %s", error.Error())
	}

	if error := json.Unmarshal(data, &d.users); error != nil {
		log.Fatalf("Error while unmarshal json %s", error.Error())
	}
}

func dataServer() *UserDataServer{
	s:= UserDataServer{}
	s.loadDataFromFile()
	return &s
}

func main() {
	listener, error := net.Listen("tcp", ":8080")
	if error != nil {
		log.Fatalf("Error while listen %s", error.Error())
	}

	grpcServer := grpc.NewServer()
	pb.RegisterUserDataServer(grpcServer, dataServer())

	if error := grpcServer.Serve(listener); error != nil {
		log.Fatalf("Error while serve %s", error.Error())
	}
}

