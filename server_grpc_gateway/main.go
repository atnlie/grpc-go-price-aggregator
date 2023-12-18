package main

import (
	pb "atn.lie/grpc/price-aggregator/modules/user"
	"context"
	"encoding/json"
	"fmt"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"log"
	"net"
	"net/http"
	"os"
	"sync"
)

type UserDataServer struct {
	pb.UnimplementedUserDataServer
	mu    sync.Mutex
	users []*pb.GetUserDataResponse
}

func (d *UserDataServer) loadDataFromFile() {
	data, err := os.ReadFile("./server_grpc_gateway/fixtures/users.json")
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
	log.Println(&s.users)

	return &s
}

func (d *UserDataServer) GetUserData(_ context.Context, user *pb.GetUserDataRequest) (*pb.GetUserDataResponse, error) {
	for _, v := range d.users {
		if v.UserId == user.UserId {
			return v, nil
		}
	}

	return nil, nil
}

func main() {
	listener, err := net.Listen("tcp", ":8880")
	fmt.Println("Starting Serve on port 8880")
	if err != nil {
		log.Fatalf("Error while listen %s", err.Error())
	}

	grpcServer := grpc.NewServer()
	pb.RegisterUserDataServer(grpcServer, dataServer())
	log.Println("Serving gRPC on 0.0.0.0:8080")
	go func() {
		log.Fatalln(grpcServer.Serve(listener))
	}()

	// client GRPC
	// Approach #1
	var opts []grpc.DialOption
	opts = append(opts, grpc.WithBlock())
	opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))

	conn, err := grpc.Dial(":8880", opts...)
	if err != nil {
		log.Fatalf("Error in Dial %s", err.Error())
	}

	// Approach #2
	//conn, err := grpc.DialContext(context.Background(),
	//	":8880",
	//	grpc.WithBlock(),
	//	grpc.WithTransportCredentials(insecure.NewCredentials()),
	//)
	fmt.Println("Listening Client gRPC on port 8880")
	if err != nil {
		log.Fatalf("Failed to dial server %s", err.Error())
	}

	gwMux := runtime.NewServeMux()
	err = pb.RegisterUserDataHandler(context.Background(), gwMux, conn)
	if err != nil {
		log.Fatalf("Failed to register gateway %s", err.Error())
	}
	gwSever := &http.Server{Addr: ":8881", Handler: gwMux}
	log.Println("Serving gRPC-Gateway on http://0.0.0.0:8881")
	log.Fatalln(gwSever.ListenAndServe())
}
