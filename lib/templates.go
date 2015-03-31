package lib

import (
	"encoding/json"

	"github.com/draaglom/GleepostAPI/lib/gp"
)

//CreateTemplateFromPost saves a Post as a Template, so it can be used again.
func (api *API) CreateTemplateFromPost(post gp.PostFull) (templateID gp.TemplateID, err error) {
	template, err := json.MarshalIndent(post, "", "\t")
	templateID, err = api.db.CreateTemplate(1, string(template))
	return
}

//CreatePostFromTemplate creates a new post in this network, generating it from this template.
func (api *API) CreatePostFromTemplate(network gp.NetworkID, template string) (post gp.PostID, err error) {
	//Parse template
	//Insert this network

	return
}
