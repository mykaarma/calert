package google_chat

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	alertmgrtmpl "github.com/prometheus/alertmanager/template"
)

const (
	maxMsgSize = 4096
)

// prepareMessage accepts an Alert object and templates out with the user provided template.
// It also splits the alerts if the combined size exceeds the limit of 4096 bytes by
// G-Chat Webhook API
func (m *GoogleChatManager) prepareMessage(alert alertmgrtmpl.Alert) ([]ChatMessage, error) {
	var (
		to  bytes.Buffer
		msg ChatMessage
	)

	messages := make([]ChatMessage, 0)

	// Render template
	err := m.msgTmpl.ExecuteTemplate(&to, "googlechat.custom.message", alert)
	if err != nil {
		m.lo.Error("Error parsing values in template", "error", err)
		return messages, err
	}

	// Log the output for debugging
	rendered := to.String()
	m.lo.Info("Rendered template output", "content", rendered)

	// Parse JSON
	var cards map[string]interface{}
	if err := json.Unmarshal(to.Bytes(), &cards); err != nil {
		m.lo.Error("invalid card JSON", "error", err)
		return messages, err
	}

	// Set the message
	msg.Cards = []map[string]interface{}{cards}
	m.lo.Info("message card to be sent", msg.Cards[0])
	messages = append(messages, msg)

	return messages, nil
}

// sendMessage pushes out a notification to Google Chat space.
func (m *GoogleChatManager) sendMessage(msg ChatMessage, threadKey string) error {
	// out, err := json.Marshal(msg)
	// if err != nil {
	// 	return err
	// }

	buf := new(bytes.Buffer)
	if err := json.NewEncoder(buf).Encode(msg.Cards[0]); err != nil {
		return err
	}

	// Parse the webhook URL to add `?threadKey` param.
	u, err := url.Parse(m.endpoint)
	if err != nil {
		return err
	}
	q := u.Query()
	// Default behaviour is to start a new thread for every alert.
	q.Set("messageReplyOption", "MESSAGE_REPLY_OPTION_UNSPECIFIED")
	if m.threadedReplies {
		// If threaded replies are enabled, use the threadKey to reply to the same thread.
		q.Set("messageReplyOption", "REPLY_MESSAGE_FALLBACK_TO_NEW_THREAD")
		q.Set("threadKey", threadKey)
	}
	u.RawQuery = q.Encode()
	endpoint := u.String()

	// Prepare the request.
	req, err := http.NewRequest("POST", endpoint, buf)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	// Send the request.
	m.lo.Debug("sending alert", "url", endpoint, "msg", msg.Text)
	resp, err := m.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// If response is non 200, log and throw the error.
	if resp.StatusCode != http.StatusOK {
		// Read the response body
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			// Log the error if unable to read the response body
			m.lo.Error("Failed to read response body", "error", err)
			return fmt.Errorf("failed to read response body")
		}
		// Ensure the original response body is closed
		defer resp.Body.Close()

		// Convert the body bytes to a string for logging
		responseBody := string(bodyBytes)

		// Log the status code and response body at the debug level
		m.lo.Error("Non OK HTTP Response received from Google Chat Webhook endpoint", "status", resp.StatusCode, "responseBody", responseBody)

		// Since the body has been read, if you need to use it later,
		// you may need to reassign resp.Body with a new reader
		resp.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

		return fmt.Errorf("non ok response from gchat")
	}

	return nil
}
