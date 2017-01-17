package lbotx

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/tls"
	"encoding/base64"
	"errors"
	"fmt"
	"testing"
	"time"

	"net/http"
	"net/http/httptest"

	"strings"

	"io/ioutil"

	"github.com/line/line-bot-sdk-go/linebot"
	"github.com/stretchr/testify/assert"
)

var webhookTestRequestBody = `{
    "events": [
        {
            "replyToken": "nHuyWiB7yP5Zw52FIkcQobQuGDXCTA",
            "type": "message",
            "timestamp": 1462629479859,
            "source": {
                "type": "user",
                "userId": "u206d25c2ea6bd87c17655609a1c37cb8"
            },
            "message": {
                "id": "325708",
                "type": "text",
                "text": "Hello, world"
            }
        },
        {
            "replyToken": "nHuyWiB7yP5Zw52FIkcQobQuGDXCTA",
            "type": "message",
            "timestamp": 1462629479859,
            "source": {
                "type": "group",
                "groupId": "u206d25c2ea6bd87c17655609a1c37cb8",
                "userId": "u206d25c2ea6bd87c17655609a1c37cb8"
            },
            "message": {
                "id": "325708",
                "type": "text",
                "text": "Hello, world"
            }
        },
        {
            "replyToken": "nHuyWiB7yP5Zw52FIkcQobQuGDXCTA",
            "type": "message",
            "timestamp": 1462629479859,
            "source": {
                "type": "user",
                "userId": "u206d25c2ea6bd87c17655609a1c37cb8"
            },
            "message": {
                "id": "325708",
                "type": "image"
            }
        },
        {
            "replyToken": "nHuyWiB7yP5Zw52FIkcQobQuGDXCTA",
            "type": "message",
            "timestamp": 1462629479859,
            "source": {
                "type": "user",
                "userId": "u206d25c2ea6bd87c17655609a1c37cb8"
            },
            "message": {
                "id": "325708",
                "type": "location",
                "title": "hello",
                "address": "〒150-0002 東京都渋谷区渋谷２丁目２１−１",
                "latitude": 35.65910807942215,
                "longitude": 139.70372892916203
            }
        },
        {
            "replyToken": "nHuyWiB7yP5Zw52FIkcQobQuGDXCTA",
            "type": "message",
            "timestamp": 1462629479859,
            "source": {
                "type": "user",
                "userId": "u206d25c2ea6bd87c17655609a1c37cb8"
            },
            "message": {
                "id": "325708",
                "type": "sticker",
                "packageId": "1",
                "stickerId": "1"
            }
        },
        {
            "replyToken": "nHuyWiB7yP5Zw52FIkcQobQuGDXCTA",
            "type": "follow",
            "timestamp": 1462629479859,
            "source": {
                "type": "user",
                "userId": "u206d25c2ea6bd87c17655609a1c37cb8"
            }
        },
        {
            "type": "unfollow",
            "timestamp": 1462629479859,
            "source": {
                "type": "user",
                "userId": "u206d25c2ea6bd87c17655609a1c37cb8"
            }
        },
        {
            "replyToken": "nHuyWiB7yP5Zw52FIkcQobQuGDXCTA",
            "type": "join",
            "timestamp": 1462629479859,
            "source": {
                "type": "group",
                "groupId": "cxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
            }
        },
        {
            "type": "leave",
            "timestamp": 1462629479859,
            "source": {
                "type": "group",
                "groupId": "cxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
            }
        },
        {
            "replyToken": "nHuyWiB7yP5Zw52FIkcQobQuGDXCTA",
            "type": "postback",
            "timestamp": 1462629479859,
            "source": {
                "type": "user",
                "userId": "u206d25c2ea6bd87c17655609a1c37cb8"
            },
            "postback": {
                "data": "action=buyItem&itemId=123123&color=red"
            }
        },
        {
            "replyToken": "nHuyWiB7yP5Zw52FIkcQobQuGDXCTA",
            "type": "beacon",
            "timestamp": 1462629479859,
            "source": {
                "type": "user",
                "userId": "U012345678901234567890123456789ab"
            },
            "beacon": {
                "hwid":"374591320",
                "type":"enter"
            }
        },
		{
            "replyToken": "nHuyWiB7yP5Zw52FIkcQobQuGDXCTA",
            "type": "beacon",
            "timestamp": 1462629479859,
            "source": {
                "type": "user",
                "userId": "U012345678901234567890123456789ab"
            },
            "beacon": {
                "hwid":"374591320",
                "type":"leave"
            }
        }
    ]
}
`

