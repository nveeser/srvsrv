package prettylog

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/nveeser/srvsrv/prettylog/template"
	"io"
	"log/slog"
	"os"
	"runtime"
	"slices"
	"strconv"
	"sync"
)

const (
	timeFormat      = "15:04:05.000"
	DefaultTemplate = "[{{.time}}] {{.level}} {{.source}}: {{.msg}}\n {{.json}}\n"
	JSONKey         = "json"

	reset = "\033[0m"

	black        textColor = 30
	red          textColor = 31
	green        textColor = 32
	yellow       textColor = 33
	blue         textColor = 34
	magenta      textColor = 35
	cyan         textColor = 36
	lightGray    textColor = 37
	darkGray     textColor = 90
	lightRed     textColor = 91
	lightGreen   textColor = 92
	lightYellow  textColor = 93
	lightBlue    textColor = 94
	lightMagenta textColor = 95
	lightCyan    textColor = 96
	white        textColor = 97
)

type textColor int

func colorize(colorCode textColor, v string) string {
	return fmt.Sprintf("\033[%sm%s%s", strconv.Itoa(int(colorCode)), v, reset)
}

type Options struct {
	OutputFormat  string
	FormatOptions []template.Option
	TimeFormat    string
	Colorize      bool
	StdOptions    slog.HandlerOptions
}

type Option = template.Option

func NewPrettyHandler(w io.Writer, opts *Options) slog.Handler {
	if opts == nil {
		opts = &Options{}
	}
	if opts.OutputFormat == "" {
		opts.OutputFormat = DefaultTemplate
	}
	if opts.TimeFormat == "" {
		opts.TimeFormat = timeFormat
	}

	ktmpl, err := template.Parse(opts.OutputFormat, opts.FormatOptions...)
	if err != nil {
		panic(err.Error())
	}
	common := &common{}
	jsonOpts := opts.StdOptions
	jsonOpts.ReplaceAttr = suppressTemplateKeys(opts.StdOptions.ReplaceAttr, ktmpl.Keys())
	jsonHandler := slog.NewJSONHandler(&common.jsonBuf, &jsonOpts)

	return &handler{
		json:        jsonHandler,
		opts:        opts,
		ktmpl:       ktmpl,
		common:      common,
		replaceAttr: opts.StdOptions.ReplaceAttr,
	}
}

type handler struct {
	common      *common
	keys        []string
	json        slog.Handler
	opts        *Options
	ktmpl       *template.KeyedTemplate
	replaceAttr func([]string, slog.Attr) slog.Attr
}

func (h *handler) clone(json slog.Handler) *handler {
	return &handler{
		common:      h.common,
		json:        json,
		ktmpl:       h.ktmpl,
		opts:        h.opts,
		keys:        h.keys,
		replaceAttr: h.replaceAttr,
	}
}

func (h *handler) Enabled(ctx context.Context, l slog.Level) bool {
	return h.json.Enabled(ctx, l)
}

func (h *handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return h.clone(h.json.WithAttrs(attrs))
}

func (h *handler) WithGroup(name string) slog.Handler {
	return h.clone(h.json.WithGroup(name))
}

func (h *handler) Handle(ctx context.Context, r slog.Record) error {
	attrs := h.attributes(r)

	data := map[string]string{}
	for _, key := range h.ktmpl.Keys() {
		data[key] = ""
		attr, ok := attrs[key]
		if !ok {
			continue
		}
		if h.replaceAttr != nil {
			attr = h.replaceAttr(nil, attr)
		}
		if !attr.Equal(slog.Attr{}) {
			data[key] = h.formatAttr(r, attr)
		}
	}

	h.common.Lock()
	defer h.common.Unlock()
	jsonValue, err := h.formatRecordLocked(ctx, r)
	if err != nil {
		return err
	}
	if len(jsonValue) > 0 && h.opts.Colorize {
		jsonValue = colorize(darkGray, jsonValue)
	}
	data[JSONKey] = jsonValue
	return h.ktmpl.Execute(os.Stdout, data)
}

func (h *handler) attributes(r slog.Record) map[string]slog.Attr {
	fs := runtime.CallersFrames([]uintptr{r.PC})
	f, _ := fs.Next()
	source := fmt.Sprintf("%s:%d", f.File, f.Line)

	attrs := map[string]slog.Attr{
		slog.TimeKey:    {Key: slog.TimeKey, Value: slog.StringValue(r.Time.Format(h.opts.TimeFormat))},
		slog.MessageKey: {Key: slog.MessageKey, Value: slog.StringValue(r.Message)},
		slog.LevelKey:   {Key: slog.LevelKey, Value: slog.AnyValue(r.Level)},
		slog.SourceKey:  {Key: slog.SourceKey, Value: slog.StringValue(source)},
	}

	r.Attrs(func(attr slog.Attr) bool {
		attrs[attr.Key] = attr
		return true
	})
	return attrs
}

func (h *handler) formatAttr(r slog.Record, attr slog.Attr) string {
	switch attr.Key {
	case slog.TimeKey:
		value := attr.Value.String()
		if h.opts.Colorize {
			value = colorize(lightGray, value)
		}
		return value

	case slog.MessageKey:
		value := attr.Value.String()
		if h.opts.Colorize {
			return colorize(white, value)
		}
		return value

	case slog.LevelKey:
		value := attr.Value.String()
		if !h.opts.Colorize {
			return value
		}
		switch {
		case r.Level <= slog.LevelDebug:
			return colorize(lightGray, value)
		case r.Level <= slog.LevelInfo:
			return colorize(cyan, value)
		case r.Level < slog.LevelWarn:
			return colorize(lightBlue, value)
		case r.Level < slog.LevelError:
			return colorize(lightYellow, value)
		case r.Level <= slog.LevelError+1:
			return colorize(lightRed, value)
		case r.Level > slog.LevelError+1:
			return colorize(lightMagenta, value)
		}
	}
	return attr.Value.String()
}

type common struct {
	out     io.Writer
	jsonBuf bytes.Buffer
	sync.Mutex
}

func (h *handler) formatRecordLocked(ctx context.Context, r slog.Record) (string, error) {
	defer h.common.jsonBuf.Reset()
	if err := h.json.Handle(ctx, r); err != nil {
		return "", fmt.Errorf("error when calling inner handler's Handle: %w", err)
	}
	var attrs map[string]any
	err := json.Unmarshal(h.common.jsonBuf.Bytes(), &attrs)
	if err != nil {
		return "", fmt.Errorf("error when unmarshaling inner handler's Handle result: %w", err)
	}
	if len(attrs) == 0 {
		return "", nil
	}
	jd, err := json.MarshalIndent(attrs, "", "  ")
	if err != nil {
		return "", fmt.Errorf("error when marshaling attrs: %w", err)
	}
	return string(jd), nil
}

type replaceFn func([]string, slog.Attr) slog.Attr

func suppressTemplateKeys(next replaceFn, keys []string) replaceFn {
	return func(groups []string, a slog.Attr) slog.Attr {
		if slices.Contains(keys, a.Key) {
			return slog.Attr{}
		}
		if next == nil {
			return a
		}
		return next(groups, a)
	}
}
