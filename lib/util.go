package lib

import (
	"log"

	"time"
)

//KeepPostsInFuture checks a list of posts every PollInterval and pushes them into the future if neccessary
func (api *API) KeepPostsInFuture(pollInterval time.Duration) {
	t := time.Tick(pollInterval)
	for {
		err := api.db.KeepPostsInFuture()
		if err != nil {
			log.Println(err)
		}
		<-t
	}
}
