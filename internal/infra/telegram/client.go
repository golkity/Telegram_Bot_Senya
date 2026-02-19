package telegram

import (
	"fmt"
	"io"
	"net/http"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Client struct {
	bot *tgbotapi.BotAPI
}

func New(token string) (*Client, error) {
	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, fmt.Errorf("telegram init failed: %w", err)
	}
	bot.Debug = true
	return &Client{bot: bot}, nil
}

func (c *Client) GetBot() *tgbotapi.BotAPI {
	return c.bot
}

func (c *Client) Notify(userID int64, text string) error {
	return c.SendMessage(userID, text)
}

func (c *Client) SendMessage(chatID int64, text string) error {
	msg := tgbotapi.NewMessage(chatID, text)
	_, err := c.bot.Send(msg)
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}
	return nil
}

func (c *Client) GetFileContent(fileID string) (io.ReadCloser, error) {
	fileInfo, err := c.bot.GetFile(tgbotapi.FileConfig{FileID: fileID})
	if err != nil {
		return nil, fmt.Errorf("failed to get file info: %w", err)
	}
	link := fileInfo.Link(c.bot.Token)
	resp, err := http.Get(link)
	if err != nil {
		return nil, fmt.Errorf("http download request failed: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("bad status: %s", resp.Status)
	}
	return resp.Body, nil
}

func (c *Client) SendFile(chatID int64, fileData []byte, fileName string, caption string) error {
	fileBytes := tgbotapi.FileBytes{Name: fileName, Bytes: fileData}
	msg := tgbotapi.NewDocument(chatID, fileBytes)
	msg.Caption = caption
	_, err := c.bot.Send(msg)
	return err
}

func (c *Client) StartPolling() tgbotapi.UpdatesChannel {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	return c.bot.GetUpdatesChan(u)
}

func (c *Client) StopPolling() {
	c.bot.StopReceivingUpdates()
}
