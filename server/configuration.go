package main

import (
	"github.com/mattermost/mattermost-server/v5/model"
)

type configuration struct {
	Hashtags string

	channelIds map[string]string

	bot *model.Bot

	hashtagOptions []*model.PostActionOptions
}

func (c *configuration) Clone() *configuration {
	var configuration = configuration{}

	configuration.bot = c.bot

	configuration.channelIds = make(map[string]string)
	for key, value := range c.channelIds {
		configuration.channelIds[key] = value
	}

	configuration.hashtagOptions = []*model.PostActionOptions{}
	for _, option := range c.hashtagOptions {
		o := model.PostActionOptions{
			Text:  option.Text,
			Value: option.Value,
		}
		configuration.hashtagOptions = append(configuration.hashtagOptions, &o)
	}

	return &configuration
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
