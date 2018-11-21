// Copyright 2017 Zack Butcher.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/spf13/cobra"
)

const (
	defaultPort uint16 = 9000
	healthPath         = "/health"
	echoPath           = "/echo"
	livePath           = "/live"
	callPath           = "/call"
)

type config struct {
	servingPort, healthCheckPort, livenessPort uint16
	healthy                                    bool
	livenessDelay                              time.Duration
}

func main() {
	cfg := &config{}

	root := &cobra.Command{
		Use:   "server",
		Short: "Starts Mixer as a server",
		Run: func(cmd *cobra.Command, args []string) {
			servers := map[uint16]*http.ServeMux{
				cfg.servingPort:     http.NewServeMux(),
				cfg.healthCheckPort: http.NewServeMux(),
				cfg.livenessPort:    http.NewServeMux(),
			}

			servers[cfg.servingPort].HandleFunc(echoPath, echo)
			servers[cfg.servingPort].HandleFunc(callPath, call)
			servers[cfg.healthCheckPort].HandleFunc(healthPath, health(cfg.healthy))
			servers[cfg.livenessPort].HandleFunc(livePath, live(cfg.livenessDelay))
			// For each of the servers, wire up a default handler so we can respond to any URL
			for _, server := range servers {
				server.HandleFunc("/", catchall)
			}

			log.Printf("listening for:\n/echo:     %d\n/health:   %d\n/liveness: %d\n", cfg.servingPort, cfg.healthCheckPort, cfg.livenessPort)

			wg := sync.WaitGroup{}

			for port, server := range servers {
				wg.Add(1)

				s := server
				p := port
				go func() {
					log.Printf("Starting listener on port %d\n", p)
					err := http.ListenAndServe(toAddress(p), s)
					log.Printf("%v\n", err)
					wg.Done()
				}()
			}

			wg.Wait()
		},
	}

	root.PersistentFlags().Uint16VarP(&cfg.servingPort, "server-port", "s", defaultPort, "Main port to serve on; always on /echo")
	root.PersistentFlags().Uint16VarP(&cfg.healthCheckPort, "health-port", "c", defaultPort, "Port to serve health checks on; always on /health")
	root.PersistentFlags().Uint16VarP(&cfg.livenessPort, "liveness-port", "l", defaultPort, "Port to serve liveness checks on; always on /live")
	root.PersistentFlags().BoolVar(&cfg.healthy, "healthy", true, "If false, the health check will report unhealthy")
	root.PersistentFlags().DurationVar(&cfg.livenessDelay, "liveness-delay", time.Second, "Delay before the server reports being alive")

	if err := root.Execute(); err != nil {
		log.Printf("%v\n", err)
		os.Exit(-1)
	}
}

func live(delay time.Duration) func(w http.ResponseWriter, r *http.Request) {
	live := time.Now().Add(delay)
	log.Printf("will be live at %v given delay %v\n", live, delay)
	return func(w http.ResponseWriter, r *http.Request) {
		log.Printf("got liveness request with headers:     %v\n", r.Header)
		if time.Now().After(live) {
			w.Write([]byte("live"))
		} else {
			w.WriteHeader(http.StatusServiceUnavailable)
		}
	}
}

func health(healthy bool) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Printf("got health check request with headers: %v\n", r.Header)
		if healthy {
			w.Write([]byte("healthy"))
		} else {
			w.WriteHeader(http.StatusServiceUnavailable)
		}
	}
}

func echo(w http.ResponseWriter, r *http.Request) {
	log.Printf("got echo request with headers:         %v\n", r.Header)
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Printf("got err reading body: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
	}
	w.Write(body)
}

func call(w http.ResponseWriter, r *http.Request) {
	log.Printf("got call request with headers: %v", r.Header)
	target, err := ioutil.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		log.Printf("got err reading body: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	log.Printf("got call target: %q", target)
	req, err := http.NewRequest("GET", "http://127.0.0.1/echo", bytes.NewBufferString("Success!"))
	if err != nil {
		fmt.Fprintf(w, "GET %q failed: %v", fmt.Sprintf("%s", target), err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	req.Host = string(target)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Fprintf(w, "GET %q failed: %v", fmt.Sprintf("%s", target), err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	log.Printf("GET %q succeeded with response code %v", target, resp.StatusCode)
	bodyByte, _ := ioutil.ReadAll(resp.Body)
	fmt.Fprintf(w, "got response: %v, body: %v", resp, string(bodyByte))
}

func catchall(w http.ResponseWriter, r *http.Request) {
	log.Printf("got catch-all request with headers:    %v\n", r.Header)
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}
	w.Write([]byte(fmt.Sprintf("default handler echoing: %q", body)))
}

func toAddress(port uint16) string {
	return fmt.Sprintf(":%d", port)
}
