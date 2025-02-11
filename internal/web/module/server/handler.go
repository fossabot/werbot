package server

import (
	"context"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/werbot/werbot/internal/database/sanitize"
	"github.com/werbot/werbot/internal/message"
	"github.com/werbot/werbot/internal/utils/validator"
	"github.com/werbot/werbot/internal/web/httputil"
	"github.com/werbot/werbot/internal/web/middleware"

	pb_firewall "github.com/werbot/werbot/internal/grpc/proto/firewall"
	pb "github.com/werbot/werbot/internal/grpc/proto/server"
)

// @Summary      Show information about server or list of all servers
// @Tags         servers
// @Accept       json
// @Produce      json
// @Param        user_id     path     uuid false "Server ID. Parameter Accessible with ROLE_ADMIN rights"
// @Param        server_id   path     uuid false "Server ID"
// @Param        project_id  path     uuid true "Project ID"
// @Success      200         {object} httputil.HTTPResponse{data=pb.ListServer_Response}
// @Failure      400,401,500 {object} httputil.HTTPResponse
// @Router       /v1/servers [get]
func (h *Handler) getServer(c *fiber.Ctx) error {
	input := new(pb.GetServer_Request)
	c.QueryParser(input)
	if err := validator.ValidateStruct(input); err != nil {
		return httputil.StatusBadRequest(c, message.ErrValidateBodyParams, err)
	}

	userParameter := middleware.GetUserParameters(c)
	userID := userParameter.GetUserID(input.GetUserId())

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	rClient := pb.NewServerHandlersClient(h.grpc.Client)

	// show all project
	if input.GetServerId() == "" {
		pagination := httputil.GetPaginationFromCtx(c)
		sanitizeSQL, _ := sanitize.SQL(`project_id = $1 AND user_id = $2`, input.GetProjectId(), userID)
		servers, err := rClient.ListServers(ctx, &pb.ListServers_Request{
			Limit:  pagination.GetLimit(),
			Offset: pagination.GetOffset(),
			SortBy: pagination.GetSortBy(),
			Query:  sanitizeSQL,
		})
		if err != nil {
			return httputil.ReturnGRPCError(c, err)
		}
		if servers.GetTotal() == 0 {
			return httputil.StatusNotFound(c, message.ErrNotFound, nil)
		}
		return httputil.StatusOK(c, "List of servers available in this project", servers)
	}

	// show information about the server
	server, err := rClient.GetServer(ctx, &pb.GetServer_Request{
		UserId:    userID,
		ServerId:  input.GetServerId(),
		ProjectId: input.GetProjectId(),
	})
	if err != nil {
		return httputil.ReturnGRPCError(c, err)
	}
	if server == nil {
		return httputil.StatusNotFound(c, message.ErrNotFound, nil)
	}

	return httputil.StatusOK(c, "Information about the project", server)
}

// @Summary      Adding a new server
// @Tags         servers
// @Accept       json
// @Produce      json
// @Param        req         body     pb.CreateServer_Request{}
// @Success      200         {object} httputil.HTTPResponse{data=pb.CreateServer_Response}
// @Failure      400,401,500 {object} httputil.HTTPResponse
// @Router       /v1/servers [post]
func (h *Handler) addServer(c *fiber.Ctx) error {
	input := new(pb.CreateServer_Request)
	c.BodyParser(input)
	if err := validator.ValidateStruct(input); err != nil {
		return httputil.StatusBadRequest(c, message.ErrValidateBodyParams, err)
	}

	userParameter := middleware.GetUserParameters(c)
	userID := userParameter.GetUserID(input.GetUserId())

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	rClient := pb.NewServerHandlersClient(h.grpc.Client)

	server, err := rClient.CreateServer(ctx, &pb.CreateServer_Request{
		UserId:             userID,
		ProjectId:          input.GetProjectId(),
		Address:            input.GetAddress(),
		Port:               input.GetPort(),
		Login:              input.GetLogin(),
		Title:              input.GetTitle(),
		PrivateDescription: input.GetPrivateDescription(),
		PublicDescription:  input.GetPublicDescription(),
		Auth:               input.GetAuth(),
		Scheme:             input.GetScheme(),
		Audit:              input.GetAudit(),
		Active:             input.GetActive(),
		Password:           input.GetPassword(),
		PublicKey:          input.GetPublicKey(),
		KeyUuid:            input.GetKeyUuid(),
	})
	if err != nil {
		return httputil.ReturnGRPCError(c, err)
	}
	return httputil.StatusOK(c, "Server added", server)
}

