# lbotx - A line bot go sdk extension
==============

lbotx is a wrapper/extension to official Go line bot SDK: https://github.com/line/line-bot-sdk-go

Why lbotx? (NOTE: Document is still under construction)

## Using chaining handlers instead of nested if and switch-cases

Here is the original example of a linebot webhook:

```go 
func main() {
	bot, err := linebot.New(
		os.Getenv("CHANNEL_SECRET"),
		os.Getenv("CHANNEL_TOKEN"),
	)
	if err != nil {
		log.Fatal(err)
	}

	// Setup HTTP Server for receiving requests from LINE platform
	http.HandleFunc("/callback", func(w http.ResponseWriter, req *http.Request) {
		events, err := bot.ParseRequest(req)
		if err != nil {
			if err == linebot.ErrInvalidSignature {
				w.WriteHeader(400)
			} else {
				w.WriteHeader(500)
			}
			return
		}
		for _, event := range events {
			if event.Type == linebot.EventTypeMessage {
				switch message := event.Message.(type) {
				case *linebot.TextMessage:
					if _, err = bot.ReplyMessage(event.ReplyToken, linebot.NewTextMessage(message.Text)).Do(); err != nil {
						log.Print(err)
					}
				}
			}
		}
	})
	// This is just sample code.
	// For actual use, you must support HTTPS by using `ListenAndServeTLS`, a reverse proxy or something else.
	if err := http.ListenAndServe(":"+os.Getenv("PORT"), nil); err != nil {
		log.Fatal(err)
	}
}
```

By using chaining event handlers, your codes might like this:

```go
bot.OnText(func(context *lbotx.BotContext, msg string) (bool, error) {
	fmt.Println(msg)
	context.Messages.AddTextMessage("test1")
	context.Set("test", "test")
	tested = tested + 1
	return true, nil
})

bot.OnText(func(context *BotContext, msg string) (bool, error) {
	//Should never run when type = user
	fmt.Println("second handler")
	context.Messages.AddTextMessage("test1")
	context.Set("test", "test")
	//throw error
	return true, errors.New("Error on purpose")
})

bot.OnVideo(func(context *lbotx.BotContext, data []byte) (bool, error) {
	...
	return false, nil
})

bot.OnLocation(func(context *lbotx.BotContext, location *linebot.LocationMessage) (bool, error) {
	...
	return false, nil
})

bot.OnFollow(func(context *lbotx.BotContext) (bool, error) {
	fmt.Println("follow : " + context.GetUserId())
	user, _ := context.GetUser()
	...
	return false, nil
})
```

You don't have to handle nested if and switch cases by your own

## Utils for message

Here is one example of carousel Messages:

```go
d := NewCarouselMessageBuilder()
for i := 0; i < 5; i++ {
	col := d.AddColumn()
	col.WithImage("http://upload.wikimedia.org/wikipedia/commons/c/c4/Leaky_bucket_analogy.JPG")
	col.WithText("test")
	col.WithTitle("test")
	col.WithMessageAction("Message", "test")
	col.WithURIAction("Google", "http://www.google.com")
}

message, _ = d.Build("altText")
```