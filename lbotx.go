package lbotx

import (
	"errors"
	"net/http"
	"regexp"

	"reflect"

	"io/ioutil"

	"strings"

	"github.com/line/line-bot-sdk-go/linebot"
)

var (
	ErrorInvalidUserId  = errors.New("Invalid user id")
	ErrorUnknowJoinType = errors.New("Join event without room or group")
)

type BotContext struct {
	bot   *linebot.Client
	Event *linebot.Event

	Params      map[string]string
	Data        map[string]interface{}
	Messages    *MessageBank
	userProfile *UserProfile
}

type Bot struct {
	*linebot.Client

	handlers    []EventHandler
	errHandlers []ErrorHandler
}

type UserProfile struct {
	Id      string
	Name    string
	Picture string
	Status  string
}

type EventHandler func(context *BotContext) (bool, error)

type TextFilter func(context *BotContext, text string) bool

type TextMessageHandler func(context *BotContext, text string) (bool, error)
type BinaryDataHandler func(context *BotContext, data []byte) (bool, error)
type LocationHandler func(context *BotContext, location *linebot.LocationMessage) (bool, error)
type StickerHandler func(context *BotContext, sticker *linebot.StickerMessage) (bool, error)

type JoinHandler func(context *BotContext, joinType, id string) (bool, error)
type LeaveHandler func(context *BotContext, groupId string) (bool, error)
type PostbackHandler func(context *BotContext, data string) (bool, error)
type BeaconHandler func(context *BotContext, hwid string) (bool, error)

func NewBot(channelSecret, channelToken string, options ...linebot.ClientOption) (*Bot, error) {
	bot, e := linebot.New(channelSecret, channelToken, options...)
	if e != nil {
		return nil, e
	}

	return &Bot{Client: bot}, nil
}

func (b *Bot) NewContext(event *linebot.Event) *BotContext {
	context := &BotContext{
		b.Client,
		event,
		make(map[string]string),
		make(map[string]interface{}),
		&MessageBank{
			bot: b.Client,
		},
		nil,
	}

	return context
}

func (b *Bot) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	events, err := b.ParseRequest(req)

	if err != nil {
		if err == linebot.ErrInvalidSignature {
			w.WriteHeader(400)
		} else {
			w.WriteHeader(500)
		}
		return
	}

	for _, event := range events {
		context := b.NewContext(event)

		for _, handler := range b.handlers {
			next, e := handler(context)

			if e != nil {
				for _, errHandler := range b.errHandlers {
					errHandler(context, e)
				}
			}

			if !next || e != nil {
				break
			}
		}
		e := context.Messages.reply(event.ReplyToken)

		if e != nil {
			for _, errHandler := range b.errHandlers {
				errHandler(context, e)
			}
		}
	}
}

func (c *BotContext) Set(name string, value interface{}) {
	c.Data[name] = value
}

func (c *BotContext) Get(name string) interface{} {
	return c.Data[name]
}

func (c *BotContext) GetUser() (*UserProfile, error) {
	if c.userProfile != nil {
		return c.userProfile, nil
	}

	userId := c.Event.Source.UserID

	if userId == "" {
		return nil, ErrorInvalidUserId
	}

	resp, e := c.bot.GetProfile(userId).Do()
	if e != nil {
		return nil, e
	}

	c.userProfile = &UserProfile{
		Id:      resp.UserID,
		Name:    resp.DisplayName,
		Picture: resp.PictureURL,
		Status:  resp.StatusMessage,
	}
	return c.userProfile, nil
}

func (c *BotContext) GetUserId() string {
	userId := c.Event.Source.UserID

	return userId
}

func (b *Bot) OnEvent(handler EventHandler) {
	if b.handlers == nil {
		b.handlers = []EventHandler{handler}
	} else {
		b.handlers = append(b.handlers, handler)
	}
}

func (b *Bot) OnText(handler TextMessageHandler) {
	eventHandler := func(context *BotContext) (bool, error) {
		if context.Event.Type != linebot.EventTypeMessage {
			return true, nil //Not a message. Continue.
		}

		if reflect.TypeOf(context.Event.Message) != reflect.TypeOf((*linebot.TextMessage)(nil)) {
			return true, nil
		}

		msg := context.Event.Message.(*linebot.TextMessage)

		return handler(context, msg.Text)
	}

	b.OnEvent(eventHandler)
}

