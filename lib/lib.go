package lib

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"log"

	"time"

	"github.com/draaglom/GleepostAPI/lib/cache"
	"github.com/draaglom/GleepostAPI/lib/conf"
	"github.com/draaglom/GleepostAPI/lib/db"
	"github.com/draaglom/GleepostAPI/lib/gp"
	"github.com/draaglom/GleepostAPI/lib/mail"
	"github.com/draaglom/GleepostAPI/lib/push"
	"github.com/peterbourgon/g2s"
)

//API contains all the configuration and sub-modules the Gleepost API requires to function.
type API struct {
	cache   *cache.Cache
	db      *db.DB
	fb      *FB
	Mail    mail.Mailer
	Config  conf.Config
	pushers map[string]*push.Pusher
	statsd  g2s.Statter
}

const inviteCampaignIOS = "http://ad.apps.fm/2sQSPmGhIyIaKGZ01wtHD_E7og6fuV2oOMeOQdRqrE1xKZaHtwHb8iGWO0i4C3przjNn5v5h3werrSfj3HdREnrOdTW3xhZTjoAE5juerBQ8UiWF6mcRlxGSVB6OqmJv"
const inviteCampaignAndroid = "http://ad.apps.fm/WOIqfW3iWi3krjT_Y-U5uq5px440Px0vtrw1ww5B54zsDQMwj9gVfW3tCxpkeXdizYtt678Ci7Y3djqLAxIATdBAW28aYabvxh6AeQ1YLF8"

var (
	//You'll get this when your password is too week (ie, less than 5 chars at the moment)
	ETOOWEAK = gp.APIerror{Reason: "Password too weak!"}
	//EBADREC means you tried to recover your password with an invalid or missing password reset token.
	EBADREC = gp.APIerror{Reason: "Bad password recovery token."}
)

//New creates an API from a gp.Config
func New(conf conf.Config) (api *API) {
	api = new(API)
	api.cache = cache.New(conf.Redis)
	api.db = db.New(conf.Mysql)
	api.Config = conf
	api.fb = &FB{config: conf.Facebook}
	api.Mail = mail.New(conf.Email.FromHeader, conf.Email.From, conf.Email.User, conf.Email.Pass, conf.Email.Server, conf.Email.Port)
	return
}

//Start connects to various services & makes the API ready to go.
func (api *API) Start() {
	api.pushers = make(map[string]*push.Pusher)
	for _, psh := range api.Config.Pushers {
		log.Println(psh)
		api.pushers[psh.AppName] = push.New(psh)
	}
	statsd, err := g2s.Dial("udp", api.Config.Statsd)
	if err != nil {
		log.Printf("Statsd failed: %s\nMake sure you have the right address in your conf.json\n", err)
	} else {
		api.statsd = statsd
	}
	go api.process(transcodeQueue)
}

//Time reports the time for this stat to statsd. (use it with defer)
func (api *API) Time(start time.Time, bucket string) {
	//TODO: Move the stats stuff into its own module?
	duration := time.Since(start)
	bucket = api.statsdPrefix() + bucket
	if api.statsd != nil {
		api.statsd.Timing(1.0, bucket, duration)
	}
}

func (api *API) statsdPrefix() string {
	if api.Config.DevelopmentMode {
		return "dev."
	}
	return "prod."
}

//Count wraps a g2s.Statter giving an automatic version prefix and a single location to set the report probability.
func (api *API) Count(count int, bucket string) {
	if api.statsd != nil {
		api.statsd.Counter(1.0, api.statsdPrefix()+bucket, count)
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
