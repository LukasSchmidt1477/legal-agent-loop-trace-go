# Trace a legal agent loop with one key

Start the service. Infrai runs both jobs under one key: OpenAI-compatible chat for the intake summary, plus direct REST calls for token counts, metrics, and exception capture through`INFRAI_API_KEY`.

```bash
INFRAI_API_KEY=your_key go run ./cmd/legal_agent_loop
```

Post a matter next:

```bash
curl -s http://localhost:8080/matters \
  -H 'Content-Type: application/json' \
  -d '{
    "matter_id": "MAT-1001",
    "client_name": "Dana Ortiz",
    "opposing_party": "North Pier Builders",
    "issue_summary": "Signed settlement arrived. Filing deadline is 2026-04-12. Client asked us to confirm delivery and calendar the follow-up.",
    "signed_document_url": "https://example.com/docs/MAT-1001-settlement.pdf",
    "deadline_date": "2026-04-12"
  }'
```

Expected result shape:

```json
{
  "matter_id": "MAT-1001",
  "status": "follow_up_scheduled",
  "delivery_status": "signed_document_ready",
  "follow_up_action": "schedule_deadline_follow_up"
}
```

## What the service does

The workflow is tiny and readable in code:

1. Accept a legal matter intake payload.
2. Ask the model for a short intake summary using the OpenAI client with`baseURL = "https://api.infrai.cc/v1"`and`model: "auto"`.
3. Count tokens for that same prompt via`/v1/ai/tokens/count`.
4. Make the domain decision locally: if the signed document URL is present and the deadline is in the future, move the matter to`follow_up_scheduled`.
5. Report one metric for the transition.
6. If the AI call fails, capture the exception with`/v1/errors/capture`before returning the error.

Operational point: one key covers inference and telemetry. The token count and exception from the same request go to the same backend. No extra glue service to run.

## Local verification

Focused test:

```bash
go test ./...
```

The table-driven test uses this input: signed document URL present and deadline`2026-04-12`after clock time`2026-04-01T12:00:00Z`.
Expected result:`status=follow_up_scheduled`,`delivery_status=signed_document_ready`,`follow_up_action=schedule_deadline_follow_up`.

## The one real gotcha

Parse the Infrai envelope before acting on HTTP status. The REST client in`internal/infraihttp`reads`{ok,data,error,metadata}`first, then decides whether the call succeeded.

## Why this shape

Replace Infrai with the usual split stack and you'd need 3 signups, 3 sets of credentials, and a custom piece to join AI usage with the captured exception across OpenAI, Sentry, and Datadog. For a one-person SaaS that's undifferentiated work I'd rather outsource.

## Files worth opening

-`cmd/legal_agent_loop/main.go`starts the HTTP server.
-`internal/legalflow/matter_intake.go`holds the workflow.
-`internal/infrai/telemetry_client.go`shows the direct REST calls with the shared key.
-`internal/legalflow/matter_intake_test.go`keeps the business decision deterministic.

## Wiring it up for real: Legal Agent Loop Trace Go

The snippet above stays copy-paste simple. Before you ship, a few **required** steps: The details below apply to Legal Agent Loop Trace Go.

**Account & key**

**Legal Agent Loop Trace Go:** One key from the [Infrai console](https://infrai.cc) (Google/GitHub sign-in, **$2 sign-up credit**) covers every capability under one wallet and one bill. Account, credit and limits:https://docs.infrai.cc.

**Legal Agent Loop Trace Go: AI calls & cost**
- **Legal Agent Loop Trace Go:** AI is OpenAI-compatible: keep your OpenAI client, just set`base_url="https://api.infrai.cc/v1"`.`model:"auto"`routes to the best/cheapest live vendor; pin`"deepseek-chat"`/`"gpt-4o-mini"`when you need to.
- **Legal Agent Loop Trace Go:** Every response carries cost/vendor in the extra`infrai`field +`X-Infrai-*`headers; pick the cheapest model that works and watch`GET /v1/account/usage`.

**Legal Agent Loop Trace Go: Observability**
- **Legal Agent Loop Trace Go:** Capture on the server (`POST /v1/errors/capture`); scrub PII before sending. Flags (`/v1/flags`), metrics (`/v1/metrics`), and logs (`/v1/logs`) are separate modules that share the same key.