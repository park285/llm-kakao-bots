package util

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/buffer"
	"go.uber.org/zap/zapcore"
)

var logBufferPool = buffer.NewPool()

// NewLogger 는 동작을 수행한다.
func NewLogger(level, logFile string) (*zap.Logger, error) {
	var zapLevel zapcore.Level
	switch level {
	case "debug":
		zapLevel = zapcore.DebugLevel
	case "info":
		zapLevel = zapcore.InfoLevel
	case "warn":
		zapLevel = zapcore.WarnLevel
	case "error":
		zapLevel = zapcore.ErrorLevel
	default:
		zapLevel = zapcore.InfoLevel
	}

	if logFile != "" {
		logDir := filepath.Dir(logFile)
		if err := os.MkdirAll(logDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create log directory: %w", err)
		}
	}

	encoderConfig := zapcore.EncoderConfig{
		TimeKey:       "time",
		LevelKey:      "level",
		NameKey:       "logger",
		CallerKey:     "source",
		MessageKey:    "msg",
		StacktraceKey: "stacktrace",
		LineEnding:    zapcore.DefaultLineEnding,
		EncodeTime:    zapcore.TimeEncoderOfLayout(time.RFC3339),
		EncodeDuration: func(d time.Duration, enc zapcore.PrimitiveArrayEncoder) {
			enc.AppendString(d.String())
		},
	}

	encoder := newLogfmtEncoder(encoderConfig)
	writeSyncers := []zapcore.WriteSyncer{zapcore.AddSync(os.Stdout)}

	if logFile != "" {
		file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return nil, fmt.Errorf("failed to open log file: %w", err)
		}
		writeSyncers = append(writeSyncers, zapcore.AddSync(file))
	}

	core := zapcore.NewCore(
		encoder,
		zapcore.NewMultiWriteSyncer(writeSyncers...),
		zapLevel,
	)
	logger := zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))
	return logger, nil
}

type logfmtEncoder struct {
	cfg        zapcore.EncoderConfig
	fields     []logfmtField
	namespaces []string
}

type logfmtField struct {
	key   string
	value string
}

func newLogfmtEncoder(cfg zapcore.EncoderConfig) zapcore.Encoder {
	return &logfmtEncoder{cfg: cfg}
}

func (e *logfmtEncoder) Clone() zapcore.Encoder {
	clone := &logfmtEncoder{cfg: e.cfg}
	if len(e.namespaces) > 0 {
		clone.namespaces = append([]string(nil), e.namespaces...)
	}
	return clone
}

func (e *logfmtEncoder) EncodeEntry(entry zapcore.Entry, fields []zapcore.Field) (*buffer.Buffer, error) {
	clone := e.Clone().(*logfmtEncoder)
	for _, field := range fields {
		field.AddTo(clone)
	}

	buf := logBufferPool.Get()
	buf.AppendString(entry.Time.Format(time.RFC3339))
	buf.AppendByte(' ')
	buf.AppendString(formatLevel(entry.Level))
	buf.AppendByte(' ')
	if entry.Caller.Defined {
		buf.AppendString(formatCaller(entry.Caller))
		buf.AppendByte(' ')
	}
	buf.AppendString(entry.Message)

	for _, field := range clone.fields {
		buf.AppendByte(' ')
		buf.AppendString(field.key)
		buf.AppendByte('=')
		buf.AppendString(field.value)
	}

	if entry.Stack != "" && clone.cfg.StacktraceKey != "" {
		buf.AppendByte(' ')
		buf.AppendString(clone.cfg.StacktraceKey)
		buf.AppendByte('=')
		buf.AppendString(formatValue(entry.Stack))
	}

	lineEnding := clone.cfg.LineEnding
	if lineEnding == "" {
		lineEnding = zapcore.DefaultLineEnding
	}
	buf.AppendString(lineEnding)
	return buf, nil
}

func (e *logfmtEncoder) AddArray(key string, marshaler zapcore.ArrayMarshaler) error {
	arr := &logfmtArrayEncoder{}
	if err := marshaler.MarshalLogArray(arr); err != nil {
		return fmt.Errorf("marshal log array: %w", err)
	}
	e.addField(key, formatValue(arr.String()))
	return nil
}

func (e *logfmtEncoder) AddObject(key string, marshaler zapcore.ObjectMarshaler) error {
	obj := &logfmtObjectEncoder{}
	if err := marshaler.MarshalLogObject(obj); err != nil {
		return fmt.Errorf("marshal log object: %w", err)
	}
	e.addField(key, formatValue(obj.String()))
	return nil
}

func (e *logfmtEncoder) AddBinary(key string, val []byte) {
	e.addField(key, formatValue(string(val)))
}

func (e *logfmtEncoder) AddByteString(key string, val []byte) {
	e.addField(key, formatValue(string(val)))
}

