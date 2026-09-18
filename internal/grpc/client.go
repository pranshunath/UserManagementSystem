package grpc

import (
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	v1 "usermanagementsystem/proto"
)

// Client bundles typed gRPC service clients along with the connection handle.
type Client struct {
	conn       *grpc.ClientConn
	User       v1.UserServiceClient
	Department v1.DepartmentServiceClient
	Role       v1.RoleServiceClient
}

// NewClient dials the target gRPC server and initializes type-safe RPC client stubs.
func NewClient(target string) (*Client, error) {
	// Use insecure credentials for internal cluster/local communication (or TLS in production)
	conn, err := grpc.NewClient(target, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to dial gRPC server at %s: %w", target, err)
	}

	return &Client{
		conn:       conn,
		User:       v1.NewUserServiceClient(conn),
		Department: v1.NewDepartmentServiceClient(conn),
		Role:       v1.NewRoleServiceClient(conn),
	}, nil
}

// Close gracefully terminates the underlying gRPC client connection.
func (c *Client) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}
