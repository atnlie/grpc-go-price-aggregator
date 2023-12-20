package main

import (
	pa "atn.lie/grpc/price-aggregator/pb/auth"
	pb "atn.lie/grpc/price-aggregator/pb/user"
	auth "atn.lie/grpc/price-aggregator/server_grpc_gateway/internal"
	"github.com/google/uuid"

	"context"
	"fmt"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/jackc/pgx/v5"
	"github.com/joho/godotenv"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"log"
	"net"
	"net/http"
	"os"
)

const (
	gRPCPort   = ":8880"
	gRPCGwPort = ":8881"
)

type UserDataServer struct {
	conn *pgx.Conn
	pb.UnimplementedUserDataServer
	//mu    sync.Mutex
	//users []*pb.GetUserDataResponse
}

type AuthServiceServer struct {
	conn *pgx.Conn
	pa.UnimplementedAuthServiceServer
	//mu   sync.Mutex
	//pass []*pa.HashPassword
}

//func (dataServer *UserDataServer) loadDataFromFile() {
//	data, err := os.ReadFile("./server_grpc_gateway/fixtures/users.json")
//	if err != nil {
//		log.Fatalf("Error while read file %s", err.Error())
//	}
//
//	if err := json.Unmarshal(data, &dataServer.users); err != nil {
//		log.Fatalf("Error while unmarshal json %s", err.Error())
//	}
//}

//func dataJsonServer() *UserDataServer {
//	s := UserDataServer{}
//	s.loadDataFromFile()
//	log.Println(&s.users)
//
//	return &s
//}

func userDataMgmServer() *UserDataServer {
	return &UserDataServer{}
}

func authMgtServiceServer() *AuthServiceServer {
	return &AuthServiceServer{}
}

// Use DB as permanent storage of data
func (dataServer *UserDataServer) CreateNewUser(_ context.Context, req *pb.NewUserData) (*pb.GetUserDataResponse, error) {
	log.Printf("Data %v", req)
	qrySql := "create table if not exists users(userid uuid PRIMARY KEY, email varchar(255) not null unique, username varchar(255) not null unique, password text not null);"

	_, err := dataServer.conn.Exec(context.Background(), qrySql)
	if err != nil {
		_, err := fmt.Fprintf(os.Stderr, "Table creation failed. %v\n", err)
		if err != nil {
			return nil, err
		}
		os.Exit(1)
	}
	userId := uuid.New()
	hashPass, err := auth.HashAndSalt([]byte(req.Password))
	if err != nil {
		return nil, err
	}

	createUser := &pb.GetUserDataResponse{Userid: userId.String(), Username: req.GetUsername(), Email: req.GetEmail(),
		Password: hashPass}
	tx, err := dataServer.conn.Begin(context.Background())
	if err != nil {
		log.Fatalf("conn.Begin failed. %v", err)
	}

	_, err = tx.Exec(context.Background(),
		"INSERT INTO users(userid, email, username, password) values ($1, $2, $3, $4)",
		createUser.Userid, createUser.Email, createUser.Username, createUser.Password)
	if err != nil {
		log.Printf("Some issues %v", err.Error())
		err := tx.Rollback(context.Background())
		if err != nil {
			return nil, err
		}
		return nil, err
	}

	err = tx.Commit(context.Background())

	if err != nil {
		return nil, err
	}

	return createUser, nil
}

func (dataServer *UserDataServer) LoginUser(_ context.Context, req *pb.LoginRequest) (*pb.GetUserDataResponse, error) {
	row, err := dataServer.conn.Query(context.Background(),
		"Select userid, email, username, password from users where username = $1",
		req.GetUsername())
	if err != nil {
		return nil, err
	}
	defer row.Close()

	var activeUser *pb.GetUserDataResponse
	for row.Next() {
		user := &pb.GetUserDataResponse{}
		err = row.Scan(&user.Userid, &user.Email, &user.Username, &user.Password)
		if err != nil {
			return nil, err
		}
		activeUser = user
	}

	_, err = auth.ComparePasswords(activeUser.Password, []byte(req.Password))
	if err != nil {
		return nil, err
	}

	return activeUser, nil
}

//func (dataServer *UserDataServer) GetUserData(_ context.Context, user *pb.GetUserDataRequest) (*pb.GetUserDataResponse, error) {
//	for _, v := range dataServer.users {
//		if v.Userid == user.Userid {
//			return v, nil
//		}
//	}
//
//	return nil, nil
//}

func goEnvVar(key string) string {
	err := godotenv.Load(".env")
	if err != nil {
		log.Fatalf("Error loading .env file")
	}
	return os.Getenv(key)
}

func (authServer *AuthServiceServer) SetHashPassword(ctx context.Context, req *pa.SetPassword) (*pa.HashPassword, error) {
	newPass := []byte(req.NewPassword)
	pass, err := auth.HashAndSalt(newPass)
	if err != nil {
		return nil, err
	}

	return &pa.HashPassword{HashPassword: pass}, nil
}

func unaryInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	log.Println("Unary Interceptor ---> :", info.FullMethod)
	return handler(ctx, req)
}

func streamInterceptor(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	log.Println("Stream Interceptor ---> :", info.FullMethod)
	return handler(srv, ss)
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
	var authServer = authMgtServiceServer()
	authServer.conn = pgxConn

	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(unaryInterceptor),
		grpc.StreamInterceptor(streamInterceptor),
	)

	//pb.RegisterUserDataServer(grpcServer, dataJsonServer())
	pb.RegisterUserDataServer(grpcServer, userMgmServer)
	pa.RegisterAuthServiceServer(grpcServer, authServer)

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

	gwMuxUser := runtime.NewServeMux()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	//err = pb.RegisterUserDataHandler(context.Background(), gwMux, conn)
	err = pb.RegisterUserDataHandler(ctx, gwMuxUser, conn)
	if err != nil {
		log.Fatalf("Failed to register gateway %s", err.Error())
	}

	gwMuxAuth := runtime.NewServeMux()
	err = pa.RegisterAuthServiceHandler(context.Background(), gwMuxAuth, conn)
	if err != nil {
		log.Fatalf("Failed to register gateway %s", err.Error())
	}

	mux := http.NewServeMux()
	mux.Handle("/user/", gwMuxUser)
	mux.Handle("/auth/", gwMuxAuth)

	gwServer := &http.Server{Addr: gRPCGwPort, Handler: mux}

	log.Println("Serving gRPC-Gateway on http://0.0.0.0" + gRPCGwPort)
	log.Fatalln(gwServer.ListenAndServe())

}
