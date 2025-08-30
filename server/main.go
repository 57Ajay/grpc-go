package main

import (
	"context"
	"fmt"
	"log"
	"net"

	pb "github.com/57ajay/grpcgo/proto"
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc"
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
		log.Fatal("Failed to insert user in db.\nerr: ", err)
		return nil, err
	}

	log.Printf("Successfully inserted user with userId: %s", newUserId)

	return &pb.CreateUserResponse{Id: newUserId}, nil

}

func (s *server) ListUsers(req *pb.ListUsersRequest, stream pb.UserService_ListUsersServer) error {
	log.Println("Received ListUsers request")
	sqlQuery := `SELECT id, name, email FROM users`

	rows, err := s.db.Query(context.Background(), sqlQuery)

	if err != nil {
		log.Printf("Failed to query users: %v", err)
		return err
	}
	defer rows.Close()

	for rows.Next() {
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

func main() {
	fmt.Println("Welcome to grpcServer side.")

	ctx := context.Background()
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

	grpcServer := grpc.NewServer()
	pb.RegisterUserServiceServer(grpcServer, &server{db: dbPool})
	log.Println("gRPC server started on port :50051")

	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to server: %v", err)
	}
}
