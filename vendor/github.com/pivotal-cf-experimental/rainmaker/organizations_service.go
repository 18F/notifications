package rainmaker

import (
	"encoding/json"
	"net/http"
	"net/url"

	"github.com/pivotal-cf-experimental/rainmaker/internal/documents"
)

type OrganizationsService struct {
	config Config
}

func NewOrganizationsService(config Config) *OrganizationsService {
	return &OrganizationsService{
		config: config,
	}
}

func (service OrganizationsService) Create(name string, token string) (Organization, error) {
	_, body, err := NewClient(service.config).makeRequest(requestArguments{
		Method: "POST",
		Path:   "/v2/organizations",
		Body: documents.CreateOrganizationRequest{
			Name: name,
		},
		Token: token,
		AcceptableStatusCodes: []int{http.StatusCreated},
	})
	if err != nil {
		return Organization{}, err
	}

	var response documents.OrganizationResponse
	err = json.Unmarshal(body, &response)
	if err != nil {
		panic(err)
	}

	return newOrganizationFromResponse(service.config, response), nil
}

func (service OrganizationsService) Get(guid, token string) (Organization, error) {
	_, body, err := NewClient(service.config).makeRequest(requestArguments{
		Method: "GET",
		Path:   "/v2/organizations/" + guid,
		Token:  token,
		AcceptableStatusCodes: []int{http.StatusOK},
	})
	if err != nil {
		return Organization{}, err
	}

	var response documents.OrganizationResponse
	err = json.Unmarshal(body, &response)
	if err != nil {
		panic(err)
	}

	return newOrganizationFromResponse(service.config, response), nil
}

func (service OrganizationsService) ListUsers(guid, token string) (UsersList, error) {
	list := NewUsersList(service.config, newRequestPlan("/v2/organizations/"+guid+"/users", url.Values{}))
	err := list.Fetch(token)

	return list, err
}

func (service OrganizationsService) ListBillingManagers(guid, token string) (UsersList, error) {
	list := NewUsersList(service.config, newRequestPlan("/v2/organizations/"+guid+"/billing_managers", url.Values{}))
	err := list.Fetch(token)

	return list, err
}

func (service OrganizationsService) ListAuditors(guid, token string) (UsersList, error) {
	list := NewUsersList(service.config, newRequestPlan("/v2/organizations/"+guid+"/auditors", url.Values{}))
	err := list.Fetch(token)

	return list, err
}

func (service OrganizationsService) ListManagers(guid, token string) (UsersList, error) {
	list := NewUsersList(service.config, newRequestPlan("/v2/organizations/"+guid+"/managers", url.Values{}))
	err := list.Fetch(token)

	return list, err
}
