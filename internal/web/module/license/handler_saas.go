//go:build saas

package license

import (
	"context"
	"encoding/base64"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/werbot/werbot/internal/message"
	"github.com/werbot/werbot/internal/utils/validator"
	"github.com/werbot/werbot/internal/web/httputil"
	"github.com/werbot/werbot/internal/web/middleware"

	pb "github.com/werbot/werbot/internal/grpc/proto/license"
)

type licenseInput struct {
	License string `json:"license" validate:"required,base64"`
}

// @Summary      License expired info
// @Tags         license
// @Accept       json
// @Produce      json
// @Param        req         body     licenseInput
// @Success      200         {object} httputil.HTTPResponse
// @Failure      400,401,500 {object} httputil.HTTPResponse
// @Router       /v1/license/expired [get]
func (h *Handler) getLicenseExpired(c *fiber.Ctx) error {
	input := new(licenseInput)
	c.BodyParser(input)
	if err := validator.ValidateStruct(input); err != nil {
		return httputil.StatusBadRequest(c, message.ErrValidateBodyParams, err)
	}

	licenseDec, err := base64.StdEncoding.DecodeString(input.License)
	if err != nil {
		return httputil.StatusBadRequest(c, message.ErrBadRequest, err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	rClient := pb.NewLicenseHandlersClient(h.grpc.Client)

	expiredLic, err := rClient.GetLicenseExpired(ctx, &pb.GetLicenseExpired_Request{
		License: licenseDec,
	})
	if err != nil {
		return httputil.InternalServerError(c, "Having problems show info", nil)
	}

	return httputil.StatusOK(c, "License expired", expiredLic.Status)
}

// @Summary      Creating a new license
// @Tags         license
// @Accept       json
// @Produce      json
// @Param        req     body     pb.NewLicenseRequest
// @Success      200     {object} httputil.HTTPResponse
// @Failure      400,500 {object} httputil.HTTPResponse
// @Router       /v1/license [post]
func (h *Handler) postLicense(c *fiber.Ctx) error {
	input := new(pb.NewLicense_Request)
	c.BodyParser(input)

	if err := validator.ValidateStruct(input); err != nil {
		return httputil.StatusBadRequest(c, message.ErrValidateBodyParams, err)
	}

	dataLicense := &pb.NewLicense_Request{
		Ip:    input.GetIp(),
		Token: input.GetToken(),
	}

	userParameter := middleware.GetUserParameters(c)
	if userParameter.IsUserAdmin() {
		dataLicense = &pb.NewLicense_Request{
			Customer:   input.GetCustomer(),
			Subscriber: input.GetSubscriber(),
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	rClient := pb.NewLicenseHandlersClient(h.grpc.Client)

	// check ip from db license
	dataLic, err := rClient.NewLicense(ctx, dataLicense)
	if err != nil {
		return httputil.InternalServerError(c, "Having problems saving", nil)
	}

	licenseKey := base64.StdEncoding.EncodeToString(dataLic.License)
	return httputil.StatusOK(c, "License key", licenseKey)
}
