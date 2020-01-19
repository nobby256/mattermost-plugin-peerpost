package main

import (
	"bytes"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/mattermost/mattermost-server/v5/model"
)

type peerReportUsecase struct {
	plugin *Plugin
}

const (
	commandPeerReportUsage = "** Slash Command Help **\n\n  /peer-report [YYYY/MM/DD]\n\n  - 日付は省略可能です。\n\n  - 日付を省略した場合は今週の月曜日からの集計となります。\n\n  - 集計期間は指定した日から現在まで。"
)

type ranking struct {
	fromRanking     []userIDCountPair
	toRanking       []userIDCountPair
	reactionRanking []userIDCountPair
	hashTagRanking  []userIDCountPair
	displayNameMap  map[string]string
}

type userIDCountPair struct {
	key   string
	count int
}

func (p *peerReportUsecase) execute(args *model.CommandArgs) (*model.CommandResponse, *model.AppError) {

	fields := strings.Fields(args.Command)
	if len(fields) > 2 {
		return p.plugin.createErrorCommandResponse(commandPeerReportUsage), nil
	}

	var arg string = ""
	if len(fields) == 2 {
		arg = fields[1]
	}

	from, err := p.getFromDate(arg)
	if err != nil {
		return p.plugin.createErrorCommandResponse(err.Error()), nil
	}

	configuration := p.plugin.getConfiguration()
	channelID := configuration.channelIds[args.TeamId]

	//指定のチャンネルに投稿されたPostから各種数値を数える
	info, err := p.countPost(channelID, from)

	message := p.createReportMessage(info)

	configurtion := p.plugin.getConfiguration()
	post := model.Post{
		ChannelId: args.ChannelId,
		UserId:    configurtion.bot.UserId,
		Message:   message,
	}

	if postResult := p.plugin.API.SendEphemeralPost(args.UserId, &post); postResult == nil {
		p.plugin.API.LogError("faild to SendEphemeralPost", "err", post)
		return nil, nil
	}

	return &model.CommandResponse{}, nil
}
func (p *peerReportUsecase) addCount(countMap *map[string]int, id string) {
	if count, ok := (*countMap)[id]; ok {
		(*countMap)[id] = count + 1
	} else {
		(*countMap)[id] = 1
	}
}

func (p *peerReportUsecase) getFromDate(arg string) (time.Time, error) {
	var from time.Time
	var err error = nil
	if arg == "" {
		today, _ := time.Parse("2006-01-02", time.Now().Format("2006-01-02"))
		dayOfWeek := int(today.Weekday()+6) % 7  //0:月曜
		from = today.AddDate(0, 0, -1*dayOfWeek) //今週の月曜日を求める
	} else {
		from, err = time.Parse("2006/01/02", arg)
		if err != nil {
			err = errors.New("有効な日付ではありません。\n\n" + commandPeerReportUsage)
		}
	}
	return from, err
}

