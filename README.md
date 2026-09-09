**English** | [Русский](README.ru.md)

# KIRTA — AI Security Platform

<p align="left">
  <img src="docs/assets/badges/status.svg" height="36" alt="Status: MVP" />
  <a href="./LICENSE"><img src="docs/assets/badges/license.svg" height="36" alt="License: MIT" /></a>
</p>

<p align="left">
  <img src="https://skillicons.dev/icons?i=python,go,react,ts,postgres,docker,nginx,vite,tailwind&theme=dark" alt="KIRTA core technologies" />
</p>

**KIRTA** is an MVP platform for explainable vulnerability analysis of Python projects. It combines conventional SCA results, static facts from source code, and AI interpretation to help teams identify which findings need attention first.

**Core stack:** Go, Gin, Python, React, TypeScript, PostgreSQL, MinIO/S3, Docker, Nginx  
**Security tooling:** Syft, Grype, Python AST/call tracing  
**AI:** OpenRouter-compatible client for structured finding analysis

---

## What KIRTA addresses

Conventional SCA answers “which known CVEs affect these dependencies?” well, but does not by itself show how a particular library relates to the project's code or what a finding means in the current context.

KIRTA adds source code context to scanner results:

- builds a project SBOM;
- collects SCA findings;
- identifies the use of vulnerable libraries in Python code;
- builds a call map;
- displays source files and evidence lines;
- sends structured security context to an LLM on request;
- returns an explanation and exploitability status for an individual finding.

The main goal is **not to replace a security engineer or automatically prove exploitation**, but to reduce manual initial triage and make technical reports easier to interpret.

## MVP scope

The current implementation has deliberate limitations:

1. only Python code is analyzed;
2. call maps are built only for Python;
3. SBOMs are generated with `Syft`;
4. SCA is performed with `Grype`;
5. SLOC counting and language composition detection are supported;
6. the SCA workflow is implemented; SAST and DAST findings are not yet part of the MVP;
7. scans run synchronously;
8. AI analysis runs on demand for an individual finding;
9. the model is configured through OpenRouter; the example configuration uses `openai/gpt-oss-120b:free`.

These limitations are explicitly documented so that MVP results are not mistaken for a full replacement for a production-grade AppSec platform.

---

## Main workflow

```text
Project ZIP
   ↓
safe extraction
   ↓
Syft → SBOM
   ↓
Grype → SCA findings
   ↓
Python static analysis → imports / calls / call map
   ↓
PostgreSQL + MinIO/S3
   ↓
KIRTA UI
   ↓
On-demand LLM analysis for a finding
```

### What the user receives

- a list of detected SCA findings;
- package, version, CVE, severity, and fixed versions;
- information about package usage in the source code;
- a library call map;
- source evidence viewing in the UI;
- an AI explanation for an individual finding;
- an exploitability status qualified by the limitations of static analysis.

---

## Exploitability statuses

| Status | Meaning |
| --- | --- |
| `exploitable` | Current static facts and AI analysis indicate practical reachability of the defect |
| `not_exploitable` | The current static facts do not confirm use or reachability of the vulnerable scenario |
| `unknown` | There is insufficient data for a confident conclusion; manual review is required |

> KIRTA does not claim that static analysis guarantees exploitation or the absence of risk. The verdict is a prioritization aid, not proof of complete security.

---

## Why AI is used

Without an LLM, KIRTA can already display technical facts: package, version, CVE, severity, fixed versions, imports, and a call map. An engineer still has to interpret that context manually.

AI acts as an **interpreter of structured security facts**, not a source of truth:

- receives finding data and a limited call map;
- analyzes indications of practical reachability;
- returns a strictly structured response;
- produces a short explanation;
- helps determine whether a finding should be fixed now, deferred, or sent for manual review.

The backend client is configured for predictable, structured responses:

- `response_format: json_schema`;
- `strict: true`;
- `temperature: 0`;
- model response validation;
- limits on the amount of call map data sent to the LLM.

