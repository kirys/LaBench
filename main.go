package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// ─── API Types ────────────────────────────────────────────────────────────────

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatRequest struct {
	Model       string        `json:"model"`
	Messages    []ChatMessage `json:"messages"`
	Stream      bool          `json:"stream"`
	MaxTokens   *int          `json:"max_tokens,omitempty"`
	Temperature *float64      `json:"temperature,omitempty"`
}

type ChatResponse struct {
Choices []struct {
	Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
	} `json:"message"`
} `json:"choices"`
Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
} `json:"usage"`
Timings struct {
	PromptPerSecond   float64 `json:"prompt_per_second"`
	PredictedPerSecond float64 `json:"predicted_per_second"`
} `json:"timings"`
}

type TestResult struct {
	Name       string
	Duration   time.Duration
	Status     string
	ErrMsg     string
	PromptTok  int
	RespTok    int
	TokPerSec  float64
	PrefillTok float64 // prompt_per_second (server-reported)
	DecodeTok  float64 // predicted_per_second (server-reported)
	RespBytes  int
}
// ─── Test Execution ───────────────────────────────────────────────────────────

func runTest(client *http.Client, endpoint, model, prompt string, maxTokens int, verbose bool) TestResult {
	res := TestResult{Status: "ok"}

	temp := 0.0
	body := ChatRequest{
		Model:    model,
		Messages: []ChatMessage{{Role: "user", Content: prompt}},
		Stream:   false,
		Temperature: &temp,
	}
	if maxTokens > 0 {
		body.MaxTokens = &maxTokens
	}

	payload, err := json.Marshal(body)
	if err != nil {
		res.Status = "error"
		res.ErrMsg = "marshal: " + err.Error()
		return res
	}

	start := time.Now()
	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		res.Duration = time.Since(start)
		res.Status = "error"
		res.ErrMsg = "new request: " + err.Error()
		return res
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	res.Duration = time.Since(start)
	if err != nil {
		res.Status = "error"
		res.ErrMsg = "http: " + err.Error()
		return res
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	res.RespBytes = len(respBody)

	if resp.StatusCode != http.StatusOK {
		res.Status = "error"
		res.ErrMsg = fmt.Sprintf("status %d: %s", resp.StatusCode, truncate(string(respBody), 200))
		return res
	}

	var chatResp ChatResponse
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		res.Status = "error"
		res.ErrMsg = "unmarshal: " + err.Error()
		return res
	}

	res.PromptTok = chatResp.Usage.PromptTokens
	res.RespTok = chatResp.Usage.CompletionTokens
	res.PrefillTok = chatResp.Timings.PromptPerSecond
	res.DecodeTok = chatResp.Timings.PredictedPerSecond
	if res.Duration > 0 && res.RespTok > 0 {
		res.TokPerSec = float64(res.RespTok) / res.Duration.Seconds()
		// Fallback for servers that don't report timings (e.g. non-llama.cpp)
		if res.DecodeTok == 0 {
			res.DecodeTok = res.TokPerSec
		}
	}

	if verbose && len(chatResp.Choices) > 0 {
		fmt.Printf("  ┌─ %s\n", strings.Repeat("─", 50))
		fmt.Printf("  │ %s\n", truncate(chatResp.Choices[0].Message.Content, 300))
		fmt.Printf("  └─\n")
	}

	return res
}

// ─── Reporting ────────────────────────────────────────────────────────────────

