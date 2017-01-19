package lbotx

import (
	"errors"
	"text/template"

	"net/url"

	"bytes"

	"github.com/line/line-bot-sdk-go/linebot"
)

type MessageBank struct {
	messages []linebot.Message
	bot      *linebot.Client
}

var (
	ErrorTooManyMessages        = errors.New("Can only send 5 messages at a time")
	ErrorNoAction               = errors.New("There is no action for this message")
	ErrorInvalidMapSize         = errors.New("Image map width/height should not be 0")
	ErrorInvalidUrl             = errors.New("Invalid Url. Url scheme should be https.")
	ErrorMissingParam           = errors.New("Missing mandatory parameter")
	ErrorTextExceedLimit        = errors.New("Text length is exceed limitation")
	ErrorTooManyActions         = errors.New("Too many actions")
	ErrorTooManyColumn          = errors.New("Too many columns")
	ErrorActionNumNotConsistent = errors.New("Number of actions is not consistent in each column")
	ErrorNoColumnTemplate       = errors.New("You did not setup column template first")
)

func (mb *MessageBank) AddMessage(m linebot.Message) error {
	if mb != nil {
		if len(mb.messages) > 5 {
			return ErrorTooManyMessages
		}
		mb.messages = append(mb.messages, m)
	} else {
		mb.messages = []linebot.Message{m}
	}
	return nil
}

func (mb *MessageBank) AddTextMessage(msg string) error {
	return mb.AddMessage(linebot.NewTextMessage(msg))
}

func (mb *MessageBank) AddStickerMessage(packageId, stickerId string) error {
	return mb.AddMessage(linebot.NewStickerMessage(packageId, stickerId))
}

func (mb *MessageBank) AddLocationMessage(title, address string, latitude, longitude float64) error {
	return mb.AddMessage(linebot.NewLocationMessage(title, address, latitude, longitude))
}

func (mb *MessageBank) AddAudioMessage(contentUrl string, duration int) error {
	return mb.AddMessage(linebot.NewAudioMessage(contentUrl, duration))
}

func (mb *MessageBank) AddVideoMessage(contentUrl, previewUrl string) error {
	return mb.AddMessage(linebot.NewVideoMessage(contentUrl, previewUrl))
}

func (mb *MessageBank) AddImageMessage(contentUrl, previewUrl string) error {
	return mb.AddMessage(linebot.NewImageMessage(contentUrl, previewUrl))
}

func (mb *MessageBank) Len() int {
	if mb.messages == nil {
		return 0
	}

	return len(mb.messages)
}

func (mb *MessageBank) reply(reply_token string) error {
	if len(mb.messages) > 0 {
		if _, err := mb.bot.ReplyMessage(reply_token, mb.messages...).Do(); err != nil {
			return err
		}
	}
	return nil
}

func (mb *MessageBank) push(to string) error {
	if _, err := mb.bot.PushMessage(to, mb.messages...).Do(); err != nil {
		return err
	}
	return nil
}

type PostMan struct {
	MessageBank
}

func (pm *PostMan) SendImmediately(tos ...string) ([]string, error) {
	success := []string{}
	var err error = nil

	for _, to := range tos {
		err = pm.push(to)

		if err != nil {
			break
		} else {
			success = append(success, to)
		}
	}

	if err == nil {
		//flush all messages
		pm.messages = []linebot.Message{}
	}
	return success, err
}

func validateActionTexts(iaction interface{}) error {
	switch action := iaction.(type) {
	case *linebot.URITemplateAction:
		if len([]rune(action.Label)) > 20 {
			return ErrorTextExceedLimit
		}
	case *linebot.MessageTemplateAction:
		if len([]rune(action.Label)) > 20 {
			return ErrorTextExceedLimit
		}
		if len([]rune(action.Text)) > 300 {
			return ErrorTextExceedLimit
		}
	case *linebot.PostbackTemplateAction:
		if len([]rune(action.Label)) > 20 {
			return ErrorTextExceedLimit
		}
		if len([]rune(action.Text)) > 300 {
			return ErrorTextExceedLimit
		}
		if len([]rune(action.Data)) > 300 {
			return ErrorTextExceedLimit
		}
	case *linebot.MessageImagemapAction:
		if len([]rune(action.Text)) > 400 {
			return ErrorTextExceedLimit
		}
	case *linebot.URIImagemapAction:
		if len([]rune(action.LinkURL)) > 1000 {
			return ErrorTextExceedLimit
		}
	}

	return nil
}

type ImageMapBuilder struct {
	actions []linebot.ImagemapAction
	BaseUrl string
	AltText string
	Width   int
	Height  int
}

