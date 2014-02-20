package lib

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"github.com/draaglom/GleepostAPI/lib/gp"
	"io"
	"io/ioutil"
	"launchpad.net/goamz/aws"
	"launchpad.net/goamz/s3"
	"mime/multipart"
	"strings"
)

func randomFilename(extension string) (string, error) {
	hash := sha256.New()
	random := make([]byte, 32) //Number pulled out of my... ahem.
	_, err := io.ReadFull(rand.Reader, random)
	if err == nil {
		hash.Write(random)
		digest := hex.EncodeToString(hash.Sum(nil))
		return digest + extension, nil
	} else {
		return "", err
	}
}

func (api *API) getS3(network gp.NetworkId) (s *s3.S3) {
	var auth aws.Auth
	auth.AccessKey, auth.SecretKey = api.Config.AWS.KeyId, api.Config.AWS.SecretKey
	//1911 == Stanford.
	//TODO: Make the bucket a property of the university / group of universities
	if network == 1911 {
		s = s3.New(auth, aws.USWest)
	} else {
		s = s3.New(auth, aws.EUWest)
	}
	return
}

func (api *API) StoreFile(id gp.UserId, file multipart.File, header *multipart.FileHeader) (url string, err error) {
	var filename string
	var contenttype string
	switch {
	case strings.HasSuffix(header.Filename, ".jpg"):
		filename, err = randomFilename(".jpg")
		contenttype = "image/jpeg"
	case strings.HasSuffix(header.Filename, ".jpeg"):
		filename, err = randomFilename(".jpg")
		contenttype = "image/jpeg"
	case strings.HasSuffix(header.Filename, ".png"):
		filename, err = randomFilename(".png")
		contenttype = "image/png"
	default:
		return "", gp.APIerror{"Unsupported file type"}
	}
	if err != nil {
		return "", gp.APIerror{err.Error()}
	}
	//store on s3
	networks, _ := api.GetUserNetworks(id)
	var s *s3.S3
	var bucket *s3.Bucket
	switch {
	case len(networks) > 0:
		s = api.getS3(networks[0].Id)
		if networks[0].Id == 1911 {
			bucket = s.Bucket("gpcali")
		} else {
			bucket = s.Bucket("gpimg")
		}
	default:
		s = api.getS3(1)
		bucket = s.Bucket("gpimg")
	}
	data, err := ioutil.ReadAll(file)
	if err != nil {
		return "", err
	}
	err = bucket.Put(filename, data, contenttype, s3.PublicRead)
	url = bucket.URL(filename)
	if err != nil {
		return "", err
	}
	err = api.userAddUpload(id, url)
	return url, err
}

func (api *API) userAddUpload(id gp.UserId, url string) (err error) {
	return api.db.AddUpload(id, url)
}

func (api *API) UserUploadExists(id gp.UserId, url string) (exists bool, err error) {
	return api.db.UploadExists(id, url)
}
