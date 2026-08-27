package mattermost

import (
	"fmt"
)

// DM capability: send direct messages from the bot to a target user.
type DM struct {
	client *Client
}

func NewDM(client *Client) *DM {
	return &DM{client: client}
}

// Send delivers a direct message to targetUserID using botUserID as the sender.
func (s *DM) Send(botToken, botUserID, targetUserID, message string) error {
	if botToken == "" {
		return nil
	}
	if _, err := s.client.SendDM(botToken, botUserID, targetUserID, message); err != nil {
		return fmt.Errorf("send dm: %w", err)
	}
	return nil
}
