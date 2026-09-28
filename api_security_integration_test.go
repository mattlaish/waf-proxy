package main

import "testing"

const apiSecurityIntegrationContract = `{
  "openapi":"3.0.3",
  "info":{"title":"Payments","version":"1.0.0"},
  "components":{"securitySchemes":{"bearerAuth":{"type":"http","scheme":"bearer"}}},
  "security":[{"bearerAuth":[]}],
  "paths":{
    "/api/payment/{paymentId}":{
      "post":{
        "operationId":"createPayment",
        "parameters":[
          {"name":"paymentId","in":"path","required":true,"schema":{"type":"integer"}},
          {"name":"region","in":"query","schema":{"type":"string","enum":["tw","us"]}},
          {"name":"X-Tenant","in":"header","required":true,"schema":{"type":"string"}}
        ],
        "requestBody":{"content":{"application/json":{"schema":{
          "type":"object",
          "required":["amount","currency","customer_id"],
          "properties":{
            "amount":{"type":"number"},
            "currency":{"type":"string","enum":["TWD","USD"]},
            "customer_id":{"type":"string","format":"uuid"}
          }
        }}}},
        "responses":{"204":{"description":"accepted"}}
      }
    }
  }
}`

func TestAPISecurityLiveLearningFeedsContractDrift(t *testing.T) {
	apiOps := newAPIOperationStore()
	schema := newSchemaStore()
	for i := 0; i < int(schemaCandidateMinSamples); i++ {
		op := apiOps.note(
			"payments", "api.example.test", "POST", "/api/payment/12345",
			"application/json; charset=utf-8", "bearer", 200,
		)
		meta := apiObservationMeta{
			Host:       "api.example.test",
			AuthScheme: "bearer",
			Headers:    map[string][]string{"x-tenant": {"tenant-a"}},
			BodyPrefix: []byte(`{"amount":"twelve","currency":"GBP","unexpected":true}`),
		}
		samples := collectSchemaSamplesFromObservation(
			"/api/payment/12345", "region=eu", "application/json; charset=utf-8", meta,
		)
		schema.note(op, samples)
	}
	apiOps.note("payments", "api.example.test", "GET", "/api/internal/debug", "", "none", 404)

	paymentOpID := apiOperationID("payments", "POST", "/api/payment/{id}")
	candidate, ok := schema.findByOperationID(paymentOpID)
	if !ok || candidate.Status != "CANDIDATE" || candidate.SampleCount != schemaCandidateMinSamples {
		t.Fatalf("live API-2 candidate not produced: ok=%v candidate=%+v", ok, candidate)
	}

	contracts := newContractStore()
	version, err := contracts.importDocument("payments", "checkout", apiSecurityIntegrationContract)
	if err != nil {
		t.Fatal(err)
	}
	version, contractOps, ok := contracts.latestVersion(version.ContractID)
	if !ok {
		t.Fatal("latest contract version missing")
	}
	observed := apiOps.snapshot()
	bindings := matchContractOperations(version, contractOps, observed)
	events := driftAgainstLearnedSchema(version.ContractID, version, contractOps, bindings, schema, observed)
	seen := map[string]bool{}
	for _, event := range events {
		seen[event.Type] = true
	}
	for _, typ := range []string{"TYPE_DRIFT", "ENUM_DRIFT", "UNKNOWN_FIELD", "MISSING_REQUIRED_FIELD", "UNDECLARED_ENDPOINT"} {
		if !seen[typ] {
			t.Fatalf("missing %s from live learning -> contract drift events: %+v", typ, events)
		}
	}
	if seen["CONTENT_TYPE_DRIFT"] || seen["AUTH_DRIFT"] {
		t.Fatalf("matching content-type/auth should not drift: %+v", events)
	}
}