func (ib *ImageMapBuilder) addAction(action linebot.ImagemapAction) {
	if ib.actions == nil {
		ib.actions = []linebot.ImagemapAction{action}
	} else {
		ib.actions = append(ib.actions, action)
	}
}

func NewImageMapBuilder() *ImageMapBuilder {
	return &ImageMapBuilder{}
}

func NewImageMapBuilderWith(baseUrl, altText string, width, height int) *ImageMapBuilder {
	return &ImageMapBuilder{
		BaseUrl: baseUrl,
		AltText: altText,
		Width:   width,
		Height:  height,
	}
}

func (ib *ImageMapBuilder) WithMessageAction(text string, x, y, width, height int) *ImageMapBuilder {
	area := linebot.ImagemapArea{x, y, width, height}
	ib.addAction(linebot.NewMessageImagemapAction(text, area))
	return ib
}

func (ib *ImageMapBuilder) WithURIAction(linkURL string, x, y, width, height int) *ImageMapBuilder {
	area := linebot.ImagemapArea{x, y, width, height}
	ib.addAction(linebot.NewURIImagemapAction(linkURL, area))
	return ib
}

func (ib *ImageMapBuilder) Build() (linebot.Message, error) {
	if ib.actions == nil || len(ib.actions) == 0 {
		return nil, ErrorNoAction
	}

	if ib.Width == 0 || ib.Height == 0 {
		return nil, ErrorInvalidMapSize
	}

	for _, action := range ib.actions {
		var area linebot.ImagemapArea
		switch act := action.(type) {
		case *linebot.MessageImagemapAction:
			area = act.Area
		case *linebot.URIImagemapAction:
			area = act.Area
		}

		if area.Width == 0 || area.Height == 0 {
			return nil, ErrorInvalidMapSize
		}

		if e := validateActionTexts(action); e != nil {
			return nil, e
		}
	}

	return linebot.NewImagemapMessage(ib.BaseUrl, ib.AltText, linebot.ImagemapBaseSize{ib.Width, ib.Height}, ib.actions...), nil
}

type iactionable interface {
	WithMessageAction(label, text string) iactionable
	WithURIAction(label, uri string) iactionable
	WithPostbackAction(label, data, text string) iactionable
}

type actionable struct {
	actions []linebot.TemplateAction
}

func (a *actionable) addAction(action linebot.TemplateAction) {
	if a.actions == nil {
		a.actions = []linebot.TemplateAction{action}
	} else {
		a.actions = append(a.actions, action)
	}
}

func (a *actionable) WithMessageAction(label, text string) iactionable {
	a.addAction(linebot.NewMessageTemplateAction(label, text))
	return a
}

func (a *actionable) WithURIAction(label, uri string) iactionable {
	a.addAction(linebot.NewURITemplateAction(label, uri))
	return a
}

func (a *actionable) WithPostbackAction(label, data, text string) iactionable {
	a.addAction(linebot.NewPostbackTemplateAction(label, data, text))
	return a
}

type ButtonMessageBuilder struct {
	actionable
	thumbnailImageUrl string
	title             string
	text              string
}

func NewButtonMessageBuilder() *ButtonMessageBuilder {
	return &ButtonMessageBuilder{}
}

func NewButtonMessageBuilderWith(thumbnailImageUrl, title, text string) *ButtonMessageBuilder {
	b := NewButtonMessageBuilder()
	b.thumbnailImageUrl = thumbnailImageUrl
	b.title = title
	b.text = text

	return b
}

func (b *ButtonMessageBuilder) WithImage(thumbnailImageUrl string) *ButtonMessageBuilder {
	b.thumbnailImageUrl = thumbnailImageUrl
	return b
}

func (b *ButtonMessageBuilder) WithTitle(title string) *ButtonMessageBuilder {
	b.title = title
	return b
}

func (b *ButtonMessageBuilder) WithText(text string) *ButtonMessageBuilder {
	b.text = text
	return b
}

func (b *ButtonMessageBuilder) Build(altMsg string) (linebot.Message, error) {
	if b.text == "" {
		return nil, ErrorMissingParam
	}

	if b.thumbnailImageUrl != "" {
		if parsedUrl, e := url.Parse(b.thumbnailImageUrl); e != nil {
			return nil, e
		} else {
			if parsedUrl.Scheme != "https" {
				return nil, ErrorInvalidUrl
			}
		}
	}

	if len([]rune(b.title)) > 40 {
		return nil, ErrorTextExceedLimit
	}

	if b.title == "" || b.thumbnailImageUrl == "" {
		if len([]rune(b.text)) > 160 {
			return nil, ErrorTextExceedLimit
		}
	} else {
		if len([]rune(b.text)) > 60 {
			return nil, ErrorTextExceedLimit
		}
	}

	if len(b.actions) > 4 {
		return nil, ErrorTooManyActions
	}

	buttonTemplate := linebot.NewButtonsTemplate(b.thumbnailImageUrl, b.title, b.text, b.actions...)
	return linebot.NewTemplateMessage(altMsg, buttonTemplate), nil
}

