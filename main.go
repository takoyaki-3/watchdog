package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/smtp"
	"os"
	"sync"
	"time"
)

type LogRecord struct {
	IP  string    `json:"ip"`
	ID  string    `json:"id"`
	At  time.Time `json:"at"`
}

type EmailRecord struct {
	ID  string    `json:"id"`
	At  time.Time `json:"at"`
}

var accessTimes map[string]time.Time
var emailSentFlags map[string]bool
var mutex sync.Mutex

func sendEmail(id string) {
	from := os.Getenv("SMTP_FROM_EMAIL")
	to := os.Getenv("SMTP_TO_EMAIL")
	smtpServer := os.Getenv("SMTP_SERVER")
	smtpPort := os.Getenv("SMTP_PORT")
	password := os.Getenv("SMTP_PASSWORD")

	body := fmt.Sprintf("Program with ID %s has not been accessed for more than 6 minutes.", id)

	msg := "From: " + from + "\n" +
		"To: " + to + "\n" +
		"Subject: Server Alert\n\n" +
		body

	auth := smtp.PlainAuth("", from, password, smtpServer)
	err := smtp.SendMail(smtpServer+":"+smtpPort, auth, from, []string{to}, []byte(msg))
	if err != nil {
		log.Fatal(err)
	}

	emailSentFlags[id] = true

	emailRecord := EmailRecord{
		ID: id,
		At: time.Now(),
	}

	emailJSON, err := json.Marshal(emailRecord)
	if err != nil {
		log.Printf("Failed to marshal email record: %s", err)
	} else {
		log.Printf(string(emailJSON))
	}
}

func statusHandler(w http.ResponseWriter, r *http.Request) {
	mutex.Lock()
	defer mutex.Unlock()

	tmpl := template.Must(template.ParseFiles("status.html"))
	if err := tmpl.Execute(w, accessTimes); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func handler(w http.ResponseWriter, r *http.Request) {
	ipAddress := r.RemoteAddr
	id := r.URL.Query().Get("id")

	mutex.Lock()
	accessTimes[id] = time.Now()
	emailSentFlags[id] = false
	mutex.Unlock()

	logRecord := LogRecord{
		IP:  ipAddress,
		ID:  id,
		At:  time.Now(),
	}

	logJSON, err := json.Marshal(logRecord)
	if err != nil {
		log.Printf("Failed to marshal log record: %s", err)
	} else {
		log.Printf(string(logJSON))
	}
}

func main() {
	logFile, err := os.OpenFile("watchdog.log", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal(err)
	}
	defer logFile.Close()
	log.SetOutput(logFile)

	accessTimes = make(map[string]time.Time)
	emailSentFlags = make(map[string]bool)

	http.HandleFunc("/", handler)
	http.HandleFunc("/status", statusHandler)

	go http.ListenAndServe(":8080", nil)

	ticker := time.NewTicker(1 * time.Minute)

	for {
		select {
		case <-ticker.C:
			mutex.Lock()
			for id, lastAccessTime := range accessTimes {
				if time.Since(lastAccessTime) > 6*time.Minute && !emailSentFlags[id] {
					sendEmail(id)
					log.Printf("Email notification sent for ID: %s\n", id)
				}
			}
			mutex.Unlock()
		}
	}
}
