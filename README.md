# KIRTA — AI Security Platform

<p align="left">
  <img src="docs/assets/badges/status.svg" height="36" alt="Status: MVP" />
  <a href="./LICENSE"><img src="docs/assets/badges/license.svg" height="36" alt="License: MIT" /></a>
</p>

<p align="left">
  <img src="https://skillicons.dev/icons?i=python,go,react,ts,postgres,docker,nginx,vite,tailwind&theme=dark" alt="KIRTA core technologies" />
</p>

**KIRTA** — MVP-платформа для объяснимого анализа уязвимостей Python-проектов. Она объединяет результаты классического SCA-сканирования, статические факты из исходного кода и AI-интерпретацию, чтобы помочь команде быстрее понять, какие findings требуют внимания в первую очередь.

**Core stack:** Go, Gin, Python, React, TypeScript, PostgreSQL, MinIO/S3, Docker, Nginx  
**Security tooling:** Syft, Grype, Python AST/call tracing  
**AI:** OpenRouter-compatible client for structured finding analysis

---

## Что решает KIRTA

Классический SCA хорошо отвечает на вопрос «какие известные CVE есть в зависимостях?», но сам по себе не показывает, насколько конкретная библиотека связана с кодом проекта и что найденный дефект означает для текущего контекста.

KIRTA добавляет к результатам сканера контекст исходного кода:

- строит SBOM проекта;
- получает SCA findings;
- ищет использование уязвимых библиотек в Python-коде;
- строит call map;
- показывает исходные файлы и строки evidence;
- по запросу передаёт структурированный security-контекст LLM;
- возвращает объяснение и статус эксплуатируемости для отдельного finding.

Главная цель — **не заменить security-инженера и не доказать эксплуатацию автоматически**, а сократить объём ручного первичного разбора и сделать технический отчёт более объяснимым.

## MVP scope

Текущая реализация сознательно ограничена:

1. анализируется только Python-код;
2. call map строится только для Python;
3. SBOM генерируется через `Syft`;
4. SCA выполняется через `Grype`;
5. поддерживается SLOC и определение языкового состава проекта;
6. реализован SCA-сценарий — SAST и DAST findings пока не входят в MVP;
7. сканирование выполняется синхронно;
8. AI-анализ выполняется on-demand для отдельного finding;
9. модель настраивается через OpenRouter; пример конфигурации использует `openai/gpt-oss-120b:free`.

Эти ограничения явно зафиксированы, чтобы результаты MVP не воспринимались как полноценная замена промышленной AppSec-платформе.

---

## Основной сценарий

```text
ZIP проекта
   ↓
безопасная распаковка
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

### Что получает пользователь

- список найденных SCA findings;
- package, version, CVE, severity и fixed versions;
- сведения об использовании пакета в исходном коде;
- call map для библиотеки;
- просмотр source evidence в интерфейсе;
- AI-объяснение для отдельного finding;
- статус эксплуатируемости с оговоркой о пределах статического анализа.

---

## Статусы эксплуатируемости

| Статус | Значение |
| --- | --- |
| `exploitable` | В текущих статических фактах и AI-анализе есть признаки практической достижимости дефекта |
| `not_exploitable` | Текущий набор статических фактов не подтверждает использование/достижимость уязвимого сценария |
| `unknown` | Данных недостаточно для уверенного вывода; требуется ручная проверка |

> KIRTA не утверждает, что статический анализ гарантирует эксплуатацию или отсутствие риска. Вердикт — инструмент приоритизации, а не доказательство полной безопасности.

---

## Почему здесь нужен AI

Без LLM KIRTA уже может показать технические факты: package, version, CVE, severity, fixed versions, imports и call map. Но инженеру всё равно приходится интерпретировать этот контекст вручную.

AI используется как **интерпретатор структурированных security-фактов**, а не как источник истины:

- получает данные finding и ограниченный call map;
- анализирует признаки практической достижимости;
- возвращает строго структурированный ответ;
- формирует короткое объяснение;
- помогает понять, стоит ли finding чинить сейчас, отложить или отправить на ручную проверку.

Backend-клиент настроен на предсказуемый структурированный ответ:

- `response_format: json_schema`;
- `strict: true`;
- `temperature: 0`;
- валидация ответа модели;
- ограничения на объём call map, отправляемого в LLM.

---

## Архитектура

<img width="2695" height="1428" alt="KIRTA architecture" src="https://github.com/user-attachments/assets/2ffa58dd-bd7c-4df0-a04a-300c3ec19267" />

### Backend

Backend реализован на **Go + Gin** и оркестрирует scan pipeline.

Ключевые возможности:

- загрузка ZIP-архива Python-проекта;
- защита от Zip Slip при распаковке;
- проверка наличия Python-кода;
- подсчёт SLOC и языкового состава;
- запуск Syft и Grype;
- нормализация SCA-результата;
- построение call map;
- хранение scan metadata, findings и graphs в PostgreSQL;
- сохранение source files в MinIO/S3-compatible storage;
- выдача исходного кода по API;
- on-demand AI enrichment;
- OpenAPI/Swagger документация.

| Слой | Технологии |
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

Frontend — SPA на **React + TypeScript + Vite**.

В интерфейсе реализованы:

- landing page;
- demo/mock login для MVP;
- история сканирований;
- drag-and-drop загрузка ZIP;
- scan report;
- severity и exploitability badges;
- поиск и фильтрация findings;
- call map panel;
- source code modal с подсветкой evidence;
- theme switching;
- явная маршрутизация `/`, `/login`, `/scans`, `/:scanId` и служебных путей.

| Слой | Технологии |
| --- | --- |
| Core | React 18, TypeScript, Vite |
| Routing | React Router v6 |
| Async data | TanStack Query |
| State | Zustand |
| UI | Tailwind CSS, Radix UI, lucide-react |
| Code viewer | react-syntax-highlighter |
| Quality | ESLint, Prettier |

---

## Структура репозитория

```text
.
├── kirta-backend-api/                  # Go/Gin backend
│   ├── cmd/main.go
│   ├── internal/api/                   # handlers, routes, middleware
│   ├── internal/config/                # YAML configuration
│   ├── internal/domain/                # domain DTO
│   ├── internal/persistance/db/        # PostgreSQL repository
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

