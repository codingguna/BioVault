package web

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"fp-manager/internal/matching"
	"fp-manager/internal/scanner"
	"fp-manager/internal/storage"
)

type Server struct {
	sc      scanner.Scanner
	db      *storage.DB
	matcher *matching.Service
	mux     *http.ServeMux
	mu      sync.Mutex
}

func NewServer(sc scanner.Scanner, db *storage.DB) *Server {
	s := &Server{
		sc:      sc,
		db:      db,
		matcher: matching.NewService(sc, db),
		mux:     http.NewServeMux(),
	}
	s.routes()
	return s
}

func (s *Server) Handler() http.Handler {
	return s.mux
}

func (s *Server) routes() {
	s.mux.HandleFunc("/", s.handleIndex)
	s.mux.HandleFunc("/api/scanner/status", s.handleScannerStatus)
	s.mux.HandleFunc("/api/scanner/reconnect", s.handleScannerReconnect)
	s.mux.HandleFunc("/api/scanner/capture", s.handleScanCapture)
	s.mux.HandleFunc("/api/users", s.handleUsers)
	s.mux.HandleFunc("/api/users/", s.handleUserDetail)
	s.mux.HandleFunc("/api/enroll", s.handleEnroll)
	s.mux.HandleFunc("/api/verify", s.handleVerify)
	s.mux.HandleFunc("/api/identify", s.handleIdentify)
	s.mux.HandleFunc("/api/audit", s.handleAudit)
	s.mux.HandleFunc("/api/windows/bind", s.handleWindowsBind)
	s.mux.HandleFunc("/api/windows/status", s.handleWindowsStatus)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func (s *Server) handleScannerStatus(w http.ResponseWriter, r *http.Request) {
	conn := s.sc.IsConnected()
	info, _ := s.sc.DeviceInfo()

	writeJSON(w, http.StatusOK, map[string]any{
		"connected": conn,
		"device":    info,
	})
}

func (s *Server) handleScannerReconnect(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	_ = s.sc.Close()
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	if err := s.sc.Connect(ctx); err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to connect scanner: %v", err))
		return
	}

	info, _ := s.sc.DeviceInfo()
	writeJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"device":  info,
	})
}

func (s *Server) handleScanCapture(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	timeoutSec := 10
	if tStr := r.URL.Query().Get("timeout"); tStr != "" {
		if t, err := strconv.Atoi(tStr); err == nil && t > 0 {
			timeoutSec = t
		}
	}

	minQuality := 50
	if qStr := r.URL.Query().Get("min_quality"); qStr != "" {
		if q, err := strconv.Atoi(qStr); err == nil && q > 0 {
			minQuality = q
		}
	}

	ctx, cancel := context.WithTimeout(r.Context(), time.Duration(timeoutSec+2)*time.Second)
	defer cancel()

	res, err := s.sc.AutoCapture(ctx, timeoutSec, minQuality)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"quality":       res.Quality,
		"nfiq":          res.NFIQ,
		"iso_base64":    base64.StdEncoding.EncodeToString(res.ISOTemplate),
		"ansi_base64":   base64.StdEncoding.EncodeToString(res.ANSITemplate),
		"bitmap_base64": res.BitmapBase64,
	})
}

func (s *Server) handleUsers(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		users, err := s.db.ListUsers(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, users)

	case http.MethodPost:
		var req struct {
			DisplayName string `json:"display_name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "Invalid request body")
			return
		}
		if strings.TrimSpace(req.DisplayName) == "" {
			writeError(w, http.StatusBadRequest, "Display name is required")
			return
		}

		user, err := s.db.CreateUser(r.Context(), strings.TrimSpace(req.DisplayName))
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, user)

	default:
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

func (s *Server) handleUserDetail(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 3 {
		writeError(w, http.StatusBadRequest, "User ID missing")
		return
	}
	userID := parts[2]

	switch r.Method {
	case http.MethodGet:
		user, err := s.db.GetUser(r.Context(), userID)
		if err != nil {
			writeError(w, http.StatusNotFound, "User not found")
			return
		}
		templates, _ := s.db.GetUserTemplates(r.Context(), userID)
		writeJSON(w, http.StatusOK, map[string]any{
			"user":      user,
			"templates": templates,
		})

	case http.MethodDelete:
		if err := s.db.DeleteUser(r.Context(), userID); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]bool{"deleted": true})

	default:
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

func (s *Server) handleEnroll(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req struct {
		UserID      string `json:"user_id"`
		FingerLabel string `json:"finger_label"`
		ISOBase64   string `json:"iso_base64"`
		Quality     int    `json:"quality"`
		NFIQ        int    `json:"nfiq"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.UserID == "" || req.FingerLabel == "" || req.ISOBase64 == "" {
		writeError(w, http.StatusBadRequest, "user_id, finger_label, and iso_base64 are required")
		return
	}

	isoBytes, err := base64.StdEncoding.DecodeString(req.ISOBase64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid base64 in iso_base64")
		return
	}

	tmpl, err := s.db.SaveTemplate(r.Context(), req.UserID, req.FingerLabel, "ISO_19794_2", 9025, isoBytes, req.Quality, req.NFIQ)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, tmpl)
}