func TestOnTextWith(t *testing.T) {
	bot, _ := NewBot("111", "222")

	bot.OnTextWith("Hello, {{name}}. Can you give me {{thing}}?", func(context *BotContext, text string) (bool, error) {
		assert.Equal(t, context.Params["name"], "Julian")
		assert.Equal(t, context.Params["thing"], "apple")

		context.Set("test", &struct {
			Name     string
			LastName string
		}{
			Name:     "Julian",
			LastName: "Shen",
		})

		return true, nil
	})

	event := &linebot.Event{
		ReplyToken: "nHuyWiB7yP5Zw52FIkcQobQuGDXCTA",
		Type:       linebot.EventTypeMessage,
		Timestamp:  time.Unix(1462629479859, 0),
		Source: &linebot.EventSource{
			Type:   linebot.EventSourceTypeUser,
			UserID: "u206d25c2ea6bd87c17655609a1c37cb8",
		},
		Message: &linebot.TextMessage{
			ID:   "325708",
			Text: "Hello, Julian. Can you give me apple?",
		},
	}

	context := bot.NewContext(event)
	bot.handlers[0](context)
	val := context.Get("test")
	assert.NotNil(t, val)

	context = bot.NewContext(event)
	event.Message.(*linebot.TextMessage).Text = "Test another. No one will handle this"
	bot.handlers[0](context)
	val = context.Get("test")
	assert.Nil(t, val)
}

