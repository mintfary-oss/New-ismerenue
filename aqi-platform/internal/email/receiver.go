// Package email реализует IMAP-приёмник данных с датчиков качества воздуха.
// Датчики или операторы отправляют CSV-файлы на почтовый ящик платформы.
// Приёмник опрашивает ящик с заданным интервалом и загружает данные в БД.
package email

import (
	"context"
	"crypto/tls"
	"encoding/csv"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"time"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	"github.com/google/uuid"
	"github.com/mintfary/aqi-platform/internal/config"
	"github.com/mintfary/aqi-platform/internal/domain"
)

// MeasurementIngestor — интерфейс сервиса загрузки измерений.
type MeasurementIngestor interface {
	IngestBatch(ctx context.Context, items []domain.MeasurementInput) error
}

// Receiver — IMAP-приёмник данных с датчиков.
// Подключается к почтовому ящику, извлекает CSV-вложения и загружает данные.
type Receiver struct {
	cfg    config.EmailConfig
	ingest MeasurementIngestor
	logger *slog.Logger
}

// New создаёт новый IMAP Receiver.
func New(cfg config.EmailConfig, ingest MeasurementIngestor, logger *slog.Logger) *Receiver {
	return &Receiver{cfg: cfg, ingest: ingest, logger: logger}
}

// Start запускает цикл опроса почты. Блокируется до отмены контекста.
func (r *Receiver) Start(ctx context.Context) {
	if r.cfg.IMAPHost == "" {
		r.logger.Info("IMAP-приёмник отключён: imap_host не задан")
		return
	}

	interval := r.cfg.PollInterval
	if interval <= 0 {
		interval = 5 * time.Minute
	}

	r.logger.Info("IMAP-приёмник запущен",
		"host", r.cfg.IMAPHost,
		"user", r.cfg.IMAPUser,
		"interval", interval,
	)

	// Первый опрос сразу при старте.
	r.poll(ctx)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			r.logger.Info("IMAP-приёмник остановлен")
			return
		case <-ticker.C:
			r.poll(ctx)
		}
	}
}

// poll выполняет один цикл подключения и обработки писем.
func (r *Receiver) poll(ctx context.Context) {
	if err := r.process(ctx); err != nil {
		r.logger.Error("IMAP poll error", "err", err)
	}
}

// process подключается к IMAP-серверу и обрабатывает непрочитанные письма.
func (r *Receiver) process(ctx context.Context) error {
	port := r.cfg.IMAPPort
	if port == 0 {
		port = 993
	}
	addr := fmt.Sprintf("%s:%d", r.cfg.IMAPHost, port)

	// Подключение через TLS (порт 993 по умолчанию).
	c, err := client.DialTLS(addr, &tls.Config{
		ServerName: r.cfg.IMAPHost,
		MinVersion: tls.VersionTLS12,
	})
	if err != nil {
		return fmt.Errorf("imap dial: %w", err)
	}
	defer func() { _ = c.Logout() }()

	// Авторизация по логину и паролю.
	if err := c.Login(r.cfg.IMAPUser, r.cfg.IMAPPassword); err != nil {
		return fmt.Errorf("imap login: %w", err)
	}

	// Выбираем папку INBOX.
	mbox, err := c.Select("INBOX", false)
	if err != nil {
		return fmt.Errorf("imap select: %w", err)
	}

	if mbox.Messages == 0 {
		return nil // ящик пуст
	}

	// Ищем непрочитанные письма.
	criteria := imap.NewSearchCriteria()
	criteria.WithoutFlags = []string{imap.SeenFlag}

	uids, err := c.Search(criteria)
	if err != nil {
		return fmt.Errorf("imap search: %w", err)
	}
	if len(uids) == 0 {
		return nil
	}

	r.logger.Info("IMAP: найдены непрочитанные письма", "count", len(uids))

	seqSet := new(imap.SeqSet)
	seqSet.AddNum(uids...)

	messages := make(chan *imap.Message, 10)
	done := make(chan error, 1)
	go func() {
		done <- c.Fetch(seqSet, []imap.FetchItem{
			imap.FetchEnvelope,
			"BODY[]",
		}, messages)
	}()

	processed := 0
	for msg := range messages {
		if err2 := r.handleMessage(ctx, msg); err2 != nil {
			r.logger.Warn("IMAP: ошибка обработки письма",
				"subject", msg.Envelope.Subject,
				"err", err2,
			)
		} else {
			processed++
			// Помечаем письмо как прочитанное.
			markSet := new(imap.SeqSet)
			markSet.AddNum(msg.SeqNum)
			item := imap.FormatFlagsOp(imap.AddFlags, true)
			_ = c.Store(markSet, item, []interface{}{imap.SeenFlag}, nil)
		}
	}

	if err := <-done; err != nil {
		return fmt.Errorf("imap fetch: %w", err)
	}

	if processed > 0 {
		r.logger.Info("IMAP: письма обработаны", "processed", processed)
	}
	return nil
}