func printReport(results []TestResult, wallClock time.Duration) {
	const hdr = "  %-30s %8s  %6s  %6s  %10s  %10s  %9s"
	fmt.Printf(hdr, "TEST", "DUR(ms)", "P.TOK", "C.TOK", "PREFILL", "DECODE", "STATUS")
	fmt.Println()
	var okCount, errCount int
	var totalPromptTok, totalCompTok int
	var maxDur, minDur time.Duration
	var totalDur time.Duration

	for _, r := range results {
		if r.Status == "ok" {
			okCount++
		} else {
			errCount++
		}
		totalPromptTok += r.PromptTok
		totalCompTok += r.RespTok
		totalDur += r.Duration

		if minDur == 0 || r.Duration < minDur {
			minDur = r.Duration
		}
		if r.Duration > maxDur {
			maxDur = r.Duration
		}

		durMs := r.Duration.Milliseconds()
		fmt.Printf("  %-30s %8d  %6d  %6d  %10.1f  %10.1f  %9s",
			   truncate(r.Name, 30), durMs, r.PromptTok, r.RespTok, r.PrefillTok, r.DecodeTok,
			   statusStr(r),
		)
		if r.Status == "error" {
			fmt.Printf("  %s", truncate(r.ErrMsg, 60))
		}
		fmt.Println()
	}

	fmt.Printf("  %s\n", strings.Repeat("─", 84))
	fmt.Println()
	fmt.Printf("  Tests:          %d total  (%d ok, %d errors)\n", len(results), okCount, errCount)
	fmt.Printf("  Wall clock:     %s\n", wallClock.Round(time.Millisecond))
	fmt.Printf("  Total request:  %s\n", totalDur.Round(time.Millisecond))
	if len(results) > 0 {
		fmt.Printf("  Avg latency:    %s\n", (totalDur / time.Duration(len(results))).Round(time.Millisecond))
		fmt.Printf("  Min latency:    %s\n", minDur.Round(time.Millisecond))
		fmt.Printf("  Max latency:    %s\n", maxDur.Round(time.Millisecond))
	}
	fmt.Printf("  Prompt tokens:  %d\n", totalPromptTok)
	fmt.Printf("  Comp tokens:    %d\n", totalCompTok)
	if totalDur > 0 && totalCompTok > 0 {
		fmt.Printf("  Throughput:     %.1f tok/s (aggregate)\n", float64(totalCompTok)/totalDur.Seconds())
	}
	if wallClock > 0 && totalCompTok > 0 {
		fmt.Printf("  Wall throughput: %.1f tok/s\n", float64(totalCompTok)/wallClock.Seconds())
	}
}

func statusStr(r TestResult) string {
	if r.Status == "ok" {
		return "✓ ok"
	}
	return "✗ ERR"
}

func truncate(s string, n int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) <= n {
		return s
	}
	return s[:n-3] + "..."
}

// ─── Main ─────────────────────────────────────────────────────────────────────

func main() {
	server := flag.String("server", "http://localhost:8080", "Base URL of the OpenAI-compatible server")
	conc   := flag.Int("concurrency", 4, "Maximum concurrent requests")
	dir    := flag.String("dir", "./tests", "Directory containing .md test files (one per test)")
	model  := flag.String("model", "default", "Model name sent in the request body")
	timeout := flag.Duration("timeout", 120*time.Second, "Per-request timeout")
	maxTok := flag.Int("max-tokens", 0, "Set max_tokens in the request (0 = omit field)")
	verbose := flag.Bool("v", false, "Print model response text for each test")
	flag.Parse()

	if *conc < 1 {
		log.Fatal("-concurrency must be >= 1")
	}

	// Load .md files
	entries, err := os.ReadDir(*dir)
	if err != nil {
		log.Fatalf("Cannot read test directory %q: %v", *dir, err)
	}

	type testDef struct {
		name   string
		prompt string
	}
	var tests []testDef
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		path := filepath.Join(*dir, e.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			log.Printf("  WARN: skipping %s: %v", e.Name(), err)
			continue
		}
		tests = append(tests, testDef{
			name:   strings.TrimSuffix(e.Name(), ".md"),
			prompt: string(data),
		})
	}
	if len(tests) == 0 {
		log.Fatalf("No .md files found in %q", *dir)
	}
	sort.Slice(tests, func(i, j int) bool { return tests[i].name < tests[j].name })

	fmt.Printf("  Server:      %s\n", *server)
	fmt.Printf("  Model:       %s\n", *model)
	fmt.Printf("  Tests:       %d\n", len(tests))
	fmt.Printf("  Concurrency: %d\n", *conc)
	fmt.Printf("  Timeout:     %s\n", *timeout)
	fmt.Println()

	client := &http.Client{Timeout: *timeout}
	var endpoint string
	base := strings.TrimRight(*server, "/")
	if strings.HasSuffix(base, "/v1") {
		endpoint = base + "/chat/completions"
	} else {
		endpoint = base + "/v1/chat/completions"
	}

	results := make([]TestResult, len(tests))
	var wg sync.WaitGroup
	sem := make(chan struct{}, *conc)

	wallStart := time.Now()
	for i, t := range tests {
		wg.Add(1)
		go func(idx int, tc testDef) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			r := runTest(client, endpoint, *model, tc.prompt, *maxTok, *verbose)
			r.Name = tc.name
			results[idx] = r
		}(i, t)
	}
	wg.Wait()
	wallClock := time.Since(wallStart)

	fmt.Println()
	printReport(results, wallClock)
}
