//go:build saas

package grpc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgtype"

	"github.com/werbot/werbot/internal/config"
	license_lib "github.com/werbot/werbot/internal/license"

	pb_license "github.com/werbot/werbot/internal/grpc/proto/license"
)

// NewLicense is ...
func (l *license) NewLicense(ctx context.Context, in *pb_license.NewLicense_Request) (*pb_license.NewLicense_Response, error) {
	var status string
	var licServer []byte
	db.Conn.QueryRow(`SELECT "status", "license" FROM "license" WHERE "ip" = $1::inet`, in.GetIp()).Scan(&status, &licServer)

	if status == "" {
		lic, err := license_lib.SetLicense([]byte(config.GetString("KEY_PRIVATE", "")))
		if err != nil {
			return nil, errors.New("NewLicense failed")
		}

		var typeID, period, companies, servers, users int32
		var name string
		var modulesJSON pgtype.JSON
		err = db.Conn.QueryRow(`SELECT "id", "name", "period", "companies", "servers", "users", "modules" FROM "license_type" WHERE "default" = $1`, true).
			Scan(&typeID,
				&name,
				&period,
				&companies,
				&servers,
				&users,
				&modulesJSON,
			)
		if err != nil {
			return nil, errors.New("NewLicense failed")
		}

		var modules []string
		modulesJSON.AssignTo(&modules)

		now := time.Now()
		licData := &pb_license.License_Limits{
			Companies: companies,
			Servers:   servers,
			Users:     users,
			Modules:   modules,
		}
		licDataByte, err := json.Marshal(licData)
		if err != nil {
			return nil, errors.New("NewLicense failed")
		}

		customer := checkUUIDLicenseParam(in.GetCustomer())
		subscriber := checkUUIDLicenseParam(in.GetSubscriber())

		lic.License = &license_lib.License{
			Iss: fmt.Sprintf("Werbot_%s, Inc.", time.Now().Format("20060102150405")),
			Typ: name,
			Cus: customer,
			Sub: uuid.New().String(),
			Ips: in.GetIp(),
			Iat: now.UTC(),
			Exp: now.AddDate(0, 0, int(period)).UTC(),
			//Dat: []byte(`{"servers":200, "companies":20, "users":50, "modules":["success", "error", "warning"]}`),
			Dat: licDataByte,
		}

		licenseByte, err := lic.Encode()
		if err != nil {
			return nil, errors.New("NewLicense failed")
		}

		status = "trial"
		db.Conn.Exec(`INSERT INTO "public"."license" ("version", "customer_id", "subscriber_id", "type_id", "ip", "status", "issued_at", "expires_at", "license") VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
			1,
			customer,
			subscriber,
			typeID,
			in.GetIp(),
			status,
			lic.License.Iat,
			lic.License.Exp,
			licenseByte,
		)

		return &pb_license.NewLicense_Response{
			License: licenseByte,
		}, nil
	}

	//return nil, errors.New("License release blocked")
	return &pb_license.NewLicense_Response{
		License: licServer,
	}, nil
}

// GetLicenseExpired is ...
func (l *license) GetLicenseExpired(ctx context.Context, in *pb_license.GetLicenseExpired_Request) (*pb_license.GetLicenseExpired_Response, error) {
	lic, err := license_lib.GetLicense([]byte(config.GetString("KEY_PUBLIC", "")))
	if err != nil {
		return nil, errors.New("GetLicenseExpired failed")
	}

	ld, err := lic.Decode(in.GetLicense())
	if err != nil {
		return nil, errors.New("GetLicenseExpired failed Decode")
	}

	return &pb_license.GetLicenseExpired_Response{
		Status: ld.Expired(),
	}, nil
}

func checkUUIDLicenseParam(param string) string {
	if len(param) > 0 {
		return param
	}
	return uuid.New().String()
}