func (e *logfmtEncoder) AddBool(key string, val bool) {
	e.addField(key, strconv.FormatBool(val))
}

func (e *logfmtEncoder) AddComplex128(key string, val complex128) {
	e.addField(key, formatValue(strconv.FormatComplex(val, 'f', -1, 128)))
}

func (e *logfmtEncoder) AddComplex64(key string, val complex64) {
	e.addField(key, formatValue(strconv.FormatComplex(complex128(val), 'f', -1, 64)))
}

func (e *logfmtEncoder) AddDuration(key string, val time.Duration) {
	e.addField(key, formatValue(val.String()))
}

func (e *logfmtEncoder) AddFloat64(key string, val float64) {
	e.addField(key, strconv.FormatFloat(val, 'f', -1, 64))
}

func (e *logfmtEncoder) AddFloat32(key string, val float32) {
	e.addField(key, strconv.FormatFloat(float64(val), 'f', -1, 32))
}

func (e *logfmtEncoder) AddInt(key string, val int) {
	e.addField(key, strconv.Itoa(val))
}

func (e *logfmtEncoder) AddInt64(key string, val int64) {
	e.addField(key, strconv.FormatInt(val, 10))
}

func (e *logfmtEncoder) AddInt32(key string, val int32) {
	e.addField(key, strconv.FormatInt(int64(val), 10))
}

func (e *logfmtEncoder) AddInt16(key string, val int16) {
	e.addField(key, strconv.FormatInt(int64(val), 10))
}

func (e *logfmtEncoder) AddInt8(key string, val int8) {
	e.addField(key, strconv.FormatInt(int64(val), 10))
}

func (e *logfmtEncoder) AddString(key, val string) {
	e.addField(key, formatValue(val))
}

func (e *logfmtEncoder) AddTime(key string, val time.Time) {
	e.addField(key, formatValue(val.Format(time.RFC3339)))
}

func (e *logfmtEncoder) AddUint(key string, val uint) {
	e.addField(key, strconv.FormatUint(uint64(val), 10))
}

func (e *logfmtEncoder) AddUint64(key string, val uint64) {
	e.addField(key, strconv.FormatUint(val, 10))
}

func (e *logfmtEncoder) AddUint32(key string, val uint32) {
	e.addField(key, strconv.FormatUint(uint64(val), 10))
}

func (e *logfmtEncoder) AddUint16(key string, val uint16) {
	e.addField(key, strconv.FormatUint(uint64(val), 10))
}

func (e *logfmtEncoder) AddUint8(key string, val uint8) {
	e.addField(key, strconv.FormatUint(uint64(val), 10))
}

func (e *logfmtEncoder) AddUintptr(key string, val uintptr) {
	e.addField(key, strconv.FormatUint(uint64(val), 10))
}

func (e *logfmtEncoder) AddReflected(key string, val interface{}) error {
	if val == nil {
		e.addField(key, "null")
		return nil
	}
	e.addField(key, formatValue(fmt.Sprint(val)))
	return nil
}

func (e *logfmtEncoder) OpenNamespace(key string) {
	if key != "" {
		e.namespaces = append(e.namespaces, key)
	}
}

func (e *logfmtEncoder) addField(key, value string) {
	if key == "" {
		return
	}
	e.fields = append(e.fields, logfmtField{
		key:   e.fullKey(key),
		value: value,
	})
}

func (e *logfmtEncoder) fullKey(key string) string {
	if len(e.namespaces) == 0 {
		return key
	}
	parts := append(append([]string(nil), e.namespaces...), key)
	return strings.Join(parts, ".")
}

type logfmtArrayEncoder struct {
	items []string
}

func (a *logfmtArrayEncoder) AppendBool(val bool) {
	a.items = append(a.items, strconv.FormatBool(val))
}

func (a *logfmtArrayEncoder) AppendByteString(val []byte) {
	a.items = append(a.items, formatValue(string(val)))
}

func (a *logfmtArrayEncoder) AppendComplex128(val complex128) {
	a.items = append(a.items, formatValue(strconv.FormatComplex(val, 'f', -1, 128)))
}

func (a *logfmtArrayEncoder) AppendComplex64(val complex64) {
	a.items = append(a.items, formatValue(strconv.FormatComplex(complex128(val), 'f', -1, 64)))
}

func (a *logfmtArrayEncoder) AppendDuration(val time.Duration) {
	a.items = append(a.items, formatValue(val.String()))
}

func (a *logfmtArrayEncoder) AppendFloat64(val float64) {
	a.items = append(a.items, strconv.FormatFloat(val, 'f', -1, 64))
}

func (a *logfmtArrayEncoder) AppendFloat32(val float32) {
	a.items = append(a.items, strconv.FormatFloat(float64(val), 'f', -1, 32))
}

