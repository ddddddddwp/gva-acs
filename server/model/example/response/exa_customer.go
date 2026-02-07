package response

import "github.com/ddddddddwp/gva-acs/server/model/example"

type ExaCustomerResponse struct {
	Customer example.ExaCustomer `json:"customer"`
}
