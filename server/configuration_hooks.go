package main

import (
	"github.com/pkg/errors"

	"github.com/mattermost/mattermost-server/v5/model"
)

const (
	botName        = "peerbot"
	botDisplayName = "ピア太郎"
	botDescription = "ピア投稿プラグインによって作成されたbot"

	channelName        = "peer-channel"
	channelDisplayName = "ピア投稿部屋"
	channelHeader      = "The channel used by the demo plugin."
	channelPurpose     = "This channel was created by a plugin for testing."
)

func (p *Plugin) OnConfigurationChange() error {
	var configuration = new(configuration)

	// Load the public configuration fields from the Mattermost server configuration.
	if loadConfigErr := p.API.LoadPluginConfiguration(configuration); loadConfigErr != nil {
		return errors.Wrap(loadConfigErr, "failed to load plugin configuration")
	}

	if error := p.ensureBot(configuration); error != nil {
		return error
	}

	if error := p.ensureChannels(configuration); error != nil {
		return error
	}

	p.setConfiguration(configuration)

	return nil
}

func (p *Plugin) ensureBot(configuration *configuration) error {

	//無ければ作成、あれば取得
	botID, err := p.Helpers.EnsureBot(&model.Bot{
		Username:    botName,
		DisplayName: botDisplayName,
		Description: botDescription,
	})
	if err != nil {
		return err
	}

	//idからBotを取得
	bot, appError := p.API.GetBot(botID, false)
	if appError != nil {
		return errors.Wrap(appError, "failed to GetBot.")
	}
	configuration.bot = bot

	return nil
}

func (p *Plugin) ensureChannels(configuration *configuration) error {
	teams, err := p.API.GetTeams()
	if err != nil {
		return err
	}

	//全てのチームにチャンネルを作成する
	channelIds := make(map[string]string)
	for _, team := range teams {
		channel, _ := p.API.GetChannelByNameForTeamName(team.Name, channelName, false)

		if channel == nil {
			channel, err = p.API.CreateChannel(&model.Channel{
				TeamId:      team.Id,
				Type:        model.CHANNEL_OPEN,
				DisplayName: channelDisplayName,
				Name:        channelName,
				Header:      channelHeader,
				Purpose:     channelPurpose,
			})
			if err != nil {
				return err
			}
		}
		channelIds[team.Id] = channel.Id
	}
	configuration.channelIds = channelIds

	return nil
}