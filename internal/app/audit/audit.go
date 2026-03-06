// Package audit реализует систему аудита для логирования событий.
// Поддерживает запись в файл и отправку на удаленный хост.
package audit

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/anatolyi0311/cupurl/internal/app/config"
	"github.com/sirupsen/logrus"
)

// Константы действий для аудита.
const (
	Shorten string = "shorten"
	Follow  string = "follow"
)

// Event представляет событие аудита.
// generate:reset
type Event struct {
	TS     time.Time `json:"ts"`
	Action string    `json:"action"`
	UserID int       `json:"user_id"`
	URL    string    `json:"url"`
}

// Observer определяет интерфейс для системы аудита.
type Observer interface {
	Run(ctx context.Context)
	Update(event Event)
}

// Audit реализует систему аудита с поддержкой файла и HTTP.
// generate:reset
type Audit struct {
	cfg       config.ENVConfig
	eventChan chan Event

	hasFile bool
	hasURL  bool
}

// New создает новый экземпляр Audit.
func New(cfg *config.ENVConfig) (Observer, error) {
	if cfg == nil {
		return nil, fmt.Errorf("NewAudit(): empty config")
	}
	if cfg.AuditFile == "" && cfg.AuditURL == "" {
		return nil, fmt.Errorf("NewAudit(): empty config")
	}

	return &Audit{
		cfg:       *cfg,
		hasFile:   cfg.AuditFile != "",
		hasURL:    cfg.AuditURL != "",
		eventChan: make(chan Event),
	}, nil
}

// Update отправляет событие в канал для обработки.
func (a *Audit) Update(event Event) {
	if a == nil {
		return
	}
	if a.eventChan == nil {
		return
	}
	a.eventChan <- event
}

// Run запускает обработчик событий аудита.
func (a *Audit) Run(ctx context.Context) {
	if a == nil {
		logrus.Error("audit not init")
		return
	}
	if a.eventChan == nil {
		logrus.Error("audit not run")
		return
	}
	for {
		select {
		case <-ctx.Done():
			logrus.Info("audit stopped")
			return
		case event := <-a.eventChan:
			go a.sendEvent(event)
		}
	}
}

func (a *Audit) sendEvent(event Event) {
	if a.hasFile {
		if err := a.saveToFile(event); err != nil {
			logrus.Warnf("sendEvent() %v", err)
		}
	}
	if a.hasURL {
		if err := a.sendToHost(event); err != nil {
			logrus.Warnf("sendEvent() %v", err)
		}
	}
}

func (a *Audit) saveToFile(event Event) error {
	dir := filepath.Dir(a.cfg.AuditFile)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	dataEvent, err := json.MarshalIndent(event, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	if err := os.WriteFile(a.cfg.AuditFile, dataEvent, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

func (a *Audit) sendToHost(event Event) error {
	dataEvent, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	addr, _ := url.ParseRequestURI(a.cfg.AuditURL)
	body := bytes.NewReader(dataEvent)

	req, err := http.NewRequest(http.MethodPost, addr.String(), body)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.ContentLength = int64(body.Len())

	client := &http.Client{
		Timeout: 3 * time.Second,
	}

	resp, err := client.Do(req)
	if resp != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		// handle error
		return err
	}
	return nil
}

// CreateEvent создает новое событие аудита.
func CreateEvent(userID int, action, URL string) Event {
	return Event{
		TS:     time.Now(),
		Action: action,
		URL:    URL,
		UserID: userID,
	}
}
