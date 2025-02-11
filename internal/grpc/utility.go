package grpc

import (
	"context"

	pb_utility "github.com/werbot/werbot/internal/grpc/proto/utility"
)

type utility struct {
	pb_utility.UnimplementedUtilityHandlersServer
}

// GetCountry is searches for a country by first letters
func (u *utility) GetCountry(ctx context.Context, in *pb_utility.GetCountry_Request) (*pb_utility.GetCountry_Response, error) {
	countries := []*pb_utility.GetCountry_Response_Country{}

	rows, err := db.Conn.Query(`SELECT 
      "code", 
      "name" 
    FROM 
      "country" 
    WHERE 
      LOWER("name") LIKE LOWER($1) 
    ORDER BY "name" ASC LIMIT 15 OFFSET 0`, in.Name+"%")
	if err != nil {
		return nil, err
	}

	for rows.Next() {
		country := pb_utility.GetCountry_Response_Country{}

		err = rows.Scan(
			&country.Code,
			&country.Name,
		)
		if err != nil {
			return nil, err
		}

		countries = append(countries, &country)
	}

	defer rows.Close()

	return &pb_utility.GetCountry_Response{
		Countries: countries,
	}, nil
}
