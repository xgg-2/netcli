package export

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/xgg-2/netcli/internal/types"
)

var Formats = []string{"curl", "fetch", "axios", "python", "go"}

func GenerateCode(format string, e *types.RequestEntry) string {
	switch format {
	case "fetch":
		return GenerateFetch(e)
	case "axios":
		return GenerateAxios(e)
	case "python":
		return GeneratePythonRequests(e)
	case "go":
		return GenerateGo(e)
	default:
		return GenerateCurl(e)
	}
}

func GenerateCurl(e *types.RequestEntry) string {
	var b strings.Builder
	b.WriteString("curl -X " + e.Method + " \\\n  '" + e.URL + "'")
	for k, vals := range e.RequestHeaders {
		for _, v := range vals {
			b.WriteString(fmt.Sprintf(" \\\n  -H '%s: %s'", k, v))
		}
	}
	if len(e.RequestBody) > 0 && !e.IsBinaryRequest {
		escaped := strings.ReplaceAll(string(e.RequestBody), "'", "'\\''")
		b.WriteString(" \\\n  --data '" + escaped + "'")
	}
	return b.String()
}

func GenerateFetch(e *types.RequestEntry) string {
	var b strings.Builder
	bodyStr, isJSON := requestBodyAsJSON(e)

	b.WriteString("const response = await fetch('" + e.URL + "', {\n")
	b.WriteString("  method: '" + e.Method + "',\n")
	b.WriteString("  headers: {\n")
	for k, vals := range e.RequestHeaders {
		for _, v := range vals {
			b.WriteString(fmt.Sprintf("    '%s': '%s',\n", escapeJS(k), escapeJS(v)))
		}
	}
	b.WriteString("  },\n")
	if len(e.RequestBody) > 0 && !e.IsBinaryRequest {
		if isJSON {
			b.WriteString("  body: JSON.stringify(" + bodyStr + "),\n")
		} else {
			b.WriteString("  body: " + jsStringLiteral(bodyStr) + ",\n")
		}
	}
	b.WriteString("});\n")
	b.WriteString("const data = await response.json();")
	return b.String()
}

func GenerateAxios(e *types.RequestEntry) string {
	var b strings.Builder
	method := strings.ToLower(e.Method)
	bodyStr, isJSON := requestBodyAsJSON(e)

	b.WriteString("const response = await axios." + method + "(\n")
	b.WriteString("  '" + escapeJS(e.URL) + "',\n")

	if len(e.RequestBody) > 0 && !e.IsBinaryRequest {
		if isJSON {
			b.WriteString("  " + bodyStr + ",\n")
		} else {
			b.WriteString("  " + jsStringLiteral(bodyStr) + ",\n")
		}
	} else {
		b.WriteString("  null,\n")
	}

	b.WriteString("  {\n    headers: {\n")
	for k, vals := range e.RequestHeaders {
		for _, v := range vals {
			b.WriteString(fmt.Sprintf("      '%s': '%s',\n", escapeJS(k), escapeJS(v)))
		}
	}
	b.WriteString("    },\n  }\n);\n")
	return b.String()
}

func GeneratePythonRequests(e *types.RequestEntry) string {
	var b strings.Builder
	method := strings.ToLower(e.Method)
	_, isJSON := requestBodyAsJSON(e)

	if isJSON {
		b.WriteString("import json\n")
	}
	b.WriteString("import requests\n\n")

	b.WriteString("headers = {\n")
	for k, vals := range e.RequestHeaders {
		for _, v := range vals {
			b.WriteString(fmt.Sprintf("    %q: %q,\n", k, v))
		}
	}
	b.WriteString("}\n\n")

	b.WriteString(fmt.Sprintf("response = requests.%s(\n", method))
	b.WriteString(fmt.Sprintf("    %q,\n", e.URL))
	b.WriteString("    headers=headers,\n")

	if len(e.RequestBody) > 0 && !e.IsBinaryRequest {
		raw := string(e.RequestBody)
		if isJSON {
			b.WriteString(fmt.Sprintf("    json=json.loads(%q),\n", raw))
		} else {
			b.WriteString(fmt.Sprintf("    data=%q,\n", raw))
		}
	}

	b.WriteString(")")
	return b.String()
}

func GenerateGo(e *types.RequestEntry) string {
	var b strings.Builder
	hasBody := len(e.RequestBody) > 0 && !e.IsBinaryRequest

	b.WriteString("package main\n\n")
	b.WriteString("import (\n")
	b.WriteString("\t\"fmt\"\n")
	b.WriteString("\t\"net/http\"\n")
	if hasBody {
		b.WriteString("\t\"strings\"\n")
	}
	b.WriteString(")\n\n")
	b.WriteString("func main() {\n")

	if hasBody {
		b.WriteString(fmt.Sprintf("\tbody := strings.NewReader(%s)\n", goStringLiteral(string(e.RequestBody))))
		b.WriteString(fmt.Sprintf("\treq, _ := http.NewRequest(%q, %q, body)\n", e.Method, e.URL))
	} else {
		b.WriteString(fmt.Sprintf("\treq, _ := http.NewRequest(%q, %q, nil)\n", e.Method, e.URL))
	}

	for k, vals := range e.RequestHeaders {
		for _, v := range vals {
			b.WriteString(fmt.Sprintf("\treq.Header.Set(%q, %q)\n", k, v))
		}
	}

	b.WriteString("\tresp, err := http.DefaultClient.Do(req)\n")
	b.WriteString("\tif err != nil {\n")
	b.WriteString("\t\tfmt.Println(err)\n")
	b.WriteString("\t\treturn\n")
	b.WriteString("\t}\n")
	b.WriteString("\tdefer resp.Body.Close()\n")
	b.WriteString("\tfmt.Println(resp.Status)\n")
	b.WriteString("}")
	return b.String()
}

func requestBodyAsJSON(e *types.RequestEntry) (string, bool) {
	if len(e.RequestBody) == 0 || e.IsBinaryRequest {
		return "", false
	}
	raw := string(e.RequestBody)
	var v interface{}
	if err := json.Unmarshal(e.RequestBody, &v); err == nil {
		pretty, err := json.MarshalIndent(v, "", "  ")
		if err == nil {
			return string(pretty), true
		}
	}
	return raw, false
}

func escapeJS(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `'`, `\'`)
	return s
}

func jsStringLiteral(s string) string {
	if !strings.Contains(s, "`") {
		return "`" + s + "`"
	}
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	return `"` + s + `"`
}

func goStringLiteral(s string) string {
	if !strings.Contains(s, "`") {
		return "`" + s + "`"
	}
	return fmt.Sprintf("%q", s)
}
