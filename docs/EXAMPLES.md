## Examples

### Todo Agent (MVP)

- Files: `examples/todo/agent.go`
- Endpoint: `POST /api/examples/todo`

Request body

```json
{
  "RunID": "<run-id>",
  "Type": "add_task|complete_task",
  "Payload": { "title": "demo" }
}
```

Behavior
- `add_task`: Pure reducer update; emits no side-effects.
- `complete_task`: Increments done count and emits a `log` intent; the `LoggerEffect` produces a `logged` event.

Tips
- Use `ORCH_OTEL_STDOUT=1` to observe spans for handler and effect execution.
- Fetch events via `GET /api/events?run=<run-id>`.


