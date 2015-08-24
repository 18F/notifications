package web

import (
	"database/sql"
	"net/http"

	"github.com/cloudfoundry-incubator/notifications/gobble"
	"github.com/cloudfoundry-incubator/notifications/metrics"
	"github.com/cloudfoundry-incubator/notifications/uaa"
	"github.com/cloudfoundry-incubator/notifications/v2/collections"
	"github.com/cloudfoundry-incubator/notifications/v2/models"
	"github.com/cloudfoundry-incubator/notifications/v2/queue"
	"github.com/cloudfoundry-incubator/notifications/v2/web/campaigns"
	"github.com/cloudfoundry-incubator/notifications/v2/web/campaigntypes"
	"github.com/cloudfoundry-incubator/notifications/v2/web/info"
	"github.com/cloudfoundry-incubator/notifications/v2/web/middleware"
	"github.com/cloudfoundry-incubator/notifications/v2/web/senders"
	"github.com/cloudfoundry-incubator/notifications/v2/web/templates"
	"github.com/gorilla/mux"
	"github.com/nu7hatch/gouuid"
	"github.com/pivotal-cf-experimental/warrant"
	"github.com/pivotal-golang/lager"
	"github.com/ryanmoran/stack"
)

type muxer interface {
	Handle(method, path string, handler stack.Handler, middleware ...stack.Middleware)
	GetRouter() *mux.Router
	ServeHTTP(w http.ResponseWriter, req *http.Request)
}

type mother interface {
	SQLDatabase()
}

type Config struct {
	DBLoggingEnabled bool
	SkipVerifySSL    bool
	SQLDB            *sql.DB
	Logger           lager.Logger
	Queue            gobble.QueueInterface

	UAAPublicKey    string
	UAAHost         string
	UAAClientID     string
	UAAClientSecret string
}

func NewRouter(mx muxer, config Config) http.Handler {
	requestCounter := middleware.NewRequestCounter(mx.GetRouter(), metrics.DefaultLogger)
	logging := middleware.NewRequestLogging(config.Logger)
	notificationsWriteAuthenticator := middleware.NewAuthenticator(config.UAAPublicKey, "notifications.write")
	databaseAllocator := middleware.NewDatabaseAllocator(config.SQLDB, config.DBLoggingEnabled)

	warrantConfig := warrant.Config{
		Host:          config.UAAHost,
		SkipVerifySSL: config.SkipVerifySSL,
	}
	warrantUsersService := warrant.NewUsersService(warrantConfig)
	warrantClientsService := warrant.NewClientsService(warrantConfig)
	userFinder := uaa.NewUserFinder(config.UAAClientID, config.UAAClientSecret, warrantUsersService, warrantClientsService)
	campaignEnqueuer := queue.NewCampaignEnqueuer(config.Queue)

	sendersRepository := models.NewSendersRepository(uuid.NewV4)
	campaignTypesRepository := models.NewCampaignTypesRepository(uuid.NewV4)
	templatesRepository := models.NewTemplatesRepository(uuid.NewV4)

	sendersCollection := collections.NewSendersCollection(sendersRepository)
	templatesCollection := collections.NewTemplatesCollection(templatesRepository)
	campaignTypesCollection := collections.NewCampaignTypesCollection(campaignTypesRepository, sendersRepository, templatesRepository)
	campaignsCollection := collections.NewCampaignsCollection(campaignEnqueuer, campaignTypesRepository, templatesRepository, userFinder)

	info.Routes{
		RequestCounter: requestCounter,
		RequestLogging: logging,
	}.Register(mx)

	senders.Routes{
		RequestLogging:    logging,
		Authenticator:     notificationsWriteAuthenticator,
		DatabaseAllocator: databaseAllocator,
		SendersCollection: sendersCollection,
	}.Register(mx)

	campaigntypes.Routes{
		RequestLogging:          logging,
		Authenticator:           notificationsWriteAuthenticator,
		DatabaseAllocator:       databaseAllocator,
		CampaignTypesCollection: campaignTypesCollection,
	}.Register(mx)

	templates.Routes{
		RequestLogging:      logging,
		Authenticator:       notificationsWriteAuthenticator,
		DatabaseAllocator:   databaseAllocator,
		TemplatesCollection: templatesCollection,
	}.Register(mx)

	campaigns.Routes{
		RequestLogging:      logging,
		Authenticator:       notificationsWriteAuthenticator,
		DatabaseAllocator:   databaseAllocator,
		CampaignsCollection: campaignsCollection,
	}.Register(mx)

	return mx
}
