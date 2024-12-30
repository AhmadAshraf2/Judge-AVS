package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/BitDSM/BitDSM-Node/address"
	"github.com/BitDSM/BitDSM-Node/utils"
	"github.com/gorilla/mux"
)

type Service struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Status      string `json:"status"`
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func Server() {
	fmt.Println("starting API server")
	router := mux.NewRouter()
	router.Use(corsMiddleware)

	router.HandleFunc("/eigen/node", NodeHandler).Methods("GET")
	router.HandleFunc("/eigen/node/health", HealthCheckHandler).Methods("GET")
	router.HandleFunc("/eigen/node/services", ServicesHandler).Methods("GET")
	router.HandleFunc("/eigen/node/services/{service_ID}/health", ServiceHealthHandler).Methods("GET")
	router.HandleFunc("/eigen/get_address", GetAddressHandler).Methods("POST")
	http.ListenAndServe(":8080", router)
}

func GetAddressHandler(w http.ResponseWriter, r *http.Request) {
	var request struct {
		PubKey string `json:"pubKey"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	if !utils.IsValidBtcPubKey(request.PubKey) {
		http.Error(w, "Invalid Bitcoin public key", http.StatusBadRequest)
		return
	}

	newAddress, script, err := address.GenerateSimpleMultisigAddress(request.PubKey, "")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}

	response := map[string]string{
		"newAddress":    newAddress,
		"addressScript": script,
	}

	responseJSON, err := json.Marshal(response)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(responseJSON))
	return
}

func HealthCheckHandler(w http.ResponseWriter, r *http.Request) {
	service1 := runHealthCheck()
	if service1 {
		w.WriteHeader(http.StatusOK)
		return
	} else if !service1 {
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}

	w.WriteHeader(http.StatusPartialContent)
	return
}

func NodeHandler(w http.ResponseWriter, r *http.Request) {
	nodeInfo := map[string]string{
		"node_name":    "Update this",
		"spec_version": "v0.0.1",
		"node_version": "v1.0.0",
	}

	jsonResponse, err := json.Marshal(nodeInfo)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonResponse)
	return
}

func ServicesHandler(w http.ResponseWriter, r *http.Request) {
	services := []Service{}

	service1 := runHealthCheck()

	status := "Down"
	if service1 {
		status = "Up"
	}
	service := Service{
		ID:          "1",
		Name:        "RPC Server",
		Description: "RPC Server",
		Status:      status,
	}

	services = append(services, service)

	response := map[string][]Service{
		"services": services,
	}

	jsonResponse, err := json.Marshal(response)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonResponse)
	return
}

func ServiceHealthHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	serviceID := vars["service_ID"]

	service1 := runHealthCheck()
	if serviceID == "1" && service1 {
		w.WriteHeader(http.StatusOK)
		return
	}
	w.WriteHeader(http.StatusServiceUnavailable)
	return
}
