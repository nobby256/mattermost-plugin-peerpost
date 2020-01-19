package main

import (
	"strings"

	"github.com/pkg/errors"

	"github.com/mattermost/mattermost-server/v5/model"
	"github.com/mattermost/mattermost-server/v5/plugin"
)

const (
	commandPeerPost   = "peer"
	commandPeerReport = "peer-report"
)

func (p *Plugin) registerCommands() error {
	var err error

	err = p.API.RegisterCommand(&model.Command{
		Trigger:          commandPeerPost,
		AutoComplete:     true,
		AutoCompleteHint: "メンション",
		AutoCompleteDesc: "ピア投稿を行えます",
		DisplayName:      "ピア投稿 コマンド",
	})
	if err != nil {
		return errors.Wrapf(err, "failed to register %s command", commandPeerPost)
	}

	err = p.API.RegisterCommand(&model.Command{
		Trigger:          commandPeerReport,
		AutoComplete:     true,
		AutoCompleteHint: "[YYYY/MM/DD]",
		AutoCompleteDesc: "ピア投稿の各種ランキングを見ることが出来ます",
		DisplayName:      "ピア投稿レポート コマンド",
	})
	if err != nil {
		return errors.Wrapf(err, "failed to register %s command", commandPeerReport)
	}

	return nil
}

func (p *Plugin) ExecuteCommand(c *plugin.Context, args *model.CommandArgs) (*model.CommandResponse, *model.AppError) {
	var response *model.CommandResponse
	var appError *model.AppError

	trigger := strings.TrimPrefix(strings.Fields(args.Command)[0], "/")
	switch trigger {
	case commandPeerPost:
		uc := peerPostUsecase{
			plugin: p,
		}
		response, appError = uc.execute(args)
	case commandPeerReport:
		uc := peerReportUsecase{
			plugin: p,
		}
		response, appError = uc.execute(args)
	default:
		response, appError = nil, nil
	}

	return response, appError
}
