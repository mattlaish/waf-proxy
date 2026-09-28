package main

import (
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const api3TestDocV1 = `openapi: 3.0.3
info:
  title: Payments
  version: 1.0.0
components:
  securitySchemes:
    bearerAuth:
      type: http
      scheme: bearer
      bearerFormat: JWT
  schemas:
    Payment:
      type: object
      required: [amount, currency]
      properties:
        amount:
          type: number
        currency:
          type: string
          enum: [TWD, USD]
        memo:
          type: string
security:
  - bearerAuth: []
paths:
  /api/payments/{paymentId}:
    parameters:
      - name: paymentId
        in: path
        required: true
        schema:
          type: string
          format: uuid
    post:
      operationId: createPayment
      parameters:
        - name: region
          in: query
          schema:
            type: string
            enum: [tw, us]
        - name: X-Tenant
          in: header
          required: true
          schema:
            type: string
      requestBody:
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/Payment'
      responses:
        '200':
          content:
            application/json:
              schema:
                type: object
                required: [id]
                properties:
                  id:
                    type: string
                    format: uuid
`

const api3TestDocV2 = `openapi: 3.1.0
info:
  title: Payments
  version: 2.0.0
components:
  securitySchemes:
    bearerAuth:
      type: http
      scheme: bearer
security: []
paths:
  /api/payments/{paymentId}:
    parameters:
      - name: paymentId
        in: path
        required: true
        schema:
          type: string
          format: uuid
    post:
      operationId: createPayment
      parameters:
        - name: region
          in: query
          required: true
          schema:
            type: string
            enum: [tw, us, jp]
      requestBody:
        content:
          application/json:
            schema:
              type: object
              required: [amount, currency, customer_id]
              properties:
                amount:
                  type: string
                currency:
                  type: string
                  enum: [TWD, USD, JPY]
                customer_id:
                  type: string
                  format: uuid
      responses:
        '200':
          content:
            application/json:
              schema:
                type: object
                properties:
                  id:
                    type: string
  /api/refunds:
    post:
      operationId: createRefund
      responses:
        '204':
          description: created
`

func TestParseOpenAPIDocumentAndLocalRef(t *testing.T) {
	parsed, err := parseOpenAPIDocument(api3TestDocV1)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if parsed.Title != "Payments" || parsed.Version != "1.0.0" || parsed.OpenAPIVersion != "3.0.3" {
		t.Fatalf("unexpected metadata: %+v", parsed)
	}
	if len(parsed.SecuritySchemes) != 1 || stringValue(parsed.SecuritySchemes["bearerAuth"]["scheme"]) != "bearer" {
		t.Fatalf("security schemes missing: %+v", parsed.SecuritySchemes)
	}
	if len(parsed.Operations) != 1 {
		t.Fatalf("operations=%d want 1", len(parsed.Operations))
	}
	op := parsed.Operations[0]
	if op.Method != "POST" || op.Path != "/api/payments/{paymentId}" || op.OperationID != "createPayment" {
		t.Fatalf("unexpected operation: %+v", op)
	}
	fields := map[string]ContractSchemaField{}
	for _, f := range op.RequestFields {
		fields[f.Path] = f
	}
	if !fields["amount"].Required || fields["amount"].Type != "number" {
		t.Fatalf("amount field=%+v", fields["amount"])
	}
	if got := strings.Join(fields["currency"].Enum, ","); got != "TWD,USD" {
		t.Fatalf("currency enum=%q", got)
	}
	params := map[string]ContractParameter{}
	for _, p := range op.Parameters {
		params[p.In+"|"+p.Name] = p
	}
	if !params["path|paymentId"].Required || params["path|paymentId"].Format != "uuid" {
		t.Fatalf("path parameter missing: %+v", params)
	}
	if got := strings.Join(params["query|region"].Enum, ","); got != "tw,us" {
		t.Fatalf("query enum=%q", got)
	}
	if !params["header|X-Tenant"].Required {
		t.Fatalf("header parameter missing: %+v", params)
	}
	if len(op.Security) != 1 || op.Security[0].Scheme != "bearerAuth" {
		t.Fatalf("security requirement missing: %+v", op.Security)
	}
}

func TestParseOpenAPIRejectsUnsupportedOrIncomplete(t *testing.T) {
	cases := []string{
		`openapi: 2.0\ninfo: {title: old, version: v1}\npaths: {}`,
		`openapi: 3.0.3\npaths: {}`,
		`openapi: 3.0.3\ninfo: {title: t, version: v1}`,
	}
	for _, doc := range cases {
		if _, err := parseOpenAPIDocument(doc); err == nil {
			t.Fatalf("expected rejection for %q", doc)
		}
	}
}

func TestContractImportIsVersionedIdempotentAndPersistent(t *testing.T) {
	store := newContractStore()
	v1, err := store.importDocument("payments", "checkout", api3TestDocV1)
	if err != nil {
		t.Fatal(err)
	}
	v1Replay, err := store.importDocument("payments", "checkout", api3TestDocV1)
	if err != nil {
		t.Fatal(err)
	}
	if v1.ID != v1Replay.ID {
		t.Fatalf("same bytes produced different version ids: %s != %s", v1.ID, v1Replay.ID)
	}
	v2, err := store.importDocument("payments", "checkout", api3TestDocV2)
	if err != nil {
		t.Fatal(err)
	}
	if v1.ID == v2.ID {
		t.Fatal("different contract bytes reused version id")
	}
	if len(store.listVersions(v1.ContractID)) != 2 {
		t.Fatalf("versions=%d want 2", len(store.listVersions(v1.ContractID)))
	}

	configPath := filepath.Join(t.TempDir(), "config.json")
	if err := store.save(configPath); err != nil {
		t.Fatal(err)
	}
	loaded := newContractStore()
	if err := loaded.load(configPath); err != nil {
		t.Fatal(err)
	}
	got, ops, ok := loaded.latestVersion(v1.ContractID)
	if !ok || got.ID != v2.ID || len(ops) != 2 || len(got.SecuritySchemes) != 1 {
		t.Fatalf("persistent contract mismatch ok=%v version=%+v ops=%d", ok, got, len(ops))
	}
}

func TestContractOperationMatchingUsesNormalizedPathAndStableOperationID(t *testing.T) {
	version := APIContractVersion{ID: "v1"}
	contractOps := []ContractOperation{{ID: "cop", VersionID: "v1", Method: "POST", Path: "/api/payments/{paymentId}"}}
	normalized := "/api/payments/{id}"
	observed := []apiOperation{{ID: "op-123", Site: "payments", Method: "POST", Path: normalized, Fingerprint: operationFingerprint("POST", normalized), Samples: 10}}
	bindings := matchContractOperations(version, contractOps, observed)
	if len(bindings) != 1 {
		t.Fatalf("bindings=%d want 1", len(bindings))
	}
	if bindings[0].Status != "MATCHED" || bindings[0].Confidence < 0.99 {
		t.Fatalf("unexpected binding: %+v", bindings[0])
	}
	if bindings[0].APIOperationID != "op-123" {
		t.Fatalf("api operation id mismatch: %+v", bindings[0])
	}
}

func TestCompareContractVersionsCoversSchemaParametersSecurityAndResponse(t *testing.T) {
	store := newContractStore()
	v1, err := store.importDocument("payments", "checkout", api3TestDocV1)
	if err != nil {
		t.Fatal(err)
	}
	v2, err := store.importDocument("payments", "checkout", api3TestDocV2)
	if err != nil {
		t.Fatal(err)
	}
	_, ops1, _ := store.version(v1.ID)
	_, ops2, _ := store.version(v2.ID)
	diffs := compareContractVersions(v1.ContractID, v1, ops1, v2, ops2)
	seen := map[string]bool{}
	for _, diff := range diffs {
		seen[diff.ChangeType] = true
	}
	for _, typ := range []string{"TYPE_CHANGED", "FIELD_ADDED", "OPERATION_ADDED", "ENUM_CHANGED", "PARAMETER_ENUM_CHANGED", "SECURITY_REQUIREMENT_CHANGED"} {
		if !seen[typ] {
			t.Fatalf("missing %s in %+v", typ, diffs)
		}
	}
}

func TestDriftAgainstLiveLearnedSchemaTaxonomy(t *testing.T) {
	learned := newSchemaStore()
	candidate := &SchemaCandidate{
		ID: "candidate-1", OperationID: "op-payments", Status: "CANDIDATE", Confidence: 0.95, SampleCount: 100,
		Fields: []SchemaFieldObservation{
			{Path: "amount", Location: "body", TypeCounts: map[string]int64{"string": 100}, Samples: 100},
			{Path: "currency", Location: "body", TypeCounts: map[string]int64{"string": 100}, EnumCandidate: []string{"GBP"}, Samples: 100},
			{Path: "unexpected", Location: "body", TypeCounts: map[string]int64{"boolean": 100}, Samples: 100},
			{Path: "param_1", Location: "path", TypeCounts: map[string]int64{"string": 100}, Formats: []string{"uuid"}, Samples: 100},
			{Path: "region", Location: "query", TypeCounts: map[string]int64{"string": 100}, EnumCandidate: []string{"eu"}, Samples: 100},
			{Path: "x-tenant", Location: "header", TypeCounts: map[string]int64{"string": 100}, Samples: 100},
		}, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(), Version: 1,
	}
	learned.candidates[candidate.ID] = candidate
	parsed, err := parseOpenAPIDocument(api3TestDocV1)
	if err != nil {
		t.Fatal(err)
	}
	version := APIContractVersion{ID: "v1", ContractID: "contract", SecuritySchemes: parsed.SecuritySchemes}
	op := parsed.Operations[0]
	op.ID = "cop"
	op.VersionID = "v1"
	bindings := []OperationBinding{{ID: "binding", ContractVersionID: "v1", ContractOperationID: "cop", APIOperationID: "op-payments", Status: "MATCHED", Confidence: 1}}
	observed := []apiOperation{
		{ID: "op-payments", Method: "POST", Path: "/api/payments/{id}", Samples: 100, ContentTypes: map[string]int64{"application/xml": 100}, AuthObserved: map[string]int64{"none": 100}},
		{ID: "op-shadow", Method: "GET", Path: "/api/internal/debug", Samples: 5},
	}
	events := driftAgainstLearnedSchema("contract", version, []ContractOperation{op}, bindings, learned, observed)
	seen := map[string]bool{}
	for _, event := range events {
		seen[event.Type] = true
	}
	for _, typ := range []string{"TYPE_DRIFT", "ENUM_DRIFT", "UNKNOWN_FIELD", "CONTENT_TYPE_DRIFT", "AUTH_DRIFT", "UNDECLARED_ENDPOINT"} {
		if !seen[typ] {
			t.Fatalf("missing drift %s in %+v", typ, events)
		}
	}
}

func TestOpenAPIExportRoundTripRetainsParametersEnumsAndSecurity(t *testing.T) {
	store := newContractStore()
	v1, err := store.importDocument("payments", "checkout", api3TestDocV1)
	if err != nil {
		t.Fatal(err)
	}
	b, contentType, err := store.exportDocument(v1.ContractID, v1.ID, "yaml")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(contentType, "yaml") {
		t.Fatalf("unexpected content type %s", contentType)
	}
	parsed, err := parseOpenAPIDocument(string(b))
	if err != nil {
		t.Fatalf("export did not round-trip: %v\n%s", err, string(b))
	}
	if len(parsed.Operations) != 1 || len(parsed.SecuritySchemes) != 1 {
		t.Fatalf("roundtrip metadata missing: %+v", parsed)
	}
	op := parsed.Operations[0]
	if len(op.Parameters) != 3 || len(op.Security) != 1 {
		t.Fatalf("roundtrip operation incomplete: %+v", op)
	}
	fields := map[string]ContractSchemaField{}
	for _, f := range op.RequestFields {
		fields[f.Path] = f
	}
	if strings.Join(fields["currency"].Enum, ",") != "TWD,USD" {
		t.Fatalf("enum lost: %+v", fields["currency"])
	}
}

func TestOpenAPIRejectsExternalRefAndNormalizedDuplicate(t *testing.T) {
	external := `openapi: 3.0.3
info: {title: External, version: 1.0.0}
paths:
  /api/x:
    post:
      requestBody:
        content:
          application/json:
            schema:
              $ref: https://example.invalid/schema.json
      responses:
        '204': {description: ok}
`
	if _, err := parseOpenAPIDocument(external); err == nil || !strings.Contains(err.Error(), "external OpenAPI $ref") {
		t.Fatalf("external ref not rejected: %v", err)
	}
	duplicate := `openapi: 3.0.3
info: {title: Duplicate, version: 1.0.0}
paths:
  /api/users/{id}:
    get:
      responses: {'204': {description: ok}}
  /api/users/{userId}:
    get:
      responses: {'204': {description: ok}}
`
	if _, err := parseOpenAPIDocument(duplicate); err == nil || !strings.Contains(err.Error(), "ambiguous normalized operation") {
		t.Fatalf("normalized duplicate not rejected: %v", err)
	}
}

func TestDriftUndeclaredEndpointIsScopedToMatchedApplicationTraffic(t *testing.T) {
	learned := newSchemaStore()
	learned.candidates["candidate"] = &SchemaCandidate{ID: "candidate", OperationID: "op-main", Status: "CANDIDATE", Fields: []SchemaFieldObservation{}}
	version := APIContractVersion{ID: "v1", ContractID: "c"}
	contractOps := []ContractOperation{{ID: "cop", Method: "GET", Path: "/api/main"}}
	bindings := []OperationBinding{{ID: "b", ContractOperationID: "cop", APIOperationID: "op-main", Status: "MATCHED"}}
	observed := []apiOperation{
		{ID: "op-main", Site: "site-a", Host: "a.example", Method: "GET", Path: "/api/main"},
		{ID: "shadow-a", Site: "site-a", Host: "a.example", Method: "GET", Path: "/api/shadow"},
		{ID: "unrelated-b", Site: "site-b", Host: "b.example", Method: "GET", Path: "/api/admin"},
	}
	events := driftAgainstLearnedSchema("c", version, contractOps, bindings, learned, observed)
	var shadow, unrelated bool
	for _, event := range events {
		if event.Type != "UNDECLARED_ENDPOINT" {
			continue
		}
		shadow = shadow || strings.Contains(event.Location, "/api/shadow")
		unrelated = unrelated || strings.Contains(event.Location, "/api/admin")
	}
	if !shadow || unrelated {
		t.Fatalf("scope filtering incorrect: %+v", events)
	}
}

func TestAuthRequirementDriftHonorsOpenAPIOrAndAnonymousSemantics(t *testing.T) {
	schemes := map[string]map[string]any{
		"bearerAuth": {"type": "http", "scheme": "bearer"},
		"apiKeyAuth": {"type": "apiKey", "in": "header", "name": "X-API-Key"},
	}
	andReq := []ContractSecurityRequirement{{Scheme: "bearerAuth", Group: 0}, {Scheme: "apiKeyAuth", Group: 0}}
	if !authRequirementDrift(schemes, andReq, map[string]int64{"bearer": 10}) {
		t.Fatal("bearer alone must not satisfy bearer+apiKey AND requirement")
	}
	if authRequirementDrift(schemes, andReq, map[string]int64{"apikey+bearer": 10}) {
		t.Fatal("combined evidence should satisfy bearer+apiKey AND requirement")
	}
	orReq := []ContractSecurityRequirement{{Scheme: "bearerAuth", Group: 0}, {Scheme: "apiKeyAuth", Group: 1}}
	if authRequirementDrift(schemes, orReq, map[string]int64{"apikey": 10}) {
		t.Fatal("apiKey should satisfy OR alternative")
	}
	anonymous := []ContractSecurityRequirement{{Group: 0}, {Scheme: "bearerAuth", Group: 1}}
	if authRequirementDrift(schemes, anonymous, map[string]int64{"none": 10}) {
		t.Fatal("anonymous OpenAPI alternative must not drift")
	}
}

func TestDeclaredMultipleContentTypesAcceptAnyObservedIntersection(t *testing.T) {
	op := ContractOperation{RequestMedia: "application/json", RequestMediaTypes: []string{"application/json", "application/xml"}}
	if !observedContentTypeMatches(op.RequestMediaTypes, op.RequestMedia, map[string]int64{"application/xml": 5}) {
		t.Fatal("declared XML alternative should match observed XML")
	}
	if observedContentTypeMatches(op.RequestMediaTypes, op.RequestMedia, map[string]int64{"text/plain": 5}) {
		t.Fatal("undeclared text/plain should not match")
	}
}
