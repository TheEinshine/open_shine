package web

import (
	"net/http"
	"strconv"

	"github.com/TheEinshine/open_shine/db"
)

func (s *Server) handleListSubscribers(w http.ResponseWriter, r *http.Request) {
	subs, err := s.store.ListSubscribers()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if subs == nil {
		subs = []db.Subscriber{}
	}
	writeJSON(w, http.StatusOK, subs)
}

func (s *Server) handleAddSubscriber(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Email string `json:"email"`
	}
	if !readJSON(w, r, &in) {
		return
	}
	if in.Email == "" {
		writeError(w, http.StatusBadRequest, "email is required")
		return
	}
	if err := s.store.AddSubscriber(in.Email); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func (s *Server) handleDeleteSubscriber(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := s.store.DeleteSubscriber(id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusOK)
}
