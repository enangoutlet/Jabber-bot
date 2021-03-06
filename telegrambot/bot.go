package telegrambot

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strconv"
)

const (
	BASE_URL = "https://api.telegram.org/bot"
)

type Sender struct {
	Id int `json:"id"`
}

type User struct {
	Sender
	FirstName string `json:"first_name"`
	Username  string `json:"username"`
}

type MessageReply struct {
	MessageId int `json:"message_id"`
}

type Message struct {
	Date        int          `json:"date"`
	Text        string       `json:"text"`
	MessageId   int          `json:"message_id"`
	From        User         `json:"from"`
	Chat        Sender       `json:"chat"`
	ReplyTo     MessageReply `json:"reply_to_message"`
	ForwardDate int          `json:"forward_date"`
}

type Update struct {
	Id  int     `json:"update_id"`
	Msg Message `json:"message"`
}

type Bot struct {
	Token    string
	OnUpdate func(update *Update)
}

type BotResult map[string]interface{}
type ServerResponse struct {
	ok          bool
	description string
	result      BotResult
}

func (bot *Bot) Hook(w http.ResponseWriter, r *http.Request) {
	defer func() {
		if err := recover(); err != nil {
			log.Println("Something is wrong in hook", err)
		}

		//telegram server must not know about our problems
		fmt.Fprintf(w, "OK\n")
	}()
	var update Update
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Println("Webhook body read error")
		return
	}

	err = json.Unmarshal(body, &update)
	if err != nil {
		log.Println("Webhook json parse error")
		return
	}

	if bot.OnUpdate != nil {
		bot.OnUpdate(&update)
	}
}

func (bot *Bot) Command(cmd string,
	params *url.Values) *ServerResponse {

	var result map[string]interface{}
	var err error
	var resp *http.Response

	//construct url
	cmd_url := BASE_URL + bot.Token + "/" + cmd
	if params == nil {
		resp, err = http.Get(cmd_url)
	} else {
		resp, err = http.PostForm(cmd_url, *params)
	}
	if err != nil {
		log.Printf("Request error with cmd %s: %s", cmd, err)
		return nil
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	json.Unmarshal(body, &result)

	var ok, exists bool
	if ok, exists = result["ok"].(bool); !exists {
		log.Printf("Incorrect answer from telegram: %s", string(body))
		return nil
	}

	serverResponse := &ServerResponse{ok: ok}
	if !ok {
		log.Printf("Something is wrong with request: %s",
			result["description"].(string))
		serverResponse.description = result["description"].(string)
	} else {
		if st, exists := result["result"].(map[string]interface{}); exists {
			serverResponse.result = BotResult(st)
		}
	}
	return serverResponse
}

func (bot *Bot) GetMe() (bool, BotResult) {
	resp := bot.Command("getMe", nil)
	if resp != nil && resp.ok {
		return true, resp.result
	}
	return false, nil
}

func (bot *Bot) SendMessage(chat_id int, text string) int {
	values := url.Values{}
	values.Set("chat_id", strconv.Itoa(chat_id))
	values.Set("text", text)
	resp := bot.Command("sendMessage", &values)
	if msg_id, ok := resp.result["message_id"]; ok {
		return int(msg_id.(float64))
	}
	return 0
}

func (bot *Bot) SendReplyMessage(chat_id int, text string) int {
	values := url.Values{}
	values.Set("chat_id", strconv.Itoa(chat_id))
	values.Set("text", text)
	values.Set("reply_markup", `{"force_reply": true, "selective": false}`)
	resp := bot.Command("sendMessage", &values)
	if msg_id, ok := resp.result["message_id"]; ok {
		return int(msg_id.(float64))
	}
	return 0
}

func (bot *Bot) SetWebhook(hookurl string) bool {
	values := url.Values{}
	values.Set("url", hookurl)
	resp := bot.Command("setWebhook", &values)
	if resp != nil && resp.ok {
		return true
	}
	return false
}
