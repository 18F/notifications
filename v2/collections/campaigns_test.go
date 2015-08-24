package collections_test

import (
	"errors"

	"github.com/cloudfoundry-incubator/notifications/testing/mocks"
	"github.com/cloudfoundry-incubator/notifications/v2/collections"
	"github.com/cloudfoundry-incubator/notifications/v2/models"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("CampaignsCollection", func() {
	Describe("Create", func() {
		Context("when the audience is a user", func() {
			var (
				database          *mocks.Database
				enqueuer          *mocks.CampaignEnqueuer
				collection        collections.CampaignsCollection
				campaignTypesRepo *mocks.CampaignTypesRepository
				templatesRepo     *mocks.TemplatesRepository
				userFinder        *mocks.UserFinder
			)

			BeforeEach(func() {
				database = mocks.NewDatabase()
				enqueuer = mocks.NewCampaignEnqueuer()
				campaignTypesRepo = mocks.NewCampaignTypesRepository()
				templatesRepo = mocks.NewTemplatesRepository()
				userFinder = mocks.NewUserFinder()
				userFinder.ExistsCall.Returns.Exists = true

				collection = collections.NewCampaignsCollection(enqueuer, campaignTypesRepo, templatesRepo, userFinder)
			})

			Context("enqueuing a campaignJob", func() {
				It("returns a campaignID after enqueuing the campaign with its type", func() {
					campaign := collections.Campaign{
						SendTo:         map[string]string{"user": "some-guid"},
						CampaignTypeID: "some-id",
						Text:           "some-test",
						HTML:           "no-html",
						Subject:        "some-subject",
						TemplateID:     "whoa-a-template-id",
						ReplyTo:        "nothing@example.com",
						ClientID:       "some-client-id",
					}

					enqueuedCampaign, err := collection.Create(database.Connection(), campaign)
					Expect(err).NotTo(HaveOccurred())

					Expect(enqueuer.EnqueueCall.Receives.Campaign).To(Equal(collections.Campaign{
						ID:             "some-random-id",
						SendTo:         map[string]string{"user": "some-guid"},
						CampaignTypeID: "some-id",
						Text:           "some-test",
						HTML:           "no-html",
						Subject:        "some-subject",
						TemplateID:     "whoa-a-template-id",
						ReplyTo:        "nothing@example.com",
						ClientID:       "some-client-id",
					}))
					Expect(enqueuer.EnqueueCall.Receives.JobType).To(Equal("campaign"))

					Expect(enqueuedCampaign.ID).To(Equal("some-random-id"))
					Expect(err).NotTo(HaveOccurred())
				})
			})

			It("gets the template off of the campaign type if the templateID is blank", func() {
				campaignTypesRepo.GetCall.Returns.CampaignType = models.CampaignType{
					TemplateID: "campaign-type-template-id",
				}

				campaign := collections.Campaign{
					SendTo:         map[string]string{"user": "some-guid"},
					CampaignTypeID: "some-id",
					Text:           "some-test",
					HTML:           "no-html",
					Subject:        "some-subject",
					TemplateID:     "",
					ReplyTo:        "nothing@example.com",
					ClientID:       "some-client-id",
				}

				_, err := collection.Create(database.Connection(), campaign)
				Expect(err).NotTo(HaveOccurred())

				Expect(campaignTypesRepo.GetCall.Receives.Connection).To(Equal(database.Connection()))
				Expect(campaignTypesRepo.GetCall.Receives.CampaignTypeID).To(Equal("some-id"))

				Expect(enqueuer.EnqueueCall.Receives.Campaign).To(Equal(collections.Campaign{
					ID:             "some-random-id",
					SendTo:         map[string]string{"user": "some-guid"},
					CampaignTypeID: "some-id",
					Text:           "some-test",
					HTML:           "no-html",
					Subject:        "some-subject",
					TemplateID:     "campaign-type-template-id",
					ReplyTo:        "nothing@example.com",
					ClientID:       "some-client-id",
				}))
			})

			Context("when the user does not exist", func() {
				It("returns a not found error", func() {
					campaign := collections.Campaign{
						SendTo:         map[string]string{"user": "missing-user"},
						CampaignTypeID: "some-id",
						Text:           "some-test",
						HTML:           "no-html",
						Subject:        "some-subject",
						TemplateID:     "whoa-a-template-id",
						ReplyTo:        "nothing@example.com",
						ClientID:       "some-client-id",
					}

					userFinder.ExistsCall.Returns.Exists = false

					_, err := collection.Create(database.Connection(), campaign)
					Expect(err).To(MatchError(collections.NotFoundError{errors.New("The user \"missing-user\" cannot be found")}))
				})
			})

			Context("when an error happens", func() {
				Context("when enqueue fails", func() {
					It("returns the error to the caller", func() {
						campaign := collections.Campaign{
							SendTo:         map[string]string{"user": "some-guid"},
							CampaignTypeID: "some-id",
							Text:           "some-test",
							HTML:           "no-html",
							Subject:        "some-subject",
							TemplateID:     "whoa-a-template-id",
							ReplyTo:        "nothing@example.com",
							ClientID:       "another-client-id",
						}
						enqueuer.EnqueueCall.Returns.Err = errors.New("enqueue failed")

						_, err := collection.Create(database.Connection(), campaign)

						Expect(err).To(Equal(collections.PersistenceError{Err: errors.New("enqueue failed")}))
					})
				})

				Context("when checking if the template exists", func() {
					var campaign collections.Campaign
					BeforeEach(func() {
						campaign = collections.Campaign{
							SendTo:         map[string]string{"user": "some-guid"},
							CampaignTypeID: "some-id",
							Text:           "some-test",
							HTML:           "no-html",
							Subject:        "some-subject",
							TemplateID:     "error",
							ReplyTo:        "nothing@example.com",
							ClientID:       "some-client-id",
						}
					})

					It("returns an error if the templateID is not found", func() {
						templatesRepo.GetCall.Returns.Error = models.RecordNotFoundError{}

						_, err := collection.Create(database.Connection(), campaign)
						Expect(err).To(MatchError(collections.NotFoundError{models.RecordNotFoundError{}}))
					})

					It("returns a persistence error if there is some other error", func() {
						dbError := errors.New("the database is shutting off")
						templatesRepo.GetCall.Returns.Error = dbError

						_, err := collection.Create(database.Connection(), campaign)
						Expect(err).To(MatchError(collections.PersistenceError{dbError}))
					})
				})

				Context("when checking if the campaign type exists", func() {
					var campaign collections.Campaign

					BeforeEach(func() {
						campaign = collections.Campaign{
							SendTo:         map[string]string{"user": "some-guid"},
							CampaignTypeID: "some-id",
							Text:           "some-test",
							HTML:           "no-html",
							Subject:        "some-subject",
							TemplateID:     "error",
							ReplyTo:        "nothing@example.com",
							ClientID:       "some-client-id",
						}
					})

					It("returns an error if the campaignTypeID is not found", func() {
						campaignTypesRepo.GetCall.Returns.Err = models.RecordNotFoundError{}

						_, err := collection.Create(database.Connection(), campaign)
						Expect(err).To(MatchError(collections.NotFoundError{models.RecordNotFoundError{}}))
					})

					It("returns a persistence error if there is some other error", func() {
						dbError := errors.New("the database is shutting off")
						campaignTypesRepo.GetCall.Returns.Err = dbError

						_, err := collection.Create(database.Connection(), campaign)
						Expect(err).To(MatchError(collections.PersistenceError{dbError}))
					})
				})

				Context("when checking the user exists", func() {
					var campaign collections.Campaign

					BeforeEach(func() {
						campaign = collections.Campaign{
							SendTo:         map[string]string{"user": "some-guid"},
							CampaignTypeID: "some-id",
							Text:           "some-test",
							HTML:           "no-html",
							Subject:        "some-subject",
							TemplateID:     "error",
							ReplyTo:        "nothing@example.com",
							ClientID:       "some-client-id",
						}
					})

					It("returns an error", func() {
						userFinder.ExistsCall.Returns.Error = errors.New("some error")

						_, err := collection.Create(database.Connection(), campaign)
						Expect(err).To(MatchError(collections.UnknownError{errors.New("some error")}))
					})
				})
			})
		})
	})
})
