package grpc

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	googlegrpc "google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"usermanagementsystem/internal/model"
	"usermanagementsystem/internal/service"
	v1 "usermanagementsystem/proto"
)

// Server implements the generated gRPC service interfaces.
type Server struct {
	v1.UnimplementedUserServiceServer
	v1.UnimplementedDepartmentServiceServer
	v1.UnimplementedRoleServiceServer

	userService service.UserService
	deptService service.DepartmentService
	roleService service.RoleService
	authService service.AuthService
}

// NewServer instantiates a new gRPC Server with injected domain services.
func NewServer(
	userService service.UserService,
	deptService service.DepartmentService,
	roleService service.RoleService,
	authService service.AuthService,
) *Server {
	return &Server{
		userService: userService,
		deptService: deptService,
		roleService: roleService,
		authService: authService,
	}
}

// UnaryServerLoggingInterceptor extracts the x-request-id metadata header propagated by
// the API Gateway and logs execution duration for distributed tracing.
func UnaryServerLoggingInterceptor() googlegrpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *googlegrpc.UnaryServerInfo,
		handler googlegrpc.UnaryHandler,
	) (interface{}, error) {
		start := time.Now()
		reqID := "local"
		if md, ok := metadata.FromIncomingContext(ctx); ok {
			if vals := md.Get("x-request-id"); len(vals) > 0 {
				reqID = vals[0]
			}
		}

		resp, err := handler(ctx, req)
		duration := time.Since(start)

		if err != nil {
			fmt.Printf("[gRPC] [ERR] %-35s | req_id=%s | latency=%10v | err=%v\n", info.FullMethod, reqID, duration, err)
		} else {
			fmt.Printf("[gRPC] [OK]  %-35s | req_id=%s | latency=%10v\n", info.FullMethod, reqID, duration)
		}

		return resp, err
	}
}

// ==========================================
// User Service RPCs
// ==========================================

func (s *Server) RegisterUser(ctx context.Context, req *v1.CreateUserRequest) (*v1.UserResponse, error) {
	var deptID *int
	if req.DepartmentId != nil {
		id := int(*req.DepartmentId)
		deptID = &id
	}

	roleIDs := make([]int, len(req.RoleIds))
	for i, rID := range req.RoleIds {
		roleIDs[i] = int(rID)
	}

	input := service.CreateUserInput{
		Name:         req.Name,
		Email:        req.Email,
		Password:     req.Password,
		DepartmentID: deptID,
		RoleIDs:      roleIDs,
		Status:       req.Status,
	}

	user, err := s.userService.CreateUser(ctx, input)
	if err != nil {
		return nil, mapServiceErrorToGRPC(err)
	}

	return &v1.UserResponse{
		User: mapModelToProtoUser(user),
	}, nil
}
func (s *Server) CreateUser(ctx context.Context, req *v1.CreateUserRequest) (*v1.UserResponse, error) {
	var deptID *int
	if req.DepartmentId != nil {
		id := int(*req.DepartmentId)
		deptID = &id
	}

	roleIDs := make([]int, len(req.RoleIds))
	for i, rID := range req.RoleIds {
		roleIDs[i] = int(rID)
	}

	input := service.CreateUserInput{
		Name:         req.Name,
		Email:        req.Email,
		Password:     req.Password,
		DepartmentID: deptID,
		RoleIDs:      roleIDs,
		Status:       req.Status,
	}

	user, err := s.userService.CreateUser(ctx, input)
	if err != nil {
		return nil, mapServiceErrorToGRPC(err)
	}

	return &v1.UserResponse{User: mapModelToProtoUser(user)}, nil
}

func (s *Server) GetUser(ctx context.Context, req *v1.GetUserRequest) (*v1.UserResponse, error) {
	user, err := s.userService.GetUser(ctx, int(req.Id))
	if err != nil {
		return nil, mapServiceErrorToGRPC(err)
	}

	return &v1.UserResponse{User: mapModelToProtoUser(user)}, nil
}

func (s *Server) ListUsers(ctx context.Context, req *v1.ListUsersRequest) (*v1.ListUsersResponse, error) {
	page := int(req.Page)
	if page <= 0 {
		page = 1
	}
	limit := int(req.Limit)
	if limit <= 0 {
		limit = 10
	} else if limit > 100 {
		limit = 100
	}
	offset := (page - 1) * limit

	var deptID *int
	if req.DepartmentId != nil {
		id := int(*req.DepartmentId)
		deptID = &id
	}

	var roleID *int
	if req.RoleId != nil {
		r := int(*req.RoleId)
		roleID = &r
	}

	filter := model.UserFilter{
		Search:       req.Search,
		DepartmentID: deptID,
		RoleID:       roleID,
		Status:       req.Status,
		SortBy:       req.SortBy,
		SortOrder:    req.SortOrder,
		Limit:        limit,
		Offset:       offset,
	}

	users, totalCount, err := s.userService.ListUsers(ctx, filter)
	if err != nil {
		return nil, mapServiceErrorToGRPC(err)
	}

	protoUsers := make([]*v1.User, len(users))
	for i := range users {
		protoUsers[i] = mapModelToProtoUser(&users[i])
	}

	totalPages := 0
	if totalCount > 0 {
		totalPages = int(math.Ceil(float64(totalCount) / float64(limit)))
	}

	return &v1.ListUsersResponse{
		Users:      protoUsers,
		TotalCount: int32(totalCount),
		Page:       int32(page),
		Limit:      int32(limit),
		TotalPages: int32(totalPages),
	}, nil
}

