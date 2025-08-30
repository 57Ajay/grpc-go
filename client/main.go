package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"time"

	pb "github.com/57ajay/grpcgo/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
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
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	req := &pb.CreateUserRequest{
		Name:  "Ajay Upadhyay" + time.Now().Format("20060102150405"),
		Email: time.Now().Format("20060102150405") + "@example.com",
	}

	log.Printf("Sending request to server to create a user with name: %s", req.Name)
	res, err := grpcClient.CreateUser(ctx, req)

	if err != nil {
		log.Fatal("Error creating user.\n", err)
	}

	log.Printf("CreateUser Response: Id: %s\n", res.GetId())

	log.Println("Fetching and streaming users...")

	stream, err := grpcClient.ListUsers(ctx, &pb.ListUsersRequest{})

	if err != nil {
		log.Fatal("Error listing users.\nerr: ", err)
	}

	var userList []*pb.User

	for {
		user, err := stream.Recv()
		if err == io.EOF {
			log.Println("Finished receiving user stream.")
			break
		}
		if err != nil {
			log.Fatalf("error while receiving stream: %v", err)
		}

		// log.Printf("Received User: ID=%s, Name=%s, Email=%s", user.GetId(), user.GetName(), user.GetEmail())
		// userList = append(userList, pb.User{Id: user.GetId(), Name: user.GetName(), Email: user.GetEmail()})
		userList = append(userList, user)
	}

	fmt.Println("\n========= User List =========")
	fmt.Printf("%-40s %-25s %-30s\n", "ID", "NAME", "EMAIL")
	fmt.Println("-------------------------------------------------------------------------------------------------------------")
	for _, u := range userList {
		fmt.Printf("%-40s %-25s %-30s\n", u.Id, u.Name, u.Email)
	}
	fmt.Println("================================")

}
