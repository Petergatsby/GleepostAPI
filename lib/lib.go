package lib

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"io"
	"log"

	"github.com/draaglom/GleepostAPI/lib/conf"
	"github.com/draaglom/GleepostAPI/lib/events"
	"github.com/draaglom/GleepostAPI/lib/gp"
	"github.com/draaglom/GleepostAPI/lib/mail"
	"github.com/draaglom/GleepostAPI/lib/psc"
	"github.com/draaglom/GleepostAPI/lib/push"
	"github.com/draaglom/GleepostAPI/lib/transcode"
	"github.com/garyburd/redigo/redis"
	"github.com/peterbourgon/g2s"
)

//API contains all the configuration and sub-modules the Gleepost API requires to function.
type API struct {
	Auth          *Authenticator
	broker        *events.Broker
	db            *sql.DB
	sc            *psc.StatementCache
	fb            *FB
	Mail          mail.Mailer
	Config        conf.Config
	pushers       map[string]push.Pusher
	Statsd        PrefixStatter
	notifObserver NotificationObserver
	TW            TranscodeWorker
	Viewer        Viewer
	users         *Users
	nm            *NetworkManager
	Presences     Presences
}

const inviteCampaignIOS = "http://ad.apps.fm/2sQSPmGhIyIaKGZ01wtHD_E7og6fuV2oOMeOQdRqrE1xKZaHtwHb8iGWO0i4C3przjNn5v5h3werrSfj3HdREnrOdTW3xhZTjoAE5juerBQ8UiWF6mcRlxGSVB6OqmJv"
const inviteCampaignAndroid = "http://ad.apps.fm/WOIqfW3iWi3krjT_Y-U5uq5px440Px0vtrw1ww5B54zsDQMwj9gVfW3tCxpkeXdizYtt678Ci7Y3djqLAxIATdBAW28aYabvxh6AeQ1YLF8"

var (
	//ETOOWEAK - You'll get this when your password is too week (ie, less than 5 chars at the moment)
	ETOOWEAK = gp.APIerror{Reason: "Password too weak!"}
	//EBADREC means you tried to recover your password with an invalid or missing password reset token.
	EBADREC = gp.APIerror{Reason: "Bad password recovery token."}
)

//New creates an API from a gp.Config
func New(conf conf.Config) (api *API) {
	api = new(API)
	api.broker = events.New(conf.Redis)
	api.Config = conf
	api.fb = &FB{config: conf.Facebook}
	api.Mail = mail.New(conf.Email.FromHeader, conf.Email.From, conf.Email.User, conf.Email.Pass, conf.Email.Server, conf.Email.Port)
	db, err := sql.Open("mysql", conf.Mysql.ConnectionString())
	if err != nil {
		log.Fatal("error getting db:", err)
	}
	db.SetMaxIdleConns(100)
	api.sc = psc.NewCache(db)
	api.db = db
	api.Auth = &Authenticator{sc: api.sc, pool: redis.NewPool(events.GetDialer(conf.Redis), 100)}
	api.TW = newTranscodeWorker(db, api.sc, transcode.NewTranscoder(), api.getS3(1911).Bucket("gpcali"), api.broker)
	api.Viewer = &viewer{broker: api.broker, sc: api.sc}
	api.users = &Users{sc: api.sc}
	api.nm = &NetworkManager{sc: api.sc}
	api.Presences = Presences{broker: api.broker, sc: api.sc}
	return
}

//Start connects to various services & makes the API ready to go.
func (api *API) Start() {
	api.pushers = make(map[string]push.Pusher)
	if len(api.Config.Pushers) == 0 {
		log.Println("No pushers configured. Are you sure this is right?")
	}
	for _, psh := range api.Config.Pushers {
		api.pushers[psh.AppName] = push.New(psh)
	}
	gp, ok := api.pushers["gleepost"]
	if ok {
		api.notifObserver = NewObserver(api.db, api.broker, gp, api.sc, api.users, api.nm)
	} else {
		log.Println("No \"gleepost\" pusher; using blackhole pusher")
		api.notifObserver = NewObserver(api.db, api.broker, push.NewFake(), api.sc, api.users, api.nm)
	}
	statsd, err := g2s.Dial("udp", api.Config.Statsd)
	if err != nil {
		log.Printf("Statsd failed: %s\n", err)
	} else {
		api.Statsd = PrefixStatter{statter: statsd, DevelopmentMode: api.Config.DevelopmentMode}
		api.users.statter = api.Statsd
		api.notifObserver.stats = api.Statsd
		api.Presences.Statsd = api.Statsd
	}
}

//RandomString generates a long, random string (currently hex encoded, for some unknown reason.)
//TODO: base64 url-encode instead.
func randomString() (random string, err error) {
	hash := sha256.New()
	randombuf := make([]byte, 32) //Number pulled out of my... ahem.
	_, err = io.ReadFull(rand.Reader, randombuf)
	if err != nil {
		return
	}
	hash.Write(randombuf)
	random = hex.EncodeToString(hash.Sum(nil))
	return
}
