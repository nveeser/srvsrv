// Package prettylog provides an slog.Handler which wraps the std library
// TextHandler or JSONHandler and provides template formatting of the attributes
// and optional colorization of output designed specifically for terminal output
// of a command line rather than formatted output for a server log.
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
	"sync"
)

// FormatMode defines how to format attributes which are not mentioned in the
// OutputFormat
type FormatMode string

const (
	FormatText       FormatMode = "mode-text"
	FormatJSON                  = "mode-json"
	FormatJSONIndent            = "mode-json-indent"
)

var defaultTemplates = map[FormatMode]string{
	FormatText:       `[{.time}] {.level} {.source}: {.msg}{.extra_attrs | pre " "}`,
	FormatJSON:       `[{.time}] {.level} {.source}: {.msg}{.extra_attrs | pre " "}`,
	FormatJSONIndent: `[{.time}] {.level} {.source}: {.msg}{.extra_attrs | pre "\n" | post "\n"}`,
}

const (
	timeFormat    = "15:04:05.000"
	extraAttrsKey = "extra_attrs"
)

// Options configures the PrettyHandler output.
type Options struct {
	// Defines the format of the attributes. See the defaults and tests in the templates package
	OutputFormat string
	// Specifies the mode for formatting the remaining attributes within the record
	// which are not specified within the OutputFormat
	FormatMode FormatMode
	// Specifies the format of the timestamp
	TimeFormat string
	// Enables color using ANSI color esacpe codes
	Colorize bool
	// Enables source. See slog.HandlerOptions
	AddSource bool
	// Sets the log level. See slog.HandlerOptions
	Level slog.Leveler
	// Sets the log level. See slog.HandlerOptions
	ReplaceAttr func(groups []string, a slog.Attr) slog.Attr
}

// NewPrettyHandler returns an slog.Handler which formats log records
// for human readability when running locally.
func NewPrettyHandler(w io.Writer, opts *Options) slog.Handler {
	if w == nil {
		w = os.Stdout
	}
	if opts == nil {
		opts = &Options{}
	}
	if opts.FormatMode == "" {
		opts.FormatMode = FormatJSON
	}
	if opts.OutputFormat == "" {
		opts.OutputFormat = defaultTemplates[opts.FormatMode]
	}
	if opts.TimeFormat == "" {
		opts.TimeFormat = timeFormat
	}

	ktmpl, err := template.Parse(opts.OutputFormat)
	if err != nil {
		panic(err.Error())
	}
	common := &protected{out: w}
	var delegate slog.Handler
	delegateOpts := &slog.HandlerOptions{
		Level:       opts.Level,
		AddSource:   opts.AddSource,
		ReplaceAttr: suppressTemplateKeys(opts.ReplaceAttr, ktmpl.Keys()),
	}
	switch opts.FormatMode {
	case FormatJSON:
		delegate = slog.NewJSONHandler(&common.buffer, delegateOpts)
	case FormatJSONIndent:
		delegate = slog.NewJSONHandler(&common.buffer, delegateOpts)
	case FormatText:
		delegate = slog.NewTextHandler(&common.buffer, delegateOpts)
	}

	return &handler{
		delegate:    delegate,
		opts:        opts,
		ktmpl:       ktmpl,
		protected:   common,
		replaceAttr: opts.ReplaceAttr,
	}
}

type handler struct {
	protected   *protected
	keys        []string
	delegate    slog.Handler
	opts        *Options
	ktmpl       *template.KeyedTemplate
	replaceAttr func([]string, slog.Attr) slog.Attr
}

func (h *handler) clone(delegate slog.Handler) *handler {
	return &handler{
		protected:   h.protected,
		keys:        h.keys,
		delegate:    delegate,
		opts:        h.opts,
		ktmpl:       h.ktmpl,
		replaceAttr: h.replaceAttr,
	}
}

// Enabled implements slog.Handler
func (h *handler) Enabled(ctx context.Context, l slog.Level) bool {
	return h.delegate.Enabled(ctx, l)
}

// WithAttrs implements slog.Handler
func (h *handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return h.clone(h.delegate.WithAttrs(attrs))
}

// WithGroup implements slog.Handler
func (h *handler) WithGroup(name string) slog.Handler {
	return h.clone(h.delegate.WithGroup(name))
}

// Handle implements slog.Handler
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

	h.protected.Lock()
	defer h.protected.Unlock()
	formattedValue, err := h.formatRecordLocked(ctx, r)
	if err != nil {
		return err
	}
	data[extraAttrsKey] = formattedValue
	return h.ktmpl.Execute(h.protected.out, data)
}

func (h *handler) attributes(r slog.Record) map[string]slog.Attr {
	source := callSource(r)
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

func callSource(r slog.Record) string {
	if r.PC == 0 {
		return "NoFileInfo:0"
	}
	fs := runtime.CallersFrames([]uintptr{r.PC})
	f, _ := fs.Next()
	return fmt.Sprintf("%s:%d", f.File, f.Line)
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

// protected embeds sync.Mutex and wraps the io.Writer and buffer that are protected
// by the mutex.
type protected struct {
	out    io.Writer
	buffer bytes.Buffer
	sync.Mutex
}

func (h *handler) formatRecordLocked(ctx context.Context, r slog.Record) (string, error) {
	defer h.protected.buffer.Reset()
	if err := h.delegate.Handle(ctx, r); err != nil {
		return "", fmt.Errorf("error when calling inner handler's Handle: %w", err)
	}
	formatted := h.protected.buffer.String()
	if h.opts.FormatMode == FormatJSONIndent {
		var err error
		formatted, err = reformatJSON(h.protected.buffer.Bytes())
		if err != nil {
			return "", err
		}
	}
	if len(formatted) > 0 && h.opts.Colorize {
		formatted = colorize(darkGray, formatted)
	}
	return formatted, nil
}

func reformatJSON(s []byte) (string, error) {
	var attrs map[string]any
	err := json.Unmarshal(s, &attrs)
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
