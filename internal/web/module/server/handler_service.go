package server

import (
	"context"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/werbot/werbot/internal/message"
	"github.com/werbot/werbot/internal/utils/validator"
	"github.com/werbot/werbot/internal/web/httputil"

	pb "github.com/werbot/werbot/internal/grpc/proto/server"
)

// @Summary      Adding a new server
// @Tags         servers
// @Accept       json
// @Produce      json
// @Param        req         body     pb.CreateServer_Request{}
// @Success      200         {object} httputil.HTTPResponse{data=pb.CreateServer_Response}
// @Failure      400,401,500 {object} httputil.HTTPResponse
// @Router       /v1/service/server [post]
func (h *Handler) addServiceServer(c *fiber.Ctx) error {
	input := new(pb.CreateServer_Request)
	c.BodyParser(&input)
	if err := validator.ValidateStruct(&input); err != nil {
		return httputil.StatusBadRequest(c, message.ErrValidateBodyParams, err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	rClient := pb.NewServerHandlersClient(h.grpc.Client)

	server, err := rClient.CreateServer(ctx, &pb.CreateServer_Request{
		ProjectId: input.GetProjectId(),
		Address:   strings.TrimSpace(input.GetAddress()),
		Port:      input.GetPort(),
		Login:     strings.TrimSpace(input.GetLogin()),
		Scheme:    pb.ServerScheme(pb.ServerAuth_KEY),
	})
	if err != nil {
		return httputil.ReturnGRPCError(c, err)
	}
	return httputil.StatusOK(c, "Server key", server.KeyPublic)
}

func (h *Handler) patchServiceServerStatus(c *fiber.Ctx) error {
	return httputil.StatusOK(c, "Server status", "online")
}
