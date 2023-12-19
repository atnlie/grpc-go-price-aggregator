package main

import (
	pb "atn.lie/grpc/price-aggregator/modules/user"
	"context"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/jackc/pgx/v5"
	"github.com/joho/godotenv"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"log"
	"net"
	"net/http"
	"os"
	"sync"
)

const (
	gRPCPort   = ":8880"
	gRPCGwPort = ":8881"
)

type UserDataServer struct {
	conn *pgx.Conn
	pb.UnimplementedUserDataServer
	mu    sync.Mutex
	users []*pb.GetUserDataResponse
}

func (dataServer *UserDataServer) loadDataFromFile() {
	data, err := os.ReadFile("./server_grpc_gateway/fixtures/users.json")
	if err != nil {
		log.Fatalf("Error while read file %s", err.Error())
	}

	if err := json.Unmarshal(data, &dataServer.users); err != nil {
		log.Fatalf("Error while unmarshal json %s", err.Error())
	}
}

func dataJsonServer() *UserDataServer {
	s := UserDataServer{}
	s.loadDataFromFile()
	log.Println(&s.users)

	return &s
}

func userDataMgmServer() *UserDataServer {
	return &UserDataServer{}
}

// Use DB as permanent storage of data
func (dataServer *UserDataServer) createNewUser(ctx context.Context, req *pb.NewUserData) (*pb.NewUserData, error) {
	log.Printf("Data %v", req.GetUserName())
	qrySql := "create table if not exists " +
		"users(userid uuid PRIMARY KEY,email varchar(255) not null unique, username varchar(255) not null unique, password text not null);"

	_, err := dataServer.conn.Exec(context.Background(), qrySql)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Table creation failed. %v\n", err)
		os.Exit(1)
	}
	userId := uuid.New()
	createUser := &pb.NewUserData{UserId: userId.String(), UserName: req.GetUserName(), UserEmail: req.GetUserEmail(),
		Password: req.Password}
	tx, err := dataServer.conn.Begin(context.Background())
	if err != nil {
		log.Fatalf("conn.Begin failed. %v", err)
	}

	_, err = tx.Exec(context.Background(), "Insert into users(userId, email, username, password) values ($1, $2, $3, $4)",
		createUser.UserId, createUser.UserEmail, createUser.UserName, createUser.Password)
	if err != nil {
		log.Fatalf("tx.Exec failed. %v", err)
	}
	return createUser, nil
}

func (dataServer *UserDataServer) LoginUser(ctx context.Context, req *pb.LoginRequest) (*pb.GetUserDataResponse, error) {
	row, err := dataServer.conn.Query(context.Background(), "Select userid, email, username, password from users where username = $1 and password = $2",
		req.GetUsername(), req.GetPassword())
	if err != nil {
		return nil, err
	}
	defer row.Close()

	var activeUser *pb.GetUserDataResponse
	for row.Next() {
		user := &pb.GetUserDataResponse{}
		err = row.Scan(&user.UserId, &user.UserEmail, &user.UserName, &user.Password)
		if err != nil {
			return nil, err
		}
		activeUser = user
	}

	return activeUser, nil
}

func (dataServer *UserDataServer) GetUserData(_ context.Context, user *pb.GetUserDataRequest) (*pb.GetUserDataResponse, error) {
	for _, v := range dataServer.users {
		if v.UserId == user.UserId {
			return v, nil
		}
	}

	return nil, nil
}

func goEnvVar(key string) string {
	err := godotenv.Load(".env")
	if err != nil {
		log.Fatalf("Error loading .env file")
	}
	return os.Getenv(key)
}

func main() {
	listener, err := net.Listen("tcp", gRPCPort)
	fmt.Println("Starting Serve on port " + gRPCPort)
	if err != nil {
		log.Fatalf("Error while listen %s", err.Error())
	}

	dbPass := goEnvVar("DB_PASSWORD")
	dbPort := goEnvVar("PORT")
	dbName := goEnvVar("DB_NAME")
	dbUrl := "postgres://postgres:" + dbPass + "@localhost:" + dbPort + "/" + dbName

	pgxConn, err := pgx.Connect(context.Background(), dbUrl)
	if err != nil {
		log.Fatalf("Unable to connect db %v", err)
	}

	defer pgxConn.Close(context.Background())
	var userMgmServer = userDataMgmServer()
	userMgmServer.conn = pgxConn

	grpcServer := grpc.NewServer()
	//pb.RegisterUserDataServer(grpcServer, dataJsonServer())
	pb.RegisterUserDataServer(grpcServer, userMgmServer)
	log.Println("Serving gRPC on 0.0.0.0" + gRPCPort)
	go func() {
		log.Fatalln(grpcServer.Serve(listener))
	}()

	// client GRPC
	// Approach #1
	var opts []grpc.DialOption
	opts = append(opts, grpc.WithBlock())
	opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))

	conn, err := grpc.Dial(gRPCPort, opts...)
	if err != nil {
		log.Fatalf("Error in Dial %s", err.Error())
	}

	// Approach #2
	//conn, err := grpc.DialContext(context.Background(),
	//	":8880",
	//	grpc.WithBlock(),
	//	grpc.WithTransportCredentials(insecure.NewCredentials()),
	//)
	fmt.Println("Listening Client gRPC on port " + gRPCPort)
	if err != nil {
		log.Fatalf("Failed to dial server %s", err.Error())
	}

	gwMux := runtime.NewServeMux()
	err = pb.RegisterUserDataHandler(context.Background(), gwMux, conn)
	if err != nil {
		log.Fatalf("Failed to register gateway %s", err.Error())
	}
	gwSever := &http.Server{Addr: gRPCGwPort, Handler: gwMux}
	log.Println("Serving gRPC-Gateway on http://0.0.0.0" + gRPCGwPort)
	log.Fatalln(gwSever.ListenAndServe())
}
