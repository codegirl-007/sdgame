package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/websocket"
	claude "github.com/potproject/claude-sdk-go"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type MessageReceived struct {
	Message       string `json:"message"`
	DesignPayload string `json:"designPayload"`
}

func Messages(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}
	defer conn.Close()

	client := claude.NewClient(os.Getenv("CLAUDE_API_KEY"))

	for {
		messageType, message, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		fmt.Printf("message: %s", message)

		var messageReceived MessageReceived
		err = json.Unmarshal(message, &messageReceived)
		if err != nil {
			fmt.Printf("error unmarshalling response: %v", err)
			continue
		}

		if messageReceived.Message == "" {
			messageReceived.Message = "<user did not send text>"
		} else {
			messageReceived.Message = string(message)
		}

		prompt := fmt.Sprintf("You are a tutor that helps people learn system design. You will be given a JSON payload that looks like %s. The nodes are the components a user can put into their design and the connections will tell you how they are connected. The level name identifies what problem they are working on as well as a difficulty level. Each level has an easy, medium or hard setting. Also in the payload, there is a list of components that a user can use to build their design. Your hints and responses should only refer to these components and not refer to things that the user cannot use. Always refer to the nodes by their type. Please craft your response as if you're talking to the user. And do not reference the payload as \"payload\" but as their design. Also, please do not show the payload in your response. Do not refer to components as node-0 or whatever. Always refer to the type of component they are. Always assume that the source of traffic for any system is a user. The user component will not be visible in teh payload. Also make sure you use html to format your answer. Do not over format your response. Only use p tags. Format lists using proper lists html. Anytime the user sends a different payload back to you, make note of what is correct. Never give the actual answer, only helpful hints. If the available components do not allow the user to feasibly solve the system design problem, you should mention it and then tell them what exactly is missing from the list.", messageReceived.DesignPayload)

		m := claude.RequestBodyMessages{
			Model:     "claude-3-7-sonnet-20250219",
			MaxTokens: 1024,
			SystemTypeText: []claude.RequestBodySystemTypeText{
				claude.UseSystemCacheEphemeral(prompt),
			},
			Messages: []claude.RequestBodyMessagesMessages{
				{
					Role: claude.MessagesRoleUser,
					ContentTypeText: []claude.RequestBodyMessagesMessagesContentTypeText{
						{
							Text:         messageReceived.Message,
							CacheControl: claude.UseCacheEphemeral(),
						},
					},
				},
			},
		}

		ctx := context.Background()
		res, err := client.CreateMessages(ctx, m)
		if err != nil {
			fmt.Printf("error creating messages: %v", err)
		}

		// Echo the message back to client
		err = conn.WriteMessage(messageType, []byte(res.Content[0].Text))
		if err != nil {
			log.Printf("Write error: %v", err)
			break
		}
	}

	log.Println("Client disconnected")
}
