package google_chat

import (
	"log/slog"
	"sync"
	"time"

	"github.com/gofrs/uuid"
	"github.com/mr-karan/calert/internal/metrics"
	alertmgrtmpl "github.com/prometheus/alertmanager/template"
)

// ActiveAlerts represents a map of alerts unique fingerprint hash
// with their details.
type ActiveAlerts struct {
	lo      *slog.Logger
	metrics *metrics.Manager
	sync.RWMutex
	alerts map[string]AlertDetails
}

// AlertDetails represents some internal fields required
// for dispatching alerts or cleaning up based on TTL.
type AlertDetails struct {
	StartsAt time.Time
	UUID     uuid.UUID
}

// ChatMessage represents the structure for sending a
// Text message in Google Chat Webhook endpoint.
// https://developers.google.com/chat/api/guides/message-formats/basic
type ChatMessage struct {
	Text  string                   `json:"text"`
	Cards []map[string]interface{} `json:"cards,omitempty"`
}

type CardMessage struct {
	Cards []Card `json:"cards"`
}

type Card struct {
	Header   *Header   `json:"header,omitempty"`
	Sections []Section `json:"sections"`
}

type Header struct {
	Title    string `json:"title,omitempty"`
	Subtitle string `json:"subtitle,omitempty"`
}

type Section struct {
	Widgets []Widget `json:"widgets"`
}

type Widget struct {
	TextParagraph *TextParagraph `json:"textParagraph,omitempty"`
	KeyValue      *KeyValue      `json:"keyValue,omitempty"`
	Buttons       []ButtonWidget `json:"buttons,omitempty"`
}

type TextParagraph struct {
	Text string `json:"text"`
}

type KeyValue struct {
	TopLabel         string `json:"topLabel"`
	Content          string `json:"content"`
	ContentMultiline bool   `json:"contentMultiline"`
}

type ButtonWidget struct {
	TextButton TextButton `json:"textButton"`
}

type TextButton struct {
	Text    string  `json:"text"`
	OnClick OnClick `json:"onClick"`
}

type OnClick struct {
	OpenLink OpenLink `json:"openLink"`
}

type OpenLink struct {
	URL string `json:"url"`
}

// add adds an alert to the active alerts map.
func (d *ActiveAlerts) add(a alertmgrtmpl.Alert) error {
	d.Lock()
	defer d.Unlock()

	// Create a UUID for the alert. This UUID is
	// sent as a `threadKey` param in G-Chat API.
	// Set UUID for the alert.
	uid, err := uuid.NewV4()
	if err != nil {
		return err
	}

	// Add the alert metadata to the map.
	d.alerts[a.Fingerprint] = AlertDetails{
		UUID:     uid,
		StartsAt: a.StartsAt,
	}

	return nil
}

// loookup retrievs the UUID for the alert based on the fingerprint.
func (d *ActiveAlerts) loookup(fingerprint string) string {
	d.RLock()
	defer d.RUnlock()

	// Do a lookup for the provider by the room name and push the alerts.
	if _, ok := d.alerts[fingerprint]; !ok {
		return ""
	}
	return d.alerts[fingerprint].UUID.String()
}

// Prune iterates on a list of active alerts inside the map
// and deletes them if they exceed the specified TTL.
func (d *ActiveAlerts) Prune(ttl time.Duration) {
	d.Lock()
	defer d.Unlock()

	var (
		now     = time.Now()
		expired = now.Add(-ttl)
	)

	// Iterate on map of active alerts.
	for k, a := range d.alerts {
		// If the alert creation field is past our specified TTL, remove it from the map.
		if a.StartsAt.Before(expired) {
			d.lo.Debug("removing alert from active alerts", "fingerprint", k, "created", a.StartsAt, "expired", expired)
			delete(d.alerts, k)
		}
	}

	d.metrics.Duration(`alerts_prune_duration_seconds`, now)

}

// InitPruner is used to remove active alerts in the
// map once their TTL is reached. The cleanup activity happens at periodic intervals.
// This is a blocking function so the caller must invoke as a goroutine.
// The reason for this background worker is
// 1) Alertmanager doesn't have any unique ID for a generated alert. The use case is to send
// all the future alerts for same labels in a same thread. Labels are computed via `.fingerprint` field which is a
// unique hash based on those labels. The problem here is that all future alerts for same labels will also have same
// fingerprint. This means that even after the status is Resolved, we will continue posting to same thread if we use this
// fingerprint. This is undesirable, we ideally want each thread to have the last message as "Resolved".
// Now since there's no unique field, we maintain a map of active alerts. All the alerts will be stored here for a specified
// TTL.
// 2) Since we are storing the alerts in a map, this map will continue to grow unbounded.
// We need to have a TTL based expiry for these map keys. This is the most simple implementation to prune alerts by running this
// function as a GoRoutine and check if the alert creation timestamp has crossed our specified TTL. If it has, it'll delete the alert
// entry from the map.
// This check happens at a periodic interval specified by `pruneInterval` by the caller.
func (d *ActiveAlerts) startPruneWorker(pruneInterval time.Duration, ttl time.Duration) {
	var (
		evalTicker = time.NewTicker(pruneInterval).C
	)

	for range evalTicker {
		d.lo.Debug("pruning active alerts based on ttl")
		d.Prune(ttl)
	}
}
