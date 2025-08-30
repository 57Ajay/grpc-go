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

func createUser(ctx context.Context, grpcClient pb.UserServiceClient) {
	log.Println("--- Unary RPC ---")
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

}

func listUsers(ctx context.Context, grpcClient pb.UserServiceClient) {
	log.Println("--- Server Streaming RPC ---")

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

func createUsers(ctx context.Context, grpcClient pb.UserServiceClient) {
	log.Println("--- Client Streaming RPC ---")
	usersToCreate := []*pb.CreateUserRequest{
		{Name: "Name" + time.Now().Format("20060102150405") + "1", Email: time.Now().Format("20060102150405") + "@example1.com"},
		{Name: "Name" + time.Now().Format("20060102150405") + "2", Email: time.Now().Format("20060102150405") + "@example2.com"},
		{Name: "Name" + time.Now().Format("20060102150405") + "3", Email: time.Now().Format("20060102150405") + "@example3.com"},
	}
	createStream, err := grpcClient.CreateUsers(ctx)
	if err != nil {
		log.Fatalf("could not create users stream: %v", err)
	}

	for _, userReq := range usersToCreate {
		log.Printf("Sending user: %s", userReq.GetName())
		if err = createStream.Send(userReq); err != nil {
			log.Fatalf("failed to send user on stream: %v", err)
		}
		time.Sleep(100 * time.Millisecond)
	}
	res, err := createStream.CloseAndRecv()
	if err != nil {
		log.Fatalf("failed to receive response from CreateUsers stream: %v", err)
	}

	log.Printf("Server Response: %s (Created %d users)", res.GetResult(), res.GetCreatedUserCount())
	log.Println("---------------------------------")
}

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

	// Unary rpc
	// createUser(ctx, grpcClient)

	// server streaming rpc
	listUsers(ctx, grpcClient)

	// client streaming rpc
	// createUsers(ctx, grpcClient)

}
