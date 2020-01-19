package main

import (
	"github.com/mattermost/mattermost-server/v5/model"
)

type configuration struct {
	ChannelName string

	channelIds map[string]string

	bot *model.Bot
}

func (c *configuration) Clone() *configuration {
	channelIds := make(map[string]string)
	for key, value := range c.channelIds {
		channelIds[key] = value
	}

	return &configuration{
		ChannelName: c.ChannelName,
		bot:         c.bot,
	}
}

func (p *Plugin) getConfiguration() *configuration {
	p.configurationLock.RLock()
	defer p.configurationLock.RUnlock()

	if p.configuration == nil {
		return &configuration{}
	}

	return p.configuration
}

func (p *Plugin) setConfiguration(configuration *configuration) {
	p.configurationLock.Lock()
	defer p.configurationLock.Unlock()

	if configuration != nil && p.configuration == configuration {
		panic("setConfiguration called with the existing configuration")
	}

	p.configuration = configuration
}
