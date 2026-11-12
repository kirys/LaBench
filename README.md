# LaBench
Simple Benchmark tool for llm


go build -o llama-bench .

### Basic run
```bash
./labench -server http://localhost:8080 -concurrency 4 -dir ./tests
```

### With a specific model, 16 concurrent, verbose output, 256 max tokens
```bash
./labench \
  -server http://10.0.0.5:9997 \
  -concurrency 16 \
  -model llama-3-70b \
  -dir ./tests \
  -max-tokens 512 \
  -timeout 180s \
  -v
```
