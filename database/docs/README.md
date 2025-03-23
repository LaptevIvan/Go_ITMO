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
    3) author_id (id of it author)
    4) created_at
    5) updated_at


## The service accepts the following requests

----------------------------

### • Register author

In body of request define name of new author and service will return its id.

#### NOTE: Constraints for name:

1) name must satisfy the regular expression ^[A-Za-z0-9]+( [A-Za-z0-9]+)*$
2) name's length must be in [1; 512] symbols.


-----------------------------

### • Get author info

Define id of required author and service will return info about him (his name),
if author with given id exists, else return code status 'not found'.

-----------------------------

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

--------------------------

### • Update book

Define id of book for updating and new info about him,
and service will edit book, if he exists, else return code status 'not found'.

#### NOTE: The same restrictions apply to the IDs of book authors as in the add book request.

## How to run service

Run branch generate in MakeFile, set environment variables GRPC_PORT, GRPC_GATEWAY_PORT,
POSTGRES_HOST, POSTGRES_PORT, POSTGRES_DB, POSTGRES_USER, POSTGRES_PASSWORD, POSTGRES_MAX_CONN and
run bin file 'library' in local bin directory. 
