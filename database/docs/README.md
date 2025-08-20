# Library-service

This is a service with gRPC and a REST API for requests.

Service stores entities book and author. Each book or author has several fields
describing them.

### Author
    1) id 
    2) name

### Book:
    1) id
    2) name
    3) (optional) author_id (id of it authors)
    4) created_at
    5) updated_at


## The service accepts the following requests

------------------------------

### • Register author

In body of request define name of new author and service will return its id.

#### NOTE: Constraints for name:

1) name must satisfy the regular expression ^[A-Za-z0-9]+( [A-Za-z0-9]+)*$
2) name's length must be in [1; 512] symbols.


------------------------------

### • Get author info

Define id of required author and service will return info about him (his name),
if author with given id exists, else return code status 'not found'.

------------------------------

### • Change author info

Define id of author for updating and his new name, and service will edit author,
if he exists, else return code status 'not found'.

#### NOTE: New name must satisfy the same constraints that in request of creating author.

------------------------------

### • Get author's books

Define author's id and service will find all books, which contains
this author in it list of authors.

#### NOTE: If there is no given author in library, service will return empty list.

------------------------------

### • Add book

Define name and id of authors of new book and service will return its id.

#### NOTE: The book may not have authors, but if you specify them, each id of each specified author must be stored in the service.

------------------------------

### • Get book info

Define id of required book and service will return info about it (name, authors),
if book with given id exists, else return code status 'not found'.

------------------------------

### • Update book

Define id of book for updating and new info about him,
and service will edit book, if he exists, else return code status 'not found'.


#### NOTE: The same restrictions apply to the IDs of book authors as in the add book request.

------------------------------

## The "Outbox" pattern

The service implements the "Outbox" pattern, which allows you to send messages
asynchronously to other services. There are currently only 2 kind handlers for outbox: authorOutboxHandler
and bookOutboxHandler which just asynchronously send a POST request with the AuthorID or BookID to
OUTBOX_AUTHOR_SEND_URL or OUTBOX_BOOK_SEND_URL, respectively. But it's easy to add new kind handlers that will
perform more complex logic.


## How to run service

### 1) Run branch generate in MakeFile
### 2) Set environment variables

#### For gRPC:
1) GRPC_PORT
2) GRPC_GATEWAY_PORT

#### For database
1) POSTGRES_HOST
2) POSTGRES_PORT
3) POSTGRES_DB
4) POSTGRES_USER
5) POSTGRES_PASSWORD
6) POSTGRES_MAX_CONN

#### For Outbox (if you want to it work)
1) OUTBOX_ENABLED = true
2) OUTBOX_WORKERS
3) OUTBOX_BATCH_SIZE
4) OUTBOX_WAIT_TIME_MS
5) OUTBOX_IN_PROGRESS_TTL_MS
6) OUTBOX_AUTHOR_SEND_URL
7) OUTBOX_BOOK_SEND_URL
8) OUTBOX_ATTEMPTS_RETRY

#### For logging layers (if you do not specify the value of any variable, it will be true)
1) LOG_CONTROLLER_ENABLED
2) LOG_TRANSACTOR_ENABLED
3) LOG_USECASE_ENABLED
4) LOG_DB_REPO_ENABLED
5) LOG_OUTBOX_WORKER_ENABLED



### 3) Run bin file 'library' in local bin directory. 