// @Summary      Server update
// @Tags         servers
// @Accept       json
// @Produce      json
// @Param        req         body     pb.UpdateServer_Request{}
// @Success      200         {object} httputil.HTTPResponse
// @Failure      400,401,500 {object} httputil.HTTPResponse
// @Router       /v1/servers [patch]
func (h *Handler) patchServer(c *fiber.Ctx) error {
	input := new(pb.UpdateServer_Request)
	c.BodyParser(input)
	if err := validator.ValidateStruct(input); err != nil {
		return httputil.StatusBadRequest(c, message.ErrValidateBodyParams, err)
	}

	userParameter := middleware.GetUserParameters(c)
	userID := userParameter.GetUserID(input.GetUserId())

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	rClient := pb.NewServerHandlersClient(h.grpc.Client)

	_, err := rClient.UpdateServer(ctx, &pb.UpdateServer_Request{
		UserId:             userID,
		ServerId:           input.GetServerId(),
		ProjectId:          input.GetProjectId(),
		Address:            input.GetAddress(),
		Port:               input.GetPort(),
		Login:              input.GetLogin(),
		Title:              input.GetTitle(),
		PrivateDescription: input.GetPrivateDescription(),
		PublicDescription:  input.GetPublicDescription(),
		Audit:              input.GetAudit(),
		Active:             input.GetActive(),
	})
	if err != nil {
		return httputil.ReturnGRPCError(c, err)
	}

	// access setting
	access := new(pb.UpdateServerAccess_Request)
	c.BodyParser(access)
	if err := validator.ValidateStruct(input); err != nil {
		return httputil.StatusBadRequest(c, message.ErrValidateBodyParams, err)
	}

	// If the password is not indicated, skip the next step
	if access.Auth == pb.ServerAuth_PASSWORD && access.Password == "" {
		return httputil.StatusOK(c, "Server data updated", nil)
	}

	_, err = rClient.UpdateServerAccess(ctx, &pb.UpdateServerAccess_Request{
		UserId:    userID,
		ServerId:  input.GetServerId(),
		ProjectId: input.GetProjectId(),
		Auth:      access.GetAuth(),
		Password:  access.GetPassword(),
		PublicKey: access.GetPublicKey(),
		KeyUuid:   access.GetKeyUuid(),
	})
	if err != nil {
		return httputil.ReturnGRPCError(c, err)
	}
	return httputil.StatusOK(c, "Server data updated", nil)
}

// @Summary      Delete server
// @Tags         servers
// @Accept       json
// @Produce      json
// @Param        user_id     path     uuid true "User ID"
// @Param        project_id  path     uuid true "Project ID"
// @Param        server_id   path     uuid true "Server ID"
// @Success      200         {object} httputil.HTTPResponse
// @Failure      400,401,500 {object} httputil.HTTPResponse
// @Router       /v1/servers [delete]
func (h *Handler) deleteServer(c *fiber.Ctx) error {
	input := new(pb.DeleteServer_Request)
	c.QueryParser(input)
	if err := validator.ValidateStruct(input); err != nil {
		return httputil.StatusBadRequest(c, message.ErrValidateBodyParams, err)
	}

	userParameter := middleware.GetUserParameters(c)
	userID := userParameter.GetUserID(input.GetUserId())

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	rClient := pb.NewServerHandlersClient(h.grpc.Client)

	_, err := rClient.DeleteServer(ctx, &pb.DeleteServer_Request{
		UserId:    userID,
		ProjectId: input.GetProjectId(),
		ServerId:  input.GetServerId(),
	})
	if err != nil {
		return httputil.ReturnGRPCError(c, err)
	}
	return httputil.StatusOK(c, "Server deleted", nil)
}