func (s *Server) handleVerify(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req struct {
		UserID    string `json:"user_id"`
		ISOBase64 string `json:"iso_base64"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	var isoBytes []byte
	var err error

	if req.ISOBase64 != "" {
		isoBytes, err = base64.StdEncoding.DecodeString(req.ISOBase64)
		if err != nil {
			writeError(w, http.StatusBadRequest, "Invalid base64 in iso_base64")
			return
		}
	} else {
		// Scan live
		ctx, cancel := context.WithTimeout(r.Context(), 12*time.Second)
		defer cancel()
		captureRes, err := s.sc.AutoCapture(ctx, 10, 50)
		if err != nil {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("Live scan failed: %v", err))
			return
		}
		isoBytes = captureRes.ISOTemplate
	}

	res, err := s.matcher.VerifyUser(r.Context(), req.UserID, isoBytes)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, res)
}

func (s *Server) handleIdentify(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req struct {
		ISOBase64 string `json:"iso_base64"`
	}

	_ = json.NewDecoder(r.Body).Decode(&req)

	var isoBytes []byte
	var err error

	if req.ISOBase64 != "" {
		isoBytes, err = base64.StdEncoding.DecodeString(req.ISOBase64)
		if err != nil {
			writeError(w, http.StatusBadRequest, "Invalid base64 in iso_base64")
			return
		}
	} else {
		// Scan live
		ctx, cancel := context.WithTimeout(r.Context(), 12*time.Second)
		defer cancel()
		captureRes, err := s.sc.AutoCapture(ctx, 10, 50)
		if err != nil {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("Live scan failed: %v", err))
			return
		}
		isoBytes = captureRes.ISOTemplate
	}

	res, err := s.matcher.IdentifyUser(r.Context(), isoBytes)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, res)
}

func (s *Server) handleAudit(w http.ResponseWriter, r *http.Request) {
	limit := 50
	if lStr := r.URL.Query().Get("limit"); lStr != "" {
		if l, err := strconv.Atoi(lStr); err == nil && l > 0 {
			limit = l
		}
	}

	events, err := s.db.ListAuditEvents(r.Context(), limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, events)
}

func (s *Server) handleWindowsBind(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req struct {
		UserID        string `json:"user_id"`
		WindowsUser   string `json:"windows_user"`
		WindowsDomain string `json:"windows_domain"`
		Password      string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.UserID == "" || req.WindowsUser == "" || req.Password == "" {
		writeError(w, http.StatusBadRequest, "user_id, windows_user, and password are required")
		return
	}

	if req.WindowsDomain == "" {
		req.WindowsDomain = "."
	}

	err := s.db.SaveWindowsCredential(r.Context(), req.UserID, req.WindowsUser, req.WindowsDomain, req.Password)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"message": fmt.Sprintf("Bound to Windows account %s\\%s", req.WindowsDomain, req.WindowsUser),
	})
}

func (s *Server) handleWindowsStatus(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		writeError(w, http.StatusBadRequest, "user_id is required")
		return
	}

	cred, err := s.db.GetWindowsCredential(r.Context(), userID)
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]any{"bound": false})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"bound":          true,
		"windows_user":   cred.WindowsUser,
		"windows_domain": cred.WindowsDomain,
		"updated_at":     cred.UpdatedAt,
	})
}
