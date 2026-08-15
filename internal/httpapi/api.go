// Package httpapi exposes the REST + SSE API backing the embedded SPA.
package httpapi

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/alexindigo/localsend-nas/internal/config"
	"github.com/alexindigo/localsend-nas/internal/discovery"
	"github.com/alexindigo/localsend-nas/internal/shares"
	"github.com/alexindigo/localsend-nas/internal/transfer"
	"github.com/alexindigo/localsend-nas/web"
)

type API struct {
	cfg   *config.Config
	store *shares.Store
	disc  *discovery.Discovery
	tm    *transfer.Manager
}

// Handler builds the HTTP handler: /api/* routes plus the embedded SPA.
func New(cfg *config.Config, store *shares.Store, disc *discovery.Discovery, tm *transfer.Manager) http.Handler {
	a := &API{cfg: cfg, store: store, disc: disc, tm: tm}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/shares", a.handleShares)
	mux.HandleFunc("GET /api/list", a.handleList)
	mux.HandleFunc("GET /api/devices", a.handleDevices)
	mux.HandleFunc("POST /api/devices", a.handleAddDevice)
	mux.HandleFunc("DELETE /api/devices/{fingerprint}", a.handleDeleteDevice)
	mux.HandleFunc("POST /api/send", a.handleSend)
	mux.HandleFunc("GET /api/transfers", a.handleTransfers)
	mux.HandleFunc("GET /api/transfers/{id}/events", a.handleTransferEvents)
	mux.HandleFunc("POST /api/transfers/{id}/cancel", a.handleCancelTransfer)
	mux.HandleFunc("GET /", a.handleSPA)
	return mux
}

// --- shares ---

func (a *API) handleShares(w http.ResponseWriter, r *http.Request) {
	type share struct {
		Name string `json:"name"`
		Path string `json:"path"`
	}
	out := []share{}
	for _, name := range a.store.Names() {
		p, _ := a.store.Path(name)
		out = append(out, share{Name: name, Path: p})
	}
	writeJSON(w, http.StatusOK, out)
}

func (a *API) handleList(w http.ResponseWriter, r *http.Request) {
	entries, err := a.store.List(r.URL.Query().Get("share"), r.URL.Query().Get("path"))
	if err != nil {
		switch {
		case errors.Is(err, shares.ErrUnknownShare), errors.Is(err, shares.ErrNotFound):
			writeError(w, http.StatusNotFound, err)
		case errors.Is(err, shares.ErrOutsideRoot):
			writeError(w, http.StatusForbidden, err)
		default:
			writeError(w, http.StatusBadRequest, err)
		}
		return
	}
	writeJSON(w, http.StatusOK, entries)
}

// --- devices ---

type deviceDTO struct {
	Fingerprint string    `json:"fingerprint"`
	Alias       string    `json:"alias"`
	IP          string    `json:"ip"`
	Port        int       `json:"port"`
	DeviceType  string    `json:"deviceType"`
	DeviceModel string    `json:"deviceModel"`
	LastSeen    time.Time `json:"lastSeen"`
	Manual      bool      `json:"manual"`
}

func toDTO(d discovery.Device) deviceDTO {
	return deviceDTO{
		Fingerprint: d.Info.Fingerprint,
		Alias:       d.Info.Alias,
		IP:          d.IP,
		Port:        d.Info.Port,
		DeviceType:  d.Info.DeviceType,
		DeviceModel: d.Info.DeviceModel,
		LastSeen:    d.LastSeen,
		Manual:      d.Manual,
	}
}

func (a *API) handleDevices(w http.ResponseWriter, r *http.Request) {
	out := []deviceDTO{}
	for _, d := range a.disc.Snapshot() {
		out = append(out, toDTO(d))
	}
	writeJSON(w, http.StatusOK, out)
}

func (a *API) handleAddDevice(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Address string `json:"address"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || strings.TrimSpace(body.Address) == "" {
		writeError(w, http.StatusBadRequest, errors.New("body must be {\"address\":\"host[:port]\"}"))
		return
	}
	dev, err := a.disc.Add(r.Context(), body.Address)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, toDTO(*dev))
}

func (a *API) handleDeleteDevice(w http.ResponseWriter, r *http.Request) {
	fp := r.PathValue("fingerprint")
	if _, ok := a.disc.Get(fp); !ok {
		writeError(w, http.StatusNotFound, errors.New("unknown device"))
		return
	}
	if !a.disc.Remove(fp) {
		writeError(w, http.StatusConflict, errors.New("not a manual target"))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// --- transfers ---

func (a *API) handleSend(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Target string             `json:"target"`
		Items  []transfer.ItemRef `json:"items"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Errorf("invalid body: %w", err))
		return
	}
	jobID, err := a.tm.Send(body.Target, body.Items)
	if err != nil {
		status := http.StatusBadRequest
		if strings.HasPrefix(err.Error(), "unknown target") {
			status = http.StatusNotFound
		}
		writeError(w, status, err)
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]string{"jobId": jobID})
}

func (a *API) handleTransfers(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, a.tm.List())
}

func (a *API) handleCancelTransfer(w http.ResponseWriter, r *http.Request) {
	if !a.tm.Cancel(r.PathValue("id")) {
		writeError(w, http.StatusConflict, errors.New("job not active or unknown"))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// handleTransferEvents streams job snapshots as SSE data frames until the
// job reaches a terminal state or the client disconnects.
func (a *API) handleTransferEvents(w http.ResponseWriter, r *http.Request) {
	ch, unsubscribe, ok := a.tm.Subscribe(r.PathValue("id"))
	if !ok {
		writeError(w, http.StatusNotFound, errors.New("unknown job"))
		return
	}
	defer unsubscribe()

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, errors.New("streaming unsupported"))
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)

	for {
		select {
		case <-r.Context().Done():
			return
		case job, open := <-ch:
			if !open {
				return
			}
			data, err := json.Marshal(job)
			if err != nil {
				continue
			}
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
			switch job.State {
			case transfer.StateDone, transfer.StateFailed, transfer.StateCancelled:
				return
			}
		}
	}
}

// --- SPA ---

// handleSPA serves the embedded single-page app with an index.html
// fallback for unknown paths.
func (a *API) handleSPA(w http.ResponseWriter, r *http.Request) {
	p := strings.TrimPrefix(r.URL.Path, "/")
	if p == "" {
		p = "index.html"
	}
	f, err := web.FS.Open(p)
	if err != nil {
		// SPA fallback (client-side views share one document).
		p = "index.html"
		f, err = web.FS.Open(p)
		if err != nil {
			writeError(w, http.StatusNotFound, errors.New("asset missing"))
			return
		}
	}
	defer f.Close()
	fi, err := f.Stat()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	content, err := io.ReadAll(f)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if ct := mimeByExt(p); ct != "" {
		w.Header().Set("Content-Type", ct)
	}
	http.ServeContent(w, r, p, fi.ModTime(), bytes.NewReader(content))
}

func mimeByExt(p string) string {
	switch {
	case strings.HasSuffix(p, ".html"):
		return "text/html; charset=utf-8"
	case strings.HasSuffix(p, ".css"):
		return "text/css; charset=utf-8"
	case strings.HasSuffix(p, ".js"):
		return "text/javascript; charset=utf-8"
	}
	return ""
}

// --- helpers ---

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, map[string]string{"error": err.Error()})
}
