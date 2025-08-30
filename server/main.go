package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"time"

	pb "github.com/57ajay/grpcgo/proto"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const dbConnectionString = "postgres://ajay:57ajay@localhost:5432/grpc-postgres"

type server struct {
	db *pgxpool.Pool
	pb.UnimplementedUserServiceServer
}

func (s *server) CreateUser(ctx context.Context, req *pb.CreateUserRequest) (*pb.CreateUserResponse, error) {
	log.Printf("Recieved User create Request for name: %s, email: %s.\n", req.GetName(), req.GetEmail())

	var newUserId string
	sqlQuery := `INSERT INTO users (name, email) VALUES ($1, $2) RETURNING id`
	err := s.db.QueryRow(ctx, sqlQuery, req.GetName(), req.GetEmail()).Scan(&newUserId)

	if err != nil {
		log.Printf("Failed to insert user in db.\nerr: %v\n", err)
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, status.Errorf(codes.AlreadyExists, "a user with email '%s' already exists", req.GetEmail())
		}
		return nil, status.Errorf(codes.Internal, "unexpected database error")
	}

	log.Printf("Successfully inserted user with userId: %s", newUserId)

	return &pb.CreateUserResponse{Id: newUserId}, nil

}

func (s *server) ListUsers(req *pb.ListUsersRequest, stream pb.UserService_ListUsersServer) error {
	log.Println("Received ListUsers request")
	sqlQuery := `SELECT id, name, email FROM users`
	ctx := stream.Context()

	rows, err := s.db.Query(ctx, sqlQuery)

	if err != nil {
		log.Printf("Failed to query users: %v", err)
		return err
	}
	defer rows.Close()

	for rows.Next() {
		if ctx.Err() != nil {
			log.Printf("Client canceled request. Aborting stream.")
			return ctx.Err()
		}

		var user pb.User

		if err := rows.Scan(&user.Id, &user.Name, &user.Email); err != nil {
			log.Printf("Failed to scan user row: %v", err)
			return err
		}

		if err := stream.Send(&user); err != nil {
			log.Printf("Failed to send user to stream: %v", err)
			return err
		}

	}
	if err := rows.Err(); err != nil {
		log.Printf("Error iterating user rows: %v", err)
		return err
	}

	log.Println("Finished streaming users")
	return nil

}

func (s *server) CreateUsers(stream pb.UserService_CreateUsersServer) error {
	ctx := stream.Context()
	log.Println("Received CreateUsers stream request")
	var createdUserCount int32 = 0

	for {

		if ctx.Err() != nil {
			log.Printf("Client canceled request. Aborting stream.")
			return ctx.Err()
		}

		req, err := stream.Recv()

		if err == io.EOF {
			log.Printf("Finished client stream. Total users created: %d", createdUserCount)
			return stream.SendAndClose(&pb.CreateUsersResponse{
				Result:           "Users created Successfully.",
				CreatedUserCount: createdUserCount,
			})
		}

		if err != nil {
			log.Printf("Error receiving the stream.\n err: %v", err)
			return err
		}

		log.Printf("Processing CreateUser request for name: %s", req.GetName())
		sqlQuery := `INSERT INTO users (name, email) VALUES ($1, $2)`

		_, err = s.db.Exec(ctx, sqlQuery, req.GetName(), req.GetEmail())

		if err != nil {
			log.Printf("Failed to insert user %s: %v", req.GetName(), err)
			continue
		}

		createdUserCount++

	}
}

func (s *server) UserChat(stream pb.UserService_UserChatServer) error {
	ctx := stream.Context()
	log.Println("UserChat session started")

	for {
		if ctx.Err() != nil {
			log.Printf("Client canceled request. Aborting stream.")
			return ctx.Err()
		}
		req, err := stream.Recv()
		if err == io.EOF {
			log.Println("Client closed the chat stream.")
			return nil
		}
		if err != nil {
			log.Printf("Error receiving from chat stream: %v", err)
			return err
		}

		log.Printf("Received chat message from User ID '%s': %s", req.GetUserId(), req.GetMessage())

		replyMsg := &pb.UserChatMessage{
			UserId:  "Server",
			Message: "Acknowledged your message: '" + req.GetMessage() + "'",
		}
		if err := stream.Send(replyMsg); err != nil {
			log.Printf("Error sending reply to chat stream: %v", err)
			return err
		}
	}

}

func authenticate(ctx context.Context) error {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return status.Error(codes.Unauthenticated, "metadata is not provided")
	}

	values := md["authorization"]
	if len(values) == 0 {
		return status.Error(codes.Unauthenticated, "authorization token is not provided")
	}

	secretToken := os.Getenv("SECRET_KEY")
	if len(secretToken) == 0 {
		return status.Error(codes.Internal, "server is not configured correctly")
	}

	if values[0] != "Bearer "+secretToken {
		return status.Error(codes.Unauthenticated, "authorization token is invalid")
	}

	return nil
}

// Unary Interceptor
func authInterceptor(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
	if err := authenticate(ctx); err != nil {
		return nil, err
	}
	return handler(ctx, req)
}

// Stream Interceptor
func authStreamInterceptor(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	if err := authenticate(ss.Context()); err != nil {
		return err
	}
	return handler(srv, ss)
}

func main() {
	fmt.Println("Welcome to grpcServer side.")

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	dbPool, err := pgxpool.New(ctx, dbConnectionString)

	if err != nil {
		log.Fatal("error connecting to database.\nerr: ", err)
	}

	defer dbPool.Close()
	log.Println("Successfully connected to the database")

	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	creds, err := credentials.NewServerTLSFromFile("server.crt", "server.key")
	if err != nil {
		log.Fatalf("failed to load TLS certificates: %v", err)
	}

	grpcServer := grpc.NewServer(grpc.Creds(creds), grpc.UnaryInterceptor(authInterceptor), grpc.StreamInterceptor(authStreamInterceptor))
	pb.RegisterUserServiceServer(grpcServer, &server{db: dbPool})
	log.Println("gRPC server started on port :50051")

	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to server: %v", err)
	}
}
