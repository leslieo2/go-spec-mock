# Dynamic Mocking

Dynamic mocking lets you bend responses on the fly without editing your OpenAPI document or restarting the server. Use special query parameters to override the HTTP status returned or to simulate slow networks.

## Available Query Parameters

| Parameter | Purpose | Examples |
|-----------|---------|----------|
| `__statusCode` | Force the mock to reply with a specific HTTP status code. | `?__statusCode=404`, `?__statusCode=201` |
| `__delay` | Apply artificial latency before a response is sent. Accepts numbers (milliseconds) or Go duration strings. | `?__delay=500`, `?__delay=750ms`, `?__delay=2s` |
| `__example` | Select a named example from the OpenAPI response definition. | `?__example=success`, `?__example=premiumTier` |

You can mix these parameters with ordinary query arguments. Internal `__` parameters are ignored by response generation logic (other than their intended effect) and are excluded from response cache keys, so they do not interfere with application-level filtering or caching.

## Status Code Overrides (`__statusCode`)

- Accepts any integer between `100` and `599`.
- Invalid values are ignored and logged; the response falls back to the status declared in the specification (or `200` if unspecified).
- The mock response body is selected from the example that matches the requested status. If an example for that code does not exist, the server falls back to a 2xx example when available.

```bash
# Simulate a not-found error for the /users endpoint
curl "http://localhost:8080/users?__statusCode=404"

# Exercise retry logic with a transient 503
curl "http://localhost:8080/payments?__statusCode=503"
```

## Latency Simulation (`__delay`)

- You can send a raw number (`500`) which is treated as milliseconds, or a Go duration string (`750ms`, `2s`, `1.5s`).
- Delays longer than 30 seconds are capped at 30 seconds. Negative delays are treated as no delay.
- Invalid duration strings are ignored and logged so they do not break your flow.

```bash
# Wait half a second before returning the mocked response
curl "http://localhost:8080/users?__delay=500ms"

# Provide the value in milliseconds
curl "http://localhost:8080/search?__delay=1500"
```

## Combining Parameters

```bash
curl "http://localhost:8080/orders/123?__statusCode=500&__delay=2s&region=emea"
```

The example above returns the 500 response example after a 2 second pause while still honoring real query parameters like `region=emea`.

## Selecting Named Examples (`__example`)

- Match the name defined under `content.application/json.examples` in your OpenAPI specification.
- If the named example is missing, Go-Spec-Mock falls back to the default example or schema-generated data when possible; otherwise you receive a 404 indicating no example is available.
- The selected example participates in response caching, so repeated requests for the same example are served efficiently.

```bash
# Serve the "premium" example for scenarios that need richer data
curl "http://localhost:8080/subscriptions?__statusCode=200&__example=premium"

# Combine with latency to mimic slow premium-plan provisioning
curl "http://localhost:8080/subscriptions?__example=premium&__delay=1.5s"
```

## Practical Scenarios

- **Frontend edge cases:** Trigger error templates or timeout spinners without touching backend code.
- **Automated testing:** Drive end-to-end tests that issue the same request multiple times with different status codes, named examples, and delays to verify resilience logic.
- **Demo environments:** Showcase how your client reacts to flaky networks or premium features by sharing one mock server URL that teammates can tweak with query parameters.
