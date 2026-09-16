# CPA → Pi integration

This fork maintains compatibility fixes for the following route:

```text
Pi (OpenAI Chat Completions) → CPA → devin-2api → Devin
```

CPA's `openai-compatibility` provider calls this adapter's `/v1/chat/completions`.
Register SWE-2 once as `devin/swe-2` (adapter model `swe-2`) without an additional
provider `prefix`. Pi's existing CPA model-directory integration can discover
the alias and its reasoning levels; a second Pi provider or a second copy of the Devin token is not
needed. A model appearing in the upstream directory does not prove entitlement
or that a generation will succeed.

## Changes in this fork

- SWE-2 uses a single public model ID. Chat `reasoning_effort` and Responses
  `reasoning.effort` select `medium`, `high`, or `max`; omission defaults to high.
  Only the adapter translates this to the corresponding upstream model variant.
  Response model names remain `swe-2`. Unsupported efforts return a client error.
  Explicit effort on other model families is rejected until a mapping exists.

- Responses history reconstructs consecutive assistant text/function-call items
  as one assistant turn, preserving parallel tool calls and their result IDs.
  User messages and tool results remain turn boundaries.
- All three HTTP codecs carry output-token budgets, temperature and top-p through
  the common model layer into the upstream request. Anthropic top-k is also
  forwarded. Explicit zero temperature is preserved; invalid numeric settings
  return a client error. Omitted settings keep the upstream project's defaults.
- Chat history preserves `reasoning_content`. Invalid tool-argument JSON is
  rejected instead of silently replaced with `{}`.
- Responses requests using `previous_response_id` receive an explicit error:
  clients must send the complete conversation. No response storage is provided.

These are request-conversion guarantees, not a claim that every upstream model
honors every generation parameter.

## Private host deployment

1. Generate protobuf bindings using `task generate`, then cross-compile:

   ```sh
   CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
     '-gcflags=local/devinproto/devinprotoconnect=-N -l' -trimpath -ldflags='-s -w' \
     -o bin/devin-2api-linux-amd64 ./cmd/devin-2api
   ```

   The package-specific compiler flags avoid expensive optimization of the
   upstream's large generated RPC client. Application code keeps normal Go
   optimizations. Use the same flag with `go test ./...` for offline checks on
   memory-constrained development machines.

2. Install the binary at `/data/data1/apps/devin-2api/devin-2api` (0755).
   Adapt the service's binary path for other hosts.
3. Copy `deploy/config.example.yaml` to `/etc/devin-2api/config.yaml`, fill in
   credentials privately and set mode 0600. The `swe-2` family alias routes to
   the account's SWE-2 variants. Use distinct random API and dashboard keys. Adjust or omit `proxy`
   according to the host's outbound network.
4. Install `deploy/devin-2api.service` in `/etc/systemd/system/` and enable it.
   This unit requires systemd with `LoadCredential` support (Ubuntu 22.04 works).
   The service runs as a dynamic unprivileged user and reads a private credential
   copy. It only listens on loopback; the dashboard has no public route.
5. Back up CPA's configuration, then merge the provider in `deploy/cpa.example.yaml`
   with the matching internal API key and unified model ID. CPA must
   reach the adapter on the same host network. Set per-key `proxy-url: direct`
   so loopback requests do not enter CPA's global outbound proxy.
6. Refresh Pi's existing CPA model list and choose `devin/swe-2`. Select medium,
   high or max using Pi's thinking controls (or `/gateway-thinking`). Do not infer
   model capabilities or prices from similarly named models at other providers.
   The upstream-default choice omits the effort and uses high. Numeric thinking
   budgets are not mapped. Configure context/output budgets only
   from verified catalog information or explicit local limits.

Configuration is read once at startup; credential rotation requires restarting
this service. Keep real credentials, logs and account snapshots outside Git.
Leave debug logging disabled because it records conversation content.

## Remaining limitations

- `tool_choice`, stop sequences, structured-output constraints, numeric thinking
  budgets and effort mapping for families other than SWE-2 are not implemented end to end.
- Image input must be embedded as Base64; earlier turns' images are placeholders.
- WebSocket transport handles one response per connection. Prefer HTTP/SSE for
  the CPA/Pi route.
- Catalog availability, quota, latency and model behavior need account-specific
  user validation. Offline regression checks do not establish these properties.

## Rollback

Remove or disable only CPA's `devin` provider and refresh Pi. Stop and disable
`devin-2api.service`. For a binary-only rollback, restore the previous binary and
restart this service; do not replace unrelated CPA configuration with an old copy.

The fork keeps `upstream` pointing to `leookun/devin-2api`. Preserve the original
MIT license. Do not publish upstream release tags without first changing the
inherited Docker Hub release workflow to your own registry.
