package httputil

import (
	"github.com/gofiber/fiber/v2"
	"github.com/werbot/werbot/internal/message"
	"google.golang.org/grpc/status"
)

// StatusBadRequest - HTTP error code 400
func StatusBadRequest(c *fiber.Ctx, message string, err any) error {
	return NewError(c, 400, message, err)
}

// StatusUnauthorized  - HTTP error code 401
func StatusUnauthorized(c *fiber.Ctx, message string, err any) error {
	return NewError(c, 401, message, err)
}

// StatusNotFound - HTTP error code 404
func StatusNotFound(c *fiber.Ctx, message string, err any) error {
	return NewError(c, 404, message, err)
}

// InternalServerError - HTTP error code 500
func InternalServerError(c *fiber.Ctx, message string, err any) error {
	return NewError(c, 500, message, err)
}

// NewError is ...
func NewError(c *fiber.Ctx, status int, message string, data any) error {
	return c.Status(status).JSON(HTTPResponse{
		Success: false,
		Message: message,
		Result:  data,
	})
}

// ReturnGRPCError is ...
func ReturnGRPCError(c *fiber.Ctx, err error) error {
	se, _ := status.FromError(err)

	if se.Message() == message.ErrNotFound {
		return StatusNotFound(c, message.ErrNotFound, nil)
	}

	if se.Message() != "" {
		return StatusBadRequest(c, se.Message(), nil)
	}

	return InternalServerError(c, message.ErrUnexpectedError, nil)
}
