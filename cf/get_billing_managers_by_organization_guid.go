package cf

import (
	"time"

	"github.com/cloudfoundry-incubator/notifications/metrics"
)

func (cc CloudController) GetBillingManagersByOrgGuid(guid, token string) ([]CloudControllerUser, error) {
	var ccUsers []CloudControllerUser
	then := time.Now()

	list, err := cc.client.Organizations.ListBillingManagers(guid, token)
	if err != nil {
		return ccUsers, NewFailure(0, err.Error())
	}

	users, err := list.AllUsers(token)
	if err != nil {
		return ccUsers, NewFailure(0, err.Error())
	}

	duration := time.Now().Sub(then)

	metrics.NewMetric("histogram", map[string]interface{}{
		"name":  "notifications.external-requests.cc.billing-managers-by-org-guid",
		"value": duration.Seconds(),
	}).Log()

	for _, user := range users {
		ccUsers = append(ccUsers, CloudControllerUser{
			GUID: user.GUID,
		})
	}

	return ccUsers, nil
}
