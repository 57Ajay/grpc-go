package main

import (
	"context"
	"fmt"
	pb "github.com/57ajay/grpcgo/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"log"
)

const (
	serverAddr = "localhost:50051"
)

func main() {
	fmt.Println("Welcome to grpcClient side")
	conn, err := grpc.NewClient(serverAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))

	if err != nil {
		log.Fatal("some error: ", err)
	}

	defer conn.Close()
	grpcClient := pb.NewUserServiceClient(conn)

	req := &pb.CreateUserRequest{
		Name:  "Ajay Upadhyay",
		Email: "57u.ajay@gmail.com",
	}

	log.Printf("Sending request to server to create a user with name: %s", req.Name)
	res, err := grpcClient.CreateUser(context.Background(), req)

	if err != nil {
		log.Fatal("Error creating user.\n", err)
	}

	log.Printf("CreateUser Response: Id: %s", res.GetId())

}
