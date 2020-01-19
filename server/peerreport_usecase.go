package main

import (
	"strings"

	"github.com/mattermost/mattermost-server/v5/model"
)

type peerReportUsecase struct {
	plugin *Plugin
}

const (
	peerReportUsecaseUsage = ""
)

func (p *peerReportUsecase) execute(args *model.CommandArgs) (*model.CommandResponse, *model.AppError) {

	var errorMessage string

	fields := strings.Fields(args.Command)
	if len(fields) != 2 {
		errorMessage = peerReportUsecaseUsage
	}

	return &model.CommandResponse{
		ResponseType: model.COMMAND_RESPONSE_TYPE_EPHEMERAL,
		Text:         errorMessage,
	}, nil
}
