package prettylog

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"
	"time"
)

func TestHandler(t *testing.T) {
	type modeCase struct {
		mode FormatMode
		want string
	}

	cases := []struct {
		name      string
		options   *Options
		msg       string
		args      []any
		modeCases []modeCase
	}{
		{
			name: "no args",
			msg:  "UserMessage",
			modeCases: []modeCase{
				{FormatText, `[00:00:00.000] INFO NoFileInfo:0: UserMessage `},
				{FormatJSON, `[00:00:00.000] INFO NoFileInfo:0: UserMessage {}`},
			},
		},
		{
			name: "key-values",
			msg:  "UserMessage",
			args: []any{
				"key1", "value",
				"key2", "value",
			},
			modeCases: []modeCase{
				{FormatText, `[00:00:00.000] INFO NoFileInfo:0: UserMessage key1=value key2=value`},
				{FormatJSON, `[00:00:00.000] INFO NoFileInfo:0: UserMessage {"key1":"value","key2":"value"}`},
				{FormatJSONIndent, "[00:00:00.000] INFO NoFileInfo:0: UserMessage\n{\n  \"key1\": \"value\",\n  \"key2\": \"value\"\n}"},
			},
		},
		{
			name: "colorize",
			msg:  "UserMessage",
			options: &Options{
				Colorize: true,
			},
			args: []any{
				"key1", "value",
				"key2", "value",
			},
			modeCases: []modeCase{
				{FormatJSON, "[\x1b[37m00:00:00.000\x1b[0m] \x1b[36mINFO\x1b[0m NoFileInfo:0: \x1b[97mUserMessage\x1b[0m \x1b[90m{\"key1\":\"value\",\"key2\":\"value\"}\n\x1b[0m"},
				{FormatText, "[\x1b[37m00:00:00.000\x1b[0m] \x1b[36mINFO\x1b[0m NoFileInfo:0: \x1b[97mUserMessage\x1b[0m \x1b[90mkey1=value key2=value\n\x1b[0m"},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			for _, mc := range tc.modeCases {
				var buf bytes.Buffer
				r := slog.NewRecord(time.Time{}, slog.LevelInfo, tc.msg, 0)
				r.Add(tc.args...)

				t.Run(string(mc.mode), func(t *testing.T) {
					o := tc.options
					if o == nil {
						o = &Options{}
					}
					o.FormatMode = mc.mode
					h := NewPrettyHandler(&buf, o)

					if err := h.Handle(context.Background(), r); err != nil {
						t.Errorf("Error calling Handle(): %s", err)
					}

					got := strings.TrimSuffix(buf.String(), "\n")
					if got != mc.want {
						t.Errorf("\ngot    %q \nwanted %q", got, mc.want)
					}
				})
			}
		})
	}
}
