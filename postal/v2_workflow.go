package postal

import (
	"strings"

	"github.com/cloudfoundry-incubator/notifications/mail"
	"github.com/cloudfoundry-incubator/notifications/v1/services"
	"github.com/pivotal-golang/lager"
)

type V2Workflow struct {
	mailClient           mailSender
	packager             messagePackager
	userLoader           UserLoaderInterface
	tokenLoader          tokenLoader
	messageStatusUpdater messageStatusUpdater
	database             services.DatabaseInterface
	sender               string
	domain               string
	uaaHost              string
}

type messagePackager interface {
	PrepareContext(delivery Delivery, sender, domain string) (MessageContext, error)
	Pack(context MessageContext) (mail.Message, error)
}

func NewV2Workflow(mailClient mailSender, packager messagePackager, userLoader UserLoaderInterface, tokenLoader tokenLoader, messageStatusUpdater messageStatusUpdater, database services.DatabaseInterface, sender, domain, uaaHost string) V2Workflow {
	return V2Workflow{
		mailClient:           mailClient,
		packager:             packager,
		userLoader:           userLoader,
		tokenLoader:          tokenLoader,
		messageStatusUpdater: messageStatusUpdater,
		database:             database,
		sender:               sender,
		domain:               domain,
		uaaHost:              uaaHost,
	}
}

func (w V2Workflow) Deliver(delivery Delivery, logger lager.Logger) error {
	token, err := w.tokenLoader.Load(w.uaaHost)
	if err != nil {
		return err
	}

	users, err := w.userLoader.Load([]string{delivery.UserGUID}, token)
	if err != nil {
		return err
	}

	emails := users[delivery.UserGUID].Emails
	if len(emails) > 0 {
		delivery.Email = emails[0]
	}

	if !strings.Contains(delivery.Email, "@") {
		w.messageStatusUpdater.Update(w.database.Connection(), delivery.MessageID, StatusUndeliverable, delivery.CampaignID, logger)
		return nil
	}

	context, err := w.packager.PrepareContext(delivery, w.sender, w.domain)
	if err != nil {
		return err
	}

	message, err := w.packager.Pack(context)
	if err != nil {
		return err
	}

	err = w.mailClient.Send(message, logger)
	if err != nil {
		return err
	}

	w.messageStatusUpdater.Update(w.database.Connection(), delivery.MessageID, StatusDelivered, delivery.CampaignID, logger)

	return nil
}