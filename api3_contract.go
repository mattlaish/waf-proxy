package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"
)

// API-3 OpenAPI Contract Management foundation.
// Contract intelligence only: no enforcement or blocking.
type APIContract struct {
	ID string `json:"id"`
	Name string `json:"name"`
	Application string `json:"application"`
	CreatedAt time.Time `json:"created_at"`
}

type APIContractVersion struct {
	ID string `json:"id"`
	ContractID string `json:"contract_id"`
	Version string `json:"version"`
	ContentHash string `json:"content_hash"`
	CreatedAt time.Time `json:"created_at"`
}

type ContractOperation struct {
	ID string `json:"id"`
	VersionID string `json:"version_id"`
	Method string `json:"method"`
	Path string `json:"path"`
	OperationID string `json:"operation_id"`
}

type contractStore struct {
	mu sync.RWMutex
	contracts map[string]APIContract
	versions map[string]APIContractVersion
	operations map[string][]ContractOperation
}

func newContractStore() *contractStore {
	return &contractStore{contracts: map[string]APIContract{}, versions: map[string]APIContractVersion{}, operations: map[string][]ContractOperation{}}
}

func (s *contractStore) importDocument(name, application, document string) APIContractVersion {
	h := sha256.Sum256([]byte(document))
	v := APIContractVersion{ID: hex.EncodeToString(h[:8]), ContractID: name, Version: "imported", ContentHash: hex.EncodeToString(h[:]), CreatedAt: time.Now().UTC()}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.contracts[name] = APIContract{ID:name, Name:name, Application:application, CreatedAt:time.Now().UTC()}
	s.versions[v.ID]=v
	return v
}

func (s *contractStore) list() []APIContract {
	s.mu.RLock(); defer s.mu.RUnlock()
	out:=make([]APIContract,0,len(s.contracts)); for _,c:=range s.contracts { out=append(out,c) }; return out
}

func (a *adminServer) handleContractImport(w http.ResponseWriter, r *http.Request) {
	var req struct { Name string `json:"name"`; Application string `json:"application"`; Document string `json:"document"` }
	if err:=json.NewDecoder(r.Body).Decode(&req); err!=nil || strings.TrimSpace(req.Name)=="" || req.Document=="" { http.Error(w,"invalid contract",400); return }
	v:=a.srv.contracts.importDocument(req.Name, req.Application, req.Document)
	writeJSON(w,200,v)
}

func (a *adminServer) handleContracts(w http.ResponseWriter, r *http.Request) { writeJSON(w,200,a.srv.contracts.list()) }
