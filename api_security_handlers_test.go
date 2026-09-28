package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSchemaReviewHandlerRejectsLearningPromotion(t *testing.T) {
	store := newSchemaStore()
	op := apiOperation{ID: "op-learning", Method: "POST", Path: "/api/learning"}
	store.note(op, []schemaFieldSample{{Path: "kind", Location: "body", Type: "string", EnumValue: "A", ValueHash: "a"}})
	candidate, ok := store.findByOperationID(op.ID)
	if !ok || candidate.Status != "LEARNING" {
		t.Fatalf("unexpected learning candidate: ok=%v candidate=%+v", ok, candidate)
	}

	a := &adminServer{srv: &server{schema: store}}
	req := httptest.NewRequest(http.MethodPost, "/api/security/schema/"+candidate.ID+"/review", bytes.NewBufferString(`{"decision":"approve"}`))
	req.SetPathValue("id", candidate.ID)
	rr := httptest.NewRecorder()
	a.handleSchemaReview(rr, req)
	if rr.Code != http.StatusConflict {
		t.Fatalf("status=%d body=%q, want 409", rr.Code, rr.Body.String())
	}
	unchanged, _ := store.get(candidate.ID)
	if unchanged.Status != "LEARNING" || unchanged.Review != nil {
		t.Fatalf("learning candidate mutated by rejected review: %+v", unchanged)
	}
}
