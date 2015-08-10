package fakes

import "github.com/cloudfoundry-incubator/notifications/models"

type PreferenceUpdater struct {
	ExecuteCall struct {
		Receives struct {
			Connection        models.ConnectionInterface
			Preferences       []models.Preference
			GlobalUnsubscribe bool
			UserID            string
		}
		Returns struct {
			Error error
		}
	}
}

func NewPreferenceUpdater() *PreferenceUpdater {
	return &PreferenceUpdater{}
}

func (pu *PreferenceUpdater) Execute(conn models.ConnectionInterface, preferences []models.Preference, globalUnsubscribe bool, userID string) error {
	pu.ExecuteCall.Receives.Connection = conn
	pu.ExecuteCall.Receives.Preferences = preferences
	pu.ExecuteCall.Receives.GlobalUnsubscribe = globalUnsubscribe
	pu.ExecuteCall.Receives.UserID = userID

	return pu.ExecuteCall.Returns.Error
}