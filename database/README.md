# HW 1 (database)

В этом домашнем задании вам предстоит реализовать интеграцию с базой данных в рамках сервиса **library**.
Для простоты понимания описание этого ДЗ сделано в императивном, а не декларативном стиле

## Часть 1
Ниже описана одна из возможных реализаций схемы базы данных. Вы можете сделать свою, объяснив выбор в комментариях PR

Сперва вам необходимо написать миграции к вашей базе данных.

### Migrations

Создайте директорию [db/migrations](db/migrations) с вашими миграциями, а
также [db/migrate.go](db/migrate.go)
для их применения

### Author

Создайте таблицу `author`

```sql
-- +goose Up
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE author
(
  id ...,
  name ...,
  created_at ...,
  updated_at ...
);

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION update_author_timestamp() RETURNS TRIGGER AS
$$
BEGIN
  NEW.updated_at = now();
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd


CREATE OR REPLACE TRIGGER trigger_update_author_timestamp
  BEFORE UPDATE
  ON ...
  FOR EACH ROW
EXECUTE FUNCTION update_author_timestamp();


-- +goose Down
DROP TABLE ...;
```

Отдельной миграцией создайте индекс на имя автора

```sql
-- +goose Up
CREATE INDEX ...;

-- +goose Down
DROP INDEX ...;
```

### Book

Создайте таблицу `book`

```sql
-- +goose Up
CREATE TABLE book
(
    id ...,
    name ...,
    created_at ...,
    updated_at ...
);

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION update_book_timestamp() RETURNS TRIGGER AS
$$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

CREATE OR REPLACE TRIGGER trigger_update_book_timestamp
    BEFORE UPDATE
    ON ...
    FOR EACH ROW
EXECUTE FUNCTION update_book_timestamp();

-- +goose Down
DROP TABLE ...;
```

Отдельной миграцией создайте индекс на имя книги

```sql
-- +goose Up
CREATE INDEX  ...;

-- +goose Down
DROP INDEX ...;
```

### Book to authors

Создайте таблицу `author_book`

```sql
-- +goose Up
CREATE TABLE author_book
(
    author_id ...,
    book_id ...,
    PRIMARY KEY (.. .)
);

-- +goose Down
DROP TABLE author_book;
```

- Добавьте `foreign key` для author_id и book_id.
- Поддержите каскадное удаление `ON DELETE CASCADE`, в случае удаления автора или книги в этой таблице не должны
  остаться неконсистентные записи
- Добавьте композитный `PRIMARY KEY`, состоящий из `author_id` и `book_id`

