package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/mattermost/mattermost-server/v5/plugin"
)

// ServeHTTP allows the plugin to implement the http.Handler interface. Requests destined for the
// /plugins/{id} path will be routed to the plugin.
//
// The Mattermost-User-Id header will be present if (and only if) the request is by an
// authenticated user.
//
// This demo implementation sends back whether or not the plugin hooks are currently enabled. It
// is used by the web app to recover from a network reconnection and synchronize the state of the
// plugin's hooks.
func (p *Plugin) ServeHTTP(c *plugin.Context, w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	if strings.HasPrefix(path, "/stamp/") {
		p.handleStampImage(path, w)
	} else if path == "/peer/callback" {
		uc := peerPostUsecase{
			plugin: p,
		}
		uc.handleDialogCallback(w, r)
	} else {
		http.NotFound(w, r)
	}
}

func (p *Plugin) handleStampImage(path string, w http.ResponseWriter) {
	imagePath := fmt.Sprintf("plugins/%s/assets%s", manifest.Id, path)
	data, err := ioutil.ReadFile(imagePath)
	if err == nil {
		w.Header().Set("Content-Type", "image/png")
		w.Write(data)
	} else {
		w.WriteHeader(404)
		w.Write([]byte("404 Something went wrong - " + http.StatusText(404)))
		p.API.LogInfo(path+" err = ", err.Error())
	}
}
