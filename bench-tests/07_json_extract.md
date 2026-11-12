Extract the error events from the log below. Respond with ONLY a JSON
array — no markdown fences, no explanation. Each object must have keys:
timestamp (ISO string), service, level, code, message.

2025-11-01T08:12:03Z info  gateway  request accepted id=9f2 path=/users
2025-11-01T08:12:04Z warn  auth     slow token check 412ms
2025-11-01T08:12:09Z error gateway  E503 upstream timeout after 3000ms
2025-11-01T08:12:10Z error db       E1042 connection pool exhausted
2025-11-01T08:12:11Z info  gateway  request accepted id=9f3 path=/orders
2025-11-01T08:12:15Z error auth     E2011 invalid api key format
2025-11-01T08:12:16Z info  db       pool recovered