| Method | Endpoint | Назначение |
| --- | --- | --- |
| `POST` | `/v1/scan` | Загрузить ZIP и запустить анализ |
| `GET` | `/v1/scans` | Получить список сканирований |
| `GET` | `/v1/scans/{id}` | Получить полный scan report |
| `GET` | `/v1/scans/{id}/graphs?package=<name>` | Получить call map библиотеки |
| `GET` | `/v1/scans/{id}/files/{filepath}` | Получить исходный файл из storage |
| `POST` | `/v1/scans/{id}/findings/{finding_id}/explanation` | Запустить AI-анализ finding |

---

## Скриншоты

### Landing

![KIRTA landing page](docs/assets/landing.png)

### Загрузка проекта

![KIRTA upload dialog](docs/assets/upload.png)

### История сканирований

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

## Быстрый старт: frontend

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

Проверка качества:

```bash
npm run lint
npm run format
```

## Быстрый старт: backend

Backend требует PostgreSQL, MinIO/S3-compatible storage, Syft и Grype.

```bash
cd kirta-backend-api
cp config.example.yaml config.yaml
```

В `config.yaml` укажите параметры PostgreSQL, MinIO/S3, пути к Syft/Grype и OpenRouter API key.

Запуск:

```bash
go run ./cmd
```

По умолчанию backend доступен на:

```text
http://localhost:8080
```

Swagger UI:

```text
http://localhost:8080/swagger/index.html
```

---

## Конфигурация AI

Пример из `kirta-backend-api/config.example.yaml`:

```yaml
app:
  openrouter_api_key: "YOUR_OPENROUTER_API_KEY"
  openrouter_model: "openai/gpt-oss-120b:free"
  openrouter_base_url: "https://openrouter.ai/api/v1"
  openrouter_timeout: 20s
  openrouter_callmap_max_files: 20
  openrouter_callmap_max_calls: 200
```

AI-анализ выполняется по запросу:

```http
POST /v1/scans/{id}/findings/{finding_id}/explanation
```

---

## Оценка потенциального экономического эффекта

KIRTA создавалась вокруг гипотезы, что значительная часть стоимости vulnerability management приходится на ручной первичный triage.

Пример сценарного расчёта:

- ручной triage одного finding: около 15 минут;
- условная стоимость минуты работы специалиста: 28 ₽;
- примерная стоимость ручного разбора: 420 ₽ на finding;
- примерная стоимость одного LLM-вызова в исходной модели расчёта: около 7 ₽.

При 1000 findings это даёт ориентир порядка **420 000 ₽** ручного triage против **7 000 ₽** LLM-вызовов, то есть потенциальную разницу около **413 000 ₽**.

> Это **иллюстративный сценарий, а не измеренный production-результат KIRTA**. Фактический эффект зависит от модели, стоимости токенов, качества автоматического triage, доли findings, требующих ручной проверки, и стоимости рабочего времени команды.

Исходные материалы, использованные для оценки времени triage:

- Corgea — материалы о снижении false positives в SAST;
- Astra Security — материалы о false-positive triage в DAST.

---

## Roadmap

Ближайшее развитие MVP:

- поддержка дополнительных языков;
- расширенный AI-вердикт: confidence, priority, reason codes, recommendation;
- массовая AI-приоритизация findings;
- улучшенная визуализация цепочки достижимости.

Дальше:

- SAST findings;
- DAST findings;
- secrets / IaC / container security signals;
- интеграции с GitHub/GitLab и другими VCS;
- team workspace и role-based access;
- CI/CD mode для анализа pull requests.

---

## License

Проект распространяется под лицензией **MIT**: [`LICENSE`](./LICENSE).
