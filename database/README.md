# Database

В этом домашнем задании вам предстоит реализовать интеграцию с базой данных в рамках сервиса **library**.
Для простоты понимания описание этого ДЗ сделано в императивном, а не декларативном стиле

## Часть 1
Ниже описана одна из возможных реализаций схемы базы данных. Вы можете сделать свою, объяснив выбор в комментариях PR

Сперва вам необходимо написать миграции к вашей базе данных.

### Migrations

Создайте директорию [db/migrations](db/migrations) с вашими миграциями, а
также [db/migrations/migrate.go](db/migrations/migrate.go])
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

В файле [db/migrations/migrate.go](./db/migrations/migrate.go) напишите код, который будет накатывать миграции.
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

* Открыть pull request из ветки `hw` в ветку `main` **вашего репозитория**.

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
