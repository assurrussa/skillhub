package core

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

type CatalogRow struct {
	Name        string
	Category    string
	Triggers    string
	Description string
}

var CatalogTable = Table[CatalogRow]{
	Header:  "name\tcategory\ttriggers\tdescription",
	Columns: 4,
	Parse: func(row []string) (CatalogRow, error) {
		return CatalogRow{Name: row[0], Category: row[1], Triggers: row[2], Description: row[3]}, nil
	},
	Format: func(item CatalogRow) []string {
		return []string{item.Name, item.Category, item.Triggers, item.Description}
	},
}

type Table[T any] struct {
	Header  string
	Columns int
	Parse   func(row []string) (T, error)
	Format  func(item T) []string
}

func (t Table[T]) ReadFile(path string) ([]T, error) {
	rows, err := readTSVFile(path, t.Header, t.Columns)
	if err != nil {
		return nil, err
	}
	return t.parseRows(rows)
}

func (t Table[T]) ReadString(label, data string) ([]T, error) {
	rows, err := readTSV(strings.NewReader(data), label, t.Header, t.Columns)
	if err != nil {
		return nil, err
	}
	return t.parseRows(rows)
}

func (t Table[T]) parseRows(rows [][]string) ([]T, error) {
	out := make([]T, 0, len(rows))
	for _, row := range rows {
		item, err := t.Parse(row)
		if err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, nil
}

func (t Table[T]) WriteFile(path string, items []T) error {
	rows := make([][]string, 0, len(items))
	for _, item := range items {
		rows = append(rows, t.Format(item))
	}
	return writeTSVFile(path, t.Header, rows)
}

type fileReader interface {
	Read(p []byte) (n int, err error)
}

func readTSVFile(path, header string, columns int) ([][]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	return readTSV(file, path, header, columns)
}

func readTSV(scannerInput fileReader, label, header string, columns int) ([][]string, error) {
	scanner := bufio.NewScanner(scannerInput)
	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("%s is empty", label)
	}
	if scanner.Text() != header {
		return nil, fmt.Errorf("%s header must be: %s", label, strings.ReplaceAll(header, "\t", "<TAB>"))
	}
	rows := [][]string{}
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.Split(line, "\t")
		if len(parts) != columns {
			return nil, fmt.Errorf("%s row has wrong column count: %q", label, line)
		}
		rows = append(rows, parts)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return rows, nil
}

func writeTSVFile(path, header string, rows [][]string) error {
	if err := os.MkdirAll(filepathDir(path), 0o755); err != nil {
		return err
	}
	var b strings.Builder
	_, _ = b.WriteString(header)
	_ = b.WriteByte('\n')
	for _, row := range rows {
		for i, value := range row {
			if i > 0 {
				_ = b.WriteByte('\t')
			}
			_, _ = b.WriteString(sanitizeTSVField(value))
		}
		_ = b.WriteByte('\n')
	}
	return os.WriteFile(path, []byte(b.String()), 0o644)
}

func sanitizeTSVField(value string) string {
	value = strings.ReplaceAll(value, "\t", " ")
	value = strings.ReplaceAll(value, "\r", " ")
	value = strings.ReplaceAll(value, "\n", " ")
	return strings.TrimSpace(strings.Join(strings.Fields(value), " "))
}

func filepathDir(path string) string {
	if idx := strings.LastIndexAny(path, `/\`); idx >= 0 {
		return path[:idx]
	}
	return "."
}
