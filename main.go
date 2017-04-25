package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

var (
	nonPrintable = regexp.MustCompile("[^[:print:]\n\r]")
)

type test struct {
	Id        string
	Timestamp string
	Duration  string
	Name      string
	Result    int
	Message   string
}

func main() {
	input := os.Args[1]
	output := os.Args[2]

	if err := run(input, output); err != nil {
		log.Fatalln(err)
	}
}

func run(input, output string) error {
	tests, err := readStats(filepath.Join(input, "TESTS.csv"))
	if err != nil {
		return err
	}

	report, err := os.Create(output)
	if err != nil {
		return err
	}

	w := bufio.NewWriter(report)

	if _, err := fmt.Fprintf(w, "<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n<testsuites>\n\t<testsuite tests=\"%d\">\n", len(tests)); err != nil {
		return err
	}

	for _, test := range tests {
		if _, err := fmt.Fprintf(w, "\t\t<testcase classname=\"pinata\" name=\"%s\" time=\"%s\">\n", test.Name, test.Duration); err != nil {
			return err
		}

		if test.Result == 1 {
			if err != nil {
				return err
			}

			if _, err := fmt.Fprint(w, "\t\t\t<failure type=\"Error\"/>\n"); err != nil {
				return err
			}
		} else if test.Result == 2 {
			if _, err := fmt.Fprint(w, "\t\t\t<skipped/>\n"); err != nil {
				return err
			}
		}

		if test.Result != 2 {
			testLog, err := readTestLog(filepath.Join(input, test.Name+".log"))
			if err != nil {
				return err
			}

			sanitizedOutput := sanitizeOutput(testLog)

			if _, err := fmt.Fprintf(w, "\t\t\t<system-out><![CDATA[%s]]></system-out>\n", sanitizedOutput); err != nil {
				return err
			}
		}

		if _, err := fmt.Fprint(w, "\t\t</testcase>\n"); err != nil {
			return err
		}
	}

	if _, err := w.WriteString("\t</testsuite>\n</testsuites>\n"); err != nil {
		return err
	}

	return w.Flush()
}

func readStats(csvPath string) ([]test, error) {
	tests := []test{}

	file, err := os.Open(csvPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for index := 0; scanner.Scan(); index++ {
		line := scanner.Text()
		if index == 0 {
			continue // Skip header line
		}

		fields := strings.Split(line, ",")

		result, err := strconv.Atoi(fields[4])
		if err != nil {
			return nil, err
		}

		tests = append(tests, test{
			Id:        fields[0],
			Timestamp: fields[1],
			Duration:  fields[2],
			Name:      fields[3],
			Result:    result,
			Message:   fields[5],
		})
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return tests, nil
}

func readTestLog(path string) (string, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return "", nil
	}

	b, err := ioutil.ReadFile(path)
	if err != nil {
		return "", err
	}

	return string(b), nil
}

func sanitizeOutput(output string) string {
	printable := nonPrintable.ReplaceAllLiteralString(output, "")
	noOpeningData := strings.Replace(printable, "<![CDATA[", "", -1)
	noClosingData := strings.Replace(noOpeningData, "]]>", "", -1)

	return noClosingData
}
