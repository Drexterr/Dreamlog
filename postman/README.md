# Typed-journal HTTP verification

This focused OpenAPI contract supplements `docs/API_CONTRACT.md`. The stateful
mock covers typed-entry creation and entry reads; unknown endpoints return 501.
It uses fictional journals, never an account or the real AI service.

With Postman CLI 1.73+ and the mobile dependencies installed:

```powershell
postman simulation run postman/simulations/journaling.sim.yaml --port 4917 --output ndjson --no-history
```

In a second terminal:

```powershell
node scripts/test-journaling-client.cjs
```

The check transpiles the actual Axios client, replacing only SecureStore with a
Node adapter holding a fictional token. It sends real HTTP requests through the
client interceptors and checks create/read, identical retry, conflicting retry,
bad content and a missing entry. Override `ODE_SIM_URL` to change the port.
The fixed `x-mock-session` header is confined to this check. A mock cannot establish
real authentication, quota enforcement, database atomicity or AI safety; backend
tests and the existing integration CI cover those boundaries.

No owning Postman workspace is bound/published; this is a locally owned API.
Do not commit simulation logs containing data.