func (b *Bot) OnFilteredText(filter TextFilter, handler TextMessageHandler) {
	eventHandler := func(context *BotContext) (bool, error) {
		if context.Event.Type != linebot.EventTypeMessage {
			return true, nil //Not a message. Continue.
		}

		if reflect.TypeOf(context.Event.Message) != reflect.TypeOf((*linebot.TextMessage)(nil)) {
			return true, nil
		}

		msg := context.Event.Message.(*linebot.TextMessage)

		if filter(context, msg.Text) {
			return handler(context, msg.Text)
		} else {
			return true, nil
		}
	}

	b.OnEvent(eventHandler)
}

func (b *Bot) OnTextWith(template string, handler TextMessageHandler) {
	templReg, _ := regexp.Compile("\\\\{\\\\{(\\w+)\\\\}\\\\}")
	template = regexp.QuoteMeta(template)

	matches := templReg.FindAllStringSubmatch(template, -1)
	nameList := []string{}

	for _, m := range matches {
		if len(m) > 0 {
			template = strings.Replace(template, m[0], "(.+)", -1)
			nameList = append(nameList, m[1])
		}
	}

	textReg, _ := regexp.Compile(template)

	filter := func(context *BotContext, text string) bool {
		if !textReg.MatchString(text) {
			return false
		}

		results := textReg.FindStringSubmatch(text)

		context.Params = make(map[string]string)

		for i, r := range results {
			if i == 0 {
				continue
			}

			context.Params[nameList[i-1]] = r
		}

		return true
	}
	b.OnFilteredText(filter, handler)
}

func (b *Bot) OnImage(handler BinaryDataHandler) {
	eventHandler := func(context *BotContext) (bool, error) {
		if context.Event.Type != linebot.EventTypeMessage {
			return true, nil //Not a message. Continue.
		}

		if reflect.TypeOf(context.Event.Message) != reflect.TypeOf((*linebot.ImageMessage)(nil)) {
			return true, nil
		}

		msg := context.Event.Message.(*linebot.ImageMessage)

		resp, err := context.bot.GetMessageContent(msg.ID).Do()

		if err != nil {
			return false, err //Error occured. Don't continue
		}

		defer resp.Content.Close()
		data, err := ioutil.ReadAll(resp.Content)

		if err != nil {
			return false, err //Error occured. Don't continue
		}

		return handler(context, data)
	}

	b.OnEvent(eventHandler)

}

func (b *Bot) OnVideo(handler BinaryDataHandler) {
	eventHandler := func(context *BotContext) (bool, error) {
		if context.Event.Type != linebot.EventTypeMessage {
			return true, nil //Not a message. Continue.
		}

		if reflect.TypeOf(context.Event.Message) != reflect.TypeOf((*linebot.VideoMessage)(nil)) {
			return true, nil
		}

		msg := context.Event.Message.(*linebot.VideoMessage)

		resp, err := context.bot.GetMessageContent(msg.ID).Do()

		if err != nil {
			return false, err //Error occured. Don't continue
		}

		defer resp.Content.Close()
		data, err := ioutil.ReadAll(resp.Content)

		if err != nil {
			return false, err //Error occured. Don't continue
		}

		return handler(context, data)
	}

	b.OnEvent(eventHandler)

}

func (b *Bot) OnAudio(handler BinaryDataHandler) {
	eventHandler := func(context *BotContext) (bool, error) {
		if context.Event.Type != linebot.EventTypeMessage {
			return true, nil //Not a message. Continue.
		}

		if reflect.TypeOf(context.Event.Message) != reflect.TypeOf((*linebot.AudioMessage)(nil)) {
			return true, nil
		}

		msg := context.Event.Message.(*linebot.AudioMessage)

		resp, err := context.bot.GetMessageContent(msg.ID).Do()

		if err != nil {
			return false, err //Error occured. Don't continue
		}

		defer resp.Content.Close()
		data, err := ioutil.ReadAll(resp.Content)

		if err != nil {
			return false, err //Error occured. Don't continue
		}

		return handler(context, data)
	}

	b.OnEvent(eventHandler)

}