func (p *peerReportUsecase) countPost(channelID string, from time.Time) (*ranking, error) {
	var fromMilliSecond int64 = from.Unix() * 1000
	postList, appError := p.plugin.API.GetPostsSince(channelID, fromMilliSecond)
	if appError != nil {
		return nil, appError
	}

	fromCountMap := map[string]int{}
	toCountMap := map[string]int{}
	reactionCountMap := map[string]int{}
	hashTagCountMap := map[string]int{}
	rank := ranking{
		fromRanking:     []userIDCountPair{},
		toRanking:       []userIDCountPair{},
		reactionRanking: []userIDCountPair{},
		hashTagRanking:  []userIDCountPair{},
		displayNameMap:  map[string]string{},
	}

	for _, post := range postList.Posts {
		if post.Type != "custom_peer-post" {
			continue //違う投稿
		}
		if post.DeleteAt != 0 {
			continue //削除済み
		}
		value, ok := post.Props["from-to"]
		if !ok {
			continue //本来あり得ない
		}
		fromTo, ok := value.(string)
		if !ok {
			p.plugin.API.LogError("Failed to type assertion", "err", post.Props["peer-post"])
			panic(value)
		}

		//from,toで登場した数を数える
		ids := strings.Split(fromTo, " ")
		p.addCount(&fromCountMap, ids[0]) //fromとして登場した数
		p.addCount(&toCountMap, ids[1])   //toとして登場した数

		//登場したUserIDをmapで集める
		rank.displayNameMap[ids[0]] = "" //値は後で解決
		rank.displayNameMap[ids[1]] = "" //値は後で解決

		//ハッシュタグの登場回数を数える
		if post.Hashtags != "" {
			for _, tag := range strings.Split(post.Hashtags, " ") {
				p.addCount(&hashTagCountMap, tag)
			}
		}

		if post.HasReactions {
			//リアクションの数をユーザ毎に数える
			var reactions []*model.Reaction
			reactions, err := p.plugin.API.GetReactions(post.Id)
			if err != nil {
				return nil, err
			}
			for _, reaction := range reactions {
				userID := reaction.UserId
				rank.displayNameMap[userID] = ""      //値は後で解決
				p.addCount(&reactionCountMap, userID) //リアクションの数を数える
			}
		}
	}

	//ユーザIDのmapからディスプレイ名を取得する
	for userID := range rank.displayNameMap {
		user, err := p.plugin.API.GetUser(userID)
		if err != nil {
			return nil, err
		}
		rank.displayNameMap[userID] = p.plugin.getUserDisplayName(*user)
	}
	rank.fromRanking = p.sortCountMap(&fromCountMap)
	rank.toRanking = p.sortCountMap(&toCountMap)
	rank.reactionRanking = p.sortCountMap(&reactionCountMap)
	rank.hashTagRanking = p.sortCountMap(&hashTagCountMap)

	return &rank, nil
}

func (p *peerReportUsecase) createReportMessage(rank *ranking) string {

	var buf bytes.Buffer

	buf.WriteString("褒められた回数\n\n")
	buf.WriteString("| 名前 | 回数 |\n")
	buf.WriteString("| :--- | ---: |\n")
	for _, pair := range rank.toRanking {
		text := fmt.Sprintf("|%s|%d|\n", rank.displayNameMap[pair.key], pair.count)
		buf.WriteString(text)
	}

	buf.WriteString("\n\n")

	buf.WriteString("褒めた回数\n\n")
	buf.WriteString("| 名前 | 回数 |\n")
	buf.WriteString("| :--- | ---: |\n")
	for _, pair := range rank.fromRanking {
		text := fmt.Sprintf("|%s|%d|\n", rank.displayNameMap[pair.key], pair.count)
		buf.WriteString(text)
	}

	buf.WriteString("\n\n")

	buf.WriteString("リアクション回数\n\n")
	buf.WriteString("| 名前 | 回数 |\n")
	buf.WriteString("| :--- | ---: |\n")
	for _, pair := range rank.reactionRanking {
		text := fmt.Sprintf("|%s|%d|\n", rank.displayNameMap[pair.key], pair.count)
		buf.WriteString(text)
	}

	buf.WriteString("\n\n")

	buf.WriteString("ハッシュタグ使用回数\n\n")
	buf.WriteString("| ハッシュタグ | 回数 |\n")
	buf.WriteString("| :--- | ---: |\n")
	for _, pair := range rank.hashTagRanking {
		text := fmt.Sprintf("|%s|%d|\n", pair.key, pair.count)
		buf.WriteString(text)
	}

	message := buf.String()
	return message
}

func (p *peerReportUsecase) sortCountMap(countMap *map[string]int) []userIDCountPair {
	sorter := []userIDCountPair{}
	for key, value := range *countMap {
		pair := userIDCountPair{
			key:   key,
			count: value,
		}
		sorter = append(sorter, pair)
	}
	sort.Slice(sorter, func(i, j int) bool {
		var s1 userIDCountPair = sorter[i]
		var s2 userIDCountPair = sorter[j]
		var result bool
		if s1.count == s2.count {
			result = s1.key < s2.key
		} else {
			result = !(s1.count < s2.count) //降順
		}
		return result
	})
	return sorter
}
