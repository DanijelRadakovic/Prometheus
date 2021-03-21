package main

import (
	"fmt"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"log"
	"net/http"
	"time"
)

const (
	ip      = ":"
	port    = "8000"
	address = ip + port

	welcomeEndpoint  = "/"
	birthdayEndpoint = "/birthday/{name}"
	greetingEndpoint = "/greeting/{name}"
)

var (
	RequestCounter = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "go_app",
		Subsystem: "api",
		Name:      "request_counter",
		Help:      "Total HTTP requests count for specific endpoint.",
	}, []string{"path"})
)

func monitoringMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		route := mux.CurrentRoute(r)
		path, _ := route.GetPathTemplate()
		next.ServeHTTP(w, r)
		RequestCounter.WithLabelValues(path).Inc()
	})
}

func generateWelcomeMessage(rw http.ResponseWriter, _ *http.Request) {
	if _, err := rw.Write([]byte(fmt.Sprintf("Welcome!"))); err != nil {
		log.Println(err.Error())
		http.Error(rw, err.Error(), 500)
	}
}

func generateBirthdayMessage(rw http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["name"]
	greetings := fmt.Sprintf("Happy Birthday %s :)", name)
	time.Sleep(20 * time.Second)
	if _, err := rw.Write([]byte(greetings)); err != nil {
		log.Println(err.Error())
		http.Error(rw, err.Error(), 500)
	}
}

func generateGreetingMessage(rw http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["name"]
	greetings := fmt.Sprintf("Greetings %s :)", name)
	time.Sleep(5 * time.Second)
	if _, err := rw.Write([]byte(greetings)); err != nil {
		log.Println(err.Error())
		http.Error(rw, err.Error(), 500)
	}
}

func createRequestCounterMetric(name, endpoint string,
	requestFunction func(http.ResponseWriter, *http.Request)) func(http.ResponseWriter, *http.Request) {
	RequestCount := promauto.NewCounter(prometheus.CounterOpts{
		Namespace:   "go_app",
		Subsystem:   "api",
		Name:        name,
		Help:        "Total HTTP requests count for specific endpoint.",
		ConstLabels: prometheus.Labels{"path": endpoint},
	})
	return func(rw http.ResponseWriter, r *http.Request) {
		requestFunction(rw, r)
		RequestCount.Inc()
	}
}

func createRequestsInProgressMetric(name, endpoint string,
	requestFunction func(http.ResponseWriter, *http.Request)) func(http.ResponseWriter, *http.Request) {
	RequestInProgress := promauto.NewGauge(prometheus.GaugeOpts{
		Namespace:   "go_app",
		Subsystem:   "api",
		Name:        name,
		Help:        "Total HTTP requests in progress for specific endpoint.",
		ConstLabels: prometheus.Labels{"path": endpoint},
	})
	return func(rw http.ResponseWriter, r *http.Request) {
		RequestInProgress.Inc()
		requestFunction(rw, r)
		RequestInProgress.Dec()
	}
}

func createRequestLatencyMetric(name, endpoint string,
	requestFunction func(http.ResponseWriter, *http.Request)) func(http.ResponseWriter, *http.Request) {
	RequestLatency := promauto.NewHistogram(prometheus.HistogramOpts{
		Namespace:   "go_app",
		Subsystem:   "api",
		Name:        name,
		Help:        "HTTP requests latency distribution for specific endpoint.",
		ConstLabels: prometheus.Labels{"path": endpoint},
	})
	return func(rw http.ResponseWriter, r *http.Request) {
		startTime := time.Now()
		requestFunction(rw, r)
		timeTaken := time.Since(startTime)
		RequestLatency.Observe(timeTaken.Seconds())
	}
}

func main() {
	startApp()
}

func startApp() {
	router := mux.NewRouter()

	router.HandleFunc(welcomeEndpoint, generateWelcomeMessage).Methods("GET")
	router.HandleFunc(birthdayEndpoint,
		createRequestsInProgressMetric("requests_in_progress",
			birthdayEndpoint,
			generateBirthdayMessage)).
		Methods("GET")
	router.HandleFunc(greetingEndpoint,
		createRequestLatencyMetric("request_latency",
			greetingEndpoint,
			generateGreetingMessage)).
		Methods("GET")


	router.Path("/metrics").Handler(promhttp.Handler())
	router.Use(monitoringMiddleware)

	log.Println("Starting the application server...")
	if err := http.ListenAndServe(address, router); err != nil {
		log.Fatal(err.Error())
		return
	}
}
