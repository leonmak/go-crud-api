DROP TABLE IF EXISTS users CASCADE;

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "citext";

CREATE TABLE users
(
  -- uuid for dynamic tables for easier sharding
  id                    uuid primary key default uuid_generate_v4(),
  email                 citext not null unique,
  display_name          text not null,
  image_url             text,
  country_code          char(2),
  reputation            int default 0,
  auth_type             text,
  created_at            timestamp default timezone('utc', now()),
  CHECK (length(display_name) <= 42),
  CHECK (length(image_url) <= 256)
);

CREATE UNIQUE INDEX users_unique_lower_email_idx ON users (lower(email));

BEGIN;
DO $$
  DECLARE
    vUserId uuid := 'eab30e15-fded-46fc-93f4-af0cb2a0ebd8';
  BEGIN
    INSERT INTO users (id, email, display_name, country_code)
    VALUES (vUserId, 'leon.mak@u.nus.edu', 'leon', 'US');
  END
$$;
COMMIT;
