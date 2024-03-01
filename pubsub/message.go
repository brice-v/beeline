package pubsub

import "beeline/models"

type Message struct {
	topic string
	body  models.ChatMessage
}

func NewMessage(topic string, msg models.ChatMessage) *Message {
	// Returns the message object
	return &Message{
		topic: topic,
		body:  msg,
	}
}

func (m *Message) GetTopic() string {
	// returns the topic of the message
	return m.topic
}

func (m *Message) GetMessage() models.ChatMessage {
	// returns the message body.
	return m.body
}
