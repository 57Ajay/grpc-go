package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"google.golang.org/grpc/metadata"

	pb "github.com/57ajay/grpcgo/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
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

func userChat(ctx context.Context, grpcClient pb.UserServiceClient) {
	log.Println("---------Chatting with server--------")

	chatUser, err := grpcClient.UserChat(ctx)
	if err != nil {
		log.Println("Failed to start chat with server.\nErr: ", err)
	}

	waitc := make(chan struct{})

	go func() {
		for {

			res, err := chatUser.Recv()
			if err == io.EOF {
				log.Println("Server closed the stream.")
				close(waitc)
				return
			}
			if err != nil {
				log.Printf("Error receiving message: %v", err)
				close(waitc)
				return
			}
			log.Printf("Received from Server: [%s] %s", res.GetUserId(), res.GetMessage())
		}
	}()

	messagesToSend := []*pb.UserChatMessage{
		{UserId: "client-1", Message: "Hello, server!"},
		{UserId: "client-1", Message: "How are you?"},
		{UserId: "client-1", Message: "gRPC is awesome."},
	}

	for _, msg := range messagesToSend {
		log.Printf("Sending to Server: %s", msg.GetMessage())
		if err := chatUser.Send(msg); err != nil {
			log.Fatalf("Failed to send message: %v", err)
		}
		time.Sleep(1 * time.Second)
	}
	chatUser.CloseSend()
	<-waitc
	log.Println("chat finish")

}

func main() {
	fmt.Println("Welcome to grpcClient side")
	creds, err := credentials.NewClientTLSFromFile("server.crt", "localhost") // The second param is serverNameOverride
	if err != nil {
		log.Fatalf("failed to load TLS credentials: %v", err)
	}
	secretToken := os.Getenv("SECRET_KEY")
	if len(secretToken) == 0 {
		log.Fatal("PLEASE ADD SECRET_KEY")
	}
	conn, err := grpc.NewClient(serverAddr, grpc.WithTransportCredentials(creds))

	md := metadata.Pairs("authorization", "Bearer "+secretToken)
	ctx := metadata.NewOutgoingContext(context.Background(), md)

	if err != nil {
		log.Fatal("some error: ", err)
	}

	defer conn.Close()
	grpcClient := pb.NewUserServiceClient(conn)
	ctx, cancel := context.WithTimeout(ctx, time.Second*10)
	defer cancel()

	// Unary rpc
	// createUser(ctx, grpcClient)
	//
	// // server streaming rpc
	// listUsers(ctx, grpcClient)
	//
	// // client streaming rpc
	// createUsers(ctx, grpcClient)
	//
	// bidirectional streaming rpc
	userChat(ctx, grpcClient)

}
