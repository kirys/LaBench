Translate the following paragraph into Italian. Preserve the technical
tone and keep the terms "throughput", "latency", and "backpressure" in
English (italicized). Do not add commentary.

Under load, the queue depth grew linearly with arrival rate until the
worker pool saturated. After saturation, throughput flattened while
latency climbed sharply. The fix was to introduce backpressure at the
ingress gate: requests arriving faster than the service rate are now
rejected early with a 429 and a Retry-After hint, protecting the worker
pool from unbounded queue growth.