// @Summary      Get server access
// @Tags         servers
// @Accept       json
// @Produce      json
// @Param        user_id     path     uuid true "User ID"
// @Param        project_id  path     uuid true "Project ID"
// @Param        server_id   path     uuid true "Server ID"
// @Success      200         {object} httputil.HTTPResponse{data=pb.GetServerAccess_Response}
// @Failure      400,401,500 {object} httputil.HTTPResponse
// @Router       /v1/servers/access [get]
func (h *Handler) getServerAccess(c *fiber.Ctx) error {
	input := new(pb.GetServerAccess_Request)
	c.QueryParser(input)
	if err := validator.ValidateStruct(input); err != nil {
		return httputil.StatusBadRequest(c, message.ErrValidateBodyParams, err)
	}

	userParameter := middleware.GetUserParameters(c)
	userID := userParameter.GetUserID(input.GetUserId())

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	rClient := pb.NewServerHandlersClient(h.grpc.Client)

	access, err := rClient.GetServerAccess(ctx, &pb.GetServerAccess_Request{
		UserId:    userID,
		ProjectId: input.GetProjectId(),
		ServerId:  input.GetServerId(),
	})
	if err != nil {
		return httputil.ReturnGRPCError(c, err)
	}
	if access == nil {
		return httputil.StatusNotFound(c, message.ErrNotFound, nil)
	}
	return httputil.StatusOK(c, "Server access", access)
}

// @Summary      Get server activity
// @Tags         servers
// @Accept       json
// @Produce      json
// @Param        user_id     path     uuid true "User ID"
// @Param        project_id  path     uuid true "Project ID"
// @Param        server_id   path     uuid true "Server ID"
// @Success      200         {object} httputil.HTTPResponse{data=pb.GetServerActivity_Response}
// @Failure      400,401,500 {object} httputil.HTTPResponse
// @Router       /v1/servers/activity [get]
func (h *Handler) getServerActivity(c *fiber.Ctx) error {
	input := new(pb.GetServerActivity_Request)
	c.QueryParser(input)
	if err := validator.ValidateStruct(input); err != nil {
		return httputil.StatusBadRequest(c, message.ErrValidateBodyParams, err)
	}

	userParameter := middleware.GetUserParameters(c)
	userID := userParameter.GetUserID(input.GetUserId())

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	rClient := pb.NewServerHandlersClient(h.grpc.Client)

	activity, err := rClient.GetServerActivity(ctx, &pb.GetServerActivity_Request{
		UserId:    userID,
		ServerId:  input.GetServerId(),
		ProjectId: input.GetProjectId(),
	})
	if err != nil {
		return httputil.ReturnGRPCError(c, err)
	}
	if activity == nil {
		return httputil.StatusNotFound(c, message.ErrNotFound, nil)
	}
	return httputil.StatusOK(c, "Server activity", activity)
}

// @Summary      Update server activity
// @Tags         servers
// @Accept       json
// @Produce      json
// @Param        req         body     pb.UpdateServerActivity_Request{}
// @Success      200         {object} httputil.HTTPResponse
// @Failure      400,401,500 {object} httputil.HTTPResponse
// @Router       /v1/servers/activity [patch]
func (h *Handler) patchServerActivity(c *fiber.Ctx) error {
	input := new(pb.UpdateServerActivity_Request)
	c.BodyParser(input)
	if err := validator.ValidateStruct(input); err != nil {
		return httputil.StatusBadRequest(c, message.ErrValidateBodyParams, err)
	}

	userParameter := middleware.GetUserParameters(c)
	userID := userParameter.GetUserID(input.GetUserId())

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	rClient := pb.NewServerHandlersClient(h.grpc.Client)

	_, err := rClient.UpdateServerActivity(ctx, &pb.UpdateServerActivity_Request{
		UserId:    userID,
		ProjectId: input.GetProjectId(),
		ServerId:  input.GetServerId(),
		Activity:  input.GetActivity(),
	})
	if err != nil {
		return httputil.ReturnGRPCError(c, err)
	}
	return httputil.StatusOK(c, "Server activity updated", nil)
}

