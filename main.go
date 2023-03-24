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
	db, err = sql.Open("mysql", "user:password@tcp(127.0.0.1:3306)/orders")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Create router and define endpoints
	r := mux.NewRouter()
	r.HandleFunc("/orders", createOrder).Methods("POST")
	r.HandleFunc("/orders/{id}", GetOrder).Methods("GET")
	r.HandleFunc("/updateorders", UpdateOrder).Methods("POST")
	r.HandleFunc("/ordersort/{id}", GetOrdersort).Methods("GET")
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

// func getOrder(w http.ResponseWriter, r *http.Request) {
// 	id := mux.Vars(r)["id"]

// 	// Query order from database
// 	var order Order
// 	err := db.QueryRow("SELECT id, status, items, total, currency_unit FROM orders WHERE id = ?", id).
// 		Scan(&order.ID, &order.Status, &order.Items, &order.Total, &order.CurrencyUnit)
// 	if err != nil {
// 		if err == sql.ErrNoRows {
// 			http.NotFound(w, r)
// 		}
// 	}
// }

func GetOrder(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	row := db.QueryRow("SELECT id, status, items, total, currency_unit, created_at FROM orders WHERE id=?", id)
	var order Order
	var itemsJSON string
	err := row.Scan(&order.ID, &order.Status, &itemsJSON, &order.Total, &order.CurrencyUnit)
	if err == sql.ErrNoRows {
		http.NotFound(w, r)
		return
	} else if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = json.Unmarshal([]byte(itemsJSON), &order.Items)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(order)
}

// GetOrders retrieves a list of orders from the database, filtered and sorted according to the given parameters.
func GetOrdersort(w http.ResponseWriter, r *http.Request) {
	// Parse the query string parameters
	queryParams := r.URL.Query()

	// Get the filters from the query parameters
	filters := make(map[string]interface{})
	for key, value := range queryParams {
		if key != "sort" {
			filters[key] = value[0]
		}
	}

	// Get the sorting order from the query parameters
	sortField := queryParams.Get("sort")
	if sortField == "" {
		sortField = "created_at"
	}

	// Build the SQL query string
	sqlStr := "SELECT id, status, items, total, currency_unit, created_at FROM orders"
	var values []interface{}
	if len(filters) > 0 {
		sqlStr += " WHERE "
		i := 0
		for key, value := range filters {
			if i > 0 {
				sqlStr += " AND "
			}
			sqlStr += key + "=?"
			values = append(values, value)
			i++
		}
	}
	sqlStr += " ORDER BY " + sortField

	// Execute the SQL query
	rows, err := db.Query(sqlStr, values...)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	// Iterate over the rows and build the response
	orders := make([]Order, 0)
	for rows.Next() {
		var order Order
		var itemsJSON string
		err := rows.Scan(&order.ID, &order.Status, &itemsJSON, &order.Total, &order.CurrencyUnit)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		err = json.Unmarshal([]byte(itemsJSON), &order.Items)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		orders = append(orders, order)
	}
	if err := rows.Err(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Return the orders in JSON format
	json.NewEncoder(w).Encode(orders)
}

func UpdateOrder(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	var order Order
	err := json.NewDecoder(r.Body).Decode(&order)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	stmt, err := db.Prepare("UPDATE orders SET status=? WHERE id=?")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer stmt.Close()

	_, err = stmt.Exec(order.Status, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	order.ID = id
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(order)
}
