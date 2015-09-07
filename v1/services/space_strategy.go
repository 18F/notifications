package services

const SpaceEndorsement = `You received this message because you belong to the "{{.Space}}" space in the "{{.Organization}}" organization.`

type spaceUserGUIDFinder interface {
	UserGUIDsBelongingToSpace(spaceGUID, token string) (userGUIDs []string, err error)
}

type SpaceStrategy struct {
	tokenLoader        TokenLoaderInterface
	spaceLoader        SpaceLoaderInterface
	organizationLoader OrganizationLoaderInterface
	findsUserGUIDs     spaceUserGUIDFinder
	queue              enqueuer
}

func NewSpaceStrategy(tokenLoader TokenLoaderInterface, spaceLoader SpaceLoaderInterface, organizationLoader OrganizationLoaderInterface, findsUserGUIDs spaceUserGUIDFinder, queue enqueuer) SpaceStrategy {

	return SpaceStrategy{
		tokenLoader:        tokenLoader,
		spaceLoader:        spaceLoader,
		organizationLoader: organizationLoader,
		findsUserGUIDs:     findsUserGUIDs,
		queue:              queue,
	}
}

func (strategy SpaceStrategy) Dispatch(dispatch Dispatch) ([]Response, error) {
	responses := []Response{}
	options := Options{
		To:                dispatch.Message.To,
		ReplyTo:           dispatch.Message.ReplyTo,
		Subject:           dispatch.Message.Subject,
		KindID:            dispatch.Kind.ID,
		KindDescription:   dispatch.Kind.Description,
		SourceDescription: dispatch.Client.Description,
		Endorsement:       SpaceEndorsement,
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

	token, err := strategy.tokenLoader.Load(dispatch.UAAHost)
	if err != nil {
		return responses, err
	}

	userGUIDs, err := strategy.findsUserGUIDs.UserGUIDsBelongingToSpace(dispatch.GUID, token)
	if err != nil {
		return responses, err
	}

	var users []User
	for _, guid := range userGUIDs {
		users = append(users, User{GUID: guid})
	}

	space, err := strategy.spaceLoader.Load(dispatch.GUID, token)
	if err != nil {
		return responses, err
	}

	org, err := strategy.organizationLoader.Load(space.OrganizationGUID, token)
	if err != nil {
		return responses, err
	}

	responses = strategy.queue.Enqueue(dispatch.Connection, users, options, space, org, dispatch.Client.ID, dispatch.UAAHost, "", dispatch.VCAPRequest.ID, dispatch.VCAPRequest.ReceiptTime)

	return responses, nil
}