func (a *logfmtArrayEncoder) AppendInt(val int) {
	a.items = append(a.items, strconv.Itoa(val))
}

func (a *logfmtArrayEncoder) AppendInt64(val int64) {
	a.items = append(a.items, strconv.FormatInt(val, 10))
}

func (a *logfmtArrayEncoder) AppendInt32(val int32) {
	a.items = append(a.items, strconv.FormatInt(int64(val), 10))
}

func (a *logfmtArrayEncoder) AppendInt16(val int16) {
	a.items = append(a.items, strconv.FormatInt(int64(val), 10))
}

func (a *logfmtArrayEncoder) AppendInt8(val int8) {
	a.items = append(a.items, strconv.FormatInt(int64(val), 10))
}

func (a *logfmtArrayEncoder) AppendString(val string) {
	a.items = append(a.items, formatValue(val))
}

func (a *logfmtArrayEncoder) AppendTime(val time.Time) {
	a.items = append(a.items, formatValue(val.Format(time.RFC3339)))
}

func (a *logfmtArrayEncoder) AppendUint(val uint) {
	a.items = append(a.items, strconv.FormatUint(uint64(val), 10))
}

func (a *logfmtArrayEncoder) AppendUint64(val uint64) {
	a.items = append(a.items, strconv.FormatUint(val, 10))
}

func (a *logfmtArrayEncoder) AppendUint32(val uint32) {
	a.items = append(a.items, strconv.FormatUint(uint64(val), 10))
}

func (a *logfmtArrayEncoder) AppendUint16(val uint16) {
	a.items = append(a.items, strconv.FormatUint(uint64(val), 10))
}

func (a *logfmtArrayEncoder) AppendUint8(val uint8) {
	a.items = append(a.items, strconv.FormatUint(uint64(val), 10))
}

func (a *logfmtArrayEncoder) AppendUintptr(val uintptr) {
	a.items = append(a.items, strconv.FormatUint(uint64(val), 10))
}

func (a *logfmtArrayEncoder) AppendReflected(val interface{}) error {
	if val == nil {
		a.items = append(a.items, "null")
		return nil
	}
	a.items = append(a.items, formatValue(fmt.Sprint(val)))
	return nil
}

func (a *logfmtArrayEncoder) AppendObject(marshaler zapcore.ObjectMarshaler) error {
	obj := &logfmtObjectEncoder{}
	if err := marshaler.MarshalLogObject(obj); err != nil {
		return fmt.Errorf("marshal log object: %w", err)
	}
	a.items = append(a.items, formatValue(obj.String()))
	return nil
}

func (a *logfmtArrayEncoder) AppendArray(marshaler zapcore.ArrayMarshaler) error {
	arr := &logfmtArrayEncoder{}
	if err := marshaler.MarshalLogArray(arr); err != nil {
		return fmt.Errorf("marshal log array: %w", err)
	}
	a.items = append(a.items, formatValue(arr.String()))
	return nil
}

func (a *logfmtArrayEncoder) String() string {
	return "[" + strings.Join(a.items, ",") + "]"
}

type logfmtObjectEncoder struct {
	fields     []logfmtField
	namespaces []string
}

func (o *logfmtObjectEncoder) AddArray(key string, marshaler zapcore.ArrayMarshaler) error {
	arr := &logfmtArrayEncoder{}
	if err := marshaler.MarshalLogArray(arr); err != nil {
		return fmt.Errorf("marshal log array: %w", err)
	}
	o.addField(key, formatValue(arr.String()))
	return nil
}

func (o *logfmtObjectEncoder) AddObject(key string, marshaler zapcore.ObjectMarshaler) error {
	obj := &logfmtObjectEncoder{}
	if err := marshaler.MarshalLogObject(obj); err != nil {
		return fmt.Errorf("marshal log object: %w", err)
	}
	o.addField(key, formatValue(obj.String()))
	return nil
}

func (o *logfmtObjectEncoder) AddBinary(key string, val []byte) {
	o.addField(key, formatValue(string(val)))
}

func (o *logfmtObjectEncoder) AddByteString(key string, val []byte) {
	o.addField(key, formatValue(string(val)))
}

func (o *logfmtObjectEncoder) AddBool(key string, val bool) {
	o.addField(key, strconv.FormatBool(val))
}

func (o *logfmtObjectEncoder) AddComplex128(key string, val complex128) {
	o.addField(key, formatValue(strconv.FormatComplex(val, 'f', -1, 128)))
}

func (o *logfmtObjectEncoder) AddComplex64(key string, val complex64) {
	o.addField(key, formatValue(strconv.FormatComplex(complex128(val), 'f', -1, 64)))
}

func (o *logfmtObjectEncoder) AddDuration(key string, val time.Duration) {
	o.addField(key, formatValue(val.String()))
}