func (s *Server) UpdateUser(ctx context.Context, req *v1.UpdateUserRequest) (*v1.UserResponse, error) {
	var deptID *int
	if req.DepartmentId != nil {
		id := int(*req.DepartmentId)
		deptID = &id
	}

	var roleIDs []int
	if req.RoleIds != nil {
		roleIDs = make([]int, len(req.RoleIds))
		for i, rID := range req.RoleIds {
			roleIDs[i] = int(rID)
		}
	}

	input := service.UpdateUserInput{
		Name:         req.Name,
		Email:        req.Email,
		DepartmentID: deptID,
		RoleIDs:      roleIDs,
		Status:       req.Status,
	}

	user, err := s.userService.UpdateUser(ctx, int(req.Id), input)
	if err != nil {
		return nil, mapServiceErrorToGRPC(err)
	}

	return &v1.UserResponse{User: mapModelToProtoUser(user)}, nil
}

func (s *Server) DeleteUser(ctx context.Context, req *v1.DeleteUserRequest) (*v1.DeleteUserResponse, error) {
	err := s.userService.DeleteUser(ctx, int(req.Id))
	if err != nil {
		return nil, mapServiceErrorToGRPC(err)
	}

	return &v1.DeleteUserResponse{
		Success: true,
		Message: "User deleted successfully",
	}, nil
}

func (s *Server) Login(ctx context.Context, req *v1.LoginRequest) (*v1.LoginResponse, error) {
	result, err := s.authService.Login(ctx, req.Email, req.Password)
	if err != nil {
		return nil, mapServiceErrorToGRPC(err)
	}

	return &v1.LoginResponse{
		Token: result.Token,
		User:  mapModelToProtoUser(result.User),
		Roles: result.Roles,
	}, nil
}

// ==========================================
// Department Service RPCs
// ==========================================

func (s *Server) CreateDepartment(ctx context.Context, req *v1.CreateDepartmentRequest) (*v1.DepartmentResponse, error) {
	dept, err := s.deptService.CreateDepartment(ctx, req.Name, req.Description)
	if err != nil {
		return nil, mapServiceErrorToGRPC(err)
	}
	return &v1.DepartmentResponse{Department: mapModelToProtoDepartment(dept)}, nil
}

func (s *Server) GetDepartment(ctx context.Context, req *v1.GetDepartmentRequest) (*v1.DepartmentResponse, error) {
	dept, err := s.deptService.GetDepartment(ctx, int(req.Id))
	if err != nil {
		return nil, mapServiceErrorToGRPC(err)
	}
	return &v1.DepartmentResponse{Department: mapModelToProtoDepartment(dept)}, nil
}

func (s *Server) ListDepartments(ctx context.Context, _ *v1.ListDepartmentsRequest) (*v1.ListDepartmentsResponse, error) {
	depts, err := s.deptService.ListDepartments(ctx)
	if err != nil {
		return nil, mapServiceErrorToGRPC(err)
	}
	protoDepts := make([]*v1.Department, len(depts))
	for i := range depts {
		protoDepts[i] = mapModelToProtoDepartment(&depts[i])
	}
	return &v1.ListDepartmentsResponse{Departments: protoDepts}, nil
}

func (s *Server) UpdateDepartment(ctx context.Context, req *v1.UpdateDepartmentRequest) (*v1.DepartmentResponse, error) {
	dept, err := s.deptService.UpdateDepartment(ctx, int(req.Id), req.Name, req.Description)
	if err != nil {
		return nil, mapServiceErrorToGRPC(err)
	}
	return &v1.DepartmentResponse{Department: mapModelToProtoDepartment(dept)}, nil
}

func (s *Server) DeleteDepartment(ctx context.Context, req *v1.DeleteDepartmentRequest) (*v1.DeleteDepartmentResponse, error) {
	err := s.deptService.DeleteDepartment(ctx, int(req.Id))
	if err != nil {
		return nil, mapServiceErrorToGRPC(err)
	}
	return &v1.DeleteDepartmentResponse{Success: true, Message: "Department deleted successfully"}, nil
}

// ==========================================
// Role Service RPCs
// ==========================================