func (b *Bot) OnLocation(handler LocationHandler) {
	eventHandler := func(context *BotContext) (bool, error) {
		if context.Event.Type != linebot.EventTypeMessage {
			return true, nil //Not a message. Continue.
		}

		if reflect.TypeOf(context.Event.Message) != reflect.TypeOf((*linebot.LocationMessage)(nil)) {
			return true, nil
		}

		msg := context.Event.Message.(*linebot.LocationMessage)

		return handler(context, msg)
	}

	b.OnEvent(eventHandler)
}

func (b *Bot) OnSticker(handler StickerHandler) {
	eventHandler := func(context *BotContext) (bool, error) {
		if context.Event.Type != linebot.EventTypeMessage {
			return true, nil //Not a message. Continue.
		}

		if reflect.TypeOf(context.Event.Message) != reflect.TypeOf((*linebot.StickerMessage)(nil)) {
			return true, nil
		}

		msg := context.Event.Message.(*linebot.StickerMessage)

		return handler(context, msg)
	}

	b.OnEvent(eventHandler)
}

func (b *Bot) OnFollow(handler EventHandler) {
	eventHandler := func(context *BotContext) (bool, error) {
		if context.Event.Type != linebot.EventTypeFollow {
			return true, nil //Not a message. Continue.
		}

		return handler(context)
	}

	b.OnEvent(eventHandler)
}

func (b *Bot) OnUnFollow(handler EventHandler) {
	eventHandler := func(context *BotContext) (bool, error) {
		if context.Event.Type != linebot.EventTypeUnfollow {
			return true, nil //Not a message. Continue.
		}

		return handler(context)
	}

	b.OnEvent(eventHandler)
}

func (b *Bot) OnJoin(handler JoinHandler) {
	eventHandler := func(context *BotContext) (bool, error) {
		if context.Event.Type != linebot.EventTypeJoin {
			return true, nil //Not a message. Continue.
		}

		if context.Event.Source.Type != linebot.EventSourceTypeRoom && context.Event.Source.Type != linebot.EventSourceTypeGroup {
			return false, ErrorUnknowJoinType
		}

		id := context.Event.Source.GroupID
		if context.Event.Source.Type == linebot.EventSourceTypeRoom {
			id = context.Event.Source.RoomID
		}

		return handler(context, string(context.Event.Source.Type), id)
	}

	b.OnEvent(eventHandler)
}

func (b *Bot) OnLeave(handler LeaveHandler) {
	eventHandler := func(context *BotContext) (bool, error) {
		if context.Event.Type != linebot.EventTypeLeave {
			return true, nil //Not a message. Continue.
		}

		if context.Event.Source.Type != linebot.EventSourceTypeGroup {
			return false, ErrorUnknowJoinType
		}

		id := context.Event.Source.GroupID

		return handler(context, id)
	}

	b.OnEvent(eventHandler)
}

func (b *Bot) OnPostback(handler PostbackHandler) {
	eventHandler := func(context *BotContext) (bool, error) {
		if context.Event.Type != linebot.EventTypePostback {
			return true, nil //Not a message. Continue.
		}

		return handler(context, context.Event.Postback.Data)
	}

	b.OnEvent(eventHandler)
}

func (b *Bot) OnBeaconEnter(handler BeaconHandler) {
	eventHandler := func(context *BotContext) (bool, error) {
		if context.Event.Type != linebot.EventTypeBeacon {
			return true, nil //Not a message. Continue.
		}

		if context.Event.Beacon.Type != linebot.BeaconEventTypeEnter {
			return true, nil
		}

		return handler(context, context.Event.Beacon.Hwid)
	}

	b.OnEvent(eventHandler)
}

func (b *Bot) OnBeaconLeave(handler BeaconHandler) {
	eventHandler := func(context *BotContext) (bool, error) {
		if context.Event.Type != linebot.EventTypeBeacon {
			return true, nil //Not a message. Continue.
		}

		if context.Event.Beacon.Type != linebot.BeaconEventTypeLeave {
			return true, nil
		}

		return handler(context, context.Event.Beacon.Hwid)
	}

	b.OnEvent(eventHandler)
}

type ErrorHandler func(context *BotContext, err error)

func (b *Bot) OnError(handler ErrorHandler) {
	if b.errHandlers == nil {
		b.errHandlers = []ErrorHandler{handler}
	} else {
		b.errHandlers = append(b.errHandlers, handler)
	}
}
