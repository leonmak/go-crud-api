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
        - `search_text`
            - string, fuzzy text search on `title` column
        - `city_id`
            - serial, reference cities(id)
        - `poster_id` (optional)
            - string
        - `category_id` (optional)
            - serial
        - `after`, `before` (optional)
            - iso8601 format, e.g. `2011-10-05T14:48:00.000Z`
            - must use both `after` & `before` if used (for paginating)  
        - `show_inactive` (optional)
            - bool, e.g. `true`(show all) / `false` (default, show `inactive_at` is null)
        - `lat`, `lng` (optional)
            - float64, up to 64 digits, e.g. `1.3521`
        - `radius_km` (optional)
            - int, or default 10
            - requires `lat` & `lng`
- `/deal`
    - POST
        - accepts JSON payload with keys: 
        `title`,`description`,`thumbnailId`,`latitude`,`longitude`,`locationText`,
        `expectedPrice`,`categoryId`,`posterId`,`cityId`
        - returns uuid string on successful
- `/deal/{id}`
    - GET
    - PUT
    - DELETE

- `/deal/{id}/members`
- `/deal/{id}/comments`
- `/deal/{id}/images`
- `/categories/` 
- `/user/{id}`