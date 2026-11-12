Write a thread-safe LRU cache in Go.

Requirements:
- A generic-style LRU (you may use interface{} values).
- Cap constructor: NewLRU(capacity int) *LRU.
- Get(key string) (value interface{}, ok bool)
- Put(key string, value interface{})
- Len() int
- Must be safe for concurrent use (sync.Mutex or RWMutex).
- O(1) get and put.

Provide the complete Go source for a single file lru.go, plus a small
TestLRU function in the same file covering: basic put/get, eviction of
the least recently used entry, and that a Get refreshes recency.
EOF

cat > "$DIR/03_code_python_http.md" <<'EOF'
Write a minimal Python REST API client, in a single file client.py.

Requirements:
- Class ApiClient(base_url: str, api_key: str)
- Methods: get(path, params=None), post(path, json_body=None),
	delete(path), each returning the parsed JSON response.
- Use only stdlib (urllib.request) — no third-party packages.
- Raise a custom ApiError(status, message) on non-2xx responses.
- Attach the API key as an "X-Api-Key" header.
- Support a per-request timeout parameter (default 30s).

Include a __main__ demo block that hits http://127.0.0.1:8080/ping
and prints the result or the error.
