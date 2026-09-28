package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	yaml "github.com/goccy/go-yaml"
)

// API-3 OpenAPI Contract Management.
//
// This layer is contract intelligence only. It imports and versions declared
// OpenAPI contracts, correlates them with API-1 observed operations, compares
// contract versions, and reports drift against API-2 learned schema metadata.
// It never blocks a request or activates policy.

type APIContract struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Application string    `json:"application"`
	Description string    `json:"description,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type APIContractVersion struct {
	ID              string                    `json:"id"`
	ContractID      string                    `json:"contract_id"`
	Version         string                    `json:"version"`
	OpenAPIVersion  string                    `json:"openapi_version"`
	SourceType      string                    `json:"source_type"`
	ContentHash     string                    `json:"content_hash"`
	OperationCount  int                       `json:"operation_count"`
	SchemaCount     int                       `json:"schema_count"`
	SecuritySchemes map[string]map[string]any `json:"security_schemes,omitempty"`
	CreatedAt       time.Time                 `json:"created_at"`
}

type ContractSchemaField struct {
	Path     string   `json:"path"`
	Type     string   `json:"type,omitempty"`
	Format   string   `json:"format,omitempty"`
	Required bool     `json:"required,omitempty"`
	Ref      string   `json:"ref,omitempty"`
	Enum     []string `json:"enum,omitempty"`
}

type ContractParameter struct {
	Name     string   `json:"name"`
	In       string   `json:"in"` // query | path | header | cookie
	Required bool     `json:"required,omitempty"`
	Type     string   `json:"type,omitempty"`
	Format   string   `json:"format,omitempty"`
	Enum     []string `json:"enum,omitempty"`
}

type ContractSecurityRequirement struct {
	Scheme string   `json:"scheme"`
	Scopes []string `json:"scopes,omitempty"`
	Group  int      `json:"group,omitempty"`
}

type ContractOperation struct {
	ID                 string                           `json:"id"`
	VersionID          string                           `json:"version_id"`
	Method             string                           `json:"method"`
	Path               string                           `json:"path"`
	OperationID        string                           `json:"operation_id"`
	Summary            string                           `json:"summary,omitempty"`
	RequestFields      []ContractSchemaField            `json:"request_fields,omitempty"`
	ResponseFields     map[string][]ContractSchemaField `json:"response_fields,omitempty"`
	RequestMedia       string                           `json:"request_media_type,omitempty"`
	RequestMediaTypes  []string                         `json:"request_media_types,omitempty"`
	ResponseMedia      map[string]string                `json:"response_media_types,omitempty"`
	ResponseMediaTypes map[string][]string              `json:"response_media_type_sets,omitempty"`
	Parameters         []ContractParameter              `json:"parameters,omitempty"`
	Security           []ContractSecurityRequirement    `json:"security,omitempty"`
}

type OperationBinding struct {
	ID                  string   `json:"id"`
	ContractVersionID   string   `json:"contract_version_id"`
	ContractOperationID string   `json:"contract_operation_id"`
	APIOperationID      string   `json:"api_operation_id,omitempty"`
	Method              string   `json:"method"`
	Path                string   `json:"path"`
	Confidence          float64  `json:"confidence"`
	Status              string   `json:"status"`
	Reasons             []string `json:"reasons,omitempty"`
}

type ContractDiff struct {
	ID            string `json:"id"`
	ContractID    string `json:"contract_id"`
	FromVersionID string `json:"from_version_id"`
	ToVersionID   string `json:"to_version_id"`
	Location      string `json:"location"`
	ChangeType    string `json:"change_type"`
	Severity      string `json:"severity"`
	Description   string `json:"description"`
}

type DriftEvent struct {
	ID                 string    `json:"id"`
	ContractID         string    `json:"contract_id"`
	ContractVersionID  string    `json:"contract_version_id"`
	OperationBindingID string    `json:"operation_binding_id,omitempty"`
	Location           string    `json:"location"`
	Type               string    `json:"type"`
	Severity           string    `json:"severity"`
	Description        string    `json:"description"`
	DetectedAt         time.Time `json:"detected_at"`
}

type parsedContractDocument struct {
	Title           string
	Description     string
	Version         string
	OpenAPIVersion  string
	Operations      []ContractOperation
	SchemaCount     int
	SecuritySchemes map[string]map[string]any
}

type contractStore struct {
	mu sync.RWMutex

	contracts        map[string]APIContract
	versions         map[string]APIContractVersion
	contractVersions map[string][]string
	operations       map[string][]ContractOperation // version ID -> operations
	bindings         map[string][]OperationBinding  // version ID -> bindings
	diffs            map[string][]ContractDiff      // contract ID -> latest stored comparison
}

func newContractStore() *contractStore {
	return &contractStore{
		contracts:        map[string]APIContract{},
		versions:         map[string]APIContractVersion{},
		contractVersions: map[string][]string{},
		operations:       map[string][]ContractOperation{},
		bindings:         map[string][]OperationBinding{},
		diffs:            map[string][]ContractDiff{},
	}
}

var contractHTTPMethods = map[string]struct{}{
	"get": {}, "put": {}, "post": {}, "delete": {}, "options": {}, "head": {}, "patch": {}, "trace": {},
}

func shortContractHash(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:8])
}

func fullContractHash(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}

func normalizeContractPath(path string) string {
	if path == "" {
		return "/"
	}
	parts := strings.Split(path, "/")
	for i, part := range parts {
		if (strings.HasPrefix(part, "{") && strings.HasSuffix(part, "}")) || strings.HasPrefix(part, ":") || part == "*" {
			parts[i] = "{id}"
		}
	}
	return strings.Join(parts, "/")
}

func parseOpenAPIDocument(document string) (parsedContractDocument, error) {
	if strings.TrimSpace(document) == "" {
		return parsedContractDocument{}, errors.New("empty OpenAPI document")
	}
	var root map[string]any
	if err := yaml.Unmarshal([]byte(document), &root); err != nil {
		return parsedContractDocument{}, fmt.Errorf("invalid OpenAPI document: %w", err)
	}
	if err := validateContractRefsLocalOnly(root); err != nil {
		return parsedContractDocument{}, err
	}

	openapi := stringValue(root["openapi"])
	if !(strings.HasPrefix(openapi, "3.0.") || strings.HasPrefix(openapi, "3.1.")) {
		return parsedContractDocument{}, fmt.Errorf("unsupported OpenAPI version %q", openapi)
	}
	info, ok := mapValue(root["info"])
	if !ok {
		return parsedContractDocument{}, errors.New("missing OpenAPI info object")
	}
	title := strings.TrimSpace(stringValue(info["title"]))
	version := strings.TrimSpace(stringValue(info["version"]))
	if title == "" || version == "" {
		return parsedContractDocument{}, errors.New("OpenAPI info.title and info.version are required")
	}
	paths, ok := mapValue(root["paths"])
	if !ok {
		return parsedContractDocument{}, errors.New("missing OpenAPI paths object")
	}

	parsed := parsedContractDocument{
		Title:           title,
		Description:     stringValue(info["description"]),
		Version:         version,
		OpenAPIVersion:  openapi,
		SecuritySchemes: parseContractSecuritySchemes(root),
	}
	globalSecurity := parseContractSecurity(root["security"])

	pathNames := sortedKeys(paths)
	seenOperations := map[string]string{}
	for _, path := range pathNames {
		pathItem, ok := mapValue(paths[path])
		if !ok {
			continue
		}
		pathParameters := parseContractParameters(root, pathItem["parameters"])
		for _, method := range sortedKeys(pathItem) {
			methodLower := strings.ToLower(method)
			if _, allowed := contractHTTPMethods[methodLower]; !allowed {
				continue
			}
			opMap, ok := mapValue(pathItem[method])
			if !ok {
				return parsedContractDocument{}, fmt.Errorf("%s %s operation is not an object", strings.ToUpper(methodLower), path)
			}
			opID := strings.TrimSpace(stringValue(opMap["operationId"]))
			if opID == "" {
				opID = strings.ToUpper(methodLower) + " " + path
			}
			normalizedKey := strings.ToUpper(methodLower) + " " + normalizeContractPath(path)
			if previous, exists := seenOperations[normalizedKey]; exists {
				return parsedContractDocument{}, fmt.Errorf("ambiguous normalized operation %s conflicts with %s", normalizedKey, previous)
			}
			seenOperations[normalizedKey] = strings.ToUpper(methodLower) + " " + path
			op := ContractOperation{
				Method:             strings.ToUpper(methodLower),
				Path:               path,
				OperationID:        opID,
				Summary:            stringValue(opMap["summary"]),
				ResponseFields:     map[string][]ContractSchemaField{},
				ResponseMedia:      map[string]string{},
				ResponseMediaTypes: map[string][]string{},
				Parameters:         mergeContractParameters(pathParameters, parseContractParameters(root, opMap["parameters"])),
				Security:           append([]ContractSecurityRequirement(nil), globalSecurity...),
			}
			if _, present := opMap["security"]; present {
				op.Security = parseContractSecurity(opMap["security"])
			}

			if requestBody, ok := resolveObject(root, opMap["requestBody"], 0, map[string]bool{}); ok {
				if content, ok := mapValue(requestBody["content"]); ok {
					op.RequestMediaTypes = contentMediaTypes(content)
					media, schema := firstContentSchema(content)
					if schema != nil {
						op.RequestMedia = media
						op.RequestFields = flattenContractSchema(root, schema, "", false, 0, map[string]bool{})
						parsed.SchemaCount += len(op.RequestFields)
					}
				}
			}

			if responses, ok := mapValue(opMap["responses"]); ok {
				for _, status := range sortedKeys(responses) {
					resp, ok := resolveObject(root, responses[status], 0, map[string]bool{})
					if !ok {
						continue
					}
					content, ok := mapValue(resp["content"])
					if !ok {
						continue
					}
					op.ResponseMediaTypes[status] = contentMediaTypes(content)
					media, schema := firstContentSchema(content)
					if schema == nil {
						continue
					}
					fields := flattenContractSchema(root, schema, "", false, 0, map[string]bool{})
					op.ResponseFields[status] = fields
					op.ResponseMedia[status] = media
					parsed.SchemaCount += len(fields)
				}
			}
			parsed.Operations = append(parsed.Operations, op)
		}
	}
	return parsed, nil
}

func validateContractRefsLocalOnly(v any) error {
	switch x := v.(type) {
	case map[string]any:
		if ref, ok := x["$ref"].(string); ok && ref != "" && !strings.HasPrefix(ref, "#/") {
			return fmt.Errorf("external OpenAPI $ref is not supported: %s", ref)
		}
		for _, value := range x {
			if err := validateContractRefsLocalOnly(value); err != nil {
				return err
			}
		}
	case []any:
		for _, value := range x {
			if err := validateContractRefsLocalOnly(value); err != nil {
				return err
			}
		}
	}
	return nil
}

func stringValue(v any) string {
	s, _ := v.(string)
	return s
}

func mapValue(v any) (map[string]any, bool) {
	m, ok := v.(map[string]any)
	return m, ok
}

func sliceValue(v any) []any {
	s, _ := v.([]any)
	return s
}

func stringSlice(v any) []string {
	items := sliceValue(v)
	out := make([]string, 0, len(items))
	for _, item := range items {
		switch x := item.(type) {
		case string:
			out = append(out, x)
		case fmt.Stringer:
			out = append(out, x.String())
		case float64, bool, int, int64:
			out = append(out, fmt.Sprint(x))
		}
	}
	sort.Strings(out)
	return out
}

func parseContractSecurity(raw any) []ContractSecurityRequirement {
	var out []ContractSecurityRequirement
	for group, item := range sliceValue(raw) {
		m, ok := mapValue(item)
		if !ok {
			continue
		}
		if len(m) == 0 {
			// OpenAPI uses an empty Security Requirement Object as an
			// anonymous alternative. Preserve the group explicitly instead of
			// accidentally converting optional authentication into required auth.
			out = append(out, ContractSecurityRequirement{Group: group})
			continue
		}
		for _, scheme := range sortedKeys(m) {
			out = append(out, ContractSecurityRequirement{Scheme: scheme, Scopes: stringSlice(m[scheme]), Group: group})
		}
	}
	return out
}

func parseContractSecuritySchemes(root map[string]any) map[string]map[string]any {
	out := map[string]map[string]any{}
	components, ok := mapValue(root["components"])
	if !ok {
		return out
	}
	schemes, ok := mapValue(components["securitySchemes"])
	if !ok {
		return out
	}
	for _, name := range sortedKeys(schemes) {
		def, ok := resolveObject(root, schemes[name], 0, map[string]bool{})
		if !ok {
			continue
		}
		out[name] = cloneAnyMap(def)
	}
	return out
}

func cloneAnyMap(in map[string]any) map[string]any {
	out := make(map[string]any, len(in))
	for k, v := range in {
		switch x := v.(type) {
		case map[string]any:
			out[k] = cloneAnyMap(x)
		case []any:
			cp := make([]any, len(x))
			for i := range x {
				if m, ok := x[i].(map[string]any); ok {
					cp[i] = cloneAnyMap(m)
				} else {
					cp[i] = x[i]
				}
			}
			out[k] = cp
		default:
			out[k] = v
		}
	}
	return out
}

func parseContractParameters(root map[string]any, raw any) []ContractParameter {
	var out []ContractParameter
	for _, item := range sliceValue(raw) {
		obj, ok := resolveObject(root, item, 0, map[string]bool{})
		if !ok {
			continue
		}
		name := strings.TrimSpace(stringValue(obj["name"]))
		in := strings.ToLower(strings.TrimSpace(stringValue(obj["in"])))
		if name == "" || (in != "query" && in != "path" && in != "header" && in != "cookie") {
			continue
		}
		param := ContractParameter{Name: name, In: in, Required: boolValue(obj["required"]) || in == "path"}
		if schema, ok := resolveObject(root, obj["schema"], 0, map[string]bool{}); ok {
			param.Type = stringValue(schema["type"])
			param.Format = stringValue(schema["format"])
			param.Enum = stringSlice(schema["enum"])
		}
		out = append(out, param)
	}
	return out
}

func boolValue(v any) bool {
	b, _ := v.(bool)
	return b
}

func mergeContractParameters(base, override []ContractParameter) []ContractParameter {
	m := map[string]ContractParameter{}
	for _, p := range base {
		m[p.In+"|"+strings.ToLower(p.Name)] = p
	}
	for _, p := range override {
		m[p.In+"|"+strings.ToLower(p.Name)] = p
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]ContractParameter, 0, len(keys))
	for _, k := range keys {
		out = append(out, m[k])
	}
	return out
}

func sortedKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func contentMediaTypes(content map[string]any) []string {
	media := sortedKeys(content)
	for i := range media {
		media[i] = normalizeMediaType(media[i])
	}
	return media
}

func firstContentSchema(content map[string]any) (string, map[string]any) {
	if v, ok := content["application/json"]; ok {
		if entry, ok := mapValue(v); ok {
			if schema, ok := mapValue(entry["schema"]); ok {
				return "application/json", schema
			}
		}
	}
	for _, media := range sortedKeys(content) {
		entry, ok := mapValue(content[media])
		if !ok {
			continue
		}
		if schema, ok := mapValue(entry["schema"]); ok {
			return media, schema
		}
	}
	return "", nil
}

func resolveObject(root map[string]any, raw any, depth int, seen map[string]bool) (map[string]any, bool) {
	if depth > 16 {
		return nil, false
	}
	obj, ok := mapValue(raw)
	if !ok {
		return nil, false
	}
	ref := stringValue(obj["$ref"])
	if ref == "" {
		return obj, true
	}
	if !strings.HasPrefix(ref, "#/") || seen[ref] {
		return nil, false
	}
	seen[ref] = true
	defer delete(seen, ref)
	cur := any(root)
	for _, token := range strings.Split(strings.TrimPrefix(ref, "#/"), "/") {
		m, ok := mapValue(cur)
		if !ok {
			return nil, false
		}
		token = strings.ReplaceAll(strings.ReplaceAll(token, "~1", "/"), "~0", "~")
		cur, ok = m[token]
		if !ok {
			return nil, false
		}
	}
	return resolveObject(root, cur, depth+1, seen)
}

func flattenContractSchema(root map[string]any, raw map[string]any, prefix string, required bool, depth int, seen map[string]bool) []ContractSchemaField {
	if depth > 16 {
		return nil
	}
	ref := stringValue(raw["$ref"])
	if ref != "" {
		resolved, ok := resolveObject(root, raw, depth, seen)
		if !ok {
			return []ContractSchemaField{{Path: fallbackSchemaPath(prefix), Ref: ref, Required: required}}
		}
		fields := flattenContractSchema(root, resolved, prefix, required, depth+1, seen)
		for i := range fields {
			if fields[i].Ref == "" {
				fields[i].Ref = ref
			}
		}
		return fields
	}

	if allOf := sliceValue(raw["allOf"]); len(allOf) > 0 {
		var out []ContractSchemaField
		for _, branch := range allOf {
			if obj, ok := mapValue(branch); ok {
				out = append(out, flattenContractSchema(root, obj, prefix, required, depth+1, seen)...)
			}
		}
		return dedupeContractFields(out)
	}

	typ := stringValue(raw["type"])
	format := stringValue(raw["format"])
	properties, hasProps := mapValue(raw["properties"])
	if typ == "object" || hasProps {
		requiredSet := map[string]bool{}
		for _, item := range sliceValue(raw["required"]) {
			if name, ok := item.(string); ok {
				requiredSet[name] = true
			}
		}
		var out []ContractSchemaField
		for _, name := range sortedKeys(properties) {
			child, ok := mapValue(properties[name])
			if !ok {
				continue
			}
			path := name
			if prefix != "" {
				path = prefix + "." + name
			}
			out = append(out, flattenContractSchema(root, child, path, requiredSet[name], depth+1, seen)...)
		}
		if len(out) == 0 && prefix != "" {
			out = append(out, ContractSchemaField{Path: prefix, Type: "object", Required: required, Format: format})
		}
		return out
	}

	if typ == "array" {
		if items, ok := mapValue(raw["items"]); ok {
			itemPrefix := fallbackSchemaPath(prefix) + "[]"
			children := flattenContractSchema(root, items, itemPrefix, required, depth+1, seen)
			if len(children) > 0 {
				return children
			}
		}
		return []ContractSchemaField{{Path: fallbackSchemaPath(prefix), Type: "array", Format: format, Required: required}}
	}

	return []ContractSchemaField{{Path: fallbackSchemaPath(prefix), Type: typ, Format: format, Required: required, Enum: stringSlice(raw["enum"])}}
}

func fallbackSchemaPath(path string) string {
	if path == "" {
		return "$"
	}
	return path
}

func dedupeContractFields(fields []ContractSchemaField) []ContractSchemaField {
	seen := map[string]ContractSchemaField{}
	for _, f := range fields {
		key := f.Path + "|" + f.Type + "|" + f.Format
		if prev, ok := seen[key]; ok {
			prev.Required = prev.Required || f.Required
			if prev.Ref == "" {
				prev.Ref = f.Ref
			}
			seen[key] = prev
			continue
		}
		seen[key] = f
	}
	out := make([]ContractSchemaField, 0, len(seen))
	for _, f := range seen {
		out = append(out, f)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out
}

func (s *contractStore) importDocument(name, application, document string) (APIContractVersion, error) {
	parsed, err := parseOpenAPIDocument(document)
	if err != nil {
		return APIContractVersion{}, err
	}
	name = strings.TrimSpace(name)
	if name == "" {
		name = parsed.Title
	}
	application = strings.TrimSpace(application)
	contractID := shortContractHash(application + "|" + strings.ToLower(name))
	contentHash := fullContractHash(document)
	versionID := shortContractHash(contractID + "|" + contentHash)
	now := time.Now().UTC()

	version := APIContractVersion{
		ID:              versionID,
		ContractID:      contractID,
		Version:         parsed.Version,
		OpenAPIVersion:  parsed.OpenAPIVersion,
		SourceType:      "UPLOAD",
		ContentHash:     contentHash,
		OperationCount:  len(parsed.Operations),
		SchemaCount:     parsed.SchemaCount,
		SecuritySchemes: parsed.SecuritySchemes,
		CreatedAt:       now,
	}
	for i := range parsed.Operations {
		parsed.Operations[i].VersionID = versionID
		parsed.Operations[i].ID = shortContractHash(versionID + "|" + parsed.Operations[i].Method + "|" + parsed.Operations[i].Path)
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if existing, ok := s.versions[versionID]; ok {
		return existing, nil
	}
	contract, exists := s.contracts[contractID]
	if !exists {
		contract = APIContract{ID: contractID, Name: name, Application: application, Description: parsed.Description, CreatedAt: now}
	}
	contract.Name = name
	contract.Application = application
	contract.Description = parsed.Description
	contract.UpdatedAt = now
	s.contracts[contractID] = contract
	s.versions[versionID] = version
	s.contractVersions[contractID] = append(s.contractVersions[contractID], versionID)
	s.operations[versionID] = parsed.Operations
	return version, nil
}

func (s *contractStore) list() []APIContract {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]APIContract, 0, len(s.contracts))
	for _, c := range s.contracts {
		out = append(out, c)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Application != out[j].Application {
			return out[i].Application < out[j].Application
		}
		return out[i].Name < out[j].Name
	})
	return out
}

func (s *contractStore) get(id string) (APIContract, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	c, ok := s.contracts[id]
	return c, ok
}

func (s *contractStore) listVersions(contractID string) []APIContractVersion {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ids := s.contractVersions[contractID]
	out := make([]APIContractVersion, 0, len(ids))
	for _, id := range ids {
		if v, ok := s.versions[id]; ok {
			out = append(out, v)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.Before(out[j].CreatedAt) })
	return out
}

func (s *contractStore) latestVersion(contractID string) (APIContractVersion, []ContractOperation, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ids := s.contractVersions[contractID]
	if len(ids) == 0 {
		return APIContractVersion{}, nil, false
	}
	id := ids[len(ids)-1]
	v, ok := s.versions[id]
	if !ok {
		return APIContractVersion{}, nil, false
	}
	ops := append([]ContractOperation(nil), s.operations[id]...)
	return v, ops, true
}

func (s *contractStore) version(id string) (APIContractVersion, []ContractOperation, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.versions[id]
	if !ok {
		return APIContractVersion{}, nil, false
	}
	return v, append([]ContractOperation(nil), s.operations[id]...), true
}

func matchContractOperations(version APIContractVersion, contractOps []ContractOperation, observed []apiOperation) []OperationBinding {
	bindings := make([]OperationBinding, 0, len(contractOps))
	for _, cop := range contractOps {
		contractPath := normalizeContractPath(cop.Path)
		contractFingerprint := operationFingerprint(cop.Method, contractPath)
		type candidate struct {
			op      apiOperation
			score   float64
			reasons []string
		}
		var candidates []candidate
		for _, aop := range observed {
			if !strings.EqualFold(cop.Method, aop.Method) {
				continue
			}
			score := 0.30
			reasons := []string{"method"}
			if normalizeContractPath(aop.Path) == contractPath {
				score += 0.50
				reasons = append(reasons, "normalized_path")
			}
			if aop.Fingerprint == contractFingerprint {
				score += 0.20
				reasons = append(reasons, "fingerprint")
			}
			if score >= 0.80 {
				candidates = append(candidates, candidate{op: aop, score: score, reasons: reasons})
			}
		}
		sort.Slice(candidates, func(i, j int) bool { return candidates[i].score > candidates[j].score })
		binding := OperationBinding{
			ID:                  shortContractHash(version.ID + "|" + cop.ID),
			ContractVersionID:   version.ID,
			ContractOperationID: cop.ID,
			Method:              cop.Method,
			Path:                cop.Path,
			Status:              "UNMATCHED",
		}
		if len(candidates) > 0 {
			binding.APIOperationID = candidates[0].op.ID
			if binding.APIOperationID == "" {
				binding.APIOperationID = candidates[0].op.Fingerprint
			}
			binding.Confidence = candidates[0].score
			binding.Reasons = candidates[0].reasons
			binding.Status = "MATCHED"
			if len(candidates) > 1 && candidates[1].score >= candidates[0].score-0.05 {
				binding.Status = "AMBIGUOUS"
				binding.Reasons = append(binding.Reasons, "multiple_high_confidence_candidates")
			}
		}
		bindings = append(bindings, binding)
	}
	return bindings
}

func (s *contractStore) saveBindings(versionID string, bindings []OperationBinding) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.bindings[versionID] = append([]OperationBinding(nil), bindings...)
}

func (s *contractStore) listBindings(contractID string) []OperationBinding {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ids := s.contractVersions[contractID]
	if len(ids) == 0 {
		return nil
	}
	return append([]OperationBinding(nil), s.bindings[ids[len(ids)-1]]...)
}

func compareContractVersions(contractID string, from APIContractVersion, fromOps []ContractOperation, to APIContractVersion, toOps []ContractOperation) []ContractDiff {
	fromMap := map[string]ContractOperation{}
	toMap := map[string]ContractOperation{}
	for _, op := range fromOps {
		fromMap[op.Method+" "+normalizeContractPath(op.Path)] = op
	}
	for _, op := range toOps {
		toMap[op.Method+" "+normalizeContractPath(op.Path)] = op
	}
	var diffs []ContractDiff
	if securitySchemesFingerprint(from.SecuritySchemes) != securitySchemesFingerprint(to.SecuritySchemes) {
		diffs = append(diffs, newContractDiff(contractID, from.ID, to.ID, "components.securitySchemes", "SECURITY_SCHEME_CHANGED", "WARNING", "declared security scheme definitions changed"))
	}
	for key, oldOp := range fromMap {
		newOp, ok := toMap[key]
		if !ok {
			diffs = append(diffs, newContractDiff(contractID, from.ID, to.ID, key, "OPERATION_REMOVED", "BREAKING", "declared operation removed"))
			continue
		}
		diffs = append(diffs, compareContractFields(contractID, from.ID, to.ID, key+" request", oldOp.RequestFields, newOp.RequestFields)...)
		statuses := map[string]bool{}
		for status := range oldOp.ResponseFields {
			statuses[status] = true
		}
		for status := range newOp.ResponseFields {
			statuses[status] = true
		}
		for status := range statuses {
			diffs = append(diffs, compareContractFields(contractID, from.ID, to.ID, key+" response "+status, oldOp.ResponseFields[status], newOp.ResponseFields[status])...)
		}
		diffs = append(diffs, compareContractParameters(contractID, from.ID, to.ID, key, oldOp.Parameters, newOp.Parameters)...)
		if contractMediaFingerprint(oldOp.RequestMediaTypes, oldOp.RequestMedia) != contractMediaFingerprint(newOp.RequestMediaTypes, newOp.RequestMedia) {
			diffs = append(diffs, newContractDiff(contractID, from.ID, to.ID, key+" request", "CONTENT_TYPE_CHANGED", "WARNING", contractMediaFingerprint(oldOp.RequestMediaTypes, oldOp.RequestMedia)+" -> "+contractMediaFingerprint(newOp.RequestMediaTypes, newOp.RequestMedia)))
		}
		responseStatuses := map[string]bool{}
		for status := range oldOp.ResponseMediaTypes {
			responseStatuses[status] = true
		}
		for status := range newOp.ResponseMediaTypes {
			responseStatuses[status] = true
		}
		for status := range oldOp.ResponseMedia {
			responseStatuses[status] = true
		}
		for status := range newOp.ResponseMedia {
			responseStatuses[status] = true
		}
		for status := range responseStatuses {
			oldMedia := contractMediaFingerprint(oldOp.ResponseMediaTypes[status], oldOp.ResponseMedia[status])
			newMedia := contractMediaFingerprint(newOp.ResponseMediaTypes[status], newOp.ResponseMedia[status])
			if oldMedia != newMedia {
				diffs = append(diffs, newContractDiff(contractID, from.ID, to.ID, key+" response "+status, "RESPONSE_CONTENT_TYPE_CHANGED", "WARNING", oldMedia+" -> "+newMedia))
			}
		}
		if securityFingerprint(oldOp.Security) != securityFingerprint(newOp.Security) {
			diffs = append(diffs, newContractDiff(contractID, from.ID, to.ID, key, "SECURITY_REQUIREMENT_CHANGED", "BREAKING", "declared security requirement changed"))
		}
	}
	for key := range toMap {
		if _, ok := fromMap[key]; !ok {
			diffs = append(diffs, newContractDiff(contractID, from.ID, to.ID, key, "OPERATION_ADDED", "INFO", "declared operation added"))
		}
	}
	sort.Slice(diffs, func(i, j int) bool { return diffs[i].ID < diffs[j].ID })
	return diffs
}

func contractMediaFingerprint(media []string, fallback string) string {
	set := map[string]bool{}
	for _, value := range media {
		if value = normalizeMediaType(value); value != "" {
			set[value] = true
		}
	}
	if len(set) == 0 {
		if value := normalizeMediaType(fallback); value != "" {
			set[value] = true
		}
	}
	keys := make([]string, 0, len(set))
	for value := range set {
		keys = append(keys, value)
	}
	sort.Strings(keys)
	return strings.Join(keys, ",")
}

func observedContentTypeMatches(declared []string, fallback string, observed map[string]int64) bool {
	want := map[string]bool{}
	for _, media := range declared {
		if media = normalizeMediaType(media); media != "" {
			want[media] = true
		}
	}
	if len(want) == 0 {
		if media := normalizeMediaType(fallback); media != "" {
			want[media] = true
		}
	}
	if len(want) == 0 {
		return true
	}
	for media, count := range observed {
		if count > 0 && want[normalizeMediaType(media)] {
			return true
		}
	}
	return false
}

func securitySchemesFingerprint(schemes map[string]map[string]any) string {
	if len(schemes) == 0 {
		return ""
	}
	b, err := json.Marshal(schemes)
	if err != nil {
		return ""
	}
	return fullContractHash(string(b))
}

func newContractDiff(contractID, fromID, toID, location, changeType, severity, description string) ContractDiff {
	return ContractDiff{
		ID:            shortContractHash(contractID + "|" + fromID + "|" + toID + "|" + location + "|" + changeType),
		ContractID:    contractID,
		FromVersionID: fromID,
		ToVersionID:   toID,
		Location:      location,
		ChangeType:    changeType,
		Severity:      severity,
		Description:   description,
	}
}

func compareContractFields(contractID, fromID, toID, location string, oldFields, newFields []ContractSchemaField) []ContractDiff {
	oldMap := map[string]ContractSchemaField{}
	newMap := map[string]ContractSchemaField{}
	for _, f := range oldFields {
		oldMap[f.Path] = f
	}
	for _, f := range newFields {
		newMap[f.Path] = f
	}
	var diffs []ContractDiff
	for path, oldField := range oldMap {
		newField, ok := newMap[path]
		loc := location + "." + path
		if !ok {
			diffs = append(diffs, newContractDiff(contractID, fromID, toID, loc, "FIELD_REMOVED", "BREAKING", "schema field removed"))
			continue
		}
		if oldField.Type != newField.Type {
			severity := "BREAKING"
			if oldField.Type == "integer" && newField.Type == "number" {
				severity = "WARNING"
			}
			diffs = append(diffs, newContractDiff(contractID, fromID, toID, loc, "TYPE_CHANGED", severity, oldField.Type+" -> "+newField.Type))
		}
		if oldField.Format != newField.Format {
			diffs = append(diffs, newContractDiff(contractID, fromID, toID, loc, "FORMAT_CHANGED", "WARNING", oldField.Format+" -> "+newField.Format))
		}
		if strings.Join(oldField.Enum, "\x00") != strings.Join(newField.Enum, "\x00") {
			diffs = append(diffs, newContractDiff(contractID, fromID, toID, loc, "ENUM_CHANGED", "WARNING", "allowed enum values changed"))
		}
		if oldField.Required != newField.Required {
			severity := "INFO"
			if !oldField.Required && newField.Required {
				severity = "BREAKING"
			}
			diffs = append(diffs, newContractDiff(contractID, fromID, toID, loc, "REQUIRED_CHANGED", severity, fmt.Sprintf("required %t -> %t", oldField.Required, newField.Required)))
		}
	}
	for path, newField := range newMap {
		if _, ok := oldMap[path]; ok {
			continue
		}
		severity := "INFO"
		if newField.Required {
			severity = "BREAKING"
		}
		diffs = append(diffs, newContractDiff(contractID, fromID, toID, location+"."+path, "FIELD_ADDED", severity, "schema field added"))
	}
	return diffs
}

func compareContractParameters(contractID, fromID, toID, location string, oldParams, newParams []ContractParameter) []ContractDiff {
	oldMap := map[string]ContractParameter{}
	newMap := map[string]ContractParameter{}
	for _, p := range oldParams {
		oldMap[p.In+"|"+strings.ToLower(p.Name)] = p
	}
	for _, p := range newParams {
		newMap[p.In+"|"+strings.ToLower(p.Name)] = p
	}
	var out []ContractDiff
	for key, oldP := range oldMap {
		newP, ok := newMap[key]
		loc := location + " parameter " + oldP.In + "." + oldP.Name
		if !ok {
			out = append(out, newContractDiff(contractID, fromID, toID, loc, "PARAMETER_REMOVED", "BREAKING", "declared parameter removed"))
			continue
		}
		if oldP.Type != newP.Type {
			out = append(out, newContractDiff(contractID, fromID, toID, loc, "PARAMETER_TYPE_CHANGED", "BREAKING", oldP.Type+" -> "+newP.Type))
		}
		if oldP.Format != newP.Format {
			out = append(out, newContractDiff(contractID, fromID, toID, loc, "PARAMETER_FORMAT_CHANGED", "WARNING", oldP.Format+" -> "+newP.Format))
		}
		if oldP.Required != newP.Required {
			sev := "INFO"
			if !oldP.Required && newP.Required {
				sev = "BREAKING"
			}
			out = append(out, newContractDiff(contractID, fromID, toID, loc, "PARAMETER_REQUIRED_CHANGED", sev, fmt.Sprintf("required %t -> %t", oldP.Required, newP.Required)))
		}
		if strings.Join(oldP.Enum, "\x00") != strings.Join(newP.Enum, "\x00") {
			out = append(out, newContractDiff(contractID, fromID, toID, loc, "PARAMETER_ENUM_CHANGED", "WARNING", "allowed enum values changed"))
		}
	}
	for key, p := range newMap {
		if _, ok := oldMap[key]; !ok {
			sev := "INFO"
			if p.Required {
				sev = "BREAKING"
			}
			out = append(out, newContractDiff(contractID, fromID, toID, location+" parameter "+p.In+"."+p.Name, "PARAMETER_ADDED", sev, "declared parameter added"))
		}
	}
	return out
}

func securityFingerprint(sec []ContractSecurityRequirement) string {
	parts := make([]string, 0, len(sec))
	for _, r := range sec {
		parts = append(parts, fmt.Sprintf("%d:%s:%s", r.Group, r.Scheme, strings.Join(r.Scopes, ",")))
	}
	sort.Strings(parts)
	return strings.Join(parts, "|")
}

func (s *contractStore) saveDiffs(contractID string, diffs []ContractDiff) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.diffs[contractID] = append([]ContractDiff(nil), diffs...)
}

func (s *contractStore) listDiffs(contractID string) []ContractDiff {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]ContractDiff(nil), s.diffs[contractID]...)
}

func driftAgainstLearnedSchema(contractID string, version APIContractVersion, operations []ContractOperation, bindings []OperationBinding, schema *schemaStore, observedSets ...[]apiOperation) []DriftEvent {
	if schema == nil {
		return nil
	}
	var observed []apiOperation
	if len(observedSets) > 0 {
		observed = observedSets[0]
	}
	observedByID := map[string]apiOperation{}
	for _, op := range observed {
		observedByID[op.ID] = op
		if op.Fingerprint != "" {
			observedByID[op.Fingerprint] = op
		}
	}
	bindingByContractOp := map[string]OperationBinding{}
	matchedObserved := map[string]bool{}
	matchedSites := map[string]bool{}
	matchedHosts := map[string]bool{}
	for _, b := range bindings {
		bindingByContractOp[b.ContractOperationID] = b
		if b.Status == "MATCHED" && b.APIOperationID != "" {
			matchedObserved[b.APIOperationID] = true
			if observedOp, ok := observedByID[b.APIOperationID]; ok {
				if observedOp.Site != "" {
					matchedSites[observedOp.Site] = true
				}
				if host := normalizeHost(observedOp.Host); host != "" {
					matchedHosts[host] = true
				}
			}
		}
	}
	now := time.Now().UTC()
	var out []DriftEvent
	for _, op := range operations {
		binding, bound := bindingByContractOp[op.ID]
		if !bound || binding.Status == "UNMATCHED" {
			out = append(out, newDriftEvent(contractID, version.ID, binding.ID, op.Method+" "+op.Path, "MISSING_ENDPOINT", "INFO", "declared endpoint has not been observed", now))
			continue
		}
		if binding.Status != "MATCHED" || binding.APIOperationID == "" {
			continue
		}
		observedOp, haveObserved := observedByID[binding.APIOperationID]
		candidate, _ := schema.findByOperationID(binding.APIOperationID)
		if candidate == nil {
			continue
		}

		declared := map[string]ContractSchemaField{}
		for _, f := range op.RequestFields {
			declared[f.Path] = f
		}
		observedBody := candidateFieldsByLocation(candidate, "body")
		for path, field := range declared {
			obs, exists := observedBody[path]
			if !exists {
				if field.Required {
					out = append(out, newDriftEvent(contractID, version.ID, binding.ID, op.Method+" "+op.Path+"."+path, "MISSING_REQUIRED_FIELD", "WARNING", "required declared field has not been observed", now))
				}
				continue
			}
			dominant := dominantObservedType(obs.TypeCounts)
			if dominant != "" && field.Type != "" && !schemaTypesCompatible(field.Type, dominant) {
				out = append(out, newDriftEvent(contractID, version.ID, binding.ID, op.Method+" "+op.Path+"."+path, "TYPE_DRIFT", "BREAKING", "declared type "+field.Type+" observed as "+dominant, now))
			}
			if len(field.Enum) > 0 && hasEnumOutside(obs.EnumCandidate, field.Enum) {
				out = append(out, newDriftEvent(contractID, version.ID, binding.ID, op.Method+" "+op.Path+"."+path, "ENUM_DRIFT", "WARNING", "observed enum candidate contains values outside the declared enum", now))
			}
		}
		for path := range observedBody {
			if _, exists := declared[path]; !exists {
				out = append(out, newDriftEvent(contractID, version.ID, binding.ID, op.Method+" "+op.Path+"."+path, "UNKNOWN_FIELD", "WARNING", "observed field is not present in declared request schema", now))
			}
		}
		for _, param := range op.Parameters {
			loc := strings.ToLower(param.In)
			obs, exists := observedContractParameter(candidate, op.Path, param)
			if !exists {
				if param.Required {
					out = append(out, newDriftEvent(contractID, version.ID, binding.ID, op.Method+" "+op.Path+" "+loc+"."+param.Name, "MISSING_REQUIRED_FIELD", "WARNING", "required declared parameter has not been observed", now))
				}
				continue
			}
			dominant := dominantObservedType(obs.TypeCounts)
			if param.Type != "" && dominant != "" && !schemaTypesCompatible(param.Type, dominant) {
				out = append(out, newDriftEvent(contractID, version.ID, binding.ID, op.Method+" "+op.Path+" "+loc+"."+param.Name, "TYPE_DRIFT", "BREAKING", "declared parameter type "+param.Type+" observed as "+dominant, now))
			}
			if len(param.Enum) > 0 && hasEnumOutside(obs.EnumCandidate, param.Enum) {
				out = append(out, newDriftEvent(contractID, version.ID, binding.ID, op.Method+" "+op.Path+" "+loc+"."+param.Name, "ENUM_DRIFT", "WARNING", "observed parameter enum candidate contains values outside the declared enum", now))
			}
		}
		if haveObserved && (op.RequestMedia != "" || len(op.RequestMediaTypes) > 0) && sumCounts(observedOp.ContentTypes) > 0 && !observedContentTypeMatches(op.RequestMediaTypes, op.RequestMedia, observedOp.ContentTypes) {
			out = append(out, newDriftEvent(contractID, version.ID, binding.ID, op.Method+" "+op.Path, "CONTENT_TYPE_DRIFT", "WARNING", "observed request content types do not intersect declared "+contractMediaFingerprint(op.RequestMediaTypes, op.RequestMedia), now))
		}
		if haveObserved && len(op.Security) > 0 && observedOp.Samples > 0 {
			if authRequirementDrift(version.SecuritySchemes, op.Security, observedOp.AuthObserved) {
				out = append(out, newDriftEvent(contractID, version.ID, binding.ID, op.Method+" "+op.Path, "AUTH_DRIFT", "WARNING", "observed authentication does not satisfy the declared security scheme family", now))
			}
		}
	}
	for _, op := range observed {
		if op.Ignored {
			continue
		}
		if matchedObserved[op.ID] || matchedObserved[op.Fingerprint] {
			continue
		}
		// A contract has no trustworthy application scope until at least one
		// operation is bound. Once a scope exists, only report shadow endpoints
		// from the same WAF site/host set; otherwise unrelated virtual hosts would
		// be mislabeled as undeclared endpoints for every imported contract.
		if len(matchedSites) == 0 && len(matchedHosts) == 0 {
			continue
		}
		inScope := matchedSites[op.Site]
		if !inScope {
			inScope = matchedHosts[normalizeHost(op.Host)]
		}
		if !inScope {
			continue
		}
		out = append(out, newDriftEvent(contractID, version.ID, "", op.Method+" "+op.Path, "UNDECLARED_ENDPOINT", "WARNING", "observed endpoint is not present in the declared contract", now))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func candidateFieldsByLocation(candidate *SchemaCandidate, location string) map[string]SchemaFieldObservation {
	out := map[string]SchemaFieldObservation{}
	if candidate == nil {
		return out
	}
	for _, f := range candidate.Fields {
		loc := f.Location
		if loc == "" {
			loc = "body"
		}
		if strings.EqualFold(loc, location) {
			out[f.Path] = f
		}
	}
	return out
}

func observedContractParameter(candidate *SchemaCandidate, operationPath string, param ContractParameter) (SchemaFieldObservation, bool) {
	fields := candidateFieldsByLocation(candidate, strings.ToLower(param.In))
	if strings.EqualFold(param.In, "path") {
		if ordinal := contractPathParameterOrdinal(operationPath, param.Name); ordinal > 0 {
			obs, ok := fields[fmt.Sprintf("param_%d", ordinal)]
			return obs, ok
		}
	}
	if strings.EqualFold(param.In, "header") {
		for name, obs := range fields {
			if strings.EqualFold(name, param.Name) {
				return obs, true
			}
		}
		return SchemaFieldObservation{}, false
	}
	obs, ok := fields[param.Name]
	return obs, ok
}

func contractPathParameterOrdinal(path, name string) int {
	ordinal := 0
	for _, segment := range strings.Split(path, "/") {
		if strings.HasPrefix(segment, "{") && strings.HasSuffix(segment, "}") {
			ordinal++
			if strings.TrimSuffix(strings.TrimPrefix(segment, "{"), "}") == name {
				return ordinal
			}
		}
	}
	return 0
}

func hasEnumOutside(observed, declared []string) bool {
	if len(observed) == 0 || len(declared) == 0 {
		return false
	}
	allowed := map[string]bool{}
	for _, v := range declared {
		allowed[v] = true
	}
	for _, v := range observed {
		if !allowed[v] {
			return true
		}
	}
	return false
}

func authRequirementDrift(schemes map[string]map[string]any, reqs []ContractSecurityRequirement, observed map[string]int64) bool {
	type groupState struct {
		required    map[string]bool
		unsupported bool
		anonymous   bool
	}
	groups := map[int]*groupState{}
	for _, req := range reqs {
		g := groups[req.Group]
		if g == nil {
			g = &groupState{required: map[string]bool{}}
			groups[req.Group] = g
		}
		if req.Scheme == "" {
			g.anonymous = true
			continue
		}
		family := contractSecurityFamily(schemes[req.Scheme])
		if family == "" {
			g.unsupported = true
			continue
		}
		g.required[family] = true
	}
	if len(groups) == 0 {
		return false
	}
	for _, g := range groups {
		if g.anonymous {
			return false
		}
	}
	for observedScheme, count := range observed {
		if count <= 0 {
			continue
		}
		families := authObservationFamilies(observedScheme)
		for _, g := range groups {
			if g.unsupported || len(g.required) == 0 {
				continue
			}
			satisfied := true
			for family := range g.required {
				if !families[family] {
					satisfied = false
					break
				}
			}
			if satisfied {
				return false
			}
		}
	}
	// If an OR-alternative uses a security scheme family this telemetry layer
	// cannot identify, absence cannot be proven and emitting AUTH_DRIFT would be
	// a false-positive assertion. API-5 can make this stricter with verified
	// identity/claim context.
	for _, g := range groups {
		if g.unsupported {
			return false
		}
	}
	return true
}

func contractSecurityFamily(def map[string]any) string {
	typ := strings.ToLower(strings.TrimSpace(stringValue(def["type"])))
	switch typ {
	case "http":
		scheme := strings.ToLower(strings.TrimSpace(stringValue(def["scheme"])))
		if scheme == "bearer" || scheme == "basic" {
			return scheme
		}
	case "apikey":
		return "apikey"
	case "mutualtls":
		return "mtls"
	case "oauth2", "openidconnect":
		return "bearer"
	}
	return ""
}

func authObservationFamilies(s string) map[string]bool {
	out := map[string]bool{}
	for _, family := range strings.Split(normalizeAuthScheme(s), "+") {
		if family != "" && family != "none" {
			out[family] = true
		}
	}
	return out
}

func schemaTypesCompatible(declared, observed string) bool {
	if declared == observed {
		return true
	}
	return declared == "number" && observed == "integer"
}

func newDriftEvent(contractID, versionID, bindingID, location, typ, severity, description string, now time.Time) DriftEvent {
	return DriftEvent{
		ID:                 shortContractHash(contractID + "|" + versionID + "|" + location + "|" + typ),
		ContractID:         contractID,
		ContractVersionID:  versionID,
		OperationBindingID: bindingID,
		Location:           location,
		Type:               typ,
		Severity:           severity,
		Description:        description,
		DetectedAt:         now,
	}
}

func (a *adminServer) handleContractImport(w http.ResponseWriter, r *http.Request) {
	if a == nil || a.srv == nil || a.srv.contracts == nil {
		http.Error(w, "contract store unavailable", http.StatusServiceUnavailable)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 8<<20)
	var req struct {
		Name        string `json:"name"`
		Application string `json:"application"`
		Document    string `json:"document"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid contract request", http.StatusBadRequest)
		return
	}
	version, err := a.srv.contracts.importDocument(req.Name, req.Application, req.Document)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}
	if err := a.srv.contracts.save(a.srv.configPath); err != nil {
		http.Error(w, "contract imported but persistence failed: "+err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSONCode(w, http.StatusCreated, version)
}

func (a *adminServer) handleContracts(w http.ResponseWriter, _ *http.Request) {
	if a == nil || a.srv == nil || a.srv.contracts == nil {
		writeJSON(w, []APIContract{})
		return
	}
	writeJSON(w, a.srv.contracts.list())
}

func (a *adminServer) handleContractDetail(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	contract, ok := a.srv.contracts.get(id)
	if !ok {
		http.NotFound(w, r)
		return
	}
	writeJSON(w, contract)
}

func (a *adminServer) handleContractVersions(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if _, ok := a.srv.contracts.get(id); !ok {
		http.NotFound(w, r)
		return
	}
	writeJSON(w, a.srv.contracts.listVersions(id))
}

func (a *adminServer) handleContractMatch(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	version, ops, ok := a.srv.contracts.latestVersion(id)
	if !ok {
		http.NotFound(w, r)
		return
	}
	var observed []apiOperation
	if a.srv.apiOps != nil {
		observed = a.srv.apiOps.snapshot()
	}
	bindings := matchContractOperations(version, ops, observed)
	a.srv.contracts.saveBindings(version.ID, bindings)
	if a.srv.objectLocators != nil {
		a.srv.objectLocators.refreshContractSources(a.srv.contracts)
	}
	if err := a.srv.contracts.save(a.srv.configPath); err != nil {
		http.Error(w, "binding persistence failed: "+err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, bindings)
}

func (a *adminServer) handleContractBindings(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if _, ok := a.srv.contracts.get(id); !ok {
		http.NotFound(w, r)
		return
	}
	writeJSON(w, a.srv.contracts.listBindings(id))
}

func (a *adminServer) handleContractCompare(w http.ResponseWriter, r *http.Request) {
	contractID := r.PathValue("id")
	var req struct {
		FromVersionID string `json:"from_version_id"`
		ToVersionID   string `json:"to_version_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid compare request", http.StatusBadRequest)
		return
	}
	from, fromOps, ok := a.srv.contracts.version(req.FromVersionID)
	if !ok || from.ContractID != contractID {
		http.Error(w, "unknown from_version_id", http.StatusBadRequest)
		return
	}
	to, toOps, ok := a.srv.contracts.version(req.ToVersionID)
	if !ok || to.ContractID != contractID {
		http.Error(w, "unknown to_version_id", http.StatusBadRequest)
		return
	}
	diffs := compareContractVersions(contractID, from, fromOps, to, toOps)
	a.srv.contracts.saveDiffs(contractID, diffs)
	if err := a.srv.contracts.save(a.srv.configPath); err != nil {
		http.Error(w, "diff persistence failed: "+err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, diffs)
}

func (a *adminServer) handleContractDiffs(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if _, ok := a.srv.contracts.get(id); !ok {
		http.NotFound(w, r)
		return
	}
	writeJSON(w, a.srv.contracts.listDiffs(id))
}

func (a *adminServer) handleContractDrift(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	version, ops, ok := a.srv.contracts.latestVersion(id)
	if !ok {
		http.NotFound(w, r)
		return
	}
	var observed []apiOperation
	if a.srv.apiOps != nil {
		observed = a.srv.apiOps.snapshot()
	}
	bindings := matchContractOperations(version, ops, observed)
	a.srv.contracts.saveBindings(version.ID, bindings)
	if a.srv.objectLocators != nil {
		a.srv.objectLocators.refreshContractSources(a.srv.contracts)
	}
	if err := a.srv.contracts.save(a.srv.configPath); err != nil {
		http.Error(w, "binding persistence failed: "+err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, driftAgainstLearnedSchema(id, version, ops, bindings, a.srv.schema, observed))
}
