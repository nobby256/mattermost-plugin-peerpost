package main

import (
	"github.com/mattermost/mattermost-server/v5/model"
	"github.com/mattermost/mattermost-server/v5/plugin"
)

func (p *Plugin) MessageWillBePosted(c *plugin.Context, post *model.Post) (*model.Post, string) {

	//このプラグインのPostである事が前提
	if post.Type == "custom_peer-post" {
		//ハッシュタグを追加
		propValue, ok := post.Props["hashtags"]
		if ok {
			hashtags, ok := propValue.(string)
			if ok {
				post.Hashtags = hashtags
			}
		}
	}

	return post, ""
}
