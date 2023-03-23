package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

var db *sql.DB

type Order struct {
	ID           string  `json:"id"`
	Status       string  `json:"status"`
	Items        []Item  `json:"items"`
	Total        float64 `json:"total"`
	CurrencyUnit string  `json:"currencyUnit"`
}

type Item struct {
	ID          string  `json:"id"`
	Description string  `json:"description"`
	Price       float64 `json:"price"`
	Quantity    int     `json:"quantity"`
}

func main() {
	// Connect to MySQL database
	var err error
	db, err = sql.Open("localhost", "orderservice:cdac@tcp(127.0.0.1:3306)/orders")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Create router and define endpoints
	r := mux.NewRouter()
	r.HandleFunc("/orders", createOrder).Methods("POST")
	r.HandleFunc("/orders/{id}", getOrder).Methods("GET")
	// r.HandleFunc("/orders", getOrders).Methods("GET")
	// r.HandleFunc("/orders/{id}", updateOrder).Methods("PUT")
	// r.HandleFunc("/orders/{id}", deleteOrder).Methods("DELETE")

	// Start server
	log.Println("Server started on :8080")
	log.Fatal(http.ListenAndServe(":8080", r))
}

func createOrder(w http.ResponseWriter, r *http.Request) {
	var order Order
	err := json.NewDecoder(r.Body).Decode(&order)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Insert order into database
	result, err := db.Exec("INSERT INTO orders (id, status, items, total, currency_unit) VALUES (?, ?, ?, ?, ?)",
		order.ID, order.Status, order.Items, order.Total, order.CurrencyUnit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Return HTTP 201 Created with order ID
	id, _ := result.LastInsertId()
	w.Header().Set("Location", "/orders/"+order.ID)
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{"id": id})
}

func getOrder(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	// Query order from database
	var order Order
	err := db.QueryRow("SELECT id, status, items, total, currency_unit FROM orders WHERE id = ?", id).
		Scan(&order.ID, &order.Status, &order.Items, &order.Total, &order.CurrencyUnit)
	if err != nil {
		if err == sql.ErrNoRows {
			http.NotFound(w, r)
		}
	}
}