// handleMessage извлекает CSV-данные из письма и загружает их.
func (r *Receiver) handleMessage(ctx context.Context, msg *imap.Message) error {
	if msg.Envelope != nil {
		r.logger.Debug("IMAP: обрабатываем письмо",
			"subject", msg.Envelope.Subject,
		)
	}

	for _, literal := range msg.Body {
		body, err := io.ReadAll(literal)
		if err != nil {
			continue
		}
		if !looksLikeCSV(body) {
			continue
		}

		records, err := r.parseCSV(body)
		if err != nil {
			r.logger.Warn("IMAP: не удалось разобрать CSV", "err", err)
			continue
		}
		if len(records) == 0 {
			continue
		}

		if err := r.ingest.IngestBatch(ctx, records); err != nil {
			return fmt.Errorf("ingest batch: %w", err)
		}
		r.logger.Info("IMAP: загружены данные из CSV", "rows", len(records))
	}
	return nil
}

// parseCSV разбирает CSV-данные измерений.
// Ожидаемый формат заголовка:
//
//	time,sensor_id,no2,o3,co,h2s,so2,pm25,temperature,humidity,pressure,wind_speed,wind_dir
func (r *Receiver) parseCSV(data []byte) ([]domain.MeasurementInput, error) {
	reader := csv.NewReader(strings.NewReader(string(data)))
	reader.TrimLeadingSpace = true
	reader.FieldsPerRecord = -1

	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("csv parse: %w", err)
	}
	if len(records) < 2 {
		return nil, fmt.Errorf("csv: только заголовок или пусто")
	}

	// Строим индекс колонок из заголовка.
	header := records[0]
	colIdx := make(map[string]int, len(header))
	for i, h := range header {
		colIdx[strings.ToLower(strings.TrimSpace(h))] = i
	}

	timeIdx, hasTime := colIdx["time"]
	sensorIdx, hasSensor := colIdx["sensor_id"]
	if !hasTime || !hasSensor {
		return nil, fmt.Errorf("csv: обязательные колонки time и sensor_id не найдены")
	}

	var items []domain.MeasurementInput
	for lineNum, row := range records[1:] {
		if len(row) == 0 {
			continue
		}

		t, parseErr := time.Parse(time.RFC3339, strings.TrimSpace(row[timeIdx]))
		if parseErr != nil {
			t, parseErr = time.Parse("2006-01-02 15:04:05", strings.TrimSpace(row[timeIdx]))
			if parseErr != nil {
				r.logger.Warn("IMAP CSV: пропускаем строку с некорректной датой",
					"line", lineNum+2, "value", row[timeIdx])
				continue
			}
		}

		sensorID, parseErr := uuid.Parse(strings.TrimSpace(row[sensorIdx]))
		if parseErr != nil {
			r.logger.Warn("IMAP CSV: пропускаем строку с некорректным sensor_id",
				"line", lineNum+2, "value", row[sensorIdx])
			continue
		}

		items = append(items, domain.MeasurementInput{
			SensorID:    sensorID,
			Time:        t.UTC(),
			NO2:         parseColFloat(row, colIdx, "no2"),
			O3:          parseColFloat(row, colIdx, "o3"),
			CO:          parseColFloat(row, colIdx, "co"),
			H2S:         parseColFloat(row, colIdx, "h2s"),
			SO2:         parseColFloat(row, colIdx, "so2"),
			PM25:        parseColFloat(row, colIdx, "pm25"),
			Temperature: parseColFloat(row, colIdx, "temperature"),
			Humidity:    parseColFloat(row, colIdx, "humidity"),
			Pressure:    parseColFloat(row, colIdx, "pressure"),
			WindSpeed:   parseColFloat(row, colIdx, "wind_speed"),
			WindDir:     parseColFloat(row, colIdx, "wind_dir"),
		})
	}
	return items, nil
}

// looksLikeCSV проверяет наличие CSV-заголовка с обязательными полями.
func looksLikeCSV(data []byte) bool {
	if len(data) == 0 {
		return false
	}
	s := strings.ToLower(string(data[:min(300, len(data))]))
	return strings.Contains(s, "sensor_id") && strings.Contains(s, "time")
}

// parseColFloat парсит float64 из ячейки CSV по имени колонки.
func parseColFloat(row []string, colIdx map[string]int, name string) *float64 {
	idx, ok := colIdx[name]
	if !ok || idx >= len(row) {
		return nil
	}
	val := strings.TrimSpace(row[idx])
	if val == "" || val == "null" || val == "NULL" {
		return nil
	}
	var f float64
	if _, err := fmt.Sscanf(val, "%f", &f); err != nil {
		return nil
	}
	return &f
}

// min возвращает меньшее из двух int.
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