type ConfirmMessageBuilder struct {
	actionable
	text string
}

func NewConfirmMessageBuilder() *ConfirmMessageBuilder {
	return &ConfirmMessageBuilder{}
}

func NewConfirmMessageBuilderWith(text string) *ConfirmMessageBuilder {
	b := NewConfirmMessageBuilder()
	b.text = text

	return b
}

func (b *ConfirmMessageBuilder) WithText(text string) *ConfirmMessageBuilder {
	b.text = text
	return b
}

func (b *ConfirmMessageBuilder) Build(altMsg string) (linebot.Message, error) {
	if b.text == "" {
		return nil, ErrorMissingParam
	}

	if len([]rune(b.text)) > 240 {
		return nil, ErrorTextExceedLimit
	}

	if len(b.actions) > 2 {
		return nil, ErrorTooManyActions
	}

	confirmTemplate := &linebot.ConfirmTemplate{
		Text:    b.text,
		Actions: b.actions,
	}
	return linebot.NewTemplateMessage(altMsg, confirmTemplate), nil
}

type CarouselColumn struct {
	*linebot.CarouselColumn
	actionable
}

type ActionTempate struct {
	actionType        linebot.TemplateActionType
	labelTemplate     *template.Template
	textOrUrlTemplate *template.Template
	dataTemplate      *template.Template
}

type ColumnTemplate struct {
	imageUrlTemplate *template.Template
	titleTemplate    *template.Template
	textTemplate     *template.Template

	actionsTemplates []*ActionTempate
}

func newColumnTemplate() *ColumnTemplate {
	return &ColumnTemplate{
		actionsTemplates: []*ActionTempate{},
	}
}

func (ct *ColumnTemplate) WithImage(imageTempl string) error {
	templ, e := template.New("colImage").Parse(imageTempl)

	if e != nil {
		return e
	}

	ct.imageUrlTemplate = templ
	return nil
}

func (ct *ColumnTemplate) WithTitle(titleTempl string) error {
	templ, e := template.New("colTitle").Parse(titleTempl)

	if e != nil {
		return e
	}

	ct.titleTemplate = templ
	return nil
}

func (ct *ColumnTemplate) WithText(textTempl string) error {
	templ, e := template.New("colText").Parse(textTempl)

	if e != nil {
		return e
	}

	ct.textTemplate = templ
	return nil
}

func (ct *ColumnTemplate) WithMessageAction(label, text string) iactionable {
	labelTempl, _ := template.New("label").Parse(label)
	textTempl, _ := template.New("text").Parse(text)

	actionTemplate := &ActionTempate{
		actionType:        linebot.TemplateActionTypeMessage,
		labelTemplate:     labelTempl,
		textOrUrlTemplate: textTempl,
	}
	ct.actionsTemplates = append(ct.actionsTemplates, actionTemplate)
	return ct
}

func (ct *ColumnTemplate) WithURIAction(label, uri string) iactionable {
	labelTempl, _ := template.New("label").Parse(label)
	uriTempl, _ := template.New("uri").Parse(uri)

	actionTemplate := &ActionTempate{
		actionType:        linebot.TemplateActionTypeURI,
		labelTemplate:     labelTempl,
		textOrUrlTemplate: uriTempl,
	}
	ct.actionsTemplates = append(ct.actionsTemplates, actionTemplate)
	return ct
}

func (ct *ColumnTemplate) WithPostbackAction(label, data, text string) iactionable {
	labelTempl, _ := template.New("label").Parse(label)
	textTempl, _ := template.New("text").Parse(label)
	dataTempl, _ := template.New("data").Parse(data)

	actionTemplate := &ActionTempate{
		actionType:        linebot.TemplateActionTypeMessage,
		labelTemplate:     labelTempl,
		textOrUrlTemplate: textTempl,
		dataTemplate:      dataTempl,
	}
	ct.actionsTemplates = append(ct.actionsTemplates, actionTemplate)
	return ct
}