func mockClient(server *httptest.Server) (*linebot.Client, error) {
	client, err := linebot.New(
		"test",
		"test",
		linebot.WithHTTPClient(&http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true,
				},
			},
		}),
		linebot.WithEndpointBase(server.URL),
	)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func TestServer(t *testing.T) {
	mockServer := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		uri := req.RequestURI
		fmt.Println(uri)
		if strings.Contains(uri, "content") {
			data := []byte{0, 0, 0, 0, 0, 0}
			w.WriteHeader(200)
			w.Write(data)
		} else if strings.Contains(uri, "reply") {
			body := req.Body

			data, e := ioutil.ReadAll(body)
			assert.Nil(t, e)

			fmt.Println(string(data))

			w.WriteHeader(200)
			w.Write([]byte("{}"))
		} else if strings.Contains(uri, "profile") {
			data := []byte(`{"userId":"U0047556f2e40dba2456887320ba7c76d","displayName":"BOT API","pictureUrl":"http://dl.profile.line.naver.jp/abcdefghijklmn","statusMessage":"Hello, LINE!"}`)
			w.WriteHeader(200)
			w.Write(data)
		}
	}))

	mockClient, _ := mockClient(mockServer)

	bot, e := NewBot("test", "test")
	bot.Client = mockClient //replace with mock
	assert.Nil(t, e)

	server := httptest.NewTLSServer(bot)
	defer server.Close()

	httpClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	tested := 0
	//Test handlers
	bot.OnText(func(context *BotContext, msg string) (bool, error) {
		fmt.Println(msg)
		assert.Equal(t, msg, "Hello, world")
		context.Messages.AddTextMessage("test1")
		context.Set("test", "test")
		tested = tested + 1
		return true, nil
	})

	bot.OnText(func(context *BotContext, msg string) (bool, error) {
		fmt.Println("second handler")
		assert.Equal(t, msg, "Hello, world")
		assert.Equal(t, context.Get("test"), "test")
		context.Messages.AddTextMessage(msg)
		context.Set("test", context.Get("test").(string)+"a")

		next := false
		if context.Event.Source.GroupID != "" {
			next = true
		}
		return next, nil
	})

	bot.OnText(func(context *BotContext, msg string) (bool, error) {
		//Should never run when type = user
		fmt.Println("third handler")
		assert.Equal(t, msg, "Hello, world")
		context.Messages.AddTextMessage("test1")
		context.Set("test", "test")
		//throw error
		return true, errors.New("Error on purpose")
	})

	bot.OnImage(func(context *BotContext, data []byte) (bool, error) {
		assert.Equal(t, len(data), 6)
		tested = tested + 1
		return false, nil
	})

	bot.OnVideo(func(context *BotContext, data []byte) (bool, error) {
		assert.Equal(t, len(data), 6)
		tested = tested + 1
		return false, nil
	})

	bot.OnAudio(func(context *BotContext, data []byte) (bool, error) {
		assert.Equal(t, len(data), 6)
		tested = tested + 1
		return false, nil
	})

	bot.OnLocation(func(context *BotContext, location *linebot.LocationMessage) (bool, error) {
		expected := &linebot.LocationMessage{
			"325708",
			"hello",
			"〒150-0002 東京都渋谷区渋谷２丁目２１−１",
			35.65910807942215,
			139.70372892916203,
		}

		assert.Equal(t, location, expected)
		tested = tested + 1
		return false, nil
	})

	bot.OnSticker(func(context *BotContext, sticker *linebot.StickerMessage) (bool, error) {
		expected := &linebot.StickerMessage{
			"325708",
			"1",
			"1",
		}
		assert.Equal(t, sticker, expected)
		tested = tested + 1
		return false, nil
	})

	bot.OnFollow(func(context *BotContext) (bool, error) {
		fmt.Println("follow : " + context.GetUserId())
		assert.Equal(t, context.GetUserId(), "u206d25c2ea6bd87c17655609a1c37cb8")
		user, _ := context.GetUser()
		assert.Equal(t, user.Name, "BOT API")
		tested = tested + 1
		return false, nil
	})

	bot.OnUnFollow(func(context *BotContext) (bool, error) {
		fmt.Println("unfollow : " + context.GetUserId())
		assert.Equal(t, context.GetUserId(), "u206d25c2ea6bd87c17655609a1c37cb8")
		tested = tested + 1
		return false, nil
	})

	bot.OnJoin(func(context *BotContext, joinType, id string) (bool, error) {
		fmt.Println(joinType)
		fmt.Println(id)
		assert.Equal(t, joinType, string(linebot.EventSourceTypeGroup))
		assert.Equal(t, id, "cxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx")
		user, e := context.GetUser()
		assert.NotNil(t, e)
		assert.Nil(t, user)
		tested = tested + 1
		return false, nil
	})

	bot.OnLeave(func(context *BotContext, id string) (bool, error) {
		fmt.Println("leaving " + id)
		assert.Equal(t, id, "cxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx")
		tested = tested + 1
		return false, nil
	})

	bot.OnBeaconEnter(func(context *BotContext, hwid string) (bool, error) {
		fmt.Println("beacon enter : " + hwid)
		assert.Equal(t, hwid, "374591320")
		tested = tested + 1
		return false, nil
	})

	bot.OnBeaconLeave(func(context *BotContext, hwid string) (bool, error) {
		fmt.Println("beacon leave : " + hwid)
		assert.Equal(t, hwid, "374591320")
		tested = tested + 1
		return false, nil
	})

	bot.OnError(func(context *BotContext, e error) {
		assert.EqualError(t, e, "Error on purpose")
	})

	// invalid signature
	{
		body := []byte(webhookTestRequestBody)
		req, err := http.NewRequest("POST", server.URL, bytes.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("X-Line-Signature", "invalidsignatue")
		res, err := httpClient.Do(req)

		assert.Nil(t, err)
		assert.Equal(t, res.StatusCode, 400)
	}

	// valid signature
	{
		body := []byte(webhookTestRequestBody)
		req, err := http.NewRequest("POST", server.URL, bytes.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		// generate signature
		mac := hmac.New(sha256.New, []byte("test"))
		mac.Write(body)

		req.Header.Set("X-Line-Signature", base64.StdEncoding.EncodeToString(mac.Sum(nil)))
		res, err := httpClient.Do(req)

		assert.Nil(t, err)
		assert.NotNil(t, res)
		assert.Equal(t, res.StatusCode, http.StatusOK)
	}
	assert.Equal(t, tested, 11)
}