### AI integration data boundary

AI analysis runs only at the user's request for a selected finding. The data sent to OpenRouter includes the CVE identifier and description, severity, and a limited portion of the call map: file paths, line numbers, call information, and short fragments of the corresponding expressions.

The complete ZIP archive, full source file contents, PostgreSQL and MinIO credentials, and the application's API key are not included in the model request. The call map size is limited by `openrouter_callmap_max_files` and `openrouter_callmap_max_calls`; the payload also indicates whether the context was truncated.

> Call map fragments may contain string literals from the analyzed calls. Before using an external AI provider, review its data handling policies and scan only projects you are authorized to analyze.

---

## Architecture

<img width="2695" height="1428" alt="KIRTA architecture" src="https://github.com/user-attachments/assets/2ffa58dd-bd7c-4df0-a04a-300c3ec19267" />

### Backend

The backend is built with **Go + Gin** and orchestrates the scan pipeline.

Key capabilities:

- uploading a Python project ZIP archive;
- Zip Slip protection during extraction;
- checking for Python code;
- calculating SLOC and language composition;
- running Syft and Grype;
- normalizing SCA results;
- building call maps;
- storing scan metadata, findings, and graphs in PostgreSQL;
- saving source files in MinIO/S3-compatible storage;
- serving source code through the API;
- on-demand AI enrichment;
- OpenAPI/Swagger documentation.

| Layer | Technologies |
| --- | --- |
| HTTP API | Go, Gin |
| Database | PostgreSQL, JSONB, pgx |
| Migrations | golang-migrate |
| Object storage | MinIO / S3-compatible storage |
| SCA | Syft, Grype |
| Static analysis | Python `ast`, import resolution, call tracing |
| AI integration | OpenRouter-compatible Chat Completions API |
| API docs | OpenAPI, Swagger UI, swaggo |

### Frontend

The frontend is an SPA built with **React + TypeScript + Vite**.

The UI includes:

- landing page;
- demo/mock login for the MVP;
- scan history;
- drag-and-drop ZIP uploads;
- scan report;
- severity and exploitability badges;
- finding search and filtering;
- call map panel;
- a source code modal with evidence highlighting;
- theme switching;
- explicit routing for `/`, `/login`, `/scans`, `/:scanId`, and utility paths.

| Layer | Technologies |
| --- | --- |
| Core | React 18, TypeScript, Vite |
| Routing | React Router v6 |
| Async data | TanStack Query |
| State | Zustand |
| UI | Tailwind CSS, Radix UI, lucide-react |
| Code viewer | react-syntax-highlighter |
| Quality | ESLint, Prettier |

---

## Repository structure

```text
.
├── kirta-backend-api/                  # Go/Gin backend
│   ├── cmd/main.go
│   ├── internal/api/                   # handlers, routes, middleware
│   ├── internal/config/                # YAML configuration
│   ├── internal/domain/                # domain DTO
│   ├── internal/persistence/db/        # PostgreSQL repository
│   ├── internal/service/               # scan pipeline, SCA, AI enrichment
│   ├── internal/storage/               # MinIO/S3 storage
│   ├── migrations/
│   ├── openapi.yaml
│   ├── sca.py
│   ├── graph.py
│   └── tracer.py
│
├── kirta-ui/                           # React + TypeScript frontend
│   ├── src/app/
│   ├── src/pages/
│   ├── src/features/
│   ├── src/repositories/
│   ├── src/components/
│   └── deploy/nginx/
│
├── tools/                              # CLI analysis helpers
├── kirta.py                            # Syft → Grype → Tiny-SCA → package trace
├── sca-tinifier.py
├── tiny-sca-schema.json
├── docs/assets/
├── LICENSE
└── README.md
```

---

## API overview

