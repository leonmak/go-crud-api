# TODOs

## Security
- CSRF

## Database
- add indices
    ```
    suggestions active_from < $1 AND $1 < inactive_by
    
    ```

## Automate
- Procedure for Changing object names is complicated :( 
    - Refactor name in .sql file
    - Refactor mapping key name & json tag in struct
    - Find other references to camelcase version in mapping keys
    - In client side update json key
    
