package auth

import (
	//pb "atn.lie/grpc/price-aggregator/modules/auth"
	//"context"
	"golang.org/x/crypto/bcrypt"
	"log"
)

//type AuthServiceServer struct {
//	pb.UnimplementedAuthServiceServer
//	//pass *[]pb.HashPassword
//}

func HashAndSalt(pass []byte) (string, error) {
	hash, err := bcrypt.GenerateFromPassword(pass, bcrypt.DefaultCost)
	if err != nil {
		log.Printf("Error Hashing password %v", err)
		return "", err
	}

	return string(hash), nil
}

func ComparePasswords(hashedPass string, pass []byte) (bool, error) {
	byteHash := []byte(hashedPass)
	err := bcrypt.CompareHashAndPassword(byteHash, pass)
	if err != nil {
		log.Printf("Err: %v", err)
		return false, err
	}
	return true, nil
}

//func (dataServer *AuthServiceServer) SetHashPassword(_ context.Context, req *pb.SetPassword) (*pb.HashPassword, error) {
//	newPass := []byte(req.NewPassword)
//	pass, err := HashAndSalt(newPass)
//	if err != nil {
//		return nil, err
//	}
//
//	return &pb.HashPassword{HashPassword: pass}, nil
//}