| Method | Endpoint | Purpose |
| --- | --- | --- |
| `POST` | `/v1/scan` | Upload a ZIP archive and start analysis |
| `GET` | `/v1/scans` | List scans |
| `GET` | `/v1/scans/{id}` | Retrieve the full scan report |
| `GET` | `/v1/scans/{id}/graphs?package=<name>` | Retrieve a library call map |
| `GET` | `/v1/scans/{id}/files/{filepath}` | Retrieve a source file from storage |
| `POST` | `/v1/scans/{id}/findings/{finding_id}/explanation` | Run AI analysis for a finding |

---

## Screenshots

### Landing

![KIRTA landing page](docs/assets/landing.png)

### Project upload

![KIRTA upload dialog](docs/assets/upload.png)

### Scan history

![KIRTA scan history](docs/assets/scans.png)

### Scan report

![KIRTA scan report](docs/assets/report.png)

### AI explanation

![KIRTA AI explanation](docs/assets/finding-ai.png)

### Security explanation

![KIRTA security explanation](docs/assets/finding-sec-ai.png)

### Call map

![KIRTA call map](docs/assets/call-map.png)

---

## Quick start: frontend

```bash
cd kirta-ui
npm install
npm run dev
```

Production build:

```bash
npm run build
npm run preview
```

Quality checks:

```bash
npm run lint
npm run format
```

## Quick start: backend

The backend requires PostgreSQL, MinIO/S3-compatible storage, Syft, and Grype.

```bash
cd kirta-backend-api
cp config.example.yaml config.yaml
```

In `config.yaml`, specify PostgreSQL and MinIO/S3 settings, paths to Syft/Grype, and an OpenRouter API key.

Start:

```bash
go run ./cmd
```

By default, the backend is available at:

```text
http://localhost:8080
```

Swagger UI:

```text
http://localhost:8080/swagger/index.html
```

---

## AI configuration

Example from `kirta-backend-api/config.example.yaml`:

```yaml
app:
  openrouter_api_key: "YOUR_OPENROUTER_API_KEY"
  openrouter_model: "openai/gpt-oss-120b:free"
  openrouter_base_url: "https://openrouter.ai/api/v1"
  openrouter_timeout: 20s
  openrouter_callmap_max_files: 20
  openrouter_callmap_max_calls: 200
```

AI analysis runs on request:

```http
POST /v1/scans/{id}/findings/{finding_id}/explanation
```

---

## Estimating potential cost savings

KIRTA was built around the hypothesis that manual initial triage accounts for a substantial share of vulnerability management costs.

An illustrative calculation:

- manual triage of one finding: approximately 15 minutes;
- assumed cost per minute of a specialist's time: RUB 28;
- estimated manual triage cost: RUB 420 per finding;
- estimated cost of one LLM call in the original calculation model: approximately RUB 7.

For 1,000 findings, this gives a rough estimate of **RUB 420,000** for manual triage versus **RUB 7,000** for LLM calls, a potential difference of approximately **RUB 413,000**.

> This is **an illustrative scenario, not a measured KIRTA production result**. Actual savings depend on the model, token costs, automated triage quality, the proportion of findings requiring manual review, and the cost of the team's time.

Source materials used to estimate triage time:

- Corgea — materials on reducing false positives in SAST;
- Astra Security — materials on false-positive triage in DAST.

---

## Roadmap

Near-term MVP development:

- support for additional languages;
- an extended AI verdict: confidence, priority, reason codes, recommendation;
- bulk AI prioritization of findings;
- improved reachability chain visualization.

Further ahead:

- SAST findings;
- DAST findings;
- secrets / IaC / container security signals;
- integrations with GitHub/GitLab and other VCS platforms;
- team workspaces and role-based access;
- a CI/CD mode for pull request analysis.

---

## License

The project is distributed under the **MIT** license: [`LICENSE`](./LICENSE).

---

## Authors

- [Nicole Zhurbenko (@nikamurkaa)](https://github.com/nikamurkaa)
- [PArk (@76parker)](https://github.com/76parker)
