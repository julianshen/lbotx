package lbotx

import (
	"fmt"
	"testing"

	"github.com/line/line-bot-sdk-go/linebot"
	"github.com/stretchr/testify/assert"
)

func TestConfirmMessages(t *testing.T) {
	b := NewConfirmMessageBuilder()
	b.WithText("test")
	b.WithMessageAction("test", "test")
	b.WithPostbackAction("test2", "I'm data", "test2")

	message, e := b.Build("test")

	if e != nil {
		fmt.Println(e)
	}

	action1 := linebot.NewMessageTemplateAction("test", "test")
	action2 := linebot.NewPostbackTemplateAction("test2", "I'm data", "test2")
	message2 := linebot.NewTemplateMessage("test", linebot.NewConfirmTemplate("test", action1, action2))

	assert.Equal(t, message, message2)

	b.WithURIAction("google", "http://www.google.com")
	message, e = b.Build("AltText")
	assert.NotNil(t, e)
	assert.Equal(t, e, ErrorTooManyActions)
}

func TestButtonMessages(t *testing.T) {
	b := NewButtonMessageBuilderWith("https://upload.wikimedia.org/wikipedia/commons/c/c4/Leaky_bucket_analogy.JPG", "Leaky Bucket", "For test")
	b.WithMessageAction("test", "test1")
	b.WithURIAction("test2", "http://www.google.com")
	b.WithPostbackAction("test3", "test3data", "test3")
	message, _ := b.Build("AltText")

	action1 := linebot.NewMessageTemplateAction("test", "test1")
	action2 := linebot.NewURITemplateAction("test2", "http://www.google.com")
	action3 := linebot.NewPostbackTemplateAction("test3", "test3data", "test3")
	botTempl := linebot.NewButtonsTemplate("https://upload.wikimedia.org/wikipedia/commons/c/c4/Leaky_bucket_analogy.JPG", "Leaky Bucket", "For test", action1, action2, action3)
	message2 := linebot.NewTemplateMessage("AltText", botTempl)
	assert.Equal(t, message, message2)

	for i := 0; i < 3; i++ {
		b.WithMessageAction("test", "test1")
	}
	message, e := b.Build("too many actions")
	assert.NotNil(t, e)
	assert.Equal(t, e, ErrorTooManyActions)

	b = NewButtonMessageBuilderWith("http://upload.wikimedia.org/wikipedia/commons/c/c4/Leaky_bucket_analogy.JPG", "Leaky Bucket", "For test")
	b.WithMessageAction("test", "test1")
	b.WithURIAction("test2", "http://www.google.com")

	message, e = b.Build("invalid image url")
	assert.NotNil(t, e)
	assert.Equal(t, e, ErrorInvalidUrl)
}

func TestCarouselMessage(t *testing.T) {
	b := NewCarouselMessageBuilder()

	for i := 0; i < 5; i++ {
		col := b.AddColumn()
		col.WithImage("http://upload.wikimedia.org/wikipedia/commons/c/c4/Leaky_bucket_analogy.JPG")
		col.WithText("test")
		col.WithTitle("test")
		col.WithMessageAction("Message", "test")
		col.WithURIAction("Google", "http://www.google.com")
	}

	message, _ := b.Build("altText")

	columns := []*linebot.CarouselColumn{}
	for i := 0; i < 5; i++ {
		action1 := linebot.NewMessageTemplateAction("Message", "test")
		action2 := linebot.NewURITemplateAction("Google", "http://www.google.com")

		col := linebot.NewCarouselColumn("http://upload.wikimedia.org/wikipedia/commons/c/c4/Leaky_bucket_analogy.JPG", "test", "test", action1, action2)
		columns = append(columns, col)
	}
	carTempl := linebot.NewCarouselTemplate(columns...)

	message2 := linebot.NewTemplateMessage("altText", carTempl)
	assert.Equal(t, message, message2)

	b.AddColumn()
	message, e := b.Build("Too many columns")
	assert.NotNil(t, e)
	assert.Equal(t, e, ErrorTooManyColumn)

	b = NewCarouselMessageBuilder()

	col := b.AddColumn()
	col.WithMessageAction("Message", "test")
	col.WithURIAction("Google", "http://www.google.com")

	message, e = b.Build("altText")
	assert.NotNil(t, e)
	assert.Equal(t, e, ErrorMissingParam)

	b = NewCarouselMessageBuilder()

	col = b.AddColumn()
	col.WithText("1111")
	col.WithMessageAction("Message", "test")
	col.WithURIAction("Google", "http://www.google.com")

	col = b.AddColumn()
	col.WithText("1111")
	col.WithMessageAction("Message", "test")
	message, e = b.Build("altText")
	assert.NotNil(t, e)
	assert.Equal(t, e, ErrorActionNumNotConsistent)
}

func TestCarouselGenerator(t *testing.T) {
	b := NewCarouselMessageBuilder()
	g := b.GetColumnGenerator()
	g.WithImage("http://myhost.com/image/{{.Index}}")
	g.WithText("Hi {{.Name}}")
	g.WithMessageAction("Press me", "I'm {{.Name}}")

	data := []struct {
		Index int
		Name  string
	}{
		{1, "John"},
		{2, "Mary"},
		{3, "Julian"},
	}

	b.GenerateColumnsWith(func(data []struct {
		Index int
		Name  string
	}) []interface{} {
		ret := make([]interface{}, len(data))
		for i, d := range data {
			ret[i] = d
		}
		return ret
	}(data)...)

	message, _ := b.Build("altText")

	columns := []*linebot.CarouselColumn{}
	for _, d := range data {
		imageUrl := fmt.Sprintf("http://myhost.com/image/%v", d.Index)
		text := fmt.Sprintf("Hi %v", d.Name)
		msg := fmt.Sprintf("I'm %v", d.Name)

		msgAction := linebot.NewMessageTemplateAction("Press me", msg)
		columns = append(columns, linebot.NewCarouselColumn(imageUrl, "", text, msgAction))
	}
	carTempl := linebot.NewCarouselTemplate(columns...)
	message2 := linebot.NewTemplateMessage("altText", carTempl)

	assert.Equal(t, message, message2)
}

func TestImageMap(t *testing.T) {
	b := NewImageMapBuilder()
	b.BaseUrl = "https://upload.wikimedia.org/wikipedia/commons/c/c4/Leaky_bucket_analogy.JPG"
	b.AltText = "altText"
	b.Width = 400
	b.Height = 400
	b.WithMessageAction("test", 10, 10, 30, 30)
	b.WithURIAction("https://www.google.com", 30, 30, 40, 40)

	message, _ := b.Build()

	b = NewImageMapBuilderWith("https://upload.wikimedia.org/wikipedia/commons/c/c4/Leaky_bucket_analogy.JPG", "altText", 400, 400)
	b.WithMessageAction("test", 10, 10, 30, 30)
	b.WithURIAction("https://www.google.com", 30, 30, 40, 40)

	message2, _ := b.Build()

	action1 := linebot.NewMessageImagemapAction("test", linebot.ImagemapArea{10, 10, 30, 30})
	action2 := linebot.NewURIImagemapAction("https://www.google.com", linebot.ImagemapArea{30, 30, 40, 40})
	message3 := linebot.NewImagemapMessage("https://upload.wikimedia.org/wikipedia/commons/c/c4/Leaky_bucket_analogy.JPG", "altText", linebot.ImagemapBaseSize{400, 400}, action1, action2)

	assert.Equal(t, message, message2)
	assert.Equal(t, message, message3)
}