func (o *logfmtObjectEncoder) AddFloat64(key string, val float64) {
	o.addField(key, strconv.FormatFloat(val, 'f', -1, 64))
}

func (o *logfmtObjectEncoder) AddFloat32(key string, val float32) {
	o.addField(key, strconv.FormatFloat(float64(val), 'f', -1, 32))
}

func (o *logfmtObjectEncoder) AddInt(key string, val int) {
	o.addField(key, strconv.Itoa(val))
}

func (o *logfmtObjectEncoder) AddInt64(key string, val int64) {
	o.addField(key, strconv.FormatInt(val, 10))
}

func (o *logfmtObjectEncoder) AddInt32(key string, val int32) {
	o.addField(key, strconv.FormatInt(int64(val), 10))
}

func (o *logfmtObjectEncoder) AddInt16(key string, val int16) {
	o.addField(key, strconv.FormatInt(int64(val), 10))
}

func (o *logfmtObjectEncoder) AddInt8(key string, val int8) {
	o.addField(key, strconv.FormatInt(int64(val), 10))
}

func (o *logfmtObjectEncoder) AddString(key, val string) {
	o.addField(key, formatValue(val))
}

func (o *logfmtObjectEncoder) AddTime(key string, val time.Time) {
	o.addField(key, formatValue(val.Format(time.RFC3339)))
}

func (o *logfmtObjectEncoder) AddUint(key string, val uint) {
	o.addField(key, strconv.FormatUint(uint64(val), 10))
}

func (o *logfmtObjectEncoder) AddUint64(key string, val uint64) {
	o.addField(key, strconv.FormatUint(val, 10))
}

func (o *logfmtObjectEncoder) AddUint32(key string, val uint32) {
	o.addField(key, strconv.FormatUint(uint64(val), 10))
}

func (o *logfmtObjectEncoder) AddUint16(key string, val uint16) {
	o.addField(key, strconv.FormatUint(uint64(val), 10))
}

func (o *logfmtObjectEncoder) AddUint8(key string, val uint8) {
	o.addField(key, strconv.FormatUint(uint64(val), 10))
}

func (o *logfmtObjectEncoder) AddUintptr(key string, val uintptr) {
	o.addField(key, strconv.FormatUint(uint64(val), 10))
}

func (o *logfmtObjectEncoder) AddReflected(key string, val interface{}) error {
	if val == nil {
		o.addField(key, "null")
		return nil
	}
	o.addField(key, formatValue(fmt.Sprint(val)))
	return nil
}

func (o *logfmtObjectEncoder) OpenNamespace(key string) {
	if key != "" {
		o.namespaces = append(o.namespaces, key)
	}
}

func (o *logfmtObjectEncoder) addField(key, value string) {
	if key == "" {
		return
	}
	o.fields = append(o.fields, logfmtField{
		key:   o.fullKey(key),
		value: value,
	})
}

func (o *logfmtObjectEncoder) fullKey(key string) string {
	if len(o.namespaces) == 0 {
		return key
	}
	parts := append(append([]string(nil), o.namespaces...), key)
	return strings.Join(parts, ".")
}

func (o *logfmtObjectEncoder) String() string {
	if len(o.fields) == 0 {
		return "{}"
	}
	parts := make([]string, 0, len(o.fields))
	for _, field := range o.fields {
		parts = append(parts, field.key+"="+field.value)
	}
	return "{" + strings.Join(parts, ",") + "}"
}

func formatLevel(level zapcore.Level) string {
	switch level {
	case zapcore.DebugLevel:
		return "DBG"
	case zapcore.InfoLevel:
		return "INF"
	case zapcore.WarnLevel:
		return "WRN"
	case zapcore.ErrorLevel:
		return "ERR"
	case zapcore.DPanicLevel:
		return "DPN"
	case zapcore.PanicLevel:
		return "PNC"
	case zapcore.FatalLevel:
		return "FTL"
	default:
		return strings.ToUpper(level.String())
	}
}

func formatCaller(caller zapcore.EntryCaller) string {
	if !caller.Defined {
		return "unknown"
	}
	file := filepath.ToSlash(caller.File)
	parts := strings.Split(file, "/")
	if len(parts) >= 2 {
		file = parts[len(parts)-2] + "/" + parts[len(parts)-1]
	} else if len(parts) == 1 {
		file = parts[0]
	}
	return file + ":" + strconv.Itoa(caller.Line)
}

func formatValue(value string) string {
	if value == "" {
		return "\"\""
	}
	if strings.IndexFunc(value, func(r rune) bool {
		return r == '"' || r == '\\' || r == '=' || r == ' ' || r == '\t' || r == '\n' || r == '\r'
	}) == -1 {
		return value
	}
	value = strings.ReplaceAll(value, "\\", "\\\\")
	value = strings.ReplaceAll(value, "\"", "\\\"")
	return "\"" + value + "\""
}
