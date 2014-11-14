package strategies

import (
	"errors"

	"github.com/cloudfoundry-incubator/notifications/cf"
	"github.com/cloudfoundry-incubator/notifications/models"
	"github.com/cloudfoundry-incubator/notifications/postal"
	"github.com/cloudfoundry-incubator/notifications/postal/utilities"
)

type UAAScopeStrategy struct {
	receiptsRepo    models.ReceiptsRepoInterface
	findsUserGUIDs  utilities.FindsUserGUIDsInterface
	tokenLoader     utilities.TokenLoaderInterface
	templatesLoader utilities.TemplatesLoaderInterface
	mailer          MailerInterface
	userLoader      utilities.UserLoaderInterface
}

func NewUAAScopeStrategy(tokenLoader utilities.TokenLoaderInterface, userLoader utilities.UserLoaderInterface,
	findsUserGUIDs utilities.FindsUserGUIDsInterface, templatesLoader utilities.TemplatesLoaderInterface, mailer MailerInterface, receiptsRepo models.ReceiptsRepoInterface) UAAScopeStrategy {

	return UAAScopeStrategy{
		receiptsRepo:    receiptsRepo,
		findsUserGUIDs:  findsUserGUIDs,
		tokenLoader:     tokenLoader,
		templatesLoader: templatesLoader,
		mailer:          mailer,
		userLoader:      userLoader,
	}
}

func (strategy UAAScopeStrategy) Dispatch(clientID, scope string, options postal.Options, conn models.ConnectionInterface) ([]Response, error) {
	responses := []Response{}

	if strategy.scopeIsDefault(scope) {
		return responses, errors.New("You cannot send a notification to a default scope")
	}

	token, err := strategy.tokenLoader.Load()
	if err != nil {
		return responses, err
	}

	userGUIDs, err := strategy.findsUserGUIDs.UserGUIDsBelongingToScope(scope)
	if err != nil {
		return responses, err
	}

	users, err := strategy.userLoader.Load(userGUIDs, token)
	if err != nil {
		return responses, err
	}

	subjectSuffix := strategy.subjectSuffix(options.Subject)
	templates, err := strategy.templatesLoader.LoadTemplates(subjectSuffix, models.UAAScopeBodyTemplateName, clientID, options.KindID)
	if err != nil {
		return responses, postal.TemplateLoadError("An email template could not be loaded")
	}

	err = strategy.receiptsRepo.CreateReceipts(conn, userGUIDs, clientID, options.KindID)
	if err != nil {
		return responses, err
	}

	responses = strategy.mailer.Deliver(conn, templates, users, options, cf.CloudControllerSpace{}, cf.CloudControllerOrganization{}, clientID, scope)

	return responses, nil
}

func (strategy UAAScopeStrategy) scopeIsDefault(scope string) bool {
	defaultScopes := []string{"cloud_controller.read", "cloud_controller.write", "openid", "approvals.me",
		"cloud_controller_service_permissions.read", "scim.me", "uaa.user", "password.write", "scim.userids", "oauth.approvals"}

	for _, singleScope := range defaultScopes {
		if scope == singleScope {
			return true
		}
	}
	return false
}

func (strategy UAAScopeStrategy) Trim(responses []byte) []byte {
	t := Trimmer{}
	return t.TrimFields(responses, EmailFieldName)
}

func (strategy UAAScopeStrategy) subjectSuffix(subject string) string {
	if subject == "" {
		return models.SubjectMissingTemplateName
	}
	return models.SubjectProvidedTemplateName
}
