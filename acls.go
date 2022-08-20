package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/tailscale/hujson"
)

var aclUpdateRetry = errors.New("If-Match condition failed")

func getAcls() (acls hujson.Value, etag string, err error) {
	req, err := http.NewRequest("GET", tsControlServer+"/api/v2/tailnet/"+tailnet+"/acl", nil)
	if err != nil {
		return hujson.Value{}, "", err
	}
	req.SetBasicAuth(tsApiKey, "")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return hujson.Value{}, "", err
	}

	if resp.StatusCode != http.StatusOK {
		return hujson.Value{}, "", nil
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return hujson.Value{}, "", err
	}

	acls, err = hujson.Parse(body)
	if err != nil {
		return hujson.Value{}, "", err
	}

	return acls, resp.Header.Get("ETag"), nil
}

func putAcls(acls hujson.Value, etag string) error {
	url := tsControlServer + "/api/v2/tailnet/" + tailnet + "/acl"
	req, err := http.NewRequest("POST", url, bytes.NewBufferString(acls.String()))
	if err != nil {
		return err
	}
	req.SetBasicAuth(tsApiKey, "")
	req.Header.Set("Content-Type", "application/hujson")
	req.Header.Set("If-Match", etag)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	if resp.StatusCode == http.StatusPreconditionFailed {
		return aclUpdateRetry
	} else if resp.StatusCode != http.StatusOK {
		return errors.New(fmt.Sprintf("HTTP POST failed: %d", resp.StatusCode))
	}

	return nil
}

func updateHosts(ctx context.Context, update map[string]string) {
	retry := true
	for retry {
		retry = false
		changed := false
		acls, etag, err := getAcls()
		if err != nil {
			log.Printf("getAcls failed: %v", err)
			break
		}
		if etag == "" {
			log.Printf("getAcls returned empty")
			break
		}

		for key, value := range update {
			patch := `[{ "op": "replace", "path": "/Hosts/` + key + `", "value": "` + value + `" }]`
			err = acls.Patch([]byte(patch))
			if err == nil {
				changed = true
			}
		}

		if changed {
			err = putAcls(acls, etag)
			if err != nil {
				if errors.Is(err, aclUpdateRetry) {
					// If-Match failed, collision in updating ACLs
					retry = true
				} else {
					log.Printf("putAcls failed: %v", err)
				}
			}
		}
	}
}
