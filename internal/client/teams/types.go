package teams

import "github.com/upbound/up-sdk-go/service/common"

type GetResponse struct {
	common.DataSet `json:"data"`
}

type CreateParameters struct {
	Name           string `json:"name"`
	OrganizationID uint   `json:"organizationId"`
}

type CreateResponse struct {
	ID string `json:"id"`
}
