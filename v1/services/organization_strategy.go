package services

import "github.com/cloudfoundry-incubator/notifications/cf"

const (
	OrganizationEndorsement     = `You received this message because you belong to the "{{.Organization}}" organization.`
	OrganizationRoleEndorsement = `You received this message because you are an {{.OrganizationRole}} in the "{{.Organization}}" organization.`
)

type orgUserGUIDFinder interface {
	UserGUIDsBelongingToOrganization(orgGUID, role, token string) (userGUIDs []string, err error)
}

type OrganizationStrategy struct {
	tokenLoader        TokenLoaderInterface
	organizationLoader OrganizationLoaderInterface
	findsUserGUIDs     orgUserGUIDFinder
	queue              enqueuer
}

func NewOrganizationStrategy(tokenLoader TokenLoaderInterface, organizationLoader OrganizationLoaderInterface, findsUserGUIDs orgUserGUIDFinder, queue enqueuer) OrganizationStrategy {
	return OrganizationStrategy{
		tokenLoader:        tokenLoader,
		organizationLoader: organizationLoader,
		findsUserGUIDs:     findsUserGUIDs,
		queue:              queue,
	}
}

func (strategy OrganizationStrategy) Dispatch(dispatch Dispatch) ([]Response, error) {
	responses := []Response{}
	options := Options{
		To:                dispatch.Message.To,
		ReplyTo:           dispatch.Message.ReplyTo,
		Subject:           dispatch.Message.Subject,
		KindID:            dispatch.Kind.ID,
		KindDescription:   dispatch.Kind.Description,
		SourceDescription: dispatch.Client.Description,
		Endorsement:       OrganizationEndorsement,
		Text:              dispatch.Message.Text,
		TemplateID:        dispatch.TemplateID,
		Role:              dispatch.Role,
		HTML: HTML{
			BodyContent:    dispatch.Message.HTML.BodyContent,
			BodyAttributes: dispatch.Message.HTML.BodyAttributes,
			Head:           dispatch.Message.HTML.Head,
			Doctype:        dispatch.Message.HTML.Doctype,
		},
	}

	if dispatch.Role != "" {
		options.Endorsement = OrganizationRoleEndorsement
	}

	token, err := strategy.tokenLoader.Load(dispatch.UAAHost)
	if err != nil {
		return responses, err
	}

	organization, err := strategy.organizationLoader.Load(dispatch.GUID, token)
	if err != nil {
		return responses, err
	}

	userGUIDs, err := strategy.findsUserGUIDs.UserGUIDsBelongingToOrganization(dispatch.GUID, options.Role, token)
	if err != nil {
		return responses, err
	}

	var users []User
	for _, guid := range userGUIDs {
		users = append(users, User{GUID: guid})
	}

	responses = strategy.queue.Enqueue(dispatch.Connection, users, options, cf.CloudControllerSpace{}, organization, dispatch.Client.ID, dispatch.UAAHost, "", dispatch.VCAPRequest.ID, dispatch.VCAPRequest.ReceiptTime)

	return responses, nil
}
