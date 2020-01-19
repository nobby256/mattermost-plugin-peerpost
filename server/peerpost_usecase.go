package main

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/mattermost/mattermost-server/v5/model"
)

type peerPostUsecase struct {
	plugin *Plugin
}

const (
	commandPeerUsage = "** ピア投稿 Slash Command Help **\n\n  /peer @ユーザ名\n\n  - 自身は指定できません。\n\n  - 複数人を指すメンションは指定できません。（ex: @all, @channel, @here）\n\n  - チーム内のメンバーのみ指定できます。"
)
const (
	dialogElementText     = "text"
	dialogElementStamp    = "stamp"
	dialogElementHashtag1 = "hashtag1"
	dialogElementHashtag2 = "hashtag2"
	dialogElementHashtag3 = "hashtag3"
)

func (p *peerPostUsecase) execute(args *model.CommandArgs) (*model.CommandResponse, *model.AppError) {

	fields := strings.Fields(args.Command)
	if len(fields) != 2 {
		return p.plugin.createErrorCommandResponse(commandPeerUsage), nil
	}

	//メンションを取得
	mention := fields[1]

	//メンションからユーザ名を取得
	var userName string
	if !strings.HasPrefix(mention, "@") {
		return p.plugin.createErrorCommandResponse(commandPeerUsage), nil
	} else if "@all" == mention || "@channel" == mention || "@here" == mention {
		return p.plugin.createErrorCommandResponse(commandPeerUsage), nil
	} else {
		userName = string([]rune(mention))[1:]
	}

	//ユーザーの妥当性確認
	var targetUser model.User
	if users, err := p.plugin.API.GetUsersByUsernames([]string{userName}); err != nil {
		return nil, err
	} else if len(users) == 0 {
		errorMessage := fmt.Sprintf("該当ユーザーを見つけることができませんでした。（%s）\n\n%s", mention, commandPeerUsage)
		return p.plugin.createErrorCommandResponse(errorMessage), nil
	} else if len(users) > 1 {
		errorMessage := fmt.Sprintf("ユーザーを一人に絞り込むことが出来ませんでした。（%s）\n\n%s", mention, commandPeerUsage)
		return p.plugin.createErrorCommandResponse(errorMessage), nil
	} else {
		targetUser = *users[0]
		if targetUser.Id == args.UserId {
			return p.plugin.createErrorCommandResponse("自身を指定することはできません。"), nil
		}
	}

	dialogRequest := p.createDialogRequest(args.TriggerId, targetUser)
	if apiError := p.plugin.API.OpenInteractiveDialog(dialogRequest); apiError != nil {
		p.plugin.API.LogError("Failed to open Interactive Dialog", "err", apiError.Error())
		return nil, apiError
	}

	return &model.CommandResponse{}, nil
}

func (p *peerPostUsecase) createDialogRequest(triggerID string, targetUser model.User) model.OpenDialogRequest {
	return model.OpenDialogRequest{
		TriggerId: triggerID,
		URL:       p.plugin.getServerHTTPURL("/peer/callback"),
		Dialog: model.Dialog{
			Title: p.plugin.getUserDisplayName(targetUser) + " さんへのメッセージ",
			Elements: []model.DialogElement{
				{
					DisplayName: "メッセージ",
					Name:        dialogElementText,
					Type:        "textarea",
					Default:     "",
					Placeholder: "今日の打合せの相談に乗ってくれてありがとう。\nおかげでうまくまとめることが出来たよ。",
					MinLength:   1,
					MaxLength:   500,
				}, {
					DisplayName: "チームハッシュタグ１",
					Name:        dialogElementHashtag1,
					Type:        "select",
					Options:     p.createHashtagOptions(),
				}, {
					DisplayName: "チームハッシュタグ２",
					Name:        dialogElementHashtag2,
					Type:        "select",
					Options:     p.createHashtagOptions(),
					Optional:    true,
				}, {
					DisplayName: "スタンプ",
					Name:        dialogElementStamp,
					Type:        "select",
					Options:     p.createStampOptions(),
				}},
			SubmitLabel:    "投稿する",
			NotifyOnCancel: true,
			State:          targetUser.Id,
		},
	}
}

