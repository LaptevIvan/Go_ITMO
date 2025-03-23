# Library service

It is service with gRPC and REST API that allow make request:

### creating authors 

User have to defined name of new author and service will return its id

### reading info about authors

User have to defined id of required author and service will return info about him, 
if author with given id exists, else return code status 'not found'

### updating authors

User have to defined id of author for updating and new info about him, 
and service will edit author, if he exists, else return code status 'not found'



### getting books of author

User have to defined id of author and service will find all books, which contains
this author in it list of authors 


### creating books

User have to defined name and authors of new book and service will return its id

### reading info about books

User have to defined id of required book and service will return info about it,
if book with given id exists, else return code status 'not found'

### updating books

User have to defined id of book for updating and new info about him,
and service will edit book, if he exists, else return code status 'not found'


