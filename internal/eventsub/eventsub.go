package eventsub

import (
	"encoding/json"
	"errors"

	"github.com/LinneB/twitchwh"
)

// DefaultClient is the default client for the Eventsub API.
// We create this just before initialising the exporter, it is global to avoid
// too high complexity within the collector package.
// todo: this is a bit of a hack, we should find a better way to handle this.
var DefaultClient *twitchwh.Client

type ChannelChatMessageEvent struct {
	Subscription Subscription `json:"subscription"`
	Event        Event        `json:"event"`
}

type Event struct {
	BroadcasterUserID           string      `json:"broadcaster_user_id"`
	BroadcasterUserLogin        string      `json:"broadcaster_user_login"`
	BroadcasterUserName         string      `json:"broadcaster_user_name"`
	ChatterUserID               string      `json:"chatter_user_id"`
	ChatterUserLogin            string      `json:"chatter_user_login"`
	ChatterUserName             string      `json:"chatter_user_name"`
	MessageID                   string      `json:"message_id"`
	Message                     Message     `json:"message"`
	Color                       string      `json:"color"`
	Badges                      []Badge     `json:"badges"`
	MessageType                 string      `json:"message_type"`
	Cheer                       interface{} `json:"cheer"`
	Reply                       interface{} `json:"reply"`
	ChannelPointsCustomRewardID interface{} `json:"channel_points_custom_reward_id"`
}

type Badge struct {
	SetID string `json:"set_id"`
	ID    string `json:"id"`
	Info  string `json:"info"`
}

type Message struct {
	Text      string     `json:"text"`
	Fragments []Fragment `json:"fragments"`
}

type Fragment struct {
	Type      string      `json:"type"`
	Text      string      `json:"text"`
	Cheermote interface{} `json:"cheermote"`
	Emote     interface{} `json:"emote"`
	Mention   interface{} `json:"mention"`
}

type Subscription struct {
	ID        string    `json:"id"`
	Status    string    `json:"status"`
	Type      string    `json:"type"`
	Version   string    `json:"version"`
	Condition Condition `json:"condition"`
	Transport Transport `json:"transport"`
	CreatedAt string    `json:"created_at"`
	Cost      int64     `json:"cost"`
}

type Condition struct {
	BroadcasterUserID string `json:"broadcaster_user_id"`
	UserID            string `json:"user_id"`
}

type Transport struct {
	Method   string `json:"method"`
	Callback string `json:"callback"`
}

var ErrEventsubDefaultClientNotSet = errors.New("eventsub default client not set")

func New(clientID, clientSecret, webhookURL, webhookSecret string) (*twitchwh.Client, error) {
	cl, err := twitchwh.New(twitchwh.ClientConfig{
		ClientID:      clientID,
		ClientSecret:  clientSecret,
		WebhookURL:    webhookURL,
		WebhookSecret: webhookSecret,
	})

	if err != nil {
		return nil, err
	}

	DefaultClient = cl
	return cl, nil
}

func On(event string, callback func(eventRaw json.RawMessage)) error {
	// juuust in case
	if DefaultClient == nil {
		return ErrEventsubDefaultClientNotSet
	}

	DefaultClient.On(event, callback)
	return nil
}
