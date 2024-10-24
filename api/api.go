package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/AhmadAshraf2/Judge-AVS/address"
	"github.com/AhmadAshraf2/Judge-AVS/utils"
	"github.com/gorilla/mux"
)

type Service struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Status      string `json:"status"`
}

func Server() {
	fmt.Println("starting API server")
	router := mux.NewRouter()

	router.HandleFunc("/eigen/node", NodeHandler).Methods("GET")
	router.HandleFunc("/eigen/node/health", HealthCheckHandler).Methods("GET")
	router.HandleFunc("/eigen/node/services", ServicesHandler).Methods("GET")
	router.HandleFunc("/eigen/node/services/{service_ID}/health", ServiceHealthHandler).Methods("GET")
	router.HandleFunc("/eigen/get_address", GetAddressHandler).Methods("Get")
	router.HandleFunc("/eigen/get_unsigned_psbt", GetUnSignedPsbtHandler).Methods("POST")
	router.HandleFunc("/eigen/submit_signed_psbt", SubmitSignedPsbtHandler).Methods("POST")

	http.ListenAndServe(":8080", router)
}

func GetUnSignedPsbtHandler(w http.ResponseWriter, r *http.Request) {
	var request struct {
		WithdrawAddr string `json:"withdrawAddr"`
		EthAddr      string `json:"ethAddr"`
	}

	if !utils.IsValidEthAddress(request.EthAddr) {
		http.Error(w, "Invalid PSBT string", http.StatusBadRequest)
		return
	}

	if !utils.IsValidBtcPubKey(request.WithdrawAddr) {
		http.Error(w, "Invalid withdraw Btc Address", http.StatusBadRequest)
		return
	}

	psbt, err := address.GetUnsignedPsbt(request.WithdrawAddr, request.EthAddr)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(psbt))
}

func SubmitSignedPsbtHandler(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Psbt string `json:"psbt"`
	}

	if !utils.IsValidPsbt(request.Psbt) {
		http.Error(w, "Invalid PSBT string", http.StatusBadRequest)
		return
	}

	p, err := address.SubmitSignedPsbt(request.Psbt)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(p))
}

func GetAddressHandler(w http.ResponseWriter, r *http.Request) {
	var request struct {
		PubKey  string `json:"pubKey"`
		EthAddr string `json:"ethAddr"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	if !utils.IsValidBtcPubKey(request.PubKey) {
		http.Error(w, "Invalid Bitcoin public key", http.StatusBadRequest)
		return
	}
	if !utils.IsValidEthAddress(request.EthAddr) {
		http.Error(w, "Invalid Eth Address", http.StatusBadRequest)
		return
	}
	newAddress, err := address.GenerateBTCAddress(request.PubKey, request.EthAddr)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(newAddress))
	return
}

func HealthCheckHandler(w http.ResponseWriter, r *http.Request) {
	service1, service2 := runHealthCheck()
	if service1 && service2 {
		w.WriteHeader(http.StatusOK)
		return
	} else if !service1 && !service2 {
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

	service1, service2 := runHealthCheck()

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

	status = "Down"
	if service2 {
		status = "Up"
	}
	service = Service{
		ID:          "2",
		Name:        "ForkScanner",
		Description: "ForkScanner",
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

	service1, service2 := runHealthCheck()
	if serviceID == "1" && service1 {
		w.WriteHeader(http.StatusOK)
		return
	}

	if serviceID == "2" && service2 {
		w.WriteHeader(http.StatusOK)
		return
	}
	w.WriteHeader(http.StatusServiceUnavailable)
	return
}
