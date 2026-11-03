package ipc

import (
	"encoding/json"
	"net"
	"net/http"
	"os"
	"syscall"
	"time"

	"xtui/core"
)

type Server struct {
	sockPath string
	listener net.Listener
	httpSrv  *http.Server
	mgr      *core.Manager
}

func StartIPCServer(sockPath string, mgr *core.Manager) (*Server, error) {
	// Clean up stale socket file if it exists
	if _, err := os.Stat(sockPath); err == nil {
		_ = os.Remove(sockPath)
	}

	listener, err := net.Listen("unix", sockPath)
	if err != nil {
		return nil, err
	}
	_ = os.Chmod(sockPath, 0666)

	mux := http.NewServeMux()
	s := &Server{
		sockPath: sockPath,
		listener: listener,
		mgr:      mgr,
	}

	mux.HandleFunc("/api/status", s.handleStatus)
	mux.HandleFunc("/api/user/add", s.handleAddUser)
	mux.HandleFunc("/api/user/edit", s.handleEditUser)
	mux.HandleFunc("/api/user/delete", s.handleDeleteUser)
	mux.HandleFunc("/api/user/toggle", s.handleToggleUser)
	mux.HandleFunc("/api/user/reset", s.handleResetUser)
	mux.HandleFunc("/api/user/reset-all", s.handleResetAll)
	mux.HandleFunc("/api/reality", s.handleUpdateReality)
	mux.HandleFunc("/api/reality/regen", s.handleRegenKeys)
	mux.HandleFunc("/api/vless", s.handleUpdateVLESS)
	mux.HandleFunc("/api/core/restart", s.handleRestart)
	mux.HandleFunc("/api/core/stop", s.handleStop)
	mux.HandleFunc("/api/core/start", s.handleStart)

	s.httpSrv = &http.Server{Handler: mux}

	go func() {
		_ = s.httpSrv.Serve(listener)
	}()

	return s, nil
}

func (s *Server) Close() error {
	var err error
	if s.httpSrv != nil {
		err = s.httpSrv.Close()
	}
	if s.listener != nil {
		_ = s.listener.Close()
	}
	_ = os.Remove(s.sockPath)
	return err
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	cfg := s.mgr.GetConfig()
	statsMap := make(map[string]core.UserStats)
	for _, u := range cfg.Users {
		statsMap[u.Email] = s.mgr.GetUserStats(u.Email)
	}

	resp := StatusResponse{
		Running:   s.mgr.IsRunning(),
		StartTime: s.mgr.GetStartTime(),
		Config:    cfg,
		Stats:     statsMap,
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleAddUser(w http.ResponseWriter, r *http.Request) {
	var req AddUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, ActionResponse{Success: false, Error: err.Error()})
		return
	}
	u, err := s.mgr.AddUser(req.Email, req.LimitGB, req.ValidDays, req.CustomUUID, req.Flow)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, ActionResponse{Success: false, Error: err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"success": true, "user": u})
}

func (s *Server) handleEditUser(w http.ResponseWriter, r *http.Request) {
	var req EditUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, ActionResponse{Success: false, Error: err.Error()})
		return
	}
	err := s.mgr.EditUser(req.Index, req.Email, req.UUID, req.Flow, req.LimitGB, req.ValidDays, req.Enabled)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, ActionResponse{Success: false, Error: err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, ActionResponse{Success: true})
}

func (s *Server) handleDeleteUser(w http.ResponseWriter, r *http.Request) {
	var req IndexRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, ActionResponse{Success: false, Error: err.Error()})
		return
	}
	err := s.mgr.DeleteUser(req.Index)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, ActionResponse{Success: false, Error: err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, ActionResponse{Success: true})
}

func (s *Server) handleToggleUser(w http.ResponseWriter, r *http.Request) {
	var req IndexRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, ActionResponse{Success: false, Error: err.Error()})
		return
	}
	err := s.mgr.ToggleUser(req.Index)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, ActionResponse{Success: false, Error: err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, ActionResponse{Success: true})
}

func (s *Server) handleResetUser(w http.ResponseWriter, r *http.Request) {
	var req IndexRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, ActionResponse{Success: false, Error: err.Error()})
		return
	}
	err := s.mgr.ResetUserTraffic(req.Index)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, ActionResponse{Success: false, Error: err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, ActionResponse{Success: true})
}

func (s *Server) handleResetAll(w http.ResponseWriter, r *http.Request) {
	err := s.mgr.ResetAllTraffic()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, ActionResponse{Success: false, Error: err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, ActionResponse{Success: true})
}

func (s *Server) handleUpdateReality(w http.ResponseWriter, r *http.Request) {
	var req UpdateRealityRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, ActionResponse{Success: false, Error: err.Error()})
		return
	}
	err := s.mgr.UpdateReality(req.Dest, req.ServerNames, req.PrivateKey, req.PublicKey, req.ShortIds, req.SpiderX)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, ActionResponse{Success: false, Error: err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, ActionResponse{Success: true})
}

func (s *Server) handleRegenKeys(w http.ResponseWriter, r *http.Request) {
	priv, pub, err := s.mgr.RegenerateRealityKeys()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, ActionResponse{Success: false, Error: err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, KeyPairResponse{PrivateKey: priv, PublicKey: pub})
}

func (s *Server) handleUpdateVLESS(w http.ResponseWriter, r *http.Request) {
	var req UpdateVLESSRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, ActionResponse{Success: false, Error: err.Error()})
		return
	}
	err := s.mgr.UpdateVLESS(req.Port, req.Listen, req.PublicHost, req.Fingerprint)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, ActionResponse{Success: false, Error: err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, ActionResponse{Success: true})
}

func (s *Server) handleRestart(w http.ResponseWriter, r *http.Request) {
	go func() {
		_ = s.mgr.Restart()
	}()
	writeJSON(w, http.StatusOK, ActionResponse{Success: true})
}

func (s *Server) handleStop(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, ActionResponse{Success: true})
	go func() {
		time.Sleep(100 * time.Millisecond)
		p, err := os.FindProcess(os.Getpid())
		if err == nil {
			_ = p.Signal(syscall.SIGTERM)
		}
	}()
}

func (s *Server) handleStart(w http.ResponseWriter, r *http.Request) {
	go func() {
		_ = s.mgr.Start()
	}()
	writeJSON(w, http.StatusOK, ActionResponse{Success: true})
}