func (p *peerPostUsecase) handleDialogCallback(w http.ResponseWriter, r *http.Request) {
	request := model.SubmitDialogRequestFromJson(r.Body)
	if request == nil {
		p.plugin.API.LogError("failed to decode SubmitDialogRequest")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if request.Cancelled {
		configuration := p.plugin.getConfiguration()
		if post := p.plugin.API.SendEphemeralPost(request.UserId,
			&model.Post{
				UserId:    configuration.bot.UserId,
				ChannelId: request.ChannelId,
				Message:   "投稿をキャンセルしました。",
			}); post == nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		} else {
			w.WriteHeader(http.StatusOK)
			return
		}
	}

	submission := request.Submission

	text, _ := submission[dialogElementText].(string)
	hashtags, _ := submission[dialogElementHashtag1].(string)
	hashtags = "#" + hashtags
	if tag, ok := submission[dialogElementHashtag2].(string); ok && tag != "" {
		hashtags = hashtags + " #" + tag
	}
	stamp, _ := submission[dialogElementStamp].(string)
	stamp = p.plugin.getServerHTTPURL(stamp)

	createUser, err := p.plugin.API.GetUser(request.UserId)
	if err != nil {
		p.plugin.API.LogError("Failed to GetUser", "err", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	targetUserID := request.State
	targetUser, err := p.plugin.API.GetUser(targetUserID)
	if err != nil {
		p.plugin.API.LogError("Failed to GetUser", "err", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	message := fmt.Sprintf("@%sさんへ\n%s\n%s", targetUser.GetDisplayName(model.SHOW_NICKNAME_FULLNAME), text, hashtags)

	configuration := p.plugin.getConfiguration()
	post := model.Post{
		ChannelId: configuration.channelIds[request.TeamId],
		//ChannelId: request.ChannelId,
		Type:   "custom_peer-post",
		UserId: configuration.bot.UserId,
		Props: model.StringInterface{
			"hashtags": hashtags,
			"from-to":  request.UserId + " " + targetUserID,
			"attachments": []*model.SlackAttachment{{
				AuthorName: createUser.GetDisplayName(model.SHOW_NICKNAME_FULLNAME),
				AuthorIcon: p.plugin.getUserProfileImageURL(createUser.Id),
				Text:       message,
				ThumbURL:   stamp,
			}},
		},
	}

	//所定のチャンネルにBotとして投稿
	postResult, err := p.plugin.API.CreatePost(&post)
	if err != nil {
		p.plugin.API.LogError("Failed to CreatePost", "err", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	//コマンドを実行したチャンネルと、投稿先のチャンネルが違っている場合は
	//完了メッセージをボットが投稿
	if request.ChannelId != configuration.channelIds[request.TeamId] {
		permalink, err := p.plugin.getPermanentLinkURL(request.TeamId, postResult.Id)
		if err != nil {
			p.plugin.API.LogError("SendEphemeralPost", "err", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if result := p.plugin.API.SendEphemeralPost(
			request.UserId,
			&model.Post{
				ChannelId: request.ChannelId,
				UserId:    configuration.bot.UserId,
				Message:   fmt.Sprintf("[こちらに投稿しました。](%s)", permalink),
			}); result == nil {
			p.plugin.API.LogError("Failed to SendEphemeralPost", "err", nil)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}
}

func (p *peerPostUsecase) writeSubmitDialogResponse(w http.ResponseWriter, response *model.SubmitDialogResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(response.ToJson()); err != nil {
		p.plugin.API.LogError("failed to write DialogResponse", "err", err.Error())
	}
}

func (p *peerPostUsecase) createStampOptions() []*model.PostActionOptions {
	return []*model.PostActionOptions{
		{Text: "カッコいい", Value: "/stamp/stamp_1.png"},
		{Text: "カワいい", Value: "/stamp/stamp_2.png"},
		{Text: "ステキ", Value: "/stamp/stamp_3.png"},
		{Text: "たしかに", Value: "/stamp/stamp_4.png"},
		{Text: "それな", Value: "/stamp/stamp_5.png"},
		{Text: "天才", Value: "/stamp/stamp_6.png"},
		{Text: "つよつよ", Value: "/stamp/stamp_7.png"},
		{Text: "わかる", Value: "/stamp/stamp_8.png"},
		{Text: "GJ", Value: "/stamp/stamp_9.png"},
		{Text: "いいね", Value: "/stamp/stamp_10.png"},
		{Text: "優秀", Value: "/stamp/stamp_11.png"},
		{Text: "がんばれ", Value: "/stamp/stamp_12.png"},
		{Text: "神", Value: "/stamp/stamp_13.png"},
		{Text: "早く寝ろ", Value: "/stamp/stamp_14.png"},
		{Text: "さすが", Value: "/stamp/stamp_15.png"},
		{Text: "養いたい", Value: "/stamp/stamp_16.png"},
		{Text: "世界一", Value: "/stamp/stamp_17.png"},
		{Text: "富", Value: "/stamp/stamp_18.png"},
		{Text: "名声", Value: "/stamp/stamp_19.png"},
		{Text: "力", Value: "/stamp/stamp_20.png"},
		{Text: "卍", Value: "/stamp/stamp_21.png"},
		{Text: "尊い", Value: "/stamp/stamp_22.png"},
		{Text: "ワロタ", Value: "/stamp/stamp_23.png"},
		{Text: "草", Value: "/stamp/stamp_24.png"},
		{Text: "養われたい", Value: "/stamp/stamp_25.png"},
		{Text: "すごい", Value: "/stamp/stamp_26.png"},
		{Text: "は？", Value: "/stamp/stamp_27.png"},
		{Text: "ハート", Value: "/stamp/stamp_28.gif"},
	}
}

func (p *peerPostUsecase) createHashtagOptions() []*model.PostActionOptions {
	return []*model.PostActionOptions{{
		Text:  "チャレンジ",
		Value: "チャレンジ",
	}, {
		Text:  "縁の下の力持ち",
		Value: "縁の下の力持ち",
	}, {
		Text:  "組織の壁を超えて",
		Value: "組織の壁を超えて",
	}}
}