// @Summary      Get server firewall
// @Tags         servers
// @Accept       json
// @Produce      json
// @Param        req         body     pb.GetServerFirewall_Request{}
// @Success      200         {object} httputil.HTTPResponse{data=pb.GetServerFirewall_Response}
// @Failure      400,401,500 {object} httputil.HTTPResponse
// @Router       /v1/servers/firewall [get]
func (h *Handler) getServerFirewall(c *fiber.Ctx) error {
	input := new(pb_firewall.GetServerFirewall_Request)
	c.QueryParser(input)
	if err := validator.ValidateStruct(input); err != nil {
		return httputil.StatusBadRequest(c, message.ErrValidateBodyParams, err)
	}

	userParameter := middleware.GetUserParameters(c)
	userID := userParameter.GetUserID(input.GetUserId())

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	rClient := pb_firewall.NewFirewallHandlersClient(h.grpc.Client)

	firewall, err := rClient.GetServerFirewall(ctx, &pb_firewall.GetServerFirewall_Request{
		UserId:    userID,
		ServerId:  input.GetServerId(),
		ProjectId: input.GetUserId(),
	})
	if err != nil {
		return httputil.ReturnGRPCError(c, err)
	}
	return httputil.StatusOK(c, "Server firewall data", firewall)
}

// @Summary      Add server firewall
// @Tags         servers
// @Accept       json
// @Produce      json
// @Param        req         body     pb_firewall.CreateServerFirewall_Request
// @Success      200         {object} httputil.HTTPResponse{data=pb_firewall.CreateServerFirewall_Response}
// @Failure      400,401,500 {object} httputil.HTTPResponse
// @Router       /v1/servers/firewall [post]
func (h *Handler) postServerFirewall(c *fiber.Ctx) error {
	input := new(pb_firewall.CreateServerFirewall_Request)
	if err := protojson.Unmarshal(c.Body(), input); err != nil {
		fmt.Print(err)
	}
	if err := validator.ValidateStruct(input); err != nil {
		return httputil.StatusBadRequest(c, message.ErrValidateBodyParams, err)
	}

	userParameter := middleware.GetUserParameters(c)
	userID := userParameter.GetUserID(input.GetUserId())

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	rClient := pb_firewall.NewFirewallHandlersClient(h.grpc.Client)

	var err error
	response := new(pb_firewall.CreateServerFirewall_Response)
	switch record := input.Record.(type) {
	case *pb_firewall.CreateServerFirewall_Request_Country:
		response, err = rClient.CreateServerFirewall(ctx, &pb_firewall.CreateServerFirewall_Request{
			UserId:    userID,
			ProjectId: input.GetProjectId(),
			ServerId:  input.GetServerId(),
			Record: &pb_firewall.CreateServerFirewall_Request_Country{
				Country: &pb_firewall.CountryCode{
					Code: record.Country.Code,
				},
			},
		})

	case *pb_firewall.CreateServerFirewall_Request_Ip:
		response, err = rClient.CreateServerFirewall(ctx, &pb_firewall.CreateServerFirewall_Request{
			UserId:    userID,
			ProjectId: input.GetProjectId(),
			ServerId:  input.GetServerId(),
			Record: &pb_firewall.CreateServerFirewall_Request_Ip{
				Ip: &pb_firewall.IpMask{
					StartIp: record.Ip.StartIp,
					EndIp:   record.Ip.EndIp,
				},
			},
		})
	default:
		return httputil.StatusBadRequest(c, "Bad rule", nil)
	}

	if err != nil {
		return httputil.ReturnGRPCError(c, err)
	}
	return httputil.StatusOK(c, "Firewall added", response)
}