Композитный `PRIMARY KEY` по умолчанию добавляет индекс на свои части, однако
его [эффективность для каждого атрибута разная](https://www.postgresql.org/docs/current/indexes-multicolumn.html).
Отдельной миграцией добавьте индекс для `book_id`

## Часть 2

В файле [db/migrate.go](./db/migrate.go) напишите код, который будет накатывать миграции.
Используйте библиотеки ниже, а также `//go:embed migrations/*.sql` для загрузки
миграций - [пример](https://github.com/pressly/goose)

```go
"github.com/jackc/pgx/v5/pgxpool"
"github.com/jackc/pgx/v5/stdlib"
"github.com/pressly/goose/v3"
"github.com/project/library/config"
```

Попробуйте поднять базу данных и проверить, что ваши миграции корректно накатываются

```
docker volumes
docker volume ls // если нужно удалить старый volume
docker volume rm ... // если нужно удалить старый volume
docker-compose up -d

docker ps -a // посмотреть контейнеры
docker stop / docker rm - для остановки и удаления контейнера
```

```
2025/03/06 15:03:14 OK   001_create_author_table.sql (5.89ms)
2025/03/06 15:03:14 OK   002_create_author_name_index.sql (8.83ms)
2025/03/06 15:03:14 OK   003_create_book_table.sql (9.78ms)
2025/03/06 15:03:14 OK   004_create_book_name_index.sql (2.51ms)
2025/03/06 15:03:14 OK   005_create_author_book_table.sql (3.28ms)
2025/03/06 15:03:14 OK   006_create_author_book_book_id_index.sql (2.99ms)
2025/03/06 15:03:14 goose: successfully migrated database to version: 6
```

## Часть 3

Поддержите в вашем конфиге параметры для подключения к базе данных

```go
type (
    Config struct {
        GRPC
        PG
    }
    
    GRPC struct {
        Port        string `env:"GRPC_PORT"`
        GatewayPort string `env:"GRPC_GATEWAY_PORT"`
    }
    
    PG struct {
        URL      string
        Host     string `env:"POSTGRES_HOST"`
        Port     string `env:"POSTGRES_PORT"`
        DB       string `env:"POSTGRES_DB"`
        User     string `env:"POSTGRES_USER"`
        Password string `env:"POSTGRES_PASSWORD"`
        MaxConn  string `env:"POSTGRES_MAX_CONN"`
    }
)
```

Пример URL:

```
postgres://user:password@host:port/dbname?sslmode=disable&pool_max_conns=10
```

## Часть 4

Добавьте новую реализацию репозитория вашего сервиса, используя поднятую базу данных.
Не забывайте про консистентность и атомарность операций. Пример:

```go
func (r *PostgresRepository) CreateBook(ctx context.Context, book entity.Book) (entity.Book, error) {
tx, err := r.db.Begin(ctx)
if err != nil {
return entity.Book{}, err
}
defer tx.Rollback(ctx)

	const queryBook = `INSERT INTO book (name) VALUES ($1) RETURNING id, created_at, updated_at`
	err = tx.QueryRow(ctx, queryBook, book.Name).Scan(&book.ID, &book.CreatedAt, &book.UpdatedAt)
	if err != nil {
		return entity.Book{}, err
	}

	const queryAuthorBooks = `INSERT INTO author_book (author_id, book_id) VALUES ($1, $2)`
	for _, authorID := range book.AuthorIDs {
		_, err := tx.Exec(ctx, queryAuthorBooks, authorID, book.ID)
		if err != nil {
			return entity.Book{}, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return entity.Book{}, err
	}

	return book, nil
}
```

* Старайтесь обойтись одним запросом там, где это возможно
* ID автора и книги должны генерироваться __на уровне базы__ через `DEFAULT uuid_generate_v4()`

## Часть 5

Добавьте в API для Book поля `created_at` и `updated_at`

```protobuf
import "google/protobuf/timestamp.proto";

message Book {
  ...
  google.protobuf.Timestamp created_at = ...;
  google.protobuf.Timestamp updated_at = ...;
}
```

# HW 2 (outbox)
## Часть 6
С этой части начинается ДЗ `outbox`. Ветка с решением должна иметь название `outbox`. Важно, чтобы в PR не было diff'a старого ДЗ.
Вы можете добиться этого, сделав rebase на `main` после проверки предыдущего ДЗ

Реализуйте паттерн `outbox`, который обсуждался на лекции

Создайте таблицу `outbox`
```sql
CREATE TYPE outbox_status as ENUM ('CREATED', 'IN_PROGRESS', 'SUCCESS');

CREATE TABLE outbox
(
    idempotency_key TEXT PRIMARY KEY,
    data            JSONB                   NOT NULL,
    status          outbox_status           NOT NULL,
    kind            INT                     NOT NULL,
    created_at      TIMESTAMP DEFAULT now() NOT NULL,
    updated_at      TIMESTAMP DEFAULT now() NOT NULL
);
```

Поддержите транзакции на уровне доменной логики

```go
type Transactor interface {
	WithTx(ctx context.Context, function func(ctx context.Context) error) error
}

func extractTx(ctx context.Context) (pgx.Tx, error) {}

func injectTx(ctx context.Context, pool *pgxpool.Pool) (context.Context, error, pgx.Tx) {}
```

Например:
```go
func (l *libraryImpl) RegisterBook(ctx context.Context, name string, authorIDs []string) (*library.AddBookResponse, error) {
    var book entity.Book
	err := l.transactor.WithTx(ctx, func(ctx context.Context) error {
		book, txErr = l.booksRepository.CreateBook(ctx, entity.Book{
			Name:      name,
			AuthorIDs: authorIDs,
		})
		
		...
		l.outboxRepository.SendMessage(ctx, idempotencyKey, repository.OutboxKindBook, serialized)
	})
	
	...
}
```


Поддержите конфиг для Outbox

```go
type Outbox struct {
    Enabled         bool          `env:"OUTBOX_ENABLED"`
    Workers         int           `env:"OUTBOX_WORKERS"`
    BatchSize       int           `env:"OUTBOX_BATCH_SIZE"`
    WaitTimeMS      time.Duration `env:"OUTBOX_WAIT_TIME_MS"`
    InProgressTTLMS time.Duration `env:"OUTBOX_IN_PROGRESS_TTL_MS"`
    AuthorSendURL   string        `env:"OUTBOX_AUTHOR_SEND_URL"`
    BookSendURL     string        `env:"OUTBOX_BOOK_SEND_URL"`
}
```

При создании книги или автора вам необходимо асинхронно отправить `POST` запрос c `AuthorID` или `BookID` на `OUTBOX_AUTHOR_SEND_URL`
или `OUTBOX_BOOK_SEND_URL`, соответственно.

# HW 3 (Observability)
## Часть 7
В этой части начинается ДЗ `observability`. Ветка с решением должна иметь название `observability`.
Важно, чтобы в PR не было diff'a из старых заданий. Вы можете добиться этого, сделав rebase на `main` после проверки предыдущих ДЗ.

Для начала научитесь запускать ваше приложение в `Docker`. Для этого напишите `Dockerfile` и конфигурацию сервиса
в `docker-compose.yml`.

*Dockerfile*
```dockerfile
FROM golang:latest

WORKDIR /application
COPY . .
RUN make generate && (GOOS=linux GOARCH=amd64 make build)
CMD ["./bin/library"]
```

*docker-compose.yml*
```yaml
library:
  build:
    context: .
    dockerfile: Dockerfile
  depends_on:
    postgres:
      condition: service_healthy
    pyroscope:
      condition: service_started
  environment:
      ...
  volumes:
      ...
  ports:
      ...
  networks:
    - internal
```

## Часть 8
В этой части вам предлагается поддержать экспорт и отображение логов вашего приложения. Объяснение схемы можно посмотреть в
[лекции](https://www.youtube.com/watch?v=AoYEJCOMaQU).

* Для хранения и обработки логов вам предлагается использовать [Loki](https://grafana.com/docs/loki/latest/)
* Для экспорта логов вашего приложения в `Loki` вам предлагается использовать [Promtail](https://lyz-code.github.io/blue-book/promtail/)
* Для отображения логов вам предлагается использовать [Grafana](https://grafana.com/docs/grafana/latest/dashboards/)

**Важно, что вы можете использовать любой другой observability стек, главное выполнить требования задания.**

Для корректной работы `Grafana` и `Promtail` необходимо написать соответствующие конфиги

*Пример конфига для Promtail*
```yaml
server:
  http_listen_port: 9080
  grpc_listen_port: 0

positions:
  filename: /tmp/positions.yaml

clients:
  - url: http://loki:3100/loki/api/v1/push

scrape_configs:
  - job_name: library-logs
    static_configs:
      - targets:
          - localhost
        labels:
          job: library
          __path__: /var/log/library.log

    pipeline_stages:
      - json:
          expressions:
            level: level
            trace_id: trace_id
            book_id: book_id
      - labels:
          level:
          trace_id:
          book_id:
```

*Пример конфига для Grafana*
```yaml
apiVersion: 1

datasources:
  - name: Prometheus
    type: prometheus
    access: proxy
    url: http://prometheus:9090
    isDefault: true
```

**В данном задании разрешается хардкодить адреса в конфигах**

* Должна быть возможность **быстро** искать логи по меткам (labels)
* Должно быть поддержано не менее 5 меток
* При проверке задания будет оцениваться поисковая полезность и кардинальность тегов, рекомендуется давать пояснения в итоговом отчёте


## Часть 9
В этой части вам предлагается настроить трейсинг с использованием [jaeger](https://www.jaegertracing.io/).

В отличие от низкоуровнего подхода на лекции, необходимо использовать
готовый grpc-интерцептор

*Внедрение интерцептора в grpc server*
```go
s := grpc.NewServer(
    grpc.UnaryInterceptor(
        otelgrpc.UnaryServerInterceptor(
            otelgrpc.WithTracerProvider(otel.GetTracerProvider()),
        ),
    ),
    grpc.StreamInterceptor(
        otelgrpc.StreamServerInterceptor(
            otelgrpc.WithTracerProvider(otel.GetTracerProvider()),
        ),
    ),
)
```

Данный интерцептор сам создаст span и пробросит `trace` из входящего запроса, либо создаст свой при необходимости.

*Получение span'a из контекста*

```go
span := trace.SpanFromContext(ctx)
defer span.End()
```

Особенности реализации
* В такой конфигурации при тестировании grpc gateway может казаться, что трейсы отображаются. Однако на практике часто пишут дополнительные
  HTTP-обработчики (`middleware`), чтобы корректно прокидывать трейсы от других сервисов. В данном задании это делать **не нужно**

* На данный момент есть несколько подходов для декларации интерцепторов, пример выше работает на версии `v0.52.0`
```go
go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.52.0
```

* В трейсах должны быть теги для поиска соответствующих логов
* При проверке задания будет оцениваться поисковая полезность и кардинальность тегов, рекомендуется давать пояснения в итоговом отчёте

## Часть 10
Реализуйте сбор метрик в вашем приложении

*Пример сервера с хендлером для метрик. Обычно его специально разделяют с основным, чтобы как минимум разделять входящий трафик + есть возможность поднять его на другом интерфейсе*
```go
func runMetricsServer(port string) {
	http.Handle("/metrics", promhttp.Handler())
	http.ListenAndServe(":"+port, nil)
}
```

Открыв `http://localhost:${METRICS_PORT}/metrics` вы сможете увидеть сырые метрики, которые отдаёт ваше приложение

*Пример метрик по умолчанию, которые отдаёт promhttp.Handler()*
```
# TYPE go_memstats_alloc_bytes_total counter
go_memstats_alloc_bytes_total 8.57147216e+08
# HELP go_memstats_buck_hash_sys_bytes Number of bytes used by the profiling bucket hash table. Equals to /memory/classes/profiling/buckets:bytes.
# TYPE go_memstats_buck_hash_sys_bytes gauge
go_memstats_buck_hash_sys_bytes 1.509895e+06
```

Напишите конфиг для `Prometheus`
```yaml
global:
  scrape_interval: 10s

scrape_configs:
  - job_name: 'library'
    static_configs:
      - targets: ['library:9000']
```

Подключите `Prometheus` к `Grafana`

*Пример конфига для Grafana*
```yaml
apiVersion: 1

datasources:
  - name: Prometheus
    type: prometheus
    access: proxy
    url: http://prometheus:9090
    isDefault: true

  - name: Loki
    type: loki
    access: proxy
    url: http://loki:3100
    jsonData:
      maxLines: 1000
```

В самой `Grafana` постройте дешборд

### Outbox
* Во всех графиках должна быть возможность посмотреть состояние по каждому kind'y
* График количества задач
* График, показывающий скорость обработки задач
* График, показывающий rate неуспешных задач

### Library handlers
* График, показывающий количество горутин `go_goroutines`
* График, показывающий live heap `go_memstats_heap_inuse_bytes`
* График, показывающий RPS. Должна быть возможность посмотреть график по конкретному эндпоинту
* График, показывающий latency. Должна быть возможность посмотреть график по конкретному эндпоинту

### Postgres
Для сбора некоторых метрик ниже можно запускать отдельные горутины при старте приложения

* График, показывающий количество записей по каждой таблице в базе
* График, показывающий rate вставки записей по каждой таблице
* (*) Можно сделать графики для latency основных операций, что требует некоторой обёртки над драйвером к базе. В данном задании такие графики **можно не строить**


Особенности
* Задание может быть оценено на частичный балл, в том числе, если вы не построили все графики
* Дешборд должен быть красивым

## Часть 11
Для сдачи данного домашнего задания вам необходимо записать видеоотчёт **до 6 минут**.
В данном отчёте вы должны будете продемонстрировать выполненное задание и дать комментарии.

**Не нужно пушить видео на GitHub, загрузите его на сторонний ресурс и приложите ссылку в комментарии к PR**

Для демонстрации работы дешбордов необходимо будет сгенерировать нагрузку, написав кастомные сценарии.
Рекомендуется использовать [k6](https://k6.io/)

*Пример*
```javascript
import http from 'k6/http';
import { check, sleep } from 'k6';
import { randomItem } from 'https://jslib.k6.io/k6-utils/1.4.0/index.js';

export const options = {
    vus: 10,
    duration: '30s',
};

const authorIds = [
    'e6672056-49ee-4aba-a9f0-21813b2963a3',
    'de41b5cb-859a-4e82-8a15-5e83a609d510',
    'fbb90d1a-5616-4e0f-9242-4b1015c94cbf',
    'a37751e6-b8db-465e-bb8b-5182e5799fc3',
    '0f84bfb0-cda5-41f6-851d-25cf7b35e5d4',
];

export default function () {
    const url = 'http://localhost:8080/v1/library/book';

    const randomAuthorId = randomItem(authorIds);

    const payload = JSON.stringify({
        name: `book-${__VU}-${__ITER}`,
        author_ids: [randomAuthorId],
    });

    const params = {
        headers: {
            'Content-Type': 'application/json',
        },
    };

    const res = http.post(url, payload, params);

    check(res, {
        'status is 200 or 201': (r) => r.status === 200 || r.status === 201,
    });

    sleep(1);
}

```

## Унификация технологий

Для удобства выполнения и проверки дз вводится ряд правил, унифицирующих используемые технологии

* Структура проекта [go-clean-template](https://github.com/evrone/go-clean-template) и
  этот [шаблон](https://github.com/itmo-org/lectures/tree/main/sem2/lecture1)
* Для генерации кода авторские [Makefile](./Makefile) и [easyp.yaml](./easyp.yaml)
* Для логирования [zap](https://github.com/uber-go/zap)
* Для валидации [protoc-gen-validate](https://github.com/bufbuild/protoc-gen-validate)
* Для поддержики REST-to-gRPC API [gRPC gateway](https://grpc-ecosystem.github.io/grpc-gateway/)
* Для миграций [goose](https://github.com/pressly/goose)
* [pgx](https://github.com/jackc/pgx) как драйвер для postgres

## Тестирование в CI

* Код тестов можно посмотреть в файле [integration_test.go](./integration-test/integration_test.go)

* Важно, чтобы ваш сервис умел корректно обрабатывать SIGINT и SIGTERM, иначе тесты могут работать некорректно
* В [Makefile](Makefile) реализованы метки **build** и **generate**, без них CI не будет работать

## Переменные окружения

В рамках вашего сервиса вы должны реализовать конфиг, который будет работать с переменными окружения

## Тесты

Необходимо сгенерировать моки и написать свои тесты, степень покрытия будет проверяться в CI

## Документация

Вам необходимо своими словами написать [README.md](./docs/README.md) в ./docs к своему сервису library

## Рекомендации

* [Пример реализации](https://github.com/itmo-org/lectures/tree/main/sem2)
* Не забывайте про логирование
* Не забывайте про консистентность в базе данных
* Используйте [тесты](./integration-test) чтобы осознать недосказанности
* Не нужно добавлять старую in-memory реализацию репозитория

## Письменные комментарии

Поскольку количество попыток сдачи ограничено, вы можете написать дополнительные комментарии в PR. Если ваше

обоснование будет достаточно разумным, это может быть учтено при выставлении баллов. Например,

* описать, почему вы написали именно такие интерфейсы

* описать, почему вы сделали именно такую валидацию

* описать, почему вы сделали именно такую схему в базе данных

## Сдача

* Открыть pull request из ветки задания в ветку `main` **вашего репозитория**.

* В описании PR заполнить количество часов, которые вы потратили на это задание.

* Отправить заявку на ревью в соответствующей форме.

* Время дедлайна фиксируется отправкой формы.

* Изменять файлы в ветке main без PR запрещено.

* Изменять файл [CI workflow](./.github/workflows/library.yaml) запрещено.

## Makefile

Для удобств локальной разработки сделан [`Makefile`](Makefile). Имеются следующие команды:

Запустить полный цикл (линтер, тесты):

```bash 

make all

```

Запустить только тесты:

```bash

make test

``` 

Запустить линтер:

```bash

make lint

```

Подтянуть новые тесты:

```bash

make update

```

При разработке на Windows рекомендуется использовать [WSL](https://learn.microsoft.com/en-us/windows/wsl/install), чтобы

была возможность пользоваться вспомогательными скриптами.
