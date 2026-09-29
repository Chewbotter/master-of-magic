package font

// Lines of styled text, see styled.go.

import (
    "strings"
)

// the text in lines that are no wider than width, in art pixels, broken between words. size is
// how large the letters are compared to the art (RelativeTextSize), 1 for the full size
func (styled *StyledFont) Wrap(text string, width float64, size float64) []string {
    if size <= 0 {
        size = 1
    }

    var lines []string
    for _, paragraph := range strings.Split(text, "\n") {
        line := ""
        for _, word := range strings.Fields(paragraph) {
            longer := word
            if line != "" {
                longer = line + " " + word
            }

            if line != "" && float64(styled.Width(longer)) * size > width {
                lines = append(lines, line)
                line = word
            } else {
                line = longer
            }
        }
        lines = append(lines, line)
    }

    return lines
}

// the width of the text in art pixels at a size
func (styled *StyledFont) WidthAt(text string, size float64) float64 {
    if size <= 0 {
        size = 1
    }
    return float64(styled.Width(text)) * size
}
