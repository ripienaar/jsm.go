package audit

import (
	"fmt"
	"log"
	"time"

	"github.com/fatih/color"
)

const (
	debugLevel = "debug"
	infoLevel  = "info"
	warnLevel  = "warn"
	errLevel   = "error"
)

// LogLine represents a single log line
type LogLine struct {
	Level string    `json:"level"`
	Time  time.Time `json:"time"`
	Body  string    `json:"body"`
}

type Logger interface {
	Debugf(format string, a ...interface{})
	Infof(format string, a ...interface{})
	Warnf(format string, a ...interface{})
	Errorf(format string, a ...interface{})
}

// Debugf logs at debug level
func (r *Run) Debugf(format string, a ...interface{}) {
	r.log(debugLevel, format, a...)
}

// Infof logs at info level
func (r *Run) Infof(format string, a ...interface{}) {
	r.log(infoLevel, format, a...)
}

// Warnf logs at warning level
func (r *Run) Warnf(format string, a ...interface{}) {
	r.log(warnLevel, format, a...)
}

// Errorf logs at warning level
func (r *Run) Errorf(format string, a ...interface{}) {
	r.log(errLevel, format, a...)
}

func (r *Run) log(level string, format string, a ...interface{}) {
	r.mu.Lock()
	defer r.mu.Unlock()

	line := LogLine{
		Level: level,
		Time:  time.Now(),
		Body:  fmt.Sprintf(format, a...),
	}

	r.Logs = append(r.Logs, line)

	if r.logger == nil {
		pl := level
		switch level {
		case infoLevel:
			pl = color.GreenString(infoLevel)
		case errLevel:
			pl = color.RedString(errLevel)
		case warnLevel:
			pl = color.YellowString(warnLevel)
		}
		log.Printf("[%s] %s", pl, line.Body)
		return
	}

	switch level {
	case debugLevel:
		r.logger.Debugf(format, a...)
	case warnLevel:
		r.logger.Warnf(format, a...)
	case errLevel:
		r.logger.Errorf(format, a...)
	default:
		r.logger.Infof(format, a...)

	}
}
