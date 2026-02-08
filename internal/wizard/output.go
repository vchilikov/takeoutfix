package wizard

import (
	"fmt"
	"io"
)

func writeLine(out io.Writer, msg string) {
	_, _ = fmt.Fprintln(out, msg)
}

func writef(out io.Writer, format string, args ...any) {
	_, _ = fmt.Fprintf(out, format, args...)
}
