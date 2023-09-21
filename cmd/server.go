/*
Copyright © 2020 Rob Callahan <robtcallahan@gmail.com>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cmd

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	cfg "register/pkg/config"
	"register/pkg/plaid_auth"

	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"github.com/rs/cors"
	"github.com/spf13/cobra"
)

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Provides a web service function",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		server()
	},
}

func init() {
	config, _ = cfg.ReadConfig(ConfigFile)
	rootCmd.AddCommand(serverCmd)

	client = getBankingClient()
}

var store = sessions.NewCookieStore([]byte(os.Getenv("SESSION_KEY")))

func server() {
	r := mux.NewRouter()

	fs := http.FileServer(http.Dir("./www/public/"))
	r.PathPrefix("/public").Handler(http.StripPrefix("/public/", fs))

	r.HandleFunc("/api/test", test).Methods("GET", "OPTIONS")
	r.HandleFunc("/api/create_link_token", client.createLinkToken).Methods("GET", "OPTIONS")
	r.HandleFunc("/api/exchange_public_token", client.exchangePublicToken).Methods("POST")
	r.HandleFunc("/api/get_accounts", client.getAccounts).Methods("GET")
	r.HandleFunc("/api/get_balance", client.getBalance).Methods("GET")
	r.HandleFunc("/api/get_transactions", client.getTransactions).Methods("GET")
	r.HandleFunc("/api/is_account_connected", isAccountConnected).Methods("GET")

	log.Println("Server will start at https://localhost:9000/")
	handler := cors.Default().Handler(r)
	log.Fatal(http.ListenAndServeTLS(":9000", config.CertFile, config.KeyFile, handler))
}

func test(w http.ResponseWriter, r *http.Request) {
	log.Println("test()")

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte("{\"hello\": \"world\"}"))
}

func (c *Client) createLinkToken(w http.ResponseWriter, r *http.Request) {
	log.Println("createLinkToken()")

	linkToken, err := plaid_auth.GetLinkToken(c.BankClient)
	if err != nil {
		log.Println(err.Error())
		fmt.Print(err)
	}
	w.Header().Set("Content-Type", "application/json")
	s := LinkToken{LinkToken: linkToken}
	json.NewEncoder(w).Encode(s)
}

func (c *Client) exchangePublicToken(w http.ResponseWriter, r *http.Request) {
	log.Println("exchangePublicToken()")

	var publicToken PublicToken

	err := json.NewDecoder(r.Body).Decode(&publicToken)
	if err != nil {
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	accessToken, err := plaid_auth.ExchangePublicToken(c.BankClient, publicToken.PublicToken, ctx)
	if err != nil {
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	err = os.WriteFile(config.PlaidTokensDir+"/"+"AccessToken.txt", []byte(accessToken+"\n"), 0644)
	if err != nil {
		log.Printf("could not write access token: %s", err.Error())
	}

	session, err := store.Get(r, "register")
	if err != nil {
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	session.Values["AccessToken"] = accessToken
	err = sessions.Save(r, w)
	if err != nil {
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (c *Client) getAccounts(w http.ResponseWriter, r *http.Request) {
	log.Println("getAccounts()")

	session, err := store.Get(r, "register")
	if err != nil {
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	accessToken := fmt.Sprintf("%s", session.Values["AccessToken"])
	if accessToken == "" {
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	balances, err := c.BankClient.GetAccounts(accessToken, ctx)
	if err != nil {
		_, _ = w.Write([]byte(err.Error()))
		return
	}
	_ = json.NewEncoder(w).Encode(balances)
}

func (c *Client) getBalance(w http.ResponseWriter, r *http.Request) {
	log.Println("getBalance()")

	session, err := store.Get(r, "register")
	if err != nil {
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	accessToken := fmt.Sprintf("%s", session.Values["AccessToken"])
	if accessToken == "" {
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	balance, err := c.BankClient.GetBalance(accessToken, "fidelity", ctx)
	if err != nil {
		_, _ = w.Write([]byte(err.Error()))
		return
	}
	_ = json.NewEncoder(w).Encode(balance)
}

func (c *Client) getTransactions(w http.ResponseWriter, r *http.Request) {
	log.Println("getTransactions()")

	session, err := store.Get(r, "register")
	if err != nil {
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	accessToken := fmt.Sprintf("%s", session.Values["AccessToken"])
	if accessToken == "" {
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// start from 2 weeks ago
	startDate := weeksAgo(2)
	endDate := today()
	transactions, err := client.BankClient.GetTransactions(options.BankIDs, startDate, endDate)
	if err != nil {
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	_ = json.NewEncoder(w).Encode(transactions)
}

func isAccountConnected(w http.ResponseWriter, r *http.Request) {
	log.Println("isAccountConnected()")
	session, err := store.Get(r, "register")
	if err != nil {
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	accessToken := fmt.Sprintf("%s", session.Values["AccessToken"])
	type ConnectedResponse struct {
		Status bool `json:"status"`
	}
	if accessToken != "" {
		//_ = json.NewEncoder(w).Encode("{status: true}")
		resp := ConnectedResponse{Status: true}
		_ = json.NewEncoder(w).Encode(resp)
	} else {
		resp := ConnectedResponse{Status: false}
		_ = json.NewEncoder(w).Encode(resp)
	}
}
