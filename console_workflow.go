package main

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
)

var errDeliveryConfigChanged = errors.New("active delivery settings changed; review the current destination and test again")

type deliveryTestRequest struct {
	ExpectedGeneration *uint64 `json:"expected_generation"`
	ExpectedToken      string  `json:"expected_token"`
}

func decodeDeliveryTest(w http.ResponseWriter, r *http.Request) (*deliveryTestRequest, bool) {
	if r.Body == nil || r.ContentLength == 0 {
		return &deliveryTestRequest{}, true
	} // compatibility for existing clients
	var input deliveryTestRequest
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1024))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil {
		http.Error(w, "invalid delivery test request", 400)
		return nil, false
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		http.Error(w, "invalid delivery test request", 400)
		return nil, false
	}
	return &input, true
}

func (a *adminServer) handleNotifyHandle(w http.ResponseWriter, r *http.Request) {
	id, _, ok := decodeIDReq(w, r)
	if !ok {
		return
	}
	if !a.srv.notify.markHandled(id) {
		http.Error(w, "notification no longer available", 404)
		return
	}
	a.audit.add(who(r).user, "notification.handled", "reviewed notification")
	writeJSON(w, map[string]any{"ok": true})
}

func (a *adminServer) handleNotifyTest(w http.ResponseWriter, r *http.Request) {
	expected, ok := decodeDeliveryTest(w, r)
	if !ok {
		return
	}
	n := a.srv.notify
	n.mu.Lock()
	cfg := n.cfg
	generation := n.delivery.current()
	token := n.delivery.configToken()
	n.mu.Unlock()
	if expected.ExpectedGeneration != nil && *expected.ExpectedGeneration != generation || expected.ExpectedToken != "" && expected.ExpectedToken != token {
		http.Error(w, errDeliveryConfigChanged.Error(), 409)
		return
	}
	if cfg.WebhookURL == "" {
		http.Error(w, "no active webhook configured; save and Apply settings first", 400)
		return
	}
	_, err := n.sendWebhook(cfg.WebhookURL, cfg.WebhookKind, notification{Level: "info", Kind: "test", Title: "WAF delivery test", Body: "Operator-requested transport test; not an attack or confirmed ticket."}, cfg, generation)
	a.audit.add(who(r).user, "notification.webhook_test", "active configuration; transport send only")
	if err != nil {
		http.Error(w, "webhook test failed; inspect delivery status", 502)
		return
	}
	writeJSON(w, map[string]any{"ok": true, "source": "active", "receipt_verified": false, "delivery": n.webhookDeliveryStats()})
}
