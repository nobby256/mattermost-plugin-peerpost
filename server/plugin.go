package main

import (
	"fmt"
	"sync"
	"time"

	"github.com/mattermost/mattermost-server/v5/model"
	"github.com/mattermost/mattermost-server/v5/plugin"
	"github.com/pkg/errors"
)

type Plugin struct {
	plugin.MattermostPlugin

	configurationLock sync.RWMutex

	configuration *configuration

	run bool
}

func (p *Plugin) waitForDebuggerAttach() {
	if p.run {
		return
	}
	loop := true
	for loop {
		println("waiting...")
		time.Sleep(time.Duration(1) * time.Second)
	}
	p.run = true
}

func (p *Plugin) getUserProfileImageURL(userID string) string {
	siteURL := p.API.GetConfig().ServiceSettings.SiteURL
	userProfileImageURL := fmt.Sprintf("%s/api/v4/users/%s/image? =0", *siteURL, userID)
	return userProfileImageURL
}

func (p *Plugin) getBotImageURL() string {
	bot := p.getConfiguration().bot
	return p.getUserProfileImageURL(bot.UserId)
}

func (p *Plugin) getUserDisplayName(user model.User) string {
	//p.API.GetConfig().ServiceSettings
	return user.GetDisplayName(model.SHOW_NICKNAME_FULLNAME)
}

func (p *Plugin) getBotDisplayName() string {
	//p.API.GetConfig().ServiceSettings
	bot := p.getConfiguration().bot
	return bot.DisplayName
}

func (p *Plugin) getPermanentLinkURL(teamID string, postID string) (string, error) {
	var team *model.Team
	team, err := p.API.GetTeam(teamID)
	if err != nil {
		return "", errors.Cause(err)
	}
	siteURL := p.API.GetConfig().ServiceSettings.SiteURL
	permaLinkURL := fmt.Sprintf("%s/%s/pl/%s", *siteURL, team.Name, postID)
	return permaLinkURL, nil
}

func (p *Plugin) getServerHTTPURL(path string) string {
	pluginID := manifest.Id
	url := fmt.Sprintf("/plugins/%s%s", pluginID, path)
	return url
}
