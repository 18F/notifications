package models_test

import (
	"errors"
	"fmt"

	"github.com/cloudfoundry-incubator/notifications/db"
	"github.com/cloudfoundry-incubator/notifications/testing/helpers"
	"github.com/cloudfoundry-incubator/notifications/testing/mocks"
	"github.com/cloudfoundry-incubator/notifications/v2/models"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("SendersRepo", func() {
	var (
		repo          models.SendersRepository
		conn          db.ConnectionInterface
		guidGenerator *mocks.IDGenerator
	)

	BeforeEach(func() {
		database := db.NewDatabase(sqlDB, db.Config{})
		helpers.TruncateTables(database)

		guidGenerator = mocks.NewIDGenerator()
		guidGenerator.GenerateCall.Returns.IDs = []string{"first-random-guid", "second-random-guid"}

		repo = models.NewSendersRepository(guidGenerator.Generate)
		conn = database.Connection()
	})

	Describe("Insert", func() {
		It("inserts the record into the database", func() {
			sender := models.Sender{
				Name:     "some-sender",
				ClientID: "some-client-id",
			}

			sender, err := repo.Insert(conn, sender)
			Expect(err).NotTo(HaveOccurred())
			Expect(sender).To(Equal(models.Sender{
				ID:       "first-random-guid",
				Name:     "some-sender",
				ClientID: "some-client-id",
			}))
		})

		Context("failure cases", func() {
			It("returns a duplicate record error when the name and client_id are taken", func() {
				sender := models.Sender{
					Name:     "some-sender",
					ClientID: "some-client-id",
				}

				_, err := repo.Insert(conn, sender)
				Expect(err).NotTo(HaveOccurred())

				_, err = repo.Insert(conn, sender)
				Expect(err).To(MatchError(models.DuplicateRecordError{errors.New("Sender with name \"some-sender\" already exists")}))
			})

			It("returns the error when the insert call fails", func() {
				connection := mocks.NewConnection()
				connection.InsertCall.Returns.Error = errors.New("something bad happened")

				sender := models.Sender{
					Name:     "some-sender",
					ClientID: "some-client-id",
				}

				_, err := repo.Insert(connection, sender)
				Expect(err).To(MatchError(errors.New("something bad happened")))
			})

			It("returns an error when the guid generator blows up", func() {
				guidGenerator.GenerateCall.Returns.Error = errors.New("failed to generate")

				_, err := repo.Insert(conn, models.Sender{})
				Expect(err).To(MatchError(errors.New("failed to generate")))
			})
		})
	})

	Describe("Update", func() {
		BeforeEach(func() {
			sender := models.Sender{
				Name:     "some-sender",
				ClientID: "some-client-id",
			}

			sender, err := repo.Insert(conn, sender)
			Expect(err).NotTo(HaveOccurred())
			Expect(sender).To(Equal(models.Sender{
				ID:       "first-random-guid",
				Name:     "some-sender",
				ClientID: "some-client-id",
			}))
		})

		It("updates the name", func() {
			sender := models.Sender{
				ID:       "first-random-guid",
				Name:     "new-sender-name",
				ClientID: "some-client-id",
			}

			sender, err := repo.Update(conn, sender)
			Expect(err).NotTo(HaveOccurred())
			Expect(sender).To(Equal(models.Sender{
				ID:       "first-random-guid",
				Name:     "new-sender-name",
				ClientID: "some-client-id",
			}))
		})

		Context("failure cases", func() {
			It("returns a duplicate record error when the name and client_id are taken", func() {
				sender := models.Sender{
					Name:     "new-sender-name",
					ClientID: "some-client-id",
				}
				sender, err := repo.Insert(conn, sender)
				Expect(err).NotTo(HaveOccurred())

				sender = models.Sender{
					ID:       "first-random-guid",
					Name:     "new-sender-name",
					ClientID: "some-client-id",
				}

				_, err = repo.Update(conn, sender)
				Expect(err).To(MatchError(models.DuplicateRecordError{errors.New("Sender with name \"new-sender-name\" already exists")}))
			})

			It("returns other errors if they occur", func() {
				connection := mocks.NewConnection()

				connection.UpdateCall.Returns.Error = errors.New("potatoes")

				sender := models.Sender{
					ID:       "first-random-guid",
					Name:     "new-sender-name",
					ClientID: "some-client-id",
				}
				_, err := repo.Update(connection, sender)
				Expect(err).To(MatchError(errors.New("potatoes")))
			})
		})
	})

	Describe("List", func() {
		It("lists the senders given a client id", func() {
			createdSender, err := repo.Insert(conn, models.Sender{
				Name:     "some-sender",
				ClientID: "some-client-id",
			})
			Expect(err).NotTo(HaveOccurred())

			_, err = repo.Insert(conn, models.Sender{
				Name:     "some-sender",
				ClientID: "other-client-id",
			})
			Expect(err).NotTo(HaveOccurred())

			senders, err := repo.List(conn, createdSender.ClientID)
			Expect(err).NotTo(HaveOccurred())
			Expect(senders).To(HaveLen(1))
			Expect(senders[0].ID).To(Equal(createdSender.ID))
		})

		It("returns any error that was encountered", func() {
			connection := mocks.NewConnection()

			connection.SelectCall.Returns.Error = errors.New("potatoes")

			_, err := repo.List(connection, "doesnt-matter")
			Expect(err).To(MatchError(errors.New("potatoes")))
		})
	})

	Describe("Get", func() {
		It("fetches the sender given a sender id", func() {
			createdSender, err := repo.Insert(conn, models.Sender{
				Name:     "some-sender",
				ClientID: "some-client-id",
			})
			Expect(err).NotTo(HaveOccurred())

			sender, err := repo.Get(conn, createdSender.ID)
			Expect(err).NotTo(HaveOccurred())
			Expect(sender).To(Equal(createdSender))
		})

		Context("failure cases", func() {
			It("fails to fetch the sender given a non-existent sender id", func() {
				_, err := repo.Insert(conn, models.Sender{
					Name:     "some-sender",
					ClientID: "some-client-id",
				})
				Expect(err).NotTo(HaveOccurred())

				_, err = repo.Get(conn, "some-other-sender-id")
				Expect(err).To(BeAssignableToTypeOf(models.RecordNotFoundError{}))
				Expect(err.Error()).To(Equal(`Sender with id "some-other-sender-id" could not be found`))
			})
		})
	})

	Describe("GetByClientIDAndName", func() {
		It("fetches the sender given a client_id and name", func() {
			createdSender, err := repo.Insert(conn, models.Sender{
				Name:     "some-sender",
				ClientID: "some-client-id",
			})
			Expect(err).NotTo(HaveOccurred())

			sender, err := repo.GetByClientIDAndName(conn, "some-client-id", "some-sender")
			Expect(err).NotTo(HaveOccurred())
			Expect(sender).To(Equal(createdSender))
		})

		It("fails to fetch the sender given a non-existent client_id and name", func() {
			_, err := repo.Insert(conn, models.Sender{
				Name:     "some-sender",
				ClientID: "some-client-id",
			})
			Expect(err).NotTo(HaveOccurred())

			_, err = repo.GetByClientIDAndName(conn, "some-other-client-id", "some-sender")
			Expect(err).To(BeAssignableToTypeOf(models.RecordNotFoundError{}))
			Expect(err.Error()).To(Equal(`Sender with client_id "some-other-client-id" and name "some-sender" could not be found`))
		})
	})

	Describe("Delete", func() {
		It("deletes the sender record", func() {
			createdSender, err := repo.Insert(conn, models.Sender{
				Name:     "some-sender",
				ClientID: "some-client-id",
			})
			Expect(err).NotTo(HaveOccurred())

			err = repo.Delete(conn, createdSender)
			Expect(err).NotTo(HaveOccurred())

			_, err = repo.Get(conn, createdSender.ID)
			Expect(err).To(MatchError(models.RecordNotFoundError{fmt.Errorf("Sender with id %q could not be found", createdSender.ID)}))
		})

		It("returns an error if the db fails", func() {
			fakeConnection := mocks.NewConnection()
			fakeConnection.DeleteCall.Returns.Error = errors.New("there was an error")

			err := repo.Delete(fakeConnection, models.Sender{})
			Expect(err).To(MatchError(errors.New("there was an error")))
		})
	})
})