// @Summary      Status firewall server
// @Tags         servers
// @Accept       json
// @Produce      json
// @Param        req         body     pb.UpdateAccessPolicy_Request{}
// @Success      200         {object} httputil.HTTPResponse
// @Failure      400,401,500 {object} httputil.HTTPResponse
// @Router       /v1/servers/firewall [patch]
func (h *Handler) patchAccessPolicy(c *fiber.Ctx) error {
	input := new(pb_firewall.UpdateAccessPolicy_Request)
	c.BodyParser(input)
	if err := validator.ValidateStruct(input); err != nil {
		return httputil.StatusBadRequest(c, message.ErrValidateBodyParams, err)
	}

	userParameter := middleware.GetUserParameters(c)
	userID := userParameter.GetUserID(input.GetUserId())

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	rClient := pb_firewall.NewFirewallHandlersClient(h.grpc.Client)

	_, err := rClient.UpdateAccessPolicy(ctx, &pb_firewall.UpdateAccessPolicy_Request{
		UserId:    userID,
		ProjectId: input.GetProjectId(),
		ServerId:  input.GetServerId(),
		Rule:      input.GetRule(),
		Status:    input.GetStatus(),
	})
	if err != nil {
		return httputil.ReturnGRPCError(c, err)
	}
	return httputil.StatusOK(c, "Firewall updated", nil)
}

// @Summary      Delete server firewall
// @Tags         servers
// @Accept       json
// @Produce      json
// @Param        req         body     pb_firewall.GetServerFirewall_Request{}
// @Success      200         {object} httputil.HTTPResponse
// @Failure      400,401,500 {object} httputil.HTTPResponse
// @Router       /v1/servers/firewall [delete]
func (h *Handler) deleteServerFirewall(c *fiber.Ctx) error {
	input := new(pb_firewall.DeleteServerFirewall_Request)
	c.BodyParser(input)
	if err := validator.ValidateStruct(input); err != nil {
		return httputil.StatusBadRequest(c, message.ErrValidateBodyParams, err)
	}

	userParameter := middleware.GetUserParameters(c)
	userID := userParameter.GetUserID(input.GetUserId())

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	rClient := pb_firewall.NewFirewallHandlersClient(h.grpc.Client)

	_, err := rClient.DeleteServerFirewall(ctx, &pb_firewall.DeleteServerFirewall_Request{
		UserId:    userID,
		ProjectId: input.GetProjectId(),
		ServerId:  input.GetServerId(),
		Rule:      input.GetRule(),
		RecordId:  input.GetRecordId(),
	})
	if err != nil {
		return httputil.ReturnGRPCError(c, err)
	}
	return httputil.StatusOK(c, "Firewall deleted", nil)
}

// @Summary      Update server status
// @Tags         servers
// @Accept       json
// @Produce      json
// @Param        req         body     pb.UpdateServerActiveStatus_Request{}
// @Success      200         {object} httputil.HTTPResponse
// @Failure      400,401,500 {object} httputil.HTTPResponse
// @Router       /v1/servers/active [patch]
func (h *Handler) patchServerStatus(c *fiber.Ctx) error {
	input := new(pb.UpdateServerActiveStatus_Request)
	c.BodyParser(input)
	if err := validator.ValidateStruct(input); err != nil {
		return httputil.StatusBadRequest(c, message.ErrValidateBodyParams, err)
	}

	userParameter := middleware.GetUserParameters(c)
	userID := userParameter.GetUserID(input.GetUserId())

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	rClient := pb.NewServerHandlersClient(h.grpc.Client)

	_, err := rClient.UpdateServerActiveStatus(ctx, &pb.UpdateServerActiveStatus_Request{
		UserId:   userID,
		ServerId: input.GetServerId(),
		Status:   input.GetStatus(),
	})
	if err != nil {
		return httputil.ReturnGRPCError(c, err)
	}

	// message section
	message := "Access to the server is open, the server is online"
	if input.GetStatus() == false {
		message = "Access to the server is closed"
	}

	return httputil.StatusOK(c, message, nil)
}
