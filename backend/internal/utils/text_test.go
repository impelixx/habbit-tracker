package utils

import "testing"

func TestTruncateText(t *testing.T) {
tests := []struct {
name      string
text      string
maxLength int
want      string
}{
{name: "short text unchanged", text: "hello", maxLength: 10, want: "hello"},
{name: "equal length unchanged", text: "hello", maxLength: 5, want: "hello"},
{name: "truncate with ellipsis", text: "hello world", maxLength: 8, want: "hello..."},
{name: "maxLength less than 3 returns original", text: "hello", maxLength: 2, want: "hello"},
{name: "maxLength negative returns original", text: "hello", maxLength: -1, want: "hello"},
}

for _, tt := range tests {
t.Run(tt.name, func(t *testing.T) {
got := TruncateText(tt.text, tt.maxLength)
if got != tt.want {
t.Errorf("TruncateText() = %q, want %q", got, tt.want)
}
})
}
}