func (s *Server) CreateRole(ctx context.Context, req *v1.CreateRoleRequest) (*v1.RoleResponse, error) {
	role, err := s.roleService.CreateRole(ctx, req.Name, req.Description)
	if err != nil {
		return nil, mapServiceErrorToGRPC(err)
	}
	return &v1.RoleResponse{Role: mapModelToProtoRole(role)}, nil
}

func (s *Server) GetRole(ctx context.Context, req *v1.GetRoleRequest) (*v1.RoleResponse, error) {
	role, err := s.roleService.GetRole(ctx, int(req.Id))
	if err != nil {
		return nil, mapServiceErrorToGRPC(err)
	}
	return &v1.RoleResponse{Role: mapModelToProtoRole(role)}, nil
}

func (s *Server) ListRoles(ctx context.Context, _ *v1.ListRolesRequest) (*v1.ListRolesResponse, error) {
	roles, err := s.roleService.ListRoles(ctx)
	if err != nil {
		return nil, mapServiceErrorToGRPC(err)
	}
	protoRoles := make([]*v1.Role, len(roles))
	for i := range roles {
		protoRoles[i] = mapModelToProtoRole(&roles[i])
	}
	return &v1.ListRolesResponse{Roles: protoRoles}, nil
}

func (s *Server) UpdateRole(ctx context.Context, req *v1.UpdateRoleRequest) (*v1.RoleResponse, error) {
	role, err := s.roleService.UpdateRole(ctx, int(req.Id), req.Name, req.Description)
	if err != nil {
		return nil, mapServiceErrorToGRPC(err)
	}
	return &v1.RoleResponse{Role: mapModelToProtoRole(role)}, nil
}

func (s *Server) DeleteRole(ctx context.Context, req *v1.DeleteRoleRequest) (*v1.DeleteRoleResponse, error) {
	err := s.roleService.DeleteRole(ctx, int(req.Id))
	if err != nil {
		return nil, mapServiceErrorToGRPC(err)
	}
	return &v1.DeleteRoleResponse{Success: true, Message: "Role deleted successfully"}, nil
}

// ==========================================
// Model to Protobuf Mapping Helpers
// ==========================================

func mapModelToProtoUser(u *model.User) *v1.User {
	if u == nil {
		return nil
	}

	var deptID *int32
	if u.DepartmentID != nil {
		id := int32(*u.DepartmentID)
		deptID = &id
	}

	protoRoles := make([]*v1.Role, len(u.Roles))
	for i := range u.Roles {
		protoRoles[i] = mapModelToProtoRole(&u.Roles[i])
	}

	return &v1.User{
		Id:           int32(u.ID),
		Name:         u.Name,
		Email:        u.Email,
		DepartmentId: deptID,
		Status:       u.Status,
		CreatedAt:    u.CreatedAt.Format(time.RFC3339),
		UpdatedAt:    u.UpdatedAt.Format(time.RFC3339),
		Department:   mapModelToProtoDepartment(u.Department),
		Roles:        protoRoles,
	}
}

func mapModelToProtoDepartment(d *model.Department) *v1.Department {
	if d == nil {
		return nil
	}
	return &v1.Department{
		Id:          int32(d.ID),
		Name:        d.Name,
		Description: d.Description,
		CreatedAt:   d.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   d.UpdatedAt.Format(time.RFC3339),
	}
}

func mapModelToProtoRole(r *model.Role) *v1.Role {
	if r == nil {
		return nil
	}
	return &v1.Role{
		Id:          int32(r.ID),
		Name:        r.Name,
		Description: r.Description,
		CreatedAt:   r.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   r.UpdatedAt.Format(time.RFC3339),
	}
}

// mapServiceErrorToGRPC translates Go domain errors into standard gRPC status codes.
func mapServiceErrorToGRPC(err error) error {
	switch {
	case errors.Is(err, service.ErrUserNotFound),
		errors.Is(err, service.ErrDepartmentNotFound),
		errors.Is(err, service.ErrRoleNotFound):
		return status.Error(codes.NotFound, err.Error())

	case errors.Is(err, service.ErrEmailAlreadyInUse):
		return status.Error(codes.AlreadyExists, err.Error())

	case errors.Is(err, service.ErrInvalidCredentials):
		return status.Error(codes.Unauthenticated, "invalid email or password")

	case errors.Is(err, service.ErrInactiveUser):
		return status.Error(codes.PermissionDenied, "user account is deactivated")

	case errors.Is(err, service.ErrEmptyName),
		errors.Is(err, service.ErrInvalidEmail),
		errors.Is(err, service.ErrPasswordTooWeak),
		errors.Is(err, service.ErrInvalidStatus):
		return status.Error(codes.InvalidArgument, err.Error())

	default:
		return status.Errorf(codes.Internal, "internal server error: %v", err)
	}
}
