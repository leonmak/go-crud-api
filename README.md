# Go CRUD 

Simple REST API for Golang.

## Getting Started
Install go dependencies:
```
dep ensure
```

Run sql scripts:
```bash
for i in `ls sql`; do 
    psql -h localhost -d groupbuy -p 5432 -f ./sql/$i
done
```

Routes:
- `/deals`
    - GET 
        - `after`, `before`, `before_floor`
            - iso8601 format, e.g. `2011-10-05T14:48:00.000Z`
        - `lat`, `lng`
            - up to 64 digits, e.g. `1.3521`
        - `show_inactive`
            - bool, e.g. `true`/`false`
        - `radius_km`
            - int, or default 10
- `/deal/{id}`
    - GET
    - POST
    - PUT
    - DELETE 
- ``