func (ct *ColumnTemplate) generate(data []interface{}) ([]*CarouselColumn, error) {
	if data == nil {
		return nil, ErrorMissingParam
	}

	columns := []*CarouselColumn{}
	for _, d := range data {
		col := &CarouselColumn{CarouselColumn: &linebot.CarouselColumn{}}
		if ct.imageUrlTemplate != nil {
			buf := bytes.NewBufferString("")
			ct.imageUrlTemplate.Execute(buf, d)
			col.ThumbnailImageURL = buf.String()
		}

		if ct.textTemplate != nil {
			buf := bytes.NewBufferString("")
			ct.textTemplate.Execute(buf, d)
			col.Text = buf.String()
		}

		if ct.titleTemplate != nil {
			buf := bytes.NewBufferString("")
			ct.titleTemplate.Execute(buf, d)
			col.Title = buf.String()
		}

		actions := []linebot.TemplateAction{}
		for _, actionTempl := range ct.actionsTemplates {
			label := ""
			textOrUrl := ""
			data := ""

			if actionTempl.labelTemplate != nil {
				buf := bytes.NewBufferString("")
				actionTempl.labelTemplate.Execute(buf, d)
				label = buf.String()
			}

			if actionTempl.textOrUrlTemplate != nil {
				buf := bytes.NewBufferString("")
				actionTempl.textOrUrlTemplate.Execute(buf, d)
				textOrUrl = buf.String()
			}

			if actionTempl.dataTemplate != nil {
				buf := bytes.NewBufferString("")
				actionTempl.dataTemplate.Execute(buf, d)
				data = buf.String()
			}

			switch actionTempl.actionType {
			case linebot.TemplateActionTypeMessage:
				actions = append(actions, linebot.NewMessageTemplateAction(label, textOrUrl))
			case linebot.TemplateActionTypeURI:
				actions = append(actions, linebot.NewURITemplateAction(label, textOrUrl))
			case linebot.TemplateActionTypePostback:
				actions = append(actions, linebot.NewPostbackTemplateAction(label, data, textOrUrl))
			}

			col.actions = actions
			columns = append(columns, col)
		}
	}
	return columns, nil
}

func (c *CarouselColumn) WithImage(imageUrl string) *CarouselColumn {
	c.CarouselColumn.ThumbnailImageURL = imageUrl
	return c
}

func (c *CarouselColumn) WithTitle(title string) *CarouselColumn {
	c.CarouselColumn.Title = title
	return c
}

func (c *CarouselColumn) WithText(text string) *CarouselColumn {
	c.CarouselColumn.Text = text
	return c
}

type CarouselMessageBuilder struct {
	columns         []*CarouselColumn
	columnGenerator *ColumnTemplate
}

func NewCarouselMessageBuilder() *CarouselMessageBuilder {
	return &CarouselMessageBuilder{}
}

func (cm *CarouselMessageBuilder) AddColumn() *CarouselColumn {
	column := &CarouselColumn{CarouselColumn: &linebot.CarouselColumn{}}
	if cm.columns == nil {
		cm.columns = []*CarouselColumn{column}
	} else {
		cm.columns = append(cm.columns, column)
	}

	return column
}

func (cm *CarouselMessageBuilder) GetColumnGenerator() *ColumnTemplate {
	cm.columnGenerator = newColumnTemplate()
	return cm.columnGenerator
}

func (cm *CarouselMessageBuilder) GenerateColumnsWith(data ...interface{}) error {
	if cm.columnGenerator == nil {
		return ErrorNoColumnTemplate
	}

	if columns, e := cm.columnGenerator.generate(data); e != nil {
		return e
	} else {
		cm.columns = append(cm.columns, columns...)
	}
	return nil
}

func (cm *CarouselMessageBuilder) Build(altMsg string) (linebot.Message, error) {
	if len(cm.columns) > 5 {
		return nil, ErrorTooManyColumn
	}

	columns := []*linebot.CarouselColumn{}
	actionCount := -1
	for _, c := range cm.columns {
		column := c.CarouselColumn
		c.CarouselColumn.Actions = c.actions

		if len(c.CarouselColumn.Actions) > 3 {
			return nil, ErrorTooManyActions
		}

		if actionCount == -1 {
			actionCount = len(c.CarouselColumn.Actions)
		} else if actionCount != len(c.CarouselColumn.Actions) {
			return nil, ErrorActionNumNotConsistent
		}

		//Validate texts in actions
		for _, action := range c.CarouselColumn.Actions {
			if e := validateActionTexts(action); e != nil {
				return nil, e
			}
		}

		//Check texts in columns
		if len([]rune(column.ThumbnailImageURL)) > 1000 {
			return nil, ErrorTextExceedLimit
		}

		if len([]rune(column.Title)) > 40 {
			return nil, ErrorTextExceedLimit
		}

		if column.Text == "" {
			return nil, ErrorMissingParam
		}

		textMaxLen := 60
		if column.ThumbnailImageURL == "" || column.Title == "" {
			textMaxLen = 120
		}

		if len([]rune(column.Text)) > textMaxLen {
			return nil, ErrorTextExceedLimit
		}

		columns = append(columns, column)
	}

	templ := linebot.NewCarouselTemplate(columns...)
	msg := linebot.NewTemplateMessage(altMsg, templ)

	return msg, nil
}